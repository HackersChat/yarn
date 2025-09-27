// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	// embed resources
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/robfig/cron"
)

const (
	// InvalidConfigValue is the constant value for invalid config values
	// which must be changed for production configurations before successful
	// startup
	InvalidConfigValue = "INVALID CONFIG VALUE - PLEASE CHANGE THIS VALUE"

	// DefaultDebug is the default debug mode
	DefaultDebug = false

	// DefaultData is the default data directory for storage
	DefaultData = "./data"

	// DefaultTLS is the default for whether to enable TLS
	DefaultTLS = false

	// DefaultTLSKey is the default path to a TLS private key (if blank uses Let's Encrypt)
	DefaultTLSKey = ""

	// DefaultTLSCert is the default path to a TLS certificate (if blank uses Let's Encrypt)
	DefaultTLSCert = ""

	// DefaultAuth is the default auther for authenticating sessions
	DefaultAuth = "session"

	// DefaultRedirectURL is the default redirect url for the auther
	DefaultRedirectURL = "/"

	// DefaultAuthHeader is the default auth header to use for proxy auth
	DefaultAuthHeader = "X-User"

	// DefaultStore is the default data store used for accounts, sessions, etc
	DefaultStore = "bitcask://yarn.db"

	// DefaultBaseURL is the default Base URL for the app used to construct feed URLs
	DefaultBaseURL = "http://0.0.0.0:8000"

	// DefaultAdminUser is the default username to grant admin privileges to
	DefaultAdminUser = "admin"

	// DefaultAdminName is the default name of the admin user used in support requests
	DefaultAdminName = "Administrator"

	// DefaultAdminEmail is the default email of the admin user used in support requests
	DefaultAdminEmail = "support@yarn.social"

	// DefaultName is the default instance name
	DefaultName = "yarn.local"

	// DefaultMetaxxx are the default set of <meta> tags used on non-specific views
	DefaultMetaTitle       = ""
	DefaultMetaAuthor      = "Yarn.social"
	DefaultMetaKeywords    = "twtxt, twt, yarn, blog, micro-blog, microblogging, social, media, decentralised, pod"
	DefaultMetaDescription = "🧶 Yarn.social is a Self-Hosted, Twitter™-like Decentralised Microblogging social media platform. No ads, no tracking, your content, your data!"

	// DefaultTheme is the default theme to use for templates and static assets
	// (en empty value means to use the builtin default theme)
	DefaultTheme = ""

	// DefaultLang is the default language to use ('en' or 'zh-cn')
	DefaultLang = "auto"

	// DefaultOpenRegistrations is the default for open user registrations
	DefaultOpenRegistrations = false

	// DefaultDisableSupport is the default for disabling support (affects support and abuse functionality)
	DefaultDisableSupport = false

	// DefaultFrontPage is the default behaviour of the pod's front page (anonymous discover view)
	DefaultFrontPage = "local"

	// DefaultDisableGzip is the default for disabling Gzip compression
	DefaultDisableGzip = false

	// DefaultDisableLogger is the default for disabling the Logger (access logs)
	DefaultDisableLogger = false

	// DefaultDisableMedia is the default for disabling Media support
	DefaultDisableMedia = false

	// DefaultDisableFfmpeg is the default for disabling ffmpeg support
	DefaultDisableFfmpeg = false

	// DefaultCookieSecret is the server's default cookie secret
	DefaultCookieSecret = InvalidConfigValue

	// DefaultTwtsPerPage is the server's default twts per page to display
	DefaultTwtsPerPage = 50

	// DefaultMaxTwtLength is the default maximum length of posts permitted
	DefaultMaxTwtLength = 1024

	// DefaultMaxAgeDays is the default maximum age of cached items in days
	DefaultMaxAgeDays = 10 // 10 days

	// DefaultFetchInterval is the default interval used by the global feed cache
	// to control when to actually fetch and update feeds.
	DefaultFetchInterval = "@every 5m"

	// DefaultOpenProfiles is the default for whether or not to have open user profiles
	DefaultOpenProfiles = false

	// DefaultMaxUploadSize is the default maximum upload size permitted
	DefaultMaxUploadSize = 1 << 24 // ~16MB (enough for high-res photos)

	// DefaultSessionCacheTTL is the server's default session cache ttl
	DefaultSessionCacheTTL = 1 * time.Hour

	// DefaultSessionExpiry is the server's default session expiry time
	DefaultSessionExpiry = 240 * time.Hour // 10 days

	// DefaultTranscoderTimeout is the default vodeo transcoding timeout
	DefaultTranscoderTimeout = 10 * time.Minute // 10mins

	// DefaultMagicLinkSecret is the jwt magic link secret
	DefaultMagicLinkSecret = InvalidConfigValue

	// Default Messaging settings
	DefaultSMTPBind = "0.0.0.0:8025"
	DefaultPOP3Bind = "0.0.0.0:8110"

	// Default SMTP configuration
	DefaultSMTPHost = InvalidConfigValue
	DefaultSMTPPort = 0
	DefaultSMTPUser = InvalidConfigValue
	DefaultSMTPPass = InvalidConfigValue
	DefaultSMTPFrom = InvalidConfigValue

	// DefaultMaxFetchLimit is the maximum fetch fetch limit in bytes
	DefaultMaxFetchLimit = 1 << 20 // ~1MB (or more than enough for months)

	// DefaultMaxFeedSize is the maximum feed size before a feed is rotated (should be smaller than MaxFetchLimit)
	DefaultMaxFeedSize = DefaultMaxFetchLimit / 2

	// DefaultAPISessionTime is the server's default session time for API tokens
	DefaultAPISessionTime = 240 * time.Hour // 10 days

	// DefaultAPISigningKey is the default API JWT signing key for tokens
	DefaultAPISigningKey = InvalidConfigValue

	// MinimumCacheFetchInterval is the smallest allowable cache fetch interval for
	// production pods, an attempt to configure a pod with a smaller value than this
	// results in a configuration validation error.
	MinimumCacheFetchInterval = 59 * time.Second

	// DefaultMediaResolution is the default resolution used to downscale media (iamges)
	// (the original is also preserved and accessible via adding the query string ?full=1)
	DefaultMediaResolution = 850 // 850px width (maintaining aspect ratio)

	// DefaultAvatarResolution is the default resolution used to downscale avatars (profiles)
	DefaultAvatarResolution = 360 // 360px width (maintaining aspect ratio)
)

