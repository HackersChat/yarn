// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// ManagePodHandler ...
func (s *Server) ManagePodHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		if r.Method == "GET" {
			s.render("managePod", r, w, ctx)
			return
		}

		name := strings.TrimSpace(r.FormValue("podName"))
		logo := strings.TrimSpace(r.FormValue("podLogo"))
		css := strings.TrimSpace(r.FormValue("podCSS"))
		js := strings.TrimSpace(r.FormValue("podJS"))
		description := strings.TrimSpace(r.FormValue("podDescription"))
		maxTwtLength := SafeParseInt(r.FormValue("maxTwtLength"), s.config.MaxTwtLength)
		twtsPerPage := SafeParseInt(r.FormValue("twtsPerPage"), s.config.TwtsPerPage)
		avatarResolution := SafeParseInt(r.FormValue("avatarResolution"), s.config.AvatarResolution)
		mediaResolution := SafeParseInt(r.FormValue("mediaResolution"), s.config.MediaResolution)
		frontPage := r.FormValue("frontPage")
		frontPageCompact := r.FormValue("frontPageCompact") == "on"
		openProfiles := r.FormValue("enableOpenProfiles") == "on"
		openRegistrations := r.FormValue("enableOpenRegistrations") == "on"
		disableSupport := r.FormValue("disableSupport") == "on"
		permittedImages := r.FormValue("permittedImages")
		blockedFeeds := r.FormValue("blockedFeeds")
		embedRules := r.FormValue("embedRules")
		enabledFeatures := r.FormValue("enabledFeatures")

		alertFloat := r.FormValue("podAlertFloat") == "on"
		alertGuest := r.FormValue("podAlertGuest") == "on"
		alertMessage := strings.TrimSpace(r.FormValue("podAlertMessage"))
		alertType := r.FormValue("podAlertType")

		displayDatesInTimezone := r.FormValue("displayDatesInTimezone")
		displayTimePreference := r.FormValue("displayTimePreference")
		openLinksInPreference := r.FormValue("openLinksInPreference")
		displayImagesPreference := r.FormValue("displayImagesPreference")
		displayMedia := r.FormValue("displayMedia") == "on"
		originalMedia := r.FormValue("originalMedia") == "on"

		// Clean lines from DOS (\r\n) to UNIX (\n)
		logo = strings.ReplaceAll(logo, "\r\n", "\n")
		css = strings.ReplaceAll(css, "\r\n", "\n")
		js = strings.ReplaceAll(js, "\r\n", "\n")

		permittedImages = strings.Trim(strings.ReplaceAll(permittedImages, "\r\n", "\n"), "\n")
		blockedFeeds = strings.Trim(strings.ReplaceAll(blockedFeeds, "\r\n", "\n"), "\n")
		enabledFeatures = strings.Trim(strings.ReplaceAll(enabledFeatures, "\r\n", "\n"), "\n")

		embedRules = strings.ReplaceAll(embedRules, "\r\n", "\n")
		embedRules = strings.ReplaceAll(embedRules, "\t", "  ")
		embedRules = strings.Trim(embedRules, "\n")

		// Update pod name
		if name != "" {
			s.config.Name = name
		} else {
			s.renderError(r, w, http.StatusBadRequest, "Pod name not specified")
			return
		}

		// Update pod logo
		if logo != "" {
			s.config.Logo = logo
		} else {
			s.renderError(r, w, http.StatusBadRequest, "Pod logo not provided")
			return
		}

		// Update CSS customisation
		s.config.CSS = css

		// Update JS customisation
		s.config.JS = js

		// Update pod description
		if description != "" {
			s.config.Description = description
		} else {
			s.renderError(r, w, http.StatusBadRequest, "Pod description not provided")
			return
		}

		// Update alert type and message
		s.config.AlertFloat = alertFloat
		s.config.AlertGuest = alertGuest
		s.config.AlertMessage = alertMessage
		s.config.AlertType = alertType

		// Update Max Twt Length
		s.config.MaxTwtLength = maxTwtLength
		// Update Twts Per Page
		s.config.TwtsPerPage = twtsPerPage
		// Update Avatar Resolution
		s.config.AvatarResolution = avatarResolution
		// Update Media Resolution
		s.config.MediaResolution = mediaResolution

		// Front Page behaviour (anonymous discover view)
		s.config.FrontPage = frontPage
		s.config.FrontPageCompact = frontPageCompact

		// Update open profiles
		s.config.OpenProfiles = openProfiles
		// Update open registrations
		s.config.OpenRegistrations = openRegistrations
		// Update disable support
		s.config.DisableSupport = disableSupport

		// Update PermittedImages
		if err := WithPermittedImages(strings.Split(permittedImages, "\n"))(s.config); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Error applying permitted images: %s", err)
			return
		}

		// Update BlockedFeeds
		if err := WithBlockedFeeds(strings.Split(blockedFeeds, "\n"))(s.config); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Error applying blocked feeds: %s", err)
			return
		}

		// Update EmbedRules
		if err := WithEmbedRules(embedRules)(s.config); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Error applying embed rules: %s", err)
			return
		}

		// Update Enabled Optional Features

		features, err := FeaturesFromStrings(strings.Split(enabledFeatures, "\n"))
		if err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Error extracting features: %s", err)
			return
		}
		if err := WithEnabledFeatures(features)(s.config); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Error applying features: %s", err)
			return
		}

		// Update Pod Settings (overrideable by Users)
		s.config.DisplayDatesInTimezone = displayDatesInTimezone
		s.config.DisplayTimePreference = displayTimePreference
		s.config.OpenLinksInPreference = openLinksInPreference
		s.config.DisplayImagesPreference = displayImagesPreference
		s.config.DisplayMedia = displayMedia
		s.config.OriginalMedia = originalMedia

		// Save config file
		if err := s.config.Settings().Save(filepath.Join(s.config.Data, "settings.yaml")); err != nil {
			log.WithError(err).Error("error saving config")
			s.renderError(r, w, http.StatusInternalServerError, "Error saving pod settings")
			return
		}

		s.renderSuccess(r, w, "Pod updated successfully")
	}
}

