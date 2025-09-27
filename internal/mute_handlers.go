// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// MuteHandler ...
func (s *Server) MuteHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := strings.TrimSpace(r.FormValue("nick"))
		url := NormalizeURL(r.FormValue("url"))
		hash := p.ByName("hash")

		if hash == "" && (nick == "" || url == "") {
			s.renderError(r, w, http.StatusBadRequest,
				"At least nick + url or hash must be specified")
			return
		}

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
			return
		}

		if nick != "" && url != "" {
			user.Mute(nick, NormalizeURL(url))
		} else if hash != "" {
			user.Mute(fmt.Sprintf("#%s", hash), hash)
		}

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError,
				"Error muting feed %s: %s", nick, url)
			return
		}

		if hash != "" {
			s.renderSuccess(r, w, "Successfully muted Twt (and its replies) %s", hash)
		} else {
			s.renderSuccess(r, w, "Successfully muted %s: %s", nick, url)
		}
	}
}

// MutedHandler ...
func (s *Server) MutedHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		s.render("muted", r, w, ctx)
	}
}

// UnmuteHandler ...
func (s *Server) UnmuteHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := strings.TrimSpace(r.FormValue("nick"))
		hash := p.ByName("hash")

		if nick == "" && hash == "" {
			s.renderError(r, w, http.StatusBadRequest, "No nick or hash specified to unmute")
			return
		}

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}

		if nick != "" {
			user.Unmute(nick)
		} else if hash != "" {
			user.Unmute(fmt.Sprintf("#%s", hash))
		}

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, "Error unmuting feed %s", nick)
			return
		}

		if hash != "" {
			s.renderSuccess(r, w, "Successfully unmuted Twt (and its replies) %s", hash)
		} else {
			s.renderSuccess(r, w, "Successfully unmuted %s", nick)
		}
	}
}