var (
	// DefaultLogo is the default logo (SVG)
	//go:embed logo.svg
	DefaultLogo string

	// DefaultCSS should be empty
	DefaultCSS string

	// DefaultJS should be empty
	DefaultJS string

	// Default Alert type and message
	DefaultAlertFloat   bool
	DefaultAlertGuest   bool
	DefaultAlertType    = "safe"
	DefaultAlertMessage string

	// DefaultTwtPrompts are the set of default prompts  for twt text(s)
	DefaultTwtPrompts = []string{
		`What's on your mind? 🤔`,
		`Share something insightful! 💡`,
		`Good day to you! 👌 What's new? 🥳`,
		`Did something cool lately? 🤔 Share it! 🤗`,
		`Hi! 👋 Don't forget to post a Twt today! 😉`,
		`Let's have a Yarn! ✋`,
	}

	// DefaultPermittedImages is the default list of image domains
	// to permit for external images to display them inline
	DefaultPermittedImages = []string{
		`imgur\.com`,
		`giphy\.com`,
		`imgs\.xkcd\.com`,
		`reactiongifs\.com`,
		`githubusercontent\.com`,
	}

	// DefaultBlockedFeeds is the default list of feed uris that are
	// blocked and prohibuted from being fetched by the global feed cache
	DefaultBlockedFeeds = []string{
		`port70\.dk`,
		`enotty\.dk`,
		`gopher\.floodgap\.com`,
		`lublin\.se`,
	}

	DefaultEmbedRules = ""

	// DefaultMaxCacheFetchers is the default maximun number of fetchers used
	// by the global feed cache during update cycles. This controls how quickly
	// feeds are updated in each feed cache cycle. The default is the number of
	// available CPUs on the system.
	DefaultMaxCacheFetchers = runtime.NumCPU()

	// DefaultQueueBufferSize is the default size of the queue for feed fetchers
	// When not specified, which defaults to 0, an appropriate value is auto-compueted
	// basd on the max fetchers configured.
	DefaultQueueBufferSize = 0

	// DefaultDisplayDatesInTimezone is the default timezone date and times are display in at the Pod level for
	// anonymous or unauthenticated users or users who have not changed their timezone rpefernece.
	DefaultDisplayDatesInTimezone = "UTC"

	// DefaultDisplayTimePreference is the default Pod level time representation (12hr or 24h) overridable by Users.
	DefaultDisplayTimePreference = "12h"

	// DefaultOpenLinksInPreference is the default Pod level behaviour for opening external links (overridable by Users).
	DefaultOpenLinksInPreference = "newwindow"

	// DisplayImagesPreference is the default Pod-level image display behaviour
	// (inline or lightbox) for displaying images (overridable by Users).
	DefaultDisplayImagesPreference = "inline"

	// DisplayMedia is the default for whether or not to display media at all or just link it
	DefaultDisplayMedia = true

	// OriginalMedia is the default for whether to link or display original media or not
	OriginalMedia bool
)

