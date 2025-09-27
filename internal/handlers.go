// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"gopkg.in/yaml.v2"
)

const (
	bookmarkletTemplate = `(function(){window.location.href="%s/?title="+document.title+"&url="+document.URL;})();`
	logoWidth           = 512
	logoHeight          = 512
)

var (
	ErrFeedImposter = errors.New("error: imposter detected, you do not own this feed")
)

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Endpoint Not Found", http.StatusNotFound)
		return
	}

	ctx := NewContext(s, r)
	ctx.Title = s.tr(ctx, "PageNotFoundTitle")
	w.WriteHeader(http.StatusNotFound)
	s.render("404", r, w, ctx)
}

// UserConfigHandler ...
func (s *Server) UserConfigHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := NormalizeUsername(p.ByName("nick"))
		if nick == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick = NormalizeUsername(nick)

		var (
			url       string
			following map[string]string
			bookmarks map[string]string
		)

		if s.db.HasUser(nick) {
			user, err := s.db.GetUser(nick)
			if err != nil {
				log.WithError(err).Errorf("error loading user object for %s", nick)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			url = user.URL
			if ctx.Authenticated || user.IsFollowingPubliclyVisible {
				following = user.Following
			}
			if ctx.Authenticated || user.IsBookmarksPubliclyVisible {
				bookmarks = user.Bookmarks
			}
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		config := struct {
			Nick      string            `json:"nick"`
			URL       string            `json:"url"`
			Following map[string]string `json:"following"`
			Bookmarks map[string]string `json:"bookmarks"`
		}{
			Nick:      nick,
			URL:       url,
			Following: following,
			Bookmarks: bookmarks,
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			log.WithError(err).Errorf("error exporting user/feed config")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/yaml")
		if r.Method == http.MethodHead {
			return
		}

		_, _ = w.Write(data)
	}
}

// AvatarHandler ...
func (s *Server) AvatarHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Cache-Control", "public, no-cache, must-revalidate")

		nick := NormalizeUsername(p.ByName("nick"))
		if nick == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if !s.db.HasUser(nick) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		fn, err := securejoin.SecureJoin(filepath.Join(s.config.Data, avatarsDir), fmt.Sprintf("%s.png", nick))
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if fileInfo, err := os.Stat(fn); err == nil {
			w.Header().Set("Etag", fmt.Sprintf("W/\"%s-%s\"", r.RequestURI, fileInfo.ModTime().Format(time.RFC3339)))
			w.Header().Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))
			http.ServeFile(w, r, fn)
			return
		}

		etag := fmt.Sprintf("W/\"%s\"", r.RequestURI)

		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Etag", etag)
		if r.Method == http.MethodHead {
			return
		}

		img, err := GenerateAvatar(s.config, nick)
		if err != nil {
			log.WithError(err).Errorf("error generating avatar for %s", nick)
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
	}
}

// WebMentionHandler ...
func (s *Server) WebMentionHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		defer r.Body.Close()
		webmentions.ServeHTTP(w, r)
	}
}

// LookupHandler ...
func (s *Server) LookupHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		prefix := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("prefix")))
		user := ctx.User
		matches := make([]struct {
			Nick   string
			Avatar string
			Domain string
		}, 0)

		if len(prefix) > 0 {
			for nick, url := range user.Following {
				if strings.HasPrefix(strings.ToLower(nick), prefix) {
					avatar, domain := GetLookupMatches(s.config, nick, url)
					matches = append(matches, struct {
						Nick   string
						Avatar string
						Domain string
					}{nick, avatar, domain})
				}
			}
		} else {
			for nick, url := range user.Following {
				avatar, domain := GetLookupMatches(s.config, nick, url)
				matches = append(matches, struct {
					Nick   string
					Avatar string
					Domain string
				}{nick, avatar, domain})
			}
		}

		data, err := json.Marshal(matches)
		if err != nil {
			log.WithError(err).Error("error serializing lookup response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// FollowersHandler returns an HTTP handler that processes requests to retrieve
// the list of followers for a user. It constructs the user's profile with
// follower information and responds with JSON if requested. If the request
// does not accept JSON, it renders a template with the user's followers.
func (s *Server) FollowersHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		profile := ctx.User.Profile(s.config.BaseURL, ctx.User)

		followers := s.followers.GetFor(ctx.User.Username)
		profile.Followers = followers
		profile.NFollowers = len(followers)

		ctx.Profile = profile

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(ctx.Profile.Followers); err != nil {
				log.WithError(err).Error("error encoding followers for display")
				http.Error(w, "Bad Request", http.StatusBadRequest)
			}
			return
		}

		trdata := map[string]interface{}{"Username": ctx.User.Username}
		ctx.Title = s.tr(ctx, "PageUserFollowersTitle", trdata)
		s.render("followers", r, w, ctx)
	}
}

