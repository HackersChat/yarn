// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/angelofallars/htmx-go"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.mills.io/sessions"
)

const (
	// MaxFailedLogins is the default maximum tolerable number of failed login attempts
	// TODO: Make this configurable via Pod Settings
	MaxFailedLogins = 3 // By default 3 failed login attempts per 5 minutes
)

// LoginHandler ...
func (s *Server) LoginHandler() httprouter.Handle {
	// #239: Throttle failed login attempts and lock user  account.
	failures := NewTTLCache(5 * time.Minute)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			if ctx.Authenticated {
				if htmx.IsHTMX(r) {
					htmx.NewResponse().
						Location("/").
						Write(w)
				} else {
					http.Redirect(w, r, "/", http.StatusFound)
				}
				return
			}
			ctx.Title = s.tr(ctx, "LoginTitle")
			ctx.Referer = r.Referer()
			s.render("login", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))
		password := r.FormValue("password")
		rememberme := r.FormValue("rememberme") == "on"

		// Error: no username or password provided
		if username == "" || password == "" {
			if htmx.IsHTMX(r) {
				htmx.NewResponse().
					Location("/login").
					Write(w)
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
			return
		}

		// Lookup user
		user, err := s.db.GetUser(username)
		if err != nil {
			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorInvalidUsername")
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
			s.render("error", r, w, ctx)
			return
		}

		// #239: Throttle failed login attempts and lock user  account.
		if failures.Get(user.Username) > MaxFailedLogins {
			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorMaxFailedLogins")
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusTooManyRequests)
			}
			s.render("error", r, w, ctx)
			return
		}

		// Validate cleartext password against KDF hash
		_, err = s.pm.Passwd([]byte(password), user.PasswordHash)
		if err != nil {
			// #239: Throttle failed login attempts and lock user  account.
			failed := failures.Inc(user.Username)
			time.Sleep(time.Duration(IntPow(2, failed)) * time.Second)

			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorInvalidPassword")
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
			s.render("error", r, w, ctx)
			return
		}

		// Check if password needs to be upgraded.
		if !s.pm.IsPreferred(user.PasswordHash) {
			log.Infof("Upgrading password for %s", username)

			hash, _ := s.pm.Passwd([]byte(password), nil)
			user.PasswordHash = hash

			// Save upgraded user password
			if err := s.db.SetUser(username, user); err != nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorSetUser"))
				return
			}
		}

		// #239: Throttle failed login attempts and lock user  account.
		failures.Reset(user.Username)

		// Lookup session
		sess := sessions.FromRequest(r)
		if sess == nil {
			if htmx.IsHTMX(r) {
				htmx.NewResponse().
					Location("/login").
					Write(w)
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
			return
		}

		// Authorize session
		_ = sess.Set("username", username)

		// Persist session?
		if rememberme {
			_ = sess.Set("persist", "1")
		}

		pageRedirection := ""
		switch user.StartPage {
		case "/", "/discover", "/mentions":
			pageRedirection = user.StartPage
		default:
			pageRedirection = r.FormValue("referer")
		}

		if htmx.IsHTMX(r) {
			htmx.NewResponse().
				Location(pageRedirection).
				Write(w)
		} else {
			http.Redirect(w, r, pageRedirection, http.StatusFound)
		}
	}
}

// LoginEmailHandler ...
func (s *Server) LoginEmailHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			if ctx.Authenticated {
				if htmx.IsHTMX(r) {
					htmx.NewResponse().
						Location("/").
						Write(w)
				} else {
					http.Redirect(w, r, "/", http.StatusFound)
				}
				return
			}
			ctx.Title = s.tr(ctx, "LoginEmailTitle")
			s.render("loginEmail", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		recovery := fmt.Sprintf("email:%s", FastHashString(email))

		if err := ValidateUsername(username); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Username validation failed: %s", err)
			return
		}

		// Check if user exist
		if !s.db.HasUser(username) {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorUserNotFound"))
			return
		}

		// Get user object from DB
		user, err := s.db.GetUser(username)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
			return
		}

		if recovery != user.Recovery {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorUserRecovery"))
			return
		}

		// Create magic link with a short expiry time of ~10m (hard-coded)

		// TODO: Make the expiry time configurable?
		expiryTime := time.Now().Add(30 * time.Minute).Unix()

		// Create magic link
		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.MapClaims{"username": username, "expiresAt": expiryTime},
		)
		tokenString, err := token.SignedString([]byte(s.config.MagicLinkSecret))
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, err.Error())
			return
		}
		parts := strings.SplitN(tokenString, ".", 3)
		tokenCache.Inc(parts[2])

		if err := SendMagicLinkAuthEmail(s.config, user, email, tokenString); err != nil {
			log.WithError(err).Errorf("unable to send magic-link-auth email to %s", user.Username)
			s.renderError(r, w, http.StatusInternalServerError,
				"Error sending magic-link-auth email! Please try again.")
			return
		}

		// Show success msg
		s.renderSuccess(r, w, s.tr(ctx, "MsgMagicLinkAuthEmailSent"))
	}
}

// MagicLinkAuthHandler ...
func (s *Server) MagicLinkAuthHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		// Get token from query string
		tokens, ok := r.URL.Query()["token"]

		// Check if valid token
		if !ok || len(tokens[0]) < 1 {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}

		magicLinkAuthToken := tokens[0]

		// Check if token is valid
		token, err := jwt.Parse(magicLinkAuthToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(s.config.MagicLinkSecret), nil
		})
		if err != nil {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
		if tokenCache.Get(token.Signature) == 0 {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
		tokenCache.Dec(token.Signature)

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			var username = fmt.Sprintf("%v", claims["username"])
			var expiresAt int = int(claims["expiresAt"].(float64))

			now := time.Now()
			secs := now.Unix()

			// Check token expiry
			if secs > int64(expiresAt) {
				s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorTokenExpired"))
				return
			}

			user, err := s.db.GetUser(username)
			if err != nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
				return
			}

			// Lookup session
			sess := sessions.FromRequest(r)
			if sess == nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			// Authorize and persist session
			_ = sess.Set("username", user.Username)
			_ = sess.Set("persist", "1")

			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
	}
}

// LogoutHandler ...
func (s *Server) LogoutHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.sm.Delete(w, r)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
