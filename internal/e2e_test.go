// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var (
	bind string
	srv  *Server

	registeredUserCounter int
)

func makeURL() string {
	return fmt.Sprintf("http://%s", bind)
}

func registerUser(t *testing.T) (username, password string) {
	srv.config.OpenRegistrations = true
	defer func() {
		srv.config.OpenRegistrations = false
	}()

	registeredUserCounter++
	username = fmt.Sprintf("user%d", registeredUserCounter)
	password = "hunter2"

	res := e(t).GET("/register").
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")
	csrfTokenCookie := res.Cookie("csrf_token")
	csrfToken := res.HTMLForm().Field("csrf_token")

	e(t).POST("/register").
		WithForm(map[string]string{
			"csrf_token": csrfToken,
			"username":   username,
			"password":   password,
			"agree":      "on",
		}).
		WithResponseCookie(csrfTokenCookie).
		Expect().
		Status(http.StatusFound).
		NoCookie("yarnd_token").
		Header("Location").Equal("/login")

	return username, password
}

func login(t *testing.T, username, password string) *Response {
	res := e(t).GET("/login").
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")
	csrfTokenCookie := res.Cookie("csrf_token")
	csrfToken := res.HTMLForm().Field("csrf_token")

	return e(t).POST("/login").
		WithForm(map[string]string{
			"csrf_token": csrfToken,
			"username":   username,
			"password":   password,
		}).
		WithResponseCookie(csrfTokenCookie).
		Expect()
}

func loginUser(t *testing.T, username, password string) *httpexpect.Cookie {
	res := login(t, username, password).Status(http.StatusFound)
	res.Header("Location").Equal("/")
	return res.Cookie("yarnd_token")
}

func TestInfo(t *testing.T) {
	e(t).GET("/info").
		Expect().
		Status(http.StatusOK).
		Body().Contains("yarnd")
}

func TestCookies_whenRequestingStartPageAnonymously_thenSendNoYarndSessionCookie(t *testing.T) {
	e(t).GET("/").
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")
}

func TestCookies_whenRequestingStartPageLoggedIn_thenSendNoYarndSessionCookie_sinceThereIsAlreadyAnActiveSession(t *testing.T) {
	username, password := registerUser(t)
	yarndTokenCookie := loginUser(t, username, password)

	e(t).GET("/").
		WithResponseCookie(yarndTokenCookie).
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")

	assertSession(t, yarndTokenCookie).
		Username(username).
		NoCaptchaText()
}

func TestCookies_whenRequestingCaptchaAnonymously_thenSendYarndSessionCookieForCaptcha(t *testing.T) {
	yarndTokenCookie := e(t).GET("/_captcha").
		Expect().
		Status(http.StatusOK).
		Cookie("yarnd_token")

	assertSession(t, yarndTokenCookie).
		NoUsername().
		HasCaptchaText()
}

func TestCookies_whenRequestingCaptchaLoggedIn_thenSendNoYarndSessionCookie_sinceThereIsAlreadyAnActiveSession(t *testing.T) {
	username, password := registerUser(t)
	yarndTokenCookie := loginUser(t, username, password)

	e(t).GET("/_captcha").
		WithResponseCookie(yarndTokenCookie).
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")

	assertSession(t, yarndTokenCookie).
		Username(username).
		HasCaptchaText()
}

func TestCookies_whenSubmittingSupportFormAnonymously_thenClearYarndSessionCookieForCaptcha(t *testing.T) {
	res := e(t).GET("/support").
		Expect().
		Status(http.StatusOK)
	csrfTokenCookie := res.Cookie("csrf_token")
	csrfToken := res.HTMLForm().Field("csrf_token")

	yarndTokenCookie := e(t).GET("/_captcha").
		Expect().
		Status(http.StatusOK).
		Cookie("yarnd_token")

	captcha := assertSession(t, yarndTokenCookie).
		NoUsername().
		CaptchaText()

	e(t).POST("/support").
		WithResponseCookie(csrfTokenCookie).
		WithResponseCookie(yarndTokenCookie).
		WithForm(map[string]string{
			"csrf_token":   csrfToken,
			"name":         "John Doe",
			"email":        "john.doe@example.com",
			"subject":      "Please get back to me",
			"message":      "I would like to donate some money, please send me your bank account details.",
			"captchaInput": captcha,
		}).
		Expect().
		Status(http.StatusOK).
		ClearCookie("yarnd_token")

	assertNoSession(t, yarndTokenCookie)
}