func NewConfig() *Config {
	conf := &Config{
		Version: version,
		Debug:   DefaultDebug,

		Name:                    DefaultName,
		Logo:                    DefaultLogo,
		CSS:                     DefaultCSS,
		Description:             DefaultMetaDescription,
		Auth:                    DefaultAuth,
		Store:                   DefaultStore,
		Theme:                   DefaultTheme,
		BaseURL:                 DefaultBaseURL,
		AdminUser:               DefaultAdminUser,
		CookieSecret:            DefaultCookieSecret,
		AlertFloat:              DefaultAlertFloat,
		AlertGuest:              DefaultAlertGuest,
		AlertMessage:            DefaultAlertMessage,
		AlertType:               DefaultAlertType,
		TwtPrompts:              DefaultTwtPrompts,
		TwtsPerPage:             DefaultTwtsPerPage,
		MaxTwtLength:            DefaultMaxTwtLength,
		FetchInterval:           DefaultFetchInterval,
		AvatarResolution:        DefaultAvatarResolution,
		MediaResolution:         DefaultMediaResolution,
		OpenProfiles:            DefaultOpenProfiles,
		OpenRegistrations:       DefaultOpenRegistrations,
		DisableSupport:          DefaultDisableSupport,
		DisableGzip:             DefaultDisableGzip,
		DisableLogger:           DefaultDisableLogger,
		DisableFfmpeg:           DefaultDisableFfmpeg,
		DisableMedia:            DefaultDisableMedia,
		Features:                NewFeatureFlags(),
		DisplayDatesInTimezone:  DefaultDisplayDatesInTimezone,
		DisplayTimePreference:   DefaultDisplayTimePreference,
		OpenLinksInPreference:   DefaultOpenLinksInPreference,
		DisplayImagesPreference: DefaultDisplayImagesPreference,
		DisplayMedia:            DefaultDisplayMedia,
		SessionExpiry:           DefaultSessionExpiry,
		MagicLinkSecret:         DefaultMagicLinkSecret,
		SMTPHost:                DefaultSMTPHost,
		SMTPPort:                DefaultSMTPPort,
		SMTPUser:                DefaultSMTPUser,
		SMTPPass:                DefaultSMTPPass,
	}

	return conf
}

// Option is a function that takes a config struct and modifies it
type Option func(*Config) error

