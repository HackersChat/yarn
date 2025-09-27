// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.mills.io/prologic/useragent"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// ExternalHandler ...
func (s *Server) ExternalHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		uri := NormalizeURL(r.URL.Query().Get("uri"))

		if uri == "" {
			s.renderError(r, w, http.StatusNotFound, s.tr(ctx, "ErrorNoExternalFeed"))
			return
		}

		if !s.cache.HasFeed(uri) {
			if !ctx.Authenticated {
				ua := useragent.Parse(r.UserAgent())
				if ua != nil && ua.Type == useragent.Browser {
					s.renderError(r, w, http.StatusNotFound, s.tr(ctx, "ErrorNoExternalFeed"))
				} else {
					http.Error(w, "Feed Not Found", http.StatusNotFound)
				}
				return
			}
			// Only fetch an external feed (if not cached) if the request is from an authenticated user.
			s.tasks.DispatchFunc(func() error {
				s.fetcher.EnqueueFeeds(
					FetchFeedRequests{
						{URI: uri, Created: time.Now()},
					},
					s.config.fetchInterval,
				)
				return nil
			})
		}

		if twter := s.cache.GetTwter(uri); twter != nil {
			ctx.Twter = *twter
		} else {
			ctx.Twter = types.NewTwter("", uri)
		}

		opts := &QueryOptions{
			Exclude: ctx.User.MutedList(),
		}

		// Compute page, limit and offset.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		opts.Limit = s.config.TwtsPerPage
		opts.Offset = (page - 1) * opts.Limit

		// Get the paginated twts from the external feed.
		twts, total, err := s.cache.GetByURL(uri, opts)
		if err != nil {
			log.WithError(err).Error("error loading external twts")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingTimeline"))
			return
		}

		ctx.Twts = twts

		// Set up the pager using the returned total.
		ctx.Pager = NewPager(page, opts.Limit, total)

		var follows types.Follows
		for n, twter := range ctx.Twter.Follow {
			follows = append(follows, types.Follow{Nick: n, URI: twter.URI})
		}

		var links types.Links
		for _, link := range ctx.Twter.Metadata["link"] {
			tokens := strings.Split(link, " ")
			if len(tokens) >= 2 {
				n := len(tokens) - 1
				url := tokens[n]
				title := strings.Join(tokens[:n], " ")
				links = append(links, types.Link{Title: title, URL: url})
			}
		}
		sort.Sort(links)

		ctx.Profile = types.Profile{
			Type: "External",

			Nick:        ctx.Twter.Nick,
			Description: ctx.Twter.Tagline,
			Avatar:      ctx.Twter.Avatar,
			URI:         ctx.Twter.URI,

			Following:  follows,
			NFollowing: ctx.Twter.Following,
			NFollowers: ctx.Twter.Followers,
			LastSeenAt: s.followers.LastSeenFor(uri, ctx.User.Username),

			ShowFollowing: true,
			ShowFollowers: true,

			Follows:    ctx.User.Follows(uri),
			FollowedBy: s.followers.IsFollowedBy(uri, ctx.User.Username),
			Muted:      ctx.User.HasMuted(uri),
			Links:      links,
		}

		if len(twts) > 0 {
			ctx.Profile.LastPostedAt = twts[0].Created()
		}

		trdata := map[string]interface{}{}
		trdata["URL"] = uri
		ctx.Title = s.tr(ctx, "PageExternalProfileTitle", trdata)

		s.render("profile", r, w, ctx)
	}
}

// ExternalFollowingHandler ...
func (s *Server) ExternalFollowingHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		uri := r.URL.Query().Get("uri")
		nick := r.URL.Query().Get("nick")

		if uri == "" {
			s.renderError(r, w, http.StatusNotFound, s.tr(ctx, "ErrorNoExternalFeed"))
			return
		}

		if !s.cache.HasFeed(uri) {
			s.tasks.DispatchFunc(func() error {
				s.fetcher.EnqueueFeeds(
					FetchFeedRequests{
						{URI: uri, Created: time.Now()},
					},
					s.config.fetchInterval,
				)
				return nil
			})
		}

		if twter := s.cache.GetTwter(uri); twter != nil {
			ctx.Twter = *twter
		} else {
			ctx.Twter = types.Twter{Nick: nick, URI: uri}
		}

		var follows types.Follows
		for nick, twter := range ctx.Twter.Follow {
			follows = append(follows, types.Follow{Nick: nick, URI: twter.URI})
		}

		ctx.Profile = types.Profile{
			Type: "External",

			Nick:        nick,
			Description: ctx.Twter.Tagline,
			Avatar:      URLForExternalAvatar(s.config, uri),
			URI:         uri,

			Following:  follows,
			NFollowing: ctx.Twter.Following,
			NFollowers: ctx.Twter.Followers,

			ShowFollowing: true,
			ShowFollowers: true,

			Follows:    ctx.User.Follows(uri),
			FollowedBy: s.followers.IsFollowedBy(uri, ctx.User.Username),
			Muted:      ctx.User.HasMuted(uri),
		}

		opts := &QueryOptions{
			Limit:   1,
			Exclude: ctx.User.MutedList(),
		}

		twts, _, err := s.cache.GetByURL(uri, opts)
		if err != nil {
			log.WithError(err).Error("error loading external twts")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingTimeline"))
			return
		}

		if len(twts) > 0 {
			ctx.Profile.LastPostedAt = twts[0].Created()
		}

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode(ctx.Profile.Following); err != nil {
				log.WithError(err).Error("error encoding user for display")
				http.Error(w, "Bad Request", http.StatusBadRequest)
			}

			return
		}

		trdata := map[string]interface{}{}
		trdata["Nick"] = nick
		trdata["URL"] = uri
		ctx.Title = s.tr(ctx, "PageExternalFollowingTitle", trdata)
		s.render("following", r, w, ctx)
	}
}

// ExternalAvatarHandler ...
func (s *Server) ExternalAvatarHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Cache-Control", "public, no-cache, must-revalidate")

		uri := NormalizeURL(r.URL.Query().Get("uri"))
		if uri == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		slug := Slugify(uri)
		fn, err := securejoin.SecureJoin(filepath.Join(s.config.Data, externalDir), fmt.Sprintf("%s.png", slug))
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if !FileExists(fn) {
			domainNick := slug

			if twter := s.cache.GetTwter(uri); twter != nil {
				domainNick = twter.DomainNick()
			}

			img, err := GenerateAvatar(s.config, domainNick)
			if err != nil {
				log.WithError(err).Errorf("error generating external avatar for %s", uri)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if r.Method == http.MethodHead {
				return
			}

			w.Header().Set("Content-Type", "image/png")
			if err := png.Encode(w, img); err != nil {
				log.WithError(err).Error("error encoding auto generated avatar")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}

		fileInfo, err := os.Stat(fn)
		if err != nil {
			log.WithError(err).Error("os.Stat() error")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Etag", fmt.Sprintf("W/\"%s-%s\"", slug, fileInfo.ModTime().Format(time.RFC3339)))
		w.Header().Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))

		http.ServeFile(w, r, fn)
	}
}
