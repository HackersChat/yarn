// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	"github.com/rickb777/accept"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// const keysDir = "keys"

// ProfileHandler ...
func (s *Server) ProfileHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		nick := NormalizeUsername(p.ByName("nick"))
		if nick == "" {
			if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" {
				s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorNoUser"))
			} else {
				http.Error(w, "No User", http.StatusBadRequest)
			}
			return
		}

		var profile types.Profile

		if s.db.HasUser(nick) {
			user, err := s.db.GetUser(nick)
			if err != nil {
				if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" {
					log.WithError(err).Errorf("error loading user object for %s", nick)
					s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingProfile"))
				} else {
					http.Error(w, "Error Loading Profile", http.StatusInternalServerError)
				}
				return
			}
			profile = user.Profile(s.config.BaseURL, ctx.User)
			profile.FollowedBy = ctx.User.Follows(profile.URI)
			profile.LastSeenAt = s.followers.LastSeenFor(profile.URI, ctx.User.Username)
		} else {
			if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" {
				ctx.Error = true
				ctx.Message = s.tr(ctx, "ErrorUserOrFeedNotFound")
				w.WriteHeader(http.StatusNotFound)
				s.render("404", r, w, ctx)
			} else {
				http.Error(w, "Feed Not Found", http.StatusNotFound)
			}
			return
		}

		ctx.Profile = profile

		ctx.Alternatives = append(ctx.Alternatives, Alternatives{
			Alternative{
				Type:  "text/plain",
				Title: fmt.Sprintf("%s's Twtxt Feed", profile.Nick),
				URL:   profile.URI,
			},
		}...)

		opts := &QueryOptions{
			Exclude: ctx.User.MutedList(),
		}

		// Replace in-memory paging with database-driven paging.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		opts.Limit = s.config.TwtsPerPage
		opts.Offset = (page - 1) * opts.Limit

		twts, total, err := s.cache.GetByURL(profile.URI, opts)
		if err != nil {
			if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" {
				log.WithError(err).Error("error loading twts by URL")
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingTimeline"))
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if len(twts) > 0 {
			profile.LastPostedAt = twts[0].Created()
		}

		// Compute and assign the pager.
		pager := NewPager(page, opts.Limit, total)
		ctx.Pager = pager
		ctx.Twts = twts

		ctx.Title = fmt.Sprintf("%s's Profile: %s", profile.Nick, profile.Description)
		if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" || htmx.IsHTMX(r) {
			s.render("profile", r, w, ctx)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(ctx.Profile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(data)
	}
}