// FollowingHandler handles HTTP requests for retrieving and displaying the list of users
// that a specified user is following. It supports both JSON and plain text responses based
// on the "Accept" header in the request. The handler checks if the requested user's following
// list is publicly visible or if the requester is the user themselves. If the user is not found
// or access is restricted, it renders appropriate error responses. Otherwise, it encodes the
// following list in the requested format and sends it back to the client.
func (s *Server) FollowingHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		nick := NormalizeUsername(p.ByName("nick"))

		if s.db.HasUser(nick) {
			user, err := s.db.GetUser(nick)
			if err != nil {
				log.WithError(err).Errorf("error loading user object for %s", nick)
				s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingProfile"))
				return
			}

			if !user.IsFollowingPubliclyVisible && !ctx.User.Is(user.URL) {
				s.render("401", r, w, ctx)
				return
			}
			ctx.Profile = user.Profile(s.config.BaseURL, ctx.User)
		} else {
			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorUserNotFound")
			s.render("404", r, w, ctx)
			return
		}

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode(ctx.Profile.Following); err != nil {
				log.WithError(err).Error("error encoding user for display")
				http.Error(w, "Bad Request", http.StatusBadRequest)
			}

			return
		} else if r.Header.Get("Accept") == "text/plain" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			for _, follow := range ctx.Profile.Following {
				_, _ = fmt.Fprintf(w, "%s\n", follow)
			}
			return
		}

		trdata := map[string]interface{}{
			"Username": nick,
		}
		ctx.Title = s.tr(ctx, "PageUserFollowingTitle", trdata)
		s.render("following", r, w, ctx)
	}
}

// TaskHandler ...
func (s *Server) TaskHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		uuid := p.ByName("uuid")

		if uuid == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		t, ok := s.tasks.Lookup(uuid)
		if !ok {
			http.Error(w, "Task Not Found", http.StatusNotFound)
			return
		}

		data, err := json.Marshal(t.Result())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)

	}
}

// PodInfoHandler ...
func (s *Server) PodInfoHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if r.Header.Get("Accept") == "application/json" {
			data, err := json.Marshal(Peer{
				Name:             s.config.Name,
				Logo:             s.config.baseURL.JoinPath("/logo").String(),
				Description:      s.config.Description,
				SoftwareVersion:  s.config.Version.FullVersion,
				BuildInformation: s.config.Version.Build,
			})
			if err != nil {
				log.WithError(err).Error("error serializing pod version response")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		} else {
			ctx := NewContext(s, r)
			s.render("info", r, w, ctx)
		}
	}
}

// PodLogoHandler ...
func (s *Server) PodLogoHandler() httprouter.Handle {
	logoCtx := struct{ PodName string }{PodName: s.config.Name}
	logoString, err := RenderPlainText(s.config.Logo, logoCtx)
	if err != nil {
		log.WithError(err).Fatal("error rendering logo")
	}

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logoReader := bytes.NewBufferString(logoString)

		if r.Header.Get("Accept") == "image/png" {
			icon, _ := oksvg.ReadIconStream(logoReader)
			icon.SetTarget(0, 0, float64(logoWidth), float64(logoHeight))
			img := image.NewRGBA(image.Rect(0, 0, logoWidth, logoHeight))
			icon.Draw(rasterx.NewDasher(logoWidth, logoHeight, rasterx.NewScannerGV(logoWidth, logoHeight, img, img.Bounds())), 1)

			w.Header().Set("Content-Type", "image/png")
			if err := png.Encode(w, img); err != nil {
				log.WithError(err).Error("error encoding auto generated avatar")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else {
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write(logoReader.Bytes())
		}
	}
}

// PodConfigHandler ...
func (s *Server) PodConfigHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		data, err := json.Marshal(s.config)
		if err != nil {
			log.WithError(err).Error("error serializing pod config response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// DeleteHandler ...
func (s *Server) DeleteHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		// Get user's primary feed twts
		twts, err := GetAllTwts(s.config, ctx.User.Username)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError,
				s.tr(ctx, "ErrorDeletingAccount"))
			return
		}

		// Parse twts to search and remove primary feed uploaded media
		for _, twt := range twts {
			mediaPaths := GetMediaNamesFromText(fmt.Sprintf("%t", twt))

			// Remove all uploaded media in a twt
			for _, mediaPath := range mediaPaths {
				// Delete .png
				fn := filepath.Join(s.config.Data, mediaDir, fmt.Sprintf("%s.png", mediaPath))
				if FileExists(fn) {
					if err := os.Remove(fn); err != nil {
						log.WithError(err).Error("error removing media")
						s.renderError(r, w, http.StatusInternalServerError,
							s.tr(ctx, "ErrorDeletingAccount"))
					}
				}
			}
		}

		// Delete user's avatar
		if fns, err := filepath.Glob(filepath.Join(s.config.Data, avatarsDir, fmt.Sprintf("%s.*", ctx.User.Username))); err == nil {
			for _, fn := range fns {
				if FileExists(fn) {
					if err := os.Remove(fn); err != nil {
						log.WithError(err).Error("error removing user's avatar")
						s.renderError(r, w, http.StatusInternalServerError,
							s.tr(ctx, "ErrorDeletingAccount"))
					}
				}
			}
		}

		// Delete user's twtxt.txt
		fn := filepath.Join(s.config.Data, feedsDir, ctx.User.Username)
		if FileExists(fn) {
			if err := os.Remove(fn); err != nil {
				log.WithError(err).Error("error removing user's feed")
				s.renderError(r, w, http.StatusInternalServerError,
					s.tr(ctx, "ErrorDeletingAccount"))
			}
		}

		// Delete user
		if err := s.db.DelUser(ctx.Username); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorDeletingAccount"))
			return
		}

		// Delete user's feed from cache
		var urls []string
		for _, feed := range ctx.User.Source() {
			urls = append(urls, feed.URI)
		}
		s.cache.DeleteFeeds(urls...)

		s.sm.Delete(w, r)
		ctx.Authenticated = false

		s.renderSuccessCtx(r, w, ctx, s.tr(ctx, "MsgDeleteAccountSuccess"))
	}
}
