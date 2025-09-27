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

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// RegisterHandler ...
func (s *Server) RegisterHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			if ctx.Authenticated {
				if htmx.IsHTMX(r) {
					htmx.NewResponse().
						Location("/").
						Write(w)
				} else {
					http.Redirect(w, r, "/", http.StatusFound)
				}
				return
			}
			if !s.config.OpenRegistrations {
				if htmx.IsHTMX(r) {
					htmx.NewResponse().
						Location("/join").
						Write(w)
				} else {
					http.Redirect(w, r, "/join", http.StatusFound)
				}
			} else {
				s.render("register", r, w, ctx)
			}
			return
		}

		if !s.config.OpenRegistrations {
			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorRegisterDisabled")
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			s.render("error", r, w, ctx)
			return
		}

		if r.FormValue("pooh") != "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := NormalizeUsername(r.FormValue("username"))
		password := r.FormValue("password")

		// XXX: We DO NOT store this! (EVER)
		email := strings.TrimSpace(r.FormValue("email"))

		if err := ValidateUsername(username); err != nil {
			ctx.Error = true
			trdata := map[string]interface{}{
				"Error": err.Error(),
			}
			ctx.Message = s.tr(ctx, "ErrorValidateUsername", trdata)
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			s.render("error", r, w, ctx)
			return
		}

		if s.db.HasUser(username) {
			ctx.Error = true
			ctx.Message = s.tr(ctx, "ErrorHasUserOrFeed")
			if htmx.IsHTMX(r) {
				htmx.NewResponse().Retarget("find .notice").Write(w)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			s.render("error", r, w, ctx)
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
			s.renderError(r, w, http.StatusBadRequest, s.tr(ctx, "ErrorUsernameExists"))
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
		user.PasswordHash = []byte(hash)
		user.Recovery = recoveryHash
		user.URL = URLForUser(s.config.BaseURL, username)
		user.CreatedAt = time.Now()

		if err := s.db.SetUser(username, user); err != nil {
			log.WithError(err).Error("error saving user object for new user")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if htmx.IsHTMX(r) {
			htmx.NewResponse().
				Location("/login").
				Write(w)
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}
