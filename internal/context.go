// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"html/template"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/angelofallars/htmx-go"
	cron "github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/justinas/nosurf"
	"github.com/theplant-retired/timezones"
	"go.mills.io/sessions"
	"go.yarn.social/types"

	"git.mills.io/yarnsocial/yarn"
)

type Link struct {
	Href string
	Rel  string
}

type Alternative struct {
	Type  string
	Title string
	URL   string
}

type Alternatives []Alternative
type Links []Link

type Meta struct {
	Title       string
	Description string
	UpdatedAt   string
	Image       string
	Author      string
	URL         string
	Keywords    string
}

// Context is a "god" object that holds a bunch of data mostly used in templates
// TODO: Refactor the shit out of this so we don't have this giant big object!!!
type Context struct {
	Debug   bool
	IsHTMX  bool
	Request *http.Request
	Filters []string

	Logo             template.HTML
	CSS              template.CSS
	JS               template.JS
	BaseURL          string
	InstanceName     string
	SoftwareVersion  SoftwareConfig
	TwtsPerPage      int
	TwtPrompt        string
	MaxTwtLength     int
	AvatarResolution int
	MediaResolution  int
	RegisterDisabled bool
	SupportDisabled  bool
	OpenProfiles     bool
	FrontPage        string
	FrontPageCompact bool
	DisableMedia     bool
	DisableFfmpeg    bool
	PermittedImages  []string
	BlockedFeeds     []string
	EmbedRules       string
	EnabledFeatures  []string

	AlertFloat   bool
	AlertGuest   bool
	AlertMessage string
	AlertType    string

	Timezones []*timezones.Zoneinfo

	Subject       string
	Username      string
	User          *User
	LastTwt       types.Twt
	Profile       types.Profile
	Authenticated bool
	IsAdmin       bool

	DisplayDatesInTimezone  string
	DisplayTimePreference   string
	OpenLinksInPreference   string
	DisplayImagesPreference string
	DisplayMedia            bool
	OriginalMedia           bool

	VisibilityCompact  bool
	VisibilityReadmore bool
	LinkVerification   bool
	StripTrackingParam bool

	CustomPrimaryColor   string
	CustomSecondaryColor string

	Error       bool
	Message     string
	Callback    string
	Lang        string // language
	AcceptLangs string // accept languages
	StartPage   string
	Theme       string // not to be confused with the config.Theme
	Commit      string

	Page    string
	View    string
	Content template.HTML

	Title        string
	Meta         Meta
	Links        Links
	Alternatives Alternatives

	Twter types.Twter
	Twts  types.Twts
	Root  types.Twt

	Pager *Pager

	// Discovered Pods peering with us
	Peers Peers

	// Background Jobs
	Jobs []cron.Entry

	// Search
	SearchQuery string
	SearchSort  []string

	// Tools
	Bookmarklet string

	// Report abuse
	ReportNick string
	ReportURL  string

	// Reset Password Token
	PasswordResetToken string

	// CSRF Token
	CSRFToken string

	// Login Referer
	Referer string

	// Prompt text
	PromptTitle    string
	PromptMessage  string
	PromptCallback string
	PromptApprove  string
	PromptCancel   string
	PromptTarget   string
}

type FeedContext struct {
	Debug bool

	InstanceName    string
	SoftwareVersion SoftwareConfig

	Profile types.Profile
	Twter   types.Twter
	Prev    string

	Authenticated bool
	Username      string
	IsAdmin       bool
	User          *User
}