// ManageUsersHandler ...
func (s *Server) ManageUsersHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		s.render("manageUsers", r, w, ctx)
	}
}

// AddUserHandler ...
func (s *Server) AddUserHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))
		// XXX: We DO NOT store this! (EVER)
		email := strings.TrimSpace(r.FormValue("email"))

		// Create a random password that nobody knows, neither the poderator
		// nor the new user. The user is expected to use the "Password Reset"
		// via e-mail for regular users to set their new password which is only
		// known to themselves.
		//
		// TODO Currently, the e-mail address is required only in the HTML
		// form, but we do not check its presence here. This is clearly a bug.
		// It should probably be optional in the HTML form, too, just like when
		// users register themselves. With the poderator's "Password Reset"
		// feature, the new password is shown on screen for the poderator, so
		// it can be passed to the user. We should probably do the same here.
		// So that the poderator doesn't have to enter a fake
		// e-mail address like "whatever@invalid". Maybe display the new
		// password only if no e-mail address has been entered.
		password := GenerateRandomToken()

		if err := ValidateUsername(username); err != nil {
			s.renderError(r, w, http.StatusBadRequest, "Username validation failed: %s", err)
			return
		}

		if s.db.HasUser(username) {
			s.renderError(
				r, w, http.StatusBadRequest,
				"User with that name already exists! Please pick another!",
			)
			return
		}

		p := filepath.Join(s.config.Data, feedsDir)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.WithError(err).Error("error creating feeds directory")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fn := filepath.Join(p, username)
		if _, err := os.Stat(fn); err == nil {
			s.renderError(r, w, http.StatusBadRequest,
				"Deleted user with that username already exists! Please pick another!")
			return
		}

		if err := os.WriteFile(fn, []byte{}, 0644); err != nil {
			log.WithError(err).Error("error creating new user feed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		hash, err := s.pm.Passwd([]byte(password), nil)
		if err != nil {
			log.WithError(err).Error("error creating password hash")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		recoveryHash := fmt.Sprintf("email:%s", FastHashString(email))

		user := NewUser()
		user.Username = username
		user.Recovery = recoveryHash
		user.PasswordHash = []byte(hash)
		user.URL = URLForUser(s.config.BaseURL, username)
		user.CreatedAt = time.Now()

		if err := s.db.SetUser(username, user); err != nil {
			log.WithError(err).Error("error saving user object for new user")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := CreatePasswordResetToken(s.config, user)
		if err != nil {
			log.WithError(err).Errorf("unable to create password reset token")
			s.renderError(r, w, http.StatusInternalServerError,
				"Error creating password reset token! Please try again.")
			return
		}

		if err := SendPasswordResetEmail(s.config, user, email, token); err != nil {
			log.WithError(err).Errorf("unable to send reset password email to %s", user.Username)
			s.renderError(r, w, http.StatusInternalServerError,
				"Error sending password reset email! Please try again.")
			return
		}

		s.renderSuccess(r, w, "User successfully created")
	}
}

// DelUserHandler ...
func (s *Server) DelUserHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))

		if err := DeleteUser(s.config, s.cache, s.db, username); err != nil {
			log.WithError(err).WithFields(log.Fields{"Username": username}).Errorf("error deleting user")
			s.renderError(r, w, http.StatusInternalServerError, "Error deleting user: %s", username)
			return
		}

		actor := r.Context().Value(UserContextKey).(*User)
		log.Infof("user %s deleted by %s from %s", username, actor.Username, r.RemoteAddr)

		s.renderSuccess(r, w, "Successfully deleted account")
	}
}