// WithDebug sets the debug mode flag
func WithDebug(debug bool) Option {
	return func(cfg *Config) error {
		cfg.Debug = debug
		return nil
	}
}

// WithTLS sets the tls flag
func WithTLS(tls bool) Option {
	return func(cfg *Config) error {
		cfg.TLS = tls
		return nil
	}
}

// WithTLSKey sets the path to a TLS private key
func WithTLSKey(tlsKey string) Option {
	return func(cfg *Config) error {
		cfg.TLSKey = tlsKey
		return nil
	}
}

// WithTLSCert sets the path to a TLS certificate
func WithTLSCert(tlsCert string) Option {
	return func(cfg *Config) error {
		cfg.TLSCert = tlsCert
		return nil
	}
}

// WithData sets the data directory to use for storage
func WithData(data string) Option {
	return func(cfg *Config) error {
		cfg.Data = data
		return nil
	}
}

var ValidAuthers = []string{"proxy", "session"}

// WithAuth sets the auther to use for authenticating sessions
func WithAuth(auth string) Option {
	return func(cfg *Config) error {
		auth = strings.ToLower(auth)
		if !HasString(ValidAuthers, auth) {
			return fmt.Errorf("error: invalid auth %q (valid options: %q", auth, ValidAuthers)
		}
		cfg.Auth = auth
		return nil
	}
}

// WithAuthHeader sets the auth header to use for proxy auth
func WithAuthHeader(authHeader string) Option {
	return func(cfg *Config) error {
		cfg.AuthHeader = authHeader
		return nil
	}
}

// WithStore sets the store to use for accounts, sessions, etc.
func WithStore(store string) Option {
	return func(cfg *Config) error {
		cfg.Store = store
		return nil
	}
}

// WithBaseURL sets the Base URL used for constructing feed URLs
func WithBaseURL(baseURL string) Option {
	return func(cfg *Config) error {
		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		cfg.BaseURL = baseURL
		cfg.baseURL = u
		return nil
	}
}

// WithAdminUser sets the Admin user used for granting special features to
func WithAdminUser(adminUser string) Option {
	return func(cfg *Config) error {
		cfg.AdminUser = adminUser
		return nil
	}
}

// WithAdminName sets the Admin name used to identify the pod operator
func WithAdminName(adminName string) Option {
	return func(cfg *Config) error {
		cfg.AdminName = adminName
		return nil
	}
}

// WithAdminEmail sets the Admin email used to contact the pod operator
func WithAdminEmail(adminEmail string) Option {
	return func(cfg *Config) error {
		cfg.AdminEmail = adminEmail
		return nil
	}
}

// WithName sets the instance's name
func WithName(name string) Option {
	return func(cfg *Config) error {
		cfg.Name = name
		return nil
	}
}

// WithDescription sets the instance's description
func WithDescription(description string) Option {
	return func(cfg *Config) error {
		cfg.Description = description
		return nil
	}
}

// WithTheme sets the theme to use for templates and static asssets
func WithTheme(theme string) Option {
	return func(cfg *Config) error {
		cfg.Theme = theme
		return nil
	}
}

// WithOpenRegistrations sets the open registrations flag
func WithOpenRegistrations(openRegistrations bool) Option {
	return func(cfg *Config) error {
		cfg.OpenRegistrations = openRegistrations
		return nil
	}
}

// WithDisableSupport disables support (support and abuse)
func WithDisableSupport(disableSupport bool) Option {
	return func(cfg *Config) error {
		cfg.DisableSupport = disableSupport
		return nil
	}
}

// WithFrontPage sets the behaviour of the pod's front page (anonymous discover view)
func WithFrontPage(frontpage string) Option {
	return func(cfg *Config) error {
		cfg.FrontPage = frontpage
		return nil
	}
}

// WithDisableGzip sets the disable Gzip flag
func WithDisableGzip(disableGzip bool) Option {
	return func(cfg *Config) error {
		cfg.DisableGzip = disableGzip
		return nil
	}
}

