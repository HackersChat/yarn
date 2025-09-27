// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhoFollowsResource(t *testing.T) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err, "listening to determine free port failed")
	bind := listener.Addr().String()
	require.NoError(t, listener.Close(), "closing listener failed")

	url := func(partialURL string, args ...interface{}) string {
		if len(args) > 0 {
			partialURL = fmt.Sprintf(partialURL, args...)
		}
		return fmt.Sprintf("http://%s%s", bind, partialURL)
	}

	cfg := NewConfig()
	cfg.BaseURL = url("/")

	db, err := NewStore(cfg.Store)
	require.NoError(t, err, "initializing store failed")
	defer func() {
		assert.NoError(t, db.Close(), "closing store failed")
	}()

	router := NewRouter()
	server := &Server{
		config: cfg,
		router: router,
		server: &http.Server{Addr: bind, Handler: router},
		db:     db,
	}
	router.GET("/whoFollows", server.WhoFollowsHandler())
	defer func() {
		assert.NoError(t, server.server.Shutdown(context.Background()), "shutting down server failed")
	}()

	followingThisFeed := map[string]string{"this": "https://example.com/twtxt.txt"}
	followingOtherFeed := map[string]string{"other": "https://example.org/other.txt"}

	registerUser := func(username string, publiclyFollowing bool, following map[string]string) *User {
		user := NewUser()
		user.Username = username
		user.URL = url("/user/%s/twtxt.txt", username)
		user.IsFollowingPubliclyVisible = publiclyFollowing
		for nick, url := range following {
			user.Follow(nick, url)
		}
		db.SetUser(username, user)
		return user
	}

	publiclyFollowing1 := registerUser("eugen", true, followingThisFeed)
	publiclyFollowing2 := registerUser("hans", true, followingThisFeed)
	privatelyFollowing := registerUser("hugo", false, followingThisFeed)
	publiclyNotFollowing := registerUser("kate", true, followingOtherFeed)
	privatelyNotFollowing := registerUser("tanja", false, followingOtherFeed)
	validToken := GenerateWhoFollowsToken("https://example.com/twtxt.txt")

	for _, testCase := range []struct {
		name                string
		url                 string
		expectedStatusCode  int
		expectedContentType string
		expectedRawBody     string
		expectedJSONBody    map[string]string
	}{
		{
			name:                "when missing token then return Bad Request",
			url:                 url("/whoFollows"),
			expectedStatusCode:  400,
			expectedContentType: "text/plain; charset=utf-8",
			expectedRawBody:     "Bad Request\n",
		},
		{
			name:                "when garbage token then return Bad Request",
			url:                 url("/whoFollows?token=$%9zu"), // invalid percent encoding
			expectedStatusCode:  400,
			expectedContentType: "text/plain; charset=utf-8",
			expectedRawBody:     "Bad Request\n",
		},
		{
			name:                "when unknown token then return Token Not Found",
			url:                 url("/whoFollows?token=abc"),
			expectedStatusCode:  404,
			expectedContentType: "text/plain; charset=utf-8",
			expectedRawBody:     "Token Not Found\n",
		},
		{
			name:                "when valid token then return publicly following followers",
			url:                 url("/whoFollows?token=%s", validToken),
			expectedStatusCode:  200,
			expectedContentType: "application/json",
			expectedJSONBody: map[string]string{
				publiclyFollowing1.Username: publiclyFollowing1.URL,
				publiclyFollowing2.Username: publiclyFollowing2.URL,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest("GET", testCase.url, nil)
			req.Header.Set("Accept", "application/json")

			router.ServeHTTP(res, req)

			assert.Equal(t, testCase.expectedStatusCode, res.Code, "HTTP status code mismatch")
			assert.Equal(t, testCase.expectedContentType, res.Header().Get("Content-Type"), "Content-Type header mismatch")
			actualRawBody := res.Body.String()
			if testCase.expectedRawBody != "" {
				assert.Equal(t, testCase.expectedRawBody, actualRawBody, "raw response body mismatch")
			} else {
				followers := make(map[string]string)
				if !assert.NoError(t, json.Unmarshal(res.Body.Bytes(), &followers), "unmarshalling JSON response body failed") {
					assert.Equal(t, testCase.expectedJSONBody, followers, "JSON response body mismatch")
				}
			}
			assert.NotContains(t, actualRawBody, privatelyFollowing.URL, "privately following URL should not be disclosed")
			assert.NotContains(t, actualRawBody, publiclyNotFollowing.URL, "publicly *not* following URL should not be included")
			assert.NotContains(t, actualRawBody, privatelyNotFollowing.URL, "privately *not* following URL should not be included")

			if testCase.expectedStatusCode == 200 {
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				assert.Equalf(t, 404, res.Code, "HTTP status code mismatch, token should only be usable once\nresponse body: %s", res.Body.String())
			}
		})
	}
}