// RstUserHandler ...
func (s *Server) RstUserHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))

		trdata := map[string]interface{}{}
		trdata["Nick"] = username

		user, err := s.db.GetUser(username)
		if err != nil {
			log.WithError(err).Errorf("error loading user object for %s", username)
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorGetUser"))
			return
		}

		newPassword := GenerateRandomToken()

		hash, err := s.pm.Passwd([]byte(newPassword), nil)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorInvalidPassword"))
			return
		}

		user.PasswordHash = []byte(hash)

		// Save user
		if err := s.db.SetUser(username, user); err != nil {
			trdata["Error"] = err.Error()
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorSetUser", trdata))
			return
		}

		s.renderSuccess(r, w, "Successfully reset password for %s to: %s", username, newPassword)
	}
}

// ManagePeersHandler ...
func (s *Server) ManagePeersHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		ctx.Peers = s.peering.ListPeers()

		s.render("managePeers", r, w, ctx)
	}
}

func (s *Server) DeletePeerHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		uri := r.URL.Query().Get("uri")

		if r.Method == "POST" {
			// FIXME: Fix this
			//s.cache.DeletePeer(uri)

			s.renderSuccess(r, w, "Successfully deleted peer %s", uri)
			return
		}

		s.peering.DeletePeer(uri)

		ctx.Peers = s.peering.ListPeers()

		s.render("delpeer", r, w, ctx)
	}
}

func (s *Server) TrustPeerHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		query := r.URL.Query()
		uri := query.Get("uri")
		trusted := query.Get("trusted") == "true"

		s.peering.TrustPeer(uri, trusted)

		if trusted {
			s.renderSuccess(r, w, "Successfully trusted peer %s", uri)
		} else {
			s.renderSuccess(r, w, "Successfully untrusted peer %s", uri)
		}
	}
}

// ManageJobsHandler ...
func (s *Server) ManageJobsHandler() httprouter.Handle {
	isAdminUser := IsAdminUserFactory(s.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if !isAdminUser(ctx.User) {
			ctx.Error = true
			ctx.Message = "You are not a Pod Owner!"
			w.WriteHeader(http.StatusForbidden)
			s.render("403", r, w, ctx)
			return
		}

		if r.Method == http.MethodPost {
			name := strings.TrimSpace(r.FormValue("name"))

			var job Job
			for _, entry := range s.cron.Entries() {
				if strings.EqualFold(name, entry.Job.(Job).String()) {
					job = entry.Job.(Job)
					break
				}
			}

			if job == nil {
				ctx.Error = true
				ctx.Message = fmt.Sprintf("No job found by that name: %s", name)
				w.WriteHeader(http.StatusNotFound)
				s.render("404", r, w, ctx)
				return
			}

			s.tasks.DispatchFunc(func() error {
				job.Run()
				return nil
			})

			s.renderSuccess(r, w, "Job %s successfully queued for execution", name)
			return
		}

		ctx.Jobs = s.cron.Entries()
		s.render("manageJobs", r, w, ctx)
	}
}