// WithDisableLogger sets the disable Logger flag
func WithDisableLogger(disableLogger bool) Option {
	return func(cfg *Config) error {
		cfg.DisableLogger = disableLogger
		return nil
	}
}

// WithDisableMedia sets the disable Media flag
func WithDisableMedia(disablemedia bool) Option {
	return func(cfg *Config) error {
		cfg.DisableMedia = disablemedia
		return nil
	}
}

// WithDisableFfmpeg sets the disable ffmpeg flag
func WithDisableFfmpeg(disableFfmpeg bool) Option {
	return func(cfg *Config) error {
		cfg.DisableFfmpeg = disableFfmpeg
		return nil
	}
}

// WithCookieSecret sets the server's cookie secret
func WithCookieSecret(secret string) Option {
	return func(cfg *Config) error {
		cfg.CookieSecret = secret
		return nil
	}
}

// WithTwtsPerPage sets the server's twts per page
func WithTwtsPerPage(twtsPerPage int) Option {
	return func(cfg *Config) error {
		cfg.TwtsPerPage = twtsPerPage
		return nil
	}
}

// WithMaxTwtLength sets the maximum length of posts permitted on the server
func WithMaxTwtLength(maxTwtLength int) Option {
	return func(cfg *Config) error {
		cfg.MaxTwtLength = maxTwtLength
		return nil
	}
}

// WithFetchInterval sets the cache fetch interval
// Accepts a string as parsed by `time.ParseDuration`
func WithFetchInterval(fetchInterval string) Option {
	return func(cfg *Config) error {
		if !strings.HasPrefix(fetchInterval, "@every ") {
			fetchInterval = "@every " + fetchInterval
		}

		schedule, err := cron.Parse(fetchInterval)
		if err != nil {
			return fmt.Errorf("error parsing cache fetch interval: %w", err)
		}
		now := time.Now()
		d := schedule.Next(now).Sub(now)

		cfg.fetchInterval = d
		cfg.FetchInterval = fetchInterval
		return nil
	}
}

// WithMaxCacheFetchers sets the maximum number of fetchers for the feed cache
func WithMaxCacheFetchers(maxCacheFetchers int) Option {
	return func(cfg *Config) error {
		cfg.MaxCacheFetchers = maxCacheFetchers
		return nil
	}
}

// WithQueueBufferSize sets the maximum size of the queue for feed fetchers
func WithQueueBufferSize(queueBufferSize int) Option {
	return func(cfg *Config) error {
		cfg.QueueBufferSize = queueBufferSize
		return nil
	}
}

// WithMaxAgeDays sets the maximum age of cached items in days
func WithMaxAgeDays(maxAgeDays int) Option {
	return func(cfg *Config) error {
		cfg.MaxAgeDays = maxAgeDays
		return nil
	}
}

// WithOpenProfiles sets whether or not to have open user profiles
func WithOpenProfiles(openProfiles bool) Option {
	return func(cfg *Config) error {
		cfg.OpenProfiles = openProfiles
		return nil
	}
}

// WithMaxUploadSize sets the maximum upload size permitted by the server
func WithMaxUploadSize(maxUploadSize int64) Option {
	return func(cfg *Config) error {
		cfg.MaxUploadSize = maxUploadSize
		return nil
	}
}

// WithSessionCacheTTL sets the server's session cache ttl
func WithSessionCacheTTL(cacheTTL time.Duration) Option {
	return func(cfg *Config) error {
		cfg.SessionCacheTTL = cacheTTL
		return nil
	}
}

// WithSessionExpiry sets the server's session expiry time
func WithSessionExpiry(expiry time.Duration) Option {
	return func(cfg *Config) error {
		cfg.SessionExpiry = expiry
		return nil
	}
}

// WithTranscoderTimeout sets the video transcoding timeout
func WithTranscoderTimeout(timeout time.Duration) Option {
	return func(cfg *Config) error {
		cfg.TranscoderTimeout = timeout
		return nil
	}
}