func TestCookies_whenSubmittingSupportFormLoggedIn_thenSendNoYarndSessionCookie_sinceThereIsStillAnActiveSession(t *testing.T) {
	username, password := registerUser(t)
	yarndTokenCookie := loginUser(t, username, password)

	res := e(t).GET("/support").
		WithResponseCookie(yarndTokenCookie).
		Expect().
		Status(http.StatusOK)
	csrfTokenCookie := res.Cookie("csrf_token")
	csrfToken := res.HTMLForm().Field("csrf_token")

	e(t).GET("/_captcha").
		WithResponseCookie(yarndTokenCookie).
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")

	captcha := assertSession(t, yarndTokenCookie).
		Username(username).
		CaptchaText()

	e(t).POST("/support").
		WithResponseCookie(csrfTokenCookie).
		WithResponseCookie(yarndTokenCookie).
		WithForm(map[string]string{
			"csrf_token":   csrfToken,
			"name":         "John Doe",
			"email":        "john.doe@example.com",
			"subject":      "Please get back to me",
			"message":      "I would like to donate some money, please send me your bank account details.",
			"captchaInput": captcha,
		}).
		Expect().
		Status(http.StatusOK).
		NoCookie("yarnd_token")

	assertSession(t, yarndTokenCookie).
		Username(username).
		NoCaptchaText()
}

func TestCookies_whenLoggingInSucceeds_thenSendYarndSessionCookie(t *testing.T) {
	// First a user needs to be registered
	username, password := registerUser(t)

	// Now the actual login must succeed
	yarndTokenCookie := loginUser(t, username, password)
	yarndTokenCookie.Value().NotEmpty()
}

func TestCookies_whenLoggingInFails_thenSendNoYarndSessionCookie(t *testing.T) {
	login(t, "username", "hunter2").
		Status(http.StatusUnauthorized).
		NoCookie("yarnd_token").
		Body().Contains("Invalid username! Hint: Register an account?")
}

func TestRegister_whenRegistrationDisabledInConfig_thenRejectRegistrationRequests(t *testing.T) {
	res := e(t).GET("/register").
		Expect().
		Status(http.StatusFound).
		NoCookie("yarnd_token")
	csrfTokenCookie := res.Cookie("csrf_token")
	res.HTMLForm().NoField("csrf_token")
	res.Body().Contains("/join")

	// Unfortunately, we can't really send a POST request right away, because
	// there is no CSRF token HTML field in the response body. So basically
	// this fails due to the CSRF token check. But, let's do it anyways.
	e(t).POST("/register").
		WithForm(map[string]string{
			"username": "reg-disabled-and-wrong-csrf-token",
			"password": "hunter2",
			"agree":    "on",
		}).
		WithResponseCookie(csrfTokenCookie).
		Expect().
		Status(http.StatusBadRequest).
		NoCookie("yarnd_token").
		Body().Equal("Bad Request\n")

	// However, we can try to get such a token from another page and…
	res = e(t).GET("/login").
		Expect().
		Status(http.StatusOK)
	csrfTokenCookie = res.Cookie("csrf_token")
	csrfToken := res.HTMLForm().Field("csrf_token")

	// …just submit that with the matching cookie in the registration request.
	e(t).POST("/register").
		WithForm(map[string]string{
			"csrf_token": csrfToken,
			"username":   "reg-disabled",
			"password":   "hunter2",
			"agree":      "on",
		}).
		WithResponseCookie(csrfTokenCookie).
		Expect().
		Status(http.StatusForbidden).
		NoCookie("yarnd_token").
		Body().Contains("Open Registrations are disabled on this pod.")

	// Login with that user must fail.
	login(t, "reg-disabled", "hunter2").
		Status(http.StatusUnauthorized).
		Body().Contains("Invalid username! Hint: Register an account?")
}

func TestRegister_whenRegistrationEnabled_thenAcceptRegistrationRequests(t *testing.T) {
	username, password := registerUser(t)

	// verify that the new user can log in
	yarndTokenCookie := loginUser(t, username, password)
	yarndTokenCookie.Value().NotEmpty()
}

func TestMain(m *testing.M) {
	testDir, err := os.MkdirTemp("", "*-yarn-e2e-test")
	if err != nil {
		log.WithError(err).Error("error creating temporary test directory")
		os.Exit(-1)
	}
	defer os.RemoveAll(testDir)

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		log.WithError(err).Error("error finding free test port")
		os.Exit(-1)
	}
	bind = listener.Addr().String()

	testStore := fmt.Sprintf("bitcask://%s/yarn.db", testDir)

	server, err := NewServer(
		bind,
		WithData(testDir),
		WithBaseURL(makeURL()),
		WithStore(testStore),
		WithCookieSecret(GenerateRandomToken()),
		WithMagicLinkSecret(GenerateRandomToken()),
		WithAPISigningKey(GenerateRandomToken()),
	)
	if err != nil {
		log.WithError(err).Error("error starting test server")
		os.Exit(-1)
	}
	srv = server

	var eg errgroup.Group

	eg.Go(func() error {
		return server.Run()
	})

	// Give time for the server to start up ...
	time.Sleep(time.Second * 3)

	os.Exit(m.Run())

	server.Shutdown(context.Background())

	if err := eg.Wait(); err != nil {
		log.WithError(err).Error("error running test server")
	}
}
