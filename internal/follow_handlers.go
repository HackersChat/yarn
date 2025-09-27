// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"net/http"
	"strings"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// FollowHandler ...
func (s *Server) FollowHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		uri := r.FormValue("url")

		if uri == "" {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorNoFeed"))
			return
		}

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}

		trdata := map[string]interface{}{}
		trdata["URI"] = uri

		if err := user.FollowAndValidate(s.config, uri); err != nil {
			trdata["Error"] = err.Error()
			s.renderError(r, w, http.StatusInternalServerError,
				s.tr(ctx, "ErrorFollowAndValidate", trdata))
			return
		}

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			trdata["Error"] = err.Error()
			s.renderError(r, w, http.StatusInternalServerError,
				s.tr(ctx, "ErrorSetUser", trdata))
			return
		}

		if htmx.IsHTMX(r) {
			htmx.NewResponse().
				Location(r.Referer()).
				Write(w)
		} else {
			s.renderSuccess(r, w, s.tr(ctx, "MsgFollowUserSuccess", trdata))
		}
	}
}

// UnfollowHandler ...
func (s *Server) UnfollowHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := strings.TrimSpace(r.FormValue("nick"))

		if nick == "" {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorNoNick"))
			return
		}

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}
		trdata := map[string]interface{}{}
		url, ok := user.Following[nick]
		trdata["Nick"] = nick
		trdata["URL"] = url
		if !ok {
			s.renderError(r, w, http.StatusNotFound, s.tr(ctx, "ErrorNoFeedByNick", trdata))
			return
		}

		if err := ctx.User.Unfollow(s.config, nick); err != nil {
			log.WithError(err).Errorf("error unfollowing %s", nick)
			trdata["Error"] = err.Error()
			s.renderError(r, w, http.StatusInternalServerError,
				s.tr(ctx, "ErrorUnfollowingFeed", trdata))
			return
		}

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError,
				s.tr(ctx, "ErrorUnfollowingFeed", trdata))
			return
		}

		if htmx.IsHTMX(r) {
			htmx.NewResponse().
				Location(r.Referer()).
				Write(w)
		} else {
			s.renderSuccess(r, w, s.tr(ctx, "MsgUnfollowSuccess", trdata))
		}
	}
}
