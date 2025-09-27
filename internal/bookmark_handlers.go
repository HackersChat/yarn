// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"net/http"
	"sort"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// BookmarkHandler ...
func (s *Server) BookmarkHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		hash := p.ByName("hash")
		if hash == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		opts := &QueryOptions{
			Exclude: ctx.User.MutedList(),
		}

		twt, _ := s.cache.Lookup(hash, opts)
		if twt.IsZero() {
			ctx.Error = true
			ctx.Message = "No matching twt found!"
			w.WriteHeader(http.StatusNotFound)
			s.render("404", r, w, ctx)
			return
		}

		ctx.User.Bookmark(twt.Hash())

		if err := s.db.SetUser(ctx.Username, ctx.User); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, "Error updating user")
			return
		}

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
			return
		}

		if htmx.IsHTMX(r) {
			ctx.Twts = types.Twts{twt}
			s.render("bookmarkBtn", r, w, ctx)
		} else {
			s.renderSuccess(r, w, "Successfully updated bookmarks")
		}
	}
}

// BookmarksHandler ...
func (s *Server) BookmarksHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := NormalizeUsername(p.ByName("nick"))

		var twts types.Twts

		opts := &QueryOptions{
			Exclude: ctx.User.MutedList(),
		}

		getTwts := func(hashes []string) (twts types.Twts) {
			for _, hash := range hashes {
				if twt, ok := s.cache.Lookup(hash, opts); ok {
					twts = append(twts, twt)
				}
			}
			return
		}

		if s.db.HasUser(nick) {
			user, err := s.db.GetUser(nick)
			if err != nil {
				log.WithError(err).Errorf("error loading user object for %s", nick)
				s.renderError(r, w, http.StatusInternalServerError, "Error loading profile")
				return
			}

			// Check bookmarks visibility.
			if !user.IsBookmarksPubliclyVisible && !ctx.User.Is(user.URL) {
				w.WriteHeader(http.StatusUnauthorized)
				s.render("401", r, w, ctx)
				return
			}
			// Get the tweets for the bookmarked hashes.
			twts = getTwts(MapKeys(user.Bookmarks))
			// Sort the bookmarks (ascending or descending order as appropriate).
			sort.Sort(twts)
		} else {
			ctx.Error = true
			ctx.Message = "User Not Found"
			w.WriteHeader(http.StatusNotFound)
			s.render("404", r, w, ctx)
			return
		}

		// Perform manual in-memory paging similar to conversation_handler.go.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		limit := s.config.TwtsPerPage
		total := len(twts)
		offset := (page - 1) * limit

		var pagedTwts types.Twts
		if offset < total {
			end := offset + limit
			if end > total {
				end = total
			}
			pagedTwts = twts[offset:end]
		}

		// Set the paged tweets and create a Pager.
		ctx.Twts = pagedTwts
		ctx.Pager = NewPager(page, limit, total)

		s.render("bookmarks", r, w, ctx)
	}
}
