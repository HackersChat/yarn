// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rickb777/accept"
	"github.com/securisec/go-keywords"
	log "github.com/sirupsen/logrus"

	"go.yarn.social/types"
)

const (
	maxPermalinkTitle = 144
)

// PermalinkHandler ...
func (s *Server) PermalinkHandler() httprouter.Handle {
	isLocal := IsLocalURLFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)
		ctx.Translate(s.translator)

		hash := p.ByName("hash")
		if len(hash) != types.TwtHashLength {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var twt types.Twt

		opts := &QueryOptions{
			Exclude: ctx.User.MutedList(),
		}

		twt, _ = s.cache.Lookup(hash, opts)
		if twt == nil || twt.IsZero() {
			if accept.PreferredContentTypeLike(r.Header, "text/html") == "text/html" {
				ctx.Error = true
				ctx.Message = s.tr(ctx, "ErrorNoMatchingTwt")
				w.WriteHeader(http.StatusNotFound)
				s.render("404", r, w, ctx)
			} else {
				http.Error(w, "No matching twt by that hash", http.StatusNotFound)
			}
			return
		}

		who := twt.Twter().DomainNick()

		var image string
		if isLocal(twt.Twter().URI) {
			image = URLForAvatar(s.config.BaseURL, twt.Twter().Nick, "")
		} else {
			image = URLForExternalAvatar(s.config, twt.Twter().URI)
		}

		when := twt.Created().Format(time.RFC3339)
		what := fmt.Sprintf("%c", twt)

		if subject, _ := GetTwtConvSubjectHash(s.cache, twt); subject != "" {
			what = strings.ReplaceAll(what, subject, "")
			what = strings.TrimSpace(what)
		}

		ks, err := keywords.Extract(what)
		if err != nil {
			log.WithError(err).Warn("error extracting keywords")
		}

		for _, m := range twt.Mentions() {
			ks = append(ks, m.Twter().Nick)
		}
		var tags types.TagList = twt.Tags()
		ks = append(ks, tags.Tags()...)

		w.Header().Set("Last-Modified", twt.Created().Format(http.TimeFormat))
		w.Header().Set("Link", fmt.Sprintf(`<%s/webmention>; rel="webmention"`, s.config.BaseURL))

		if r.Method == http.MethodHead {
			defer r.Body.Close()
			return
		}

		title := fmt.Sprintf("%s \"%s\"", who, TextWithEllipsis(what, maxPermalinkTitle))

		ctx.Title = title
		ctx.Meta = Meta{
			Title:       title,
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

		ctx.Twts = types.Twts{twt}

		var obj any

		preferred := accept.PreferredContentTypeLike(r.Header, "application/")

		if preferred == "application/json" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			obj = twt
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			s.render("permalink", r, w, ctx)
			return
		}

		data, err := json.Marshal(obj)
		if err != nil {
			log.WithError(err).Error("error serializing twt")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(data)
		// return
	}
}