// WithMagicLinkSecret sets the MagicLinkSecert used to create password reset tokens
func WithMagicLinkSecret(secret string) Option {
	return func(cfg *Config) error {
		cfg.MagicLinkSecret = secret
		return nil
	}
}

// WithSMTPHost sets the SMTPHost to use for sending email
func WithSMTPHost(host string) Option {
	return func(cfg *Config) error {
		cfg.SMTPHost = host
		return nil
	}
}

// WithSMTPPort sets the SMTPPort to use for sending email
func WithSMTPPort(port int) Option {
	return func(cfg *Config) error {
		cfg.SMTPPort = port
		return nil
	}
}

// WithSMTPUser sets the SMTPUser to use for sending email
func WithSMTPUser(user string) Option {
	return func(cfg *Config) error {
		cfg.SMTPUser = user
		return nil
	}
}

// WithSMTPPass sets the SMTPPass to use for sending email
func WithSMTPPass(pass string) Option {
	return func(cfg *Config) error {
		cfg.SMTPPass = pass
		return nil
	}
}

// WithSMTPFrom sets the SMTPFrom address to use for sending email
func WithSMTPFrom(from string) Option {
	return func(cfg *Config) error {
		cfg.SMTPFrom = from
		return nil
	}
}

// WithMaxFetchLimit sets the maximum feed fetch limit in bytes
func WithMaxFetchLimit(limit int64) Option {
	return func(cfg *Config) error {
		cfg.MaxFetchLimit = limit
		return nil
	}
}

// WithMaxFeedSize sets the maximum feed size before a feed is rotated
func WithMaxFeedSize(size int64) Option {
	return func(cfg *Config) error {
		cfg.MaxFeedSize = size
		return nil
	}
}

// WithAPISessionTime sets the API session time for tokens
func WithAPISessionTime(duration time.Duration) Option {
	return func(cfg *Config) error {
		cfg.APISessionTime = duration
		return nil
	}
}

// WithAPISigningKey sets the API JWT signing key for tokens
func WithAPISigningKey(key string) Option {
	return func(cfg *Config) error {
		cfg.APISigningKey = key
		return nil
	}
}

// WithPermittedImages sets the list of image domains
// permitted for external iamges to display inline
func WithPermittedImages(permittedImages []string) Option {
	return func(cfg *Config) error {
		cfg.PermittedImages = permittedImages
		for _, permittedImage := range permittedImages {
			if permittedImage == "" {
				continue
			}
			re, err := regexp.Compile(permittedImage)
			if err != nil {
				return err
			}
			cfg.permittedImages = append(cfg.permittedImages, re)
		}
		return nil
	}
}

// WithBlockedFeeds sets the list of feed uris blocked
// and prohibited from being fetched by the global feed cache
func WithBlockedFeeds(blockedFeeds []string) Option {
	return func(cfg *Config) error {
		cfg.BlockedFeeds = blockedFeeds
		var newBlockedFeeds []*regexp.Regexp
		for _, blockedFeed := range blockedFeeds {
			if blockedFeed == "" {
				continue
			}
			re, err := regexp.Compile(blockedFeed)
			if err != nil {
				return err
			}
			newBlockedFeeds = append(newBlockedFeeds, re)
		}
		cfg.blockedFeeds = newBlockedFeeds
		return nil
	}
}

// WithEmbedRules parses the json of URL-to-embed rewriting rules
func WithEmbedRules(embedRules string) Option {
	return func(cfg *Config) error {
		cfg.EmbedRules = embedRules

		var rules []EmbedRule
		if err := json.Unmarshal([]byte("["+embedRules+"]"), &rules); err != nil {
			return err
		}

		for i, rule := range rules {
			pattern, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return err
			}

			rules[i].pattern = pattern
		}

		cfg.embedRules = rules
		return nil
	}
}