func NewFeedContext(s *Server, req *http.Request) *FeedContext {
	conf := s.config
	db := s.db

	// context
	ctx := &FeedContext{
		Debug: conf.Debug,

		InstanceName:    conf.Name,
		SoftwareVersion: conf.Version,

		// Assume all users are anonymous (overridden below if Authenticated)
		User: &User{
			DisplayDatesInTimezone:  conf.DisplayDatesInTimezone,
			DisplayTimePreference:   conf.DisplayTimePreference,
			OpenLinksInPreference:   conf.OpenLinksInPreference,
			DisplayImagesPreference: conf.DisplayImagesPreference,
			DisplayMedia:            conf.DisplayMedia,
			OriginalMedia:           conf.OriginalMedia,
			VisibilityReadmore:      conf.VisibilityReadmore,
			StripTrackingParam:      conf.StripTrackingParam,
		},
	}

	if sess := sessions.FromRequest(req); sess != nil {
		if username, ok := sess.Get("username"); ok {
			ctx.Authenticated = true
			ctx.Username = username
			user, err := db.GetUser(ctx.Username)
			if err != nil {
				// TODO: What's the side effect of this happenning?
				log.WithError(err).Warnf("error loading user object for %s", ctx.Username)
			} else {
				ctx.Twter = types.Twter{
					Nick: user.Username,
					URI:  URLForUser(conf.BaseURL, user.Username),
				}
				ctx.User = user
				ctx.IsAdmin = strings.EqualFold(username, conf.AdminUser)

				// Every registered new user follows themselves
				if user.Following == nil {
					user.Following = make(map[string]string)
				}
				user.Following[user.Username] = user.URL

				// TODO: Use event sourcing for this?
				user.LastSeenAt = time.Now().Round(24 * time.Hour)
				if err := db.SetUser(user.Username, user); err != nil {
					log.WithError(err).Warnf("error updating user.LastSeenAt for %s", user.Username)
				}
			}
		}
	}

	return ctx
}

