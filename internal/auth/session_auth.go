// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package auth

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"go.mills.io/sessions"
)

// SessionAuth ...
type SessionAuth struct {
	RedirectURL string
}

// NewSessionAuth ...
func NewSessionAuth(redirectURL string) Auther {
	return &SessionAuth{RedirectURL: redirectURL}
}

// MustAuth returns the next handler in the chain if the session has already been authenticated
// otherwise redirects to the login page.
func (sa *SessionAuth) MustAuth(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if sess := sessions.FromRequest(r); sess != nil {
			if _, ok := sess.Get("username"); ok {
				next(w, r, p)
				return
			}
		}

		http.Redirect(w, r, sa.RedirectURL, http.StatusFound)
	}
}
