// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"net/http"
	"strings"

	"github.com/angelofallars/htmx-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/steambap/captcha"
	"go.mills.io/sessions"
)

// CaptchaHandler ...
func (s *Server) CaptchaHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		img, err := captcha.NewMathExpr(150, 50)
		if err != nil {
			log.WithError(err).Errorf("unable to get generate captcha image")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Save captcha text in session
		sess := sessions.FromRequest(r)
		if sess == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_ = sess.Set("captchaText", img.Text)

		w.Header().Set("Content-Type", "image/png")
		if err := img.WriteImage(w); err != nil {
			log.WithError(err).Errorf("error sending captcha image response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// SupportHandler ...
func (s *Server) SupportHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" {
			if s.config.DisableSupport {
				if htmx.IsHTMX(r) {
					htmx.NewResponse().
						Location("/contact").
						Write(w)
				} else {
					http.Redirect(w, r, "/contact", http.StatusFound)
				}
			} else {
				ctx.Title = s.tr(ctx, "PageSupportTitle")
				s.render("support", r, w, ctx)
			}
			return
		}

		if s.config.DisableSupport {
			http.Error(w, "Support Disabled", http.StatusForbidden)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		subject := strings.TrimSpace(r.FormValue("subject"))
		message := strings.TrimSpace(r.FormValue("message"))

		captchaInput := strings.TrimSpace(r.FormValue("captchaInput"))

		// Get session
		sess := sessions.FromRequest(r)
		if sess == nil {
			s.renderError(r, w, http.StatusBadRequest,
				"no session found, do you have cookies disabled?")
			return
		}

		// Get captcha text from session
		captchaText, isCaptchaTextAvailable := sess.Get("captchaText")
		if !isCaptchaTextAvailable {
			s.renderError(r, w, http.StatusBadRequest, "no captcha text found")
			return
		}

		if captchaInput != captchaText {
			s.renderError(r, w, http.StatusBadRequest,
				"Unable to match captcha text. Please try again.")
			return
		}

		if err := SendSupportRequestEmail(s.config, name, email, subject, message); err != nil {
			log.WithError(err).Errorf("unable to send support email for %s", email)
			s.renderError(r, w, http.StatusInternalServerError,
				"Error sending support message! Please try again.")
			return
		}

		log.Infof("support message email sent for %s", email)

		// Clean up session, so an anonymous session cookie can be removed with
		// this response
		_ = sess.Del("captchaText")

		s.renderSuccess(r, w,
			"Thank you for your message! Pod operator %s will get back to you soon!",
			s.config.AdminName)
	}
}

// ReportHandler ...
func (s *Server) ReportHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := NewContext(s, r)

		if r.Method == "GET" && s.config.DisableSupport {
			if htmx.IsHTMX(r) {
				htmx.NewResponse().
					Location("/contact").
					Write(w)
			} else {
				http.Redirect(w, r, "/contact", http.StatusFound)
			}
			return
		}

		nick := strings.TrimSpace(r.FormValue("nick"))
		url := NormalizeURL(r.FormValue("url"))

		if nick == "" || url == "" {
			s.renderError(r, w, http.StatusBadRequest, "Both nick and url must be specified")
			return
		}

		if r.Method == "GET" {
			ctx.Title = "Report abuse"
			ctx.ReportNick = nick
			ctx.ReportURL = url
			s.render("report", r, w, ctx)
			return
		}

		if s.config.DisableSupport {
			http.Error(w, "Support Disabled", http.StatusForbidden)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		category := strings.TrimSpace(r.FormValue("category"))
		message := strings.TrimSpace(r.FormValue("message"))

		captchaInput := strings.TrimSpace(r.FormValue("captchaInput"))

		// Get session
		sess := sessions.FromRequest(r)
		if sess == nil {
			s.renderError(r, w, http.StatusBadRequest,
				"no session found, do you have cookies disabled?")
			return
		}

		// Get captcha text from session
		captchaText, isCaptchaTextAvailable := sess.Get("captchaText")
		if !isCaptchaTextAvailable {
			s.renderError(r, w, http.StatusBadRequest, "no captcha text found")
			return
		}

		if captchaInput != captchaText {
			s.renderError(r, w, http.StatusBadRequest,
				"Unable to match captcha text. Please try again.")
			return
		}

		if err := SendReportAbuseEmail(s.config, nick, url, name, email, category, message); err != nil {
			log.WithError(err).Errorf("unable to send report email for %s", email)
			s.renderError(r, w, http.StatusInternalServerError,
				"Error sending report! Please try again.")
			return
		}

		s.renderSuccess(r, w,
			"Thank you for your report! Pod operator %s will get back to you soon!",
			s.config.AdminName)
	}
}
