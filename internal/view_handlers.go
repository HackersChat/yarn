// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	"github.com/rickb777/accept"
	"github.com/securisec/go-keywords"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// TimelineHandler handles HTTP requests for displaying the user's timeline. It supports
// both authenticated and unauthenticated requests by determining the appropriate set
// of twts to display based on the user's authentication status and configured front
// page settings. The function also applies any specified filters to the timeline entries
// and paginates the results. The timeline is rendered as an HTML response, and it handles
// errors in fetching and sorting the twts, displaying an error page if necessary.
func (s *Server) TimelineHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Set content type for HTML response.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == http.MethodHead {
			defer r.Body.Close()
			return
		}
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		var (
			twts  types.Twts
			total int
			err   error
		)

		// Determine the current page.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		limit := s.config.TwtsPerPage
		offset := (page - 1) * limit

		opts := &QueryOptions{
			Limit:      limit,
			Offset:     offset,
			MaxAgeDays: s.config.MaxAgeDays,
			Exclude:    ctx.User.MutedList(),
		}

		if !ctx.Authenticated {
			if s.config.FrontPage == "local" {
				twts, total, err = s.cache.GetByURL(s.config.BaseURL, opts)
				ctx.Title = s.tr(ctx, "PageLocalTimelineTitle")
			} else {
				opts.Compact = s.config.FrontPageCompact
				twts, total, err = s.cache.GetAll(opts)
				ctx.Title = s.tr(ctx, "PageDiscoverTitle")
			}
		} else {
			var feeds []string
			for _, source := range ctx.User.Sources() {
				feeds = append(feeds, source.URI)
			}
			opts.Flat = ctx.User.DisplayTimelinePreference == "flat"
			twts, total, err = s.cache.GetByFeeds(feeds, opts)
			ctx.Title = s.tr(ctx, "PageUserTimelineTitle")
		}

		if err != nil {
			log.WithError(err).Error("error loading timeline")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorTimelineLoad"))
			return
		}

		// Set paged results and pager.
		ctx.Twts = twts
		ctx.Pager = NewPager(page, limit, total)

		s.render("timeline", r, w, ctx)
	}
}

// DiscoverHandler returns a handler that renders the discover page.
//
// The handler shows a view of all twts that the user has access to, filtered
// by the user's current filter settings. The handler also shows the current
// user's last twt, if they are logged in.
//
// The handler will return a 500 error if there is an internal error loading
// the twts, or if the user is not authenticated and the feature
// FeatureFilterAndLists is not enabled.
func (s *Server) DiscoverHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		// Get discover twts from DB with paging.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		limit := s.config.TwtsPerPage
		offset := (page - 1) * limit

		opts := &QueryOptions{
			Limit:      limit,
			Offset:     offset,
			MaxAgeDays: s.config.MaxAgeDays,
			Exclude:    ctx.User.MutedList(),
		}

		twts, total, err := s.cache.GetAll(opts)
		if err != nil {
			log.WithError(err).Error("error loading discover")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingDiscover"))
			return
		}

		ctx.Title = s.tr(ctx, "PageDiscoverTitle")
		ctx.Twts = twts
		ctx.Pager = NewPager(page, limit, total)
		s.render("discover", r, w, ctx)
	}
}

// MentionsHandler handles HTTP requests for displaying a user's mentions.
// It retrieves mentions for the authenticated user, applying any specified filters
// and paginates the results. The mentions are rendered as an HTML response, displaying
// them in the order of creation. If an error occurs during sorting or paging, an error
// page is displayed. The handler supports both the feature flag for filtered views and
// non-filtered fallback.
func (s *Server) MentionsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		// Get the mentioned twts with DB paging.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		limit := s.config.TwtsPerPage
		offset := (page - 1) * limit

		opts := &QueryOptions{
			Limit:      limit,
			Offset:     offset,
			MaxAgeDays: s.config.MaxAgeDays,
			Exclude:    ctx.User.MutedList(),
		}

		twts, total, err := s.cache.GetMentions(ctx.User.URL, opts)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingMentions"))
			return
		}

		ctx.Title = s.tr(ctx, "PageMentionsTitle")
		ctx.Twts = twts
		ctx.Pager = NewPager(page, limit, total)
		s.render("mentions", r, w, ctx)
	}
}

