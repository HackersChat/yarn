// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// CreatePasswordResetToken generates a JWT token for password reset purposes.
// The token includes the username and an expiration time, which is currently
// set to 30 minutes from the current time. The token is signed using the
// MagicLinkSecret from the configuration. The token's signature part is
// incremented in the token cache to track its usage.
//
// Parameters:
//
//	conf - Configuration object containing the MagicLinkSecret.
//	u - User object containing the username.
//
// Returns:
//
//	A signed JWT token string if successful, or an error if token creation fails.
func CreatePasswordResetToken(conf *Config, u *User) (string, error) {
	// TODO: Make the token expiration configurable.
	expiryTime := time.Now().Add(30 * time.Minute).Unix()

	// Create magic link
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{"username": u.Username, "expiresAt": expiryTime},
	)
	tokenString, err := token.SignedString([]byte(conf.MagicLinkSecret))
	if err != nil {
		return "", fmt.Errorf("error creating password reset token: %v", err)
	}
	parts := strings.SplitN(tokenString, ".", 3)
	tokenCache.Inc(parts[2])

	return tokenString, nil
}

// ResetPasswordHandler ...
func (s *Server) ResetPasswordHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			ctx.Title = s.tr(ctx, "PageResetPasswordTitle")
			s.render("resetPassword", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		recovery := fmt.Sprintf("email:%s", FastHashString(email))

		if err := ValidateUsername(username); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Username validation failed: %v", err)
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

		//
		// Create magic link expiry time
		//

		token, err := CreatePasswordResetToken(s.config, user)
		if err != nil {
			log.WithError(err).Errorf("unable to create password reset token")
			s.renderError(r, w, http.StatusInternalServerError,
				"Error creating password reset token! Please try again.")
			return
		}

		if err := SendPasswordResetEmail(s.config, user, email, token); err != nil {
			log.WithError(err).Errorf("unable to send reset password email to %s", user.Username)
			s.renderError(r, w, http.StatusInternalServerError,
				"Error sending password reset email! Please try again.")
			return
		}

		// Show success msg
		s.renderSuccess(r, w, s.tr(ctx, "MsgUserRecoveryRequestSent"))
	}
}

// ResetPasswordMagicLinkHandler ...
func (s *Server) ResetPasswordMagicLinkHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		// Get token from query string
		tokens, ok := r.URL.Query()["token"]

		// Check if valid token
		if !ok || len(tokens[0]) < 1 {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}

		passwordResetToken := tokens[0]

		// Check if token is valid
		token, err := jwt.Parse(passwordResetToken, func(token *jwt.Token) (interface{}, error) {
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

			_, err := s.db.GetUser(username)
			if err != nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
				return
			}

			ctx.PasswordResetToken = passwordResetToken

			// Show newPassword page
			s.render("newPassword", r, w, ctx)
		} else {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
	}
}

// NewPasswordHandler ...
func (s *Server) NewPasswordHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			return
		}

		password := r.FormValue("password")
		passwordResetToken := r.FormValue("token")

		// Check if token is valid
		token, err := jwt.Parse(passwordResetToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(s.config.MagicLinkSecret), nil
		})
		if err != nil {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
		if tokenCache.Get(token.Signature) != 1 {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
		tokenCache.Del(token.Signature)

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

			// Reset password
			if password != "" {
				hash, err := s.pm.Passwd([]byte(password), nil)
				if err != nil {
					s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
					return
				}

				user.PasswordHash = hash

				// Save user
				if err := s.db.SetUser(username, user); err != nil {
					s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
					return
				}
			}

			// Show success msg
			s.renderSuccess(r, w, s.tr(ctx, "MsgPasswordResetSuccess"))
		} else {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorInvalidToken"))
			return
		}
	}
}
