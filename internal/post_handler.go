// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.mills.io/tasks"
	"go.yarn.social/types"
)

// PostHandler handles the creation/modification/deletion of a twt.
func (s *Server) PostHandler() httprouter.Handle {
	var appendTwt = AppendTwtFactory(s.config, s.cache, s.db)
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)
		var err error

		// Validate twt user.
		if ctx.User.Username == "" {
			log.Errorf("error loading user object for %s", ctx.Username)
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorPostingTwt"))
			return
		}

		hash := r.FormValue("hash")
		var toReplace types.Twt

		// If we are deleting the last twt, delete it and return.
		// Else, we are editing the last twt; so, delete it and continue.
		if r.Method == http.MethodDelete || hash != "" {
			twts, err := GetAllTwts(s.config, ctx.User.Username)
			if err != nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
				return
			}
			for _, twt := range twts {
				if twt.Hash() == hash {
					toReplace = twt
				}
			}

			if toReplace == nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
				return
			}

			// Delete the twt from the feed.
			if err = ReplaceTwt(s.config, ctx.User, toReplace, types.NilTwt); err != nil {
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
				return
			}

			// Delete the twt from cached feeds.
			s.cache.Delete(hash)

			// If we're deleting, just return,
			// otherwise we're editing our last Twt.
			if r.Method == http.MethodDelete {
				referer := RedirectRefererURL(r, s.config, "/")
				if strings.HasSuffix(referer, "/post") {
					// The Referer is total bullshit?! (What browser is doing this?!)
					referer = "/"
				}

				return
			}
		}

		//
		// Post a new twt (or patch/edit our Last Twt)
		//

		// Cleanup Twt text
		text := CleanTwt(r.FormValue("text"))
		if text == "" {
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorNoPostContent"))
			return
		}

		// Prepend the Twt Subject being replied to to the text
		subject := strings.TrimSpace(r.FormValue("subject"))
		log.Debugf("subject: %s", subject)
		if subject != "" {
			if extractedHash := ExtractHashFromSubject(subject); extractedHash != hash {
				text = fmt.Sprintf("(#%s) %s", extractedHash, text)
			}
		}

		var twt types.Twt = types.NilTwt

		if hash != "" && toReplace.Hash() == hash {
			twt, err = appendTwt(ctx.User, text, toReplace.Created())
		} else {
			twt, err = appendTwt(ctx.User, text)
		}

		if err != nil {
			log.WithError(err).Error("error posting twt")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorPostingTwt"))
			return
		}

		feedURL := s.config.URLForUser(ctx.User.Username)

		// Update user's own timeline with their own new post.
		if twter := s.cache.GetTwter(twt.Twter().URI); twter == nil {
			s.cache.SetTwter(twt.Twter().URI, twt.Twter())
		}
		s.cache.UpdateFeed(feedURL, "", types.Twts{twt})

		// WebMentions ...
		// TODO: Use a queue here instead?
		if _, err := s.tasks.Dispatch(tasks.NewFuncTask(func() error {
			for _, m := range twt.Mentions() {
				twter := m.Twter()
				if !s.config.IsLocalURL(twter.URI) {
					log.Debugf("queueing outgoing webmention for %s", twter)
					if err := WebMention(twter.URI, URLForTwt(s.config.BaseURL, twt.Hash())); err != nil {
						log.WithError(err).Warnf("error sending webmention to %s", twter.URI)
					}
				}
			}
			return nil
		})); err != nil {
			log.WithError(err).Warn("error submitting task for webmentions")
		}

		referer := RedirectRefererURL(r, s.config, "/")
		if strings.HasSuffix(referer, "/post") {
			// The Referer is total bullshit?! (What browser is doing this?!)
			referer = "/"
		}

		if htmx.IsHTMX(r) {
			htmx.NewResponse().
				Location(referer).
				Write(w)
		} else {
			http.Redirect(w, r, referer, http.StatusFound)
		}
	}
}

// DeleteTwtHandler ...
func (s *Server) DeleteTwtHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		// Validate twt user.
		if ctx.User.Username == "" {
			log.Errorf("error loading user object for %s", ctx.Username)
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorPostingTwt"))
			return
		}

		hash := r.FormValue("hash")

		var toReplace types.Twt

		twts, err := GetAllTwts(s.config, ctx.User.Username)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
			return
		}
		for _, twt := range twts {
			if twt.Hash() == hash {
				toReplace = twt
			}
		}

		if toReplace.IsZero() {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
			return
		}

		// Delete the twt from the feed.
		if err = ReplaceTwt(s.config, ctx.User, toReplace, types.NilTwt); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeleteLastTwt"))
			return
		}

		// Delete the twt from cached feeds.
		s.cache.Delete(hash)
	}
}
