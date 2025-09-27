// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// WhoFollowsHandler ...
func (s *Server) WhoFollowsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		ctype := "html"
		if r.Header.Get("Accept") == "application/json" {
			ctype = "json"
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			if ctype == "html" {
				s.renderError(r, w, http.StatusBadRequest, "No token supplied")
			} else {
				http.Error(w, "Bad Request", http.StatusBadRequest)
			}
			return
		}

		uri := tokenCache.GetString(token)
		if uri == "" {
			if ctype == "html" {
				s.renderError(r, w, http.StatusNotFound, "Token expired or invalid")
			} else {
				http.Error(w, "Token Not Found", http.StatusNotFound)
			}
			return
		}
		tokenCache.Del(token)

		users, err := s.db.GetAllUsers()
		if err != nil {
			log.WithError(err).Error("unable to get all users from database")
			if ctype == "html" {
				s.renderError(r, w, http.StatusInternalServerError, "Error computing followers list")
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		var (
			nick      string
			followers types.Followers
		)

		for _, user := range users {
			userURL := URLForUser(s.config.BaseURL, user.Username)

			if !user.IsFollowingPubliclyVisible && !ctx.User.Is(userURL) {
				continue
			}

			if user.Follows(uri) {
				followers = append(followers, &types.Follower{
					Nick:       user.Username,
					URI:        userURL,
					LastSeenAt: time.Now(),
				})
				if nick == "" {
					nick = user.sources[uri]
				}
			}
		}
		if nick == "" {
			nick = "unknown"
		}

		ctx.Profile = types.Profile{
			Type: "External",

			Nick:        nick,
			Description: "",
			URI:         uri,

			Follows:    true,
			FollowedBy: true,
			Muted:      false,

			Followers:  followers,
			NFollowers: len(followers),
		}

		if ctype == "json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode(followers.AsMap()); err != nil {
				log.WithError(err).Error("error encoding user for display")
				http.Error(w, "Bad Request", http.StatusBadRequest)
			}

			return
		}

		ctx.Title = fmt.Sprintf("Followers for @<%s %s>", nick, uri)
		s.render("followers", r, w, ctx)
	}
}