// NewContext returns a new request scoped context object mostly used by templates.
func NewContext(s *Server, req *http.Request) *Context {
	conf := s.config
	db := s.db

	// build logo
	logo, err := RenderLogo(conf.Logo, conf.Name)
	if err != nil {
		log.WithError(err).Error("error rendering logo")
		logo = template.HTML(DefaultLogo)
	}

	css, err := RenderCSS(conf.CSS)
	if err != nil {
		log.WithError(err).Warn("error rendering custom pod css")
		css = template.CSS(DefaultCSS)
	}

	js, err := RenderJS(conf.JS)
	if err != nil {
		log.WithError(err).Warn("error rendering custom pod js")
		js = template.JS(DefaultJS)
	}

	// context
	ctx := &Context{
		Debug:   conf.Debug,
		IsHTMX:  htmx.IsHTMX(req),
		Request: req,

		Logo:             logo,
		CSS:              css,
		JS:               js,
		BaseURL:          conf.BaseURL,
		InstanceName:     conf.Name,
		SoftwareVersion:  conf.Version,
		TwtsPerPage:      conf.TwtsPerPage,
		TwtPrompt:        conf.RandomTwtPrompt(),
		MaxTwtLength:     conf.MaxTwtLength,
		AvatarResolution: conf.AvatarResolution,
		MediaResolution:  conf.MediaResolution,
		RegisterDisabled: !conf.OpenRegistrations,
		SupportDisabled:  conf.DisableSupport,
		OpenProfiles:     conf.OpenProfiles,
		FrontPage:        conf.FrontPage,
		FrontPageCompact: conf.FrontPageCompact,
		DisableMedia:     conf.DisableMedia,
		DisableFfmpeg:    conf.DisableFfmpeg,
		LastTwt:          types.NilTwt,
		PermittedImages:  conf.PermittedImages,
		BlockedFeeds:     conf.BlockedFeeds,
		EmbedRules:       conf.EmbedRules,
		EnabledFeatures:  conf.Features.AsStrings(),

		AlertFloat:   conf.AlertFloat,
		AlertGuest:   conf.AlertGuest,
		AlertMessage: conf.AlertMessage,
		AlertType:    conf.AlertType,

		DisplayDatesInTimezone:  conf.DisplayDatesInTimezone,
		DisplayTimePreference:   conf.DisplayTimePreference,
		OpenLinksInPreference:   conf.OpenLinksInPreference,
		DisplayImagesPreference: conf.DisplayImagesPreference,
		DisplayMedia:            conf.DisplayMedia,
		OriginalMedia:           conf.OriginalMedia,

		VisibilityCompact:  conf.VisibilityCompact,
		VisibilityReadmore: conf.VisibilityReadmore,
		LinkVerification:   conf.LinkVerification,
		StripTrackingParam: conf.StripTrackingParam,

		CustomPrimaryColor:   conf.CustomPrimaryColor,
		CustomSecondaryColor: conf.CustomSecondaryColor,

		Commit:      yarn.Commit,
		StartPage:   conf.StartPage,
		Theme:       conf.Theme,
		Lang:        conf.Lang,
		AcceptLangs: req.Header.Get("Accept-Language"),

		Timezones: timezones.AllZones,

		Title: "",
		Meta: Meta{
			Title:       DefaultMetaTitle,
			Author:      DefaultMetaAuthor,
			Keywords:    DefaultMetaKeywords,
			Description: conf.Description,
		},

		// Assume all users are anonymous (overridden below if Authenticated)
		User: &User{
			DisplayDatesInTimezone:  conf.DisplayDatesInTimezone,
			DisplayTimePreference:   conf.DisplayTimePreference,
			OpenLinksInPreference:   conf.OpenLinksInPreference,
			DisplayImagesPreference: conf.DisplayImagesPreference,
			DisplayMedia:            conf.DisplayMedia,
			OriginalMedia:           conf.OriginalMedia,
			VisibilityReadmore:      conf.VisibilityReadmore,
			StripTrackingParam:      conf.StripTrackingParam,
		},
		Twter: types.Twter{},
		Root:  types.NilTwt,

		CSRFToken: nosurf.Token(req),
	}

	if sess := sessions.FromRequest(req); sess != nil {
		if username, ok := sess.Get("username"); ok {
			ctx.Authenticated = true
			ctx.Username = username
			user, err := db.GetUser(ctx.Username)
			if err != nil {
				// TODO: What's the side effect of this happenning?
				log.WithError(err).Warnf("error loading user object for %s", ctx.Username)
			} else {
				ctx.Twter = types.Twter{
					Nick: user.Username,
					URI:  URLForUser(conf.BaseURL, user.Username),
				}
				ctx.User = user
				ctx.IsAdmin = strings.EqualFold(username, conf.AdminUser)

				// Every registered new user follows themselves
				if user.Following == nil {
					user.Following = make(map[string]string)
				}
				user.Following[user.Username] = user.URL

				// TODO: Use event sourcing for this?
				user.LastSeenAt = time.Now().Round(24 * time.Hour)
				if err := db.SetUser(user.Username, user); err != nil {
					log.WithError(err).Warnf("error updating user.LastSeenAt for %s", user.Username)
				}
			}
		}
	}

	// Set the theme based on user preferences
	theme := strings.ToLower(ctx.User.Theme)
	switch theme {
	case "auto":
		ctx.Theme = ""
	case "light", "dark", "light-classic", "dark-classic":
		ctx.Theme = theme
	case "amoled":
		ctx.Theme = theme
	default:
		// Default to the configured theme
		ctx.Theme = conf.Theme
	}
	// Set user language
	lang := strings.ToLower(ctx.User.Lang)
	if lang != "" && lang != "auto" {
		ctx.Lang = lang
	}

	return ctx
}

func (ctx *Context) Translate(translator *Translator, data ...interface{}) {
	// TwtPrompt
	defualtTwtPrompts := translator.Translate(ctx, "DefaultTwtPrompts", data...)
	twtPrompts := strings.Split(defualtTwtPrompts, "\n")
	n := rand.Int() % len(twtPrompts)
	ctx.TwtPrompt = twtPrompts[n]
}
