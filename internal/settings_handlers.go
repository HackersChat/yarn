// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// SettingsHandler handles user settings requests. It supports both GET and POST
// methods. For GET requests, it retrieves and displays the user's profile and
// followers. For POST requests, it updates user settings such as email, tagline,
// password, avatar, and various display preferences. The function ensures that
// sensitive information like passwords is securely handled and never stored
// directly. It also manages avatar uploads and updates the user's avatar hash.
// Upon successful update, it renders a success message; otherwise, it handles
// errors appropriately.
func (s *Server) SettingsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			profile := ctx.User.Profile(s.config.BaseURL, ctx.User)

			followers := s.followers.GetFor(ctx.Username)
			profile.Followers = followers
			profile.NFollowers = len(followers)

			// This is always true because the user logged in only has access
			// to their own settings and should be following themselves.
			profile.FollowedBy = true

			ctx.Profile = profile

			ctx.Title = s.tr(ctx, "PageSettingsTitle")
			ctx.Bookmarklet = url.QueryEscape(fmt.Sprintf(bookmarkletTemplate, s.config.BaseURL))
			s.render("settings", r, w, ctx)
			return
		}

		// Limit request body to to abuse
		r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxUploadSize)
		defer r.Body.Close()

		// XXX: We DO NOT store this! (EVER)
		email := strings.TrimSpace(r.FormValue("email"))
		tagline := strings.ReplaceAll(strings.TrimSpace(r.FormValue("tagline")), "\r\n", "\u2028")
		password := r.FormValue("password")

		startpage := r.FormValue("startpage")
		theme := r.FormValue("theme")

		displayDatesInTimezone := r.FormValue("displayDatesInTimezone")
		displayTimePreference := r.FormValue("displayTimePreference")
		openLinksInPreference := r.FormValue("openLinksInPreference")
		displayTimelinePreference := r.FormValue("displayTimelinePreference")
		displayImagesPreference := r.FormValue("displayImagesPreference")
		displayMedia := r.FormValue("displayMedia") == "on"
		originalMedia := r.FormValue("originalMedia") == "on"

		visibilityReadmore := r.FormValue("visibilityReadmore") == "on"
		stripTrackingParam := r.FormValue("stripTrackingParam") == "on"

		isFollowingPubliclyVisible := r.FormValue("isFollowingPubliclyVisible") == "on"
		isBookmarksPubliclyVisible := r.FormValue("isBookmarksPubliclyVisible") == "on"

		avatarFile, _, err := r.FormFile("avatar_file")
		if err != nil && err != http.ErrMissingFile {
			log.WithError(err).Error("error parsing form file")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}

		if password != "" {
			hash, err := s.pm.Passwd([]byte(password), nil)
			if err != nil {
				log.WithError(err).Error("error creating password hash")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			user.PasswordHash = hash
		}

		if avatarFile != nil {
			opts := &ImageOptions{
				Resize: true,
				Width:  s.config.AvatarResolution,
				Height: s.config.AvatarResolution,
			}
			_, err = StoreUploadedImage(
				s.config, avatarFile,
				avatarsDir, ctx.Username,
				opts,
			)
			if err != nil {
				s.renderError(r, w, http.StatusInternalServerError,
					"Error updating user: %s", err)
				return
			}
			avatarFn := filepath.Join(s.config.Data, avatarsDir, fmt.Sprintf("%s.png", ctx.Username))
			if avatarHash, err := FastHashFile(avatarFn); err == nil {
				user.AvatarHash = avatarHash
			} else {
				log.WithError(err).Warnf("error updating avatar hash for %s", ctx.Username)
			}
		}

		recoveryHash := fmt.Sprintf("email:%s", FastHashString(email))

		user.Recovery = recoveryHash
		user.Tagline = tagline

		user.StartPage = startpage
		user.Theme = theme

		user.DisplayDatesInTimezone = displayDatesInTimezone
		user.DisplayTimePreference = displayTimePreference
		user.OpenLinksInPreference = openLinksInPreference
		user.DisplayImagesPreference = displayImagesPreference
		user.DisplayMedia = displayMedia
		user.OriginalMedia = originalMedia

		user.VisibilityReadmore = visibilityReadmore
		user.StripTrackingParam = stripTrackingParam

		user.DisplayTimelinePreference = displayTimelinePreference

		user.IsFollowingPubliclyVisible = isFollowingPubliclyVisible
		user.IsBookmarksPubliclyVisible = isBookmarksPubliclyVisible

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorUpdatingUser"))
			return
		}

		s.renderSuccess(r, w, s.tr(ctx, "MsgUpdateSettingsSuccess"))
	}
}

// SettingsAddLinkHandler ...
func (s *Server) SettingsAddLinkHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		// Limit request body to to abuse
		r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxUploadSize)
		defer r.Body.Close()

		linkTitle := strings.TrimSpace(r.FormValue("linkTitle"))
		linkURL := strings.TrimSpace(r.FormValue("linkURL"))

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}

		user.AddLink(linkTitle, linkURL)

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorAddLink"))
			return
		}

		s.renderSuccess(r, w, s.tr(ctx, "MsgAddLinkSuccess"))
	}
}

// SettingsRemoveLinkHandler ...
func (s *Server) SettingsRemoveLinkHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		user := ctx.User
		if user == nil {
			log.Fatalf("user not found in context")
		}

		linkTitle := strings.TrimSpace(r.FormValue("link_title"))
		user.RemoveLink(linkTitle)

		if err := s.db.SetUser(ctx.Username, user); err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorRemoveLink"))
			return
		}

		s.renderSuccess(r, w, s.tr(ctx, "MsgRemoveLinkSuccess"))
	}
}
