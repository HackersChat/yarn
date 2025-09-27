// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package auth

import (
	"net/http"
	"net/http/httputil"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.mills.io/sessions"
)

// ProxyAuth ...
type ProxyAuth struct {
	Header      string
	UserCreator UserCreator
}

// NewProxyAuth constructs a new proxy auther that uses a HTTP Header as the
// primary mechanism for authentication with the expetation that an external
// authentication system (like SSO or a Reverse Proxy) is already authentication
func NewProxyAuth(header string, userCreator UserCreator) Auther {
	return &ProxyAuth{Header: header, UserCreator: userCreator}
}

// MustAuth returns the next handler in the chain if the configured HTTP header
// exists in the request (assumes trust from the proxy), storing the username
// in the session.
func (pa *ProxyAuth) MustAuth(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debugf("req:\n%q\n", dump)

		if username := r.Header.Get(pa.Header); username != "" {
			log.Debugf("Usernaem: %q", username)
			if err := pa.UserCreator.CreateUser(username, r); err != nil {
				log.WithError(err).Errorf("error checking or creating user %q", username)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if sess := sessions.FromRequest(r); sess != nil {
				// Authorize session
				_ = sess.Set("username", username)
			}
			next(w, r, p)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}