// ConversationHandler ...
func (s *Server) ConversationHandler() httprouter.Handle {
	isLocal := IsLocalURLFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		hash := p.ByName("hash")
		if len(hash) < types.TwtHashLength {
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
		ctx.Root = twt

		var (
			who   string
			image string
		)

		twter := twt.Twter()
		if isLocal(twter.URI) {
			who = fmt.Sprintf("%s@%s", twter.Nick, s.config.LocalURL().Hostname())
			image = URLForAvatar(s.config.BaseURL, twter.Nick, "")
		} else {
			who = fmt.Sprintf("@<%s %s>", twter.Nick, twter.URI)
			image = URLForExternalAvatar(s.config, twter.URI)
		}

		when := twt.Created().Format(time.RFC3339)
		what := twt.FormatText(types.TextFmt, s.config)

		ks, err := keywords.Extract(what)
		if err != nil {
			log.WithError(err).Warn("error extracting keywords")
		}

		for _, m := range twt.Mentions() {
			ks = append(ks, m.Twter().Nick)
		}
		var tags types.TagList = twt.Tags()
		ks = append(ks, tags.Tags()...)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Link", fmt.Sprintf(`<%s/webmention>; rel="webmention"`, s.config.BaseURL))

		var twts types.Twts

		// Parse the page number from the URL query params. Default to 1.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		opts.Limit = s.config.TwtsPerPage
		opts.Offset = (page - 1) * opts.Limit

		twts, total, err := s.cache.GetBySubject("#"+hash, opts)
		if err != nil {
			ctx.Error = true
			ctx.Message = "Error loading twts"
			w.WriteHeader(http.StatusInternalServerError)
			s.render("error", r, w, ctx)
		}

		// TODO: Do this in SQL?
		if len(twts) > 0 && twts[0].Hash() != ctx.Root.Hash() {
			twts = append(twts, ctx.Root)
			sort.Sort(sort.Reverse(twts))
		}

		if accept.PreferredContentTypeLike(r.Header, "application/json") == "application/json" {
			data, err := json.Marshal(twts)
			if err != nil {
				log.WithError(err).Error("error serializing twt response")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Last-Modified", twt.Created().Format(http.TimeFormat))
			_, _ = w.Write(data)
			return
		}

		pager := NewPager(page, opts.Limit, total)

		if r.Method == http.MethodHead {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
			return
		}

		title := fmt.Sprintf("%s \"%s\"", who, TextWithEllipsis(what, maxPermalinkTitle))

		ctx.Title = title
		ctx.Meta = Meta{
			Title:       fmt.Sprintf("%s #%s", s.tr(ctx, "ConversationTitle"), twt.Hash()),
			Description: what,
			UpdatedAt:   when,
			Author:      who,
			Image:       image,
			URL:         URLForTwt(s.config.BaseURL, hash),
			Keywords:    strings.Join(ks, ", "),
		}

		if strings.HasPrefix(twt.Twter().URI, s.config.BaseURL) {
			ctx.Alternatives = append(ctx.Alternatives, Alternatives{
				Alternative{
					Type:  "text/plain",
					Title: fmt.Sprintf("%s's Twtxt Feed", twt.Twter().Nick),
					URL:   twt.Twter().URI,
				},
			}...)
		}

		ctx.Subject = "#" + twt.Hash()
		ctx.Twts = twts
		ctx.Pager = pager

		s.render("conversation", r, w, ctx)
	}
}

// SearchHandler ...
func (s *Server) SearchHandler() httprouter.Handle {
	isLocalURL := IsLocalURLFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		qs, err := url.QueryUnescape(r.URL.Query().Get("q"))
		if qs == "" || err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Redirect to External/Profile view if the query string is a valid URL that looks like a feed.
		if u, err := url.Parse(qs); err == nil && u.Scheme != "" && u.Host != "" {
			var location string
			if isLocalURL(u.String()) {
				location = UserURL(u.String())
			} else {
				location = URLForExternalProfile(s.config, u.String())
			}
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Location(location).Write(w)
			} else {
				http.Redirect(w, r, location, http.StatusFound)
			}
			return
		}

		sortBy := r.URL.Query()["s"]
		if len(sortBy) == 0 {
			// Default to sort by created date in descending order
			sortBy = []string{"-created"}
		}
		ctx.SearchQuery = qs
		ctx.SearchSort = sortBy

		// Compute paging values.
		page := SafeParseInt(r.FormValue("p"), 1)
		if page < 1 {
			page = 1
		}
		limit := s.config.TwtsPerPage
		offset := (page - 1) * limit

		// Get the paginated search results from the database.
		twts, total, err := s.cache.Search(qs, &QueryOptions{Limit: limit, Offset: offset, SortBy: sortBy[0]})
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingSearch"))
			return
		}
		ctx.Twts = twts

		// Create and assign the pager.
		ctx.Pager = NewPager(page, limit, total)

		s.render("search", r, w, ctx)
	}
}
