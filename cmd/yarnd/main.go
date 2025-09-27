// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package main is the entrypoint for the yarnd command
package main

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	_ "github.com/KimMachineGun/automemlimit"
	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/wblakecaldwell/profiler"
	_ "go.yarn.social/lextwt"

	"git.mills.io/yarnsocial/yarn"
	"git.mills.io/yarnsocial/yarn/internal"
)

type flagSliceOfFeatureType []internal.FeatureType

func (f *flagSliceOfFeatureType) String() string {
	var fs []string
	for _, feature := range *f {
		fs = append(fs, feature.String())
	}
	return strings.Join(fs, ",")
}

func (f *flagSliceOfFeatureType) Type() string {
	return "feature"
}

func (f *flagSliceOfFeatureType) Set(value string) error {
	if strings.ToLower(value) == "list" {
		fmt.Println("Available Features:")
		for _, feature := range internal.AvailableFeatures() {
			fmt.Printf(" - %s\n", feature)
		}
		fmt.Println()
		os.Exit(0)
	}

	feature, err := internal.FeatureFromString(value)
	if err != nil {
		log.Warnf("invalid feature %s", value)
		return nil
	}
	*f = append(*f, feature)
	return nil
}

var (
	bind    string
	debug   bool
	logFile string
	version bool

	// TLS options
	tls     bool
	tlsKey  string
	tlsCert string

	// Basic options
	name        string
	description string
	data        string
	store       string
	theme       string
	lang        string
	baseURL     string

	// Auth options
	auth       string
	authHeader string

	// Pod Operator
	adminUser  string
	adminName  string
	adminEmail string

	// Pod Settings
	frontPage         string
	openProfiles      bool
	openRegistrations bool
	disableSupport    bool

	disableGzip   bool
	disableLogger bool
	disableMedia  bool
	disableFfmpeg bool

	// Pod Limits
	twtsPerPage      int
	maxTwtLength     int
	maxUploadSize    int64
	maxFetchLimit    int64
	maxFeedSize      int64
	maxCacheFetchers int
	queueBufferSize  int
	maxAgeDays       int
	fetchInterval    string

	// Pod Secrets
	apiSigningKey   string
	cookieSecret    string
	magiclinkSecret string

	// Email Settings
	smtpHost string
	smtpPort int
	smtpUser string
	smtpPass string
	smtpFrom string

	// Timeouts
	sessionExpiry     time.Duration
	sessionCacheTTL   time.Duration
	apiSessionTime    time.Duration
	transcoderTimeout time.Duration

	// permittedImages, Blocklists
	permittedImages []string
	blockedFeeds    []string

	// Optional Features
	enabledFeatures flagSliceOfFeatureType
)

func initFlags() {
	flag.BoolVarP(&debug, "debug", "D", false, "enable debug logging")
	flag.StringVarP(&bind, "bind", "b", "0.0.0.0:8000", "[int]:<port> to bind to")
	flag.StringVar(&logFile, "log-file", "", "log to file instead of stderr")
	flag.BoolVarP(&version, "version", "v", false, "display version information")

	// TLS options
	flag.BoolVar(&tls, "tls", internal.DefaultTLS, "enable TLS (HTTPS)")
	flag.StringVar(&tlsKey, "tls-key", internal.DefaultTLSKey, "path to TLS private key (if blank uses Let's Encrypt)")
	flag.StringVar(&tlsCert, "tls-cert", internal.DefaultTLSCert, "path to TLS certificate (if blank uses Let's Encrypt)")

	// Basic options
	flag.StringVarP(&name, "name", "n", internal.DefaultName, "set the pod's name")
	flag.StringVarP(&description, "description", "m", internal.DefaultMetaDescription, "set the pod's description")
	flag.StringVarP(&data, "data", "d", internal.DefaultData, "data directory")
	flag.StringVarP(&store, "store", "s", internal.DefaultStore, "store to use")
	flag.StringVarP(&theme, "theme", "t", internal.DefaultTheme, "set the theme to use for templates and static assets (if not specified, uses builtin theme)")
	flag.StringVarP(&lang, "lang", "l", internal.DefaultLang, "set the default language")
	flag.StringVarP(&baseURL, "base-url", "u", internal.DefaultBaseURL, "base url to use")

	// Auth options
	flag.StringVarP(&auth, "auth", "a", internal.DefaultAuth, "auth to use (proxy or session)")
	flag.StringVar(&authHeader, "auth-header", internal.DefaultAuthHeader, "auth header to use for proxy auth")

	// Pod Operator
	flag.StringVarP(&adminName, "admin-name", "N", internal.DefaultAdminName, "default admin user name")
	flag.StringVarP(&adminEmail, "admin-email", "E", internal.DefaultAdminEmail, "default admin user email")
	flag.StringVarP(&adminUser, "admin-user", "A", internal.DefaultAdminUser, "default admin user to use")

	// Pod Settings
	flag.BoolVar(
		&disableSupport, "disable-support", internal.DefaultDisableSupport,
		"whether or not to disable support (support and abuse)",
	)
	flag.BoolVarP(
		&openRegistrations, "open-registrations", "R", internal.DefaultOpenRegistrations,
		"whether or not to have open user registrations",
	)
	flag.BoolVarP(
		&openProfiles, "open-profiles", "O", internal.DefaultOpenProfiles,
		"whether or not to have open user profiles",
	)
	flag.StringVar(
		&frontPage, "front-page", internal.DefaultFrontPage,
		"sets the behaviour of the front page (anonymous discover view)",
	)

	flag.BoolVar(
		&disableGzip, "disable-gzip", internal.DefaultDisableGzip,
		"whether or not to disable Gzip compression",
	)
	flag.BoolVar(
		&disableLogger, "disable-logger", internal.DefaultDisableLogger,
		"whether or not to disable the Logger (access logs)",
	)
	flag.BoolVar(
		&disableMedia, "disable-media", internal.DefaultDisableMedia,
		"whether or not to disable media support entirely",
	)
	flag.BoolVar(
		&disableFfmpeg, "disable-ffmpeg", internal.DefaultDisableFfmpeg,
		"whether or not to disable ffmpeg support for video and audio",
	)

	// Pod Limits
	flag.IntVarP(
		&twtsPerPage, "twts-per-page", "T", internal.DefaultTwtsPerPage,
		"maximum twts per page to display",
	)
	flag.IntVarP(
		&maxTwtLength, "max-twt-length", "L", internal.DefaultMaxTwtLength,
		"maximum length of posts",
	)
	flag.Int64VarP(
		&maxUploadSize, "max-upload-size", "U", internal.DefaultMaxUploadSize,
		"maximum upload size of media",
	)
	flag.Int64VarP(
		&maxFetchLimit, "max-fetch-limit", "F", internal.DefaultMaxFetchLimit,
		"maximum feed fetch limit in bytes",
	)
	flag.Int64VarP(
		&maxFeedSize, "max-feed-size", "S", internal.DefaultMaxFeedSize,
		"maximum feed size before a feed is rorated inbytes",
	)
	flag.IntVarP(
		&maxAgeDays, "max-age-days", "", internal.DefaultMaxAgeDays,
		"maximum age of cached twts in days",
	)

	flag.IntVarP(
		&maxCacheFetchers, "max-cache-fetchers", "", internal.DefaultMaxCacheFetchers,
		"set maximum numnber of fetchers to use for feed cache updates",
	)
	flag.IntVarP(
		&queueBufferSize, "queue-buffer-size", "", internal.DefaultQueueBufferSize,
		"buffer size for the fetcher work queue",
	)
	flag.StringVarP(
		&fetchInterval, "fetch-interval", "", internal.DefaultFetchInterval,
		"cache fetch interval (how often to update feeds) in cron syntax (https://pkg.go.dev/github.com/robfig/cron)",
	)

	// Pod Secrets
	flag.StringVar(
		&apiSigningKey, "api-signing-key", internal.DefaultAPISigningKey,
		"secret to use for signing api tokens",
	)
	flag.StringVar(
		&cookieSecret, "cookie-secret", internal.DefaultCookieSecret,
		"cookie secret to use secure sessions",
	)
	flag.StringVar(
		&magiclinkSecret, "magiclink-secret", internal.DefaultMagicLinkSecret,
		"magiclink secret to use for password reset tokens",
	)

	// Email Settings
	flag.StringVar(&smtpHost, "smtp-host", internal.DefaultSMTPHost, "SMTP Host to use for email sending")
	flag.IntVar(&smtpPort, "smtp-port", internal.DefaultSMTPPort, "SMTP Port to use for email sending")
	flag.StringVar(&smtpUser, "smtp-user", internal.DefaultSMTPUser, "SMTP User to use for email sending")
	flag.StringVar(&smtpPass, "smtp-pass", internal.DefaultSMTPPass, "SMTP Pass to use for email sending")
	flag.StringVar(&smtpFrom, "smtp-from", internal.DefaultSMTPFrom, "SMTP From to use for email sending")

	// Timeouts
	flag.DurationVar(
		&sessionExpiry, "session-expiry", internal.DefaultSessionExpiry,
		"timeout for sessions to expire",
	)
	flag.DurationVar(
		&sessionCacheTTL, "session-cache-ttl", internal.DefaultSessionCacheTTL,
		"time-to-live for cached sessions",
	)
	flag.DurationVar(
		&apiSessionTime, "api-session-time", internal.DefaultAPISessionTime,
		"timeout for api tokens to expire",
	)
	flag.DurationVar(
		&transcoderTimeout, "transcoder-timeout", internal.DefaultTranscoderTimeout,
		"timeout for the video transcoder",
	)

	flag.StringSliceVar(
		&permittedImages, "permitted-images", internal.DefaultPermittedImages,
		"permitted image domain (regexes) for display of inline images",
	)
	flag.StringSliceVar(
		&blockedFeeds, "blocked-feeds", internal.DefaultBlockedFeeds,
		"blocked feeds (regexes) to prohibit fetching",
	)

	// Optional Features
	flag.Var(&enabledFeatures, "enable-feature", "enable the named feature")
}

func flagNameFromEnvironmentName(s string) string {
	s = strings.ToLower(s)
	s = strings.Replace(s, "_", "-", -1)
	return s
}

func parseArgs() error {
	for _, v := range os.Environ() {
		vals := strings.SplitN(v, "=", 2)
		flagName := flagNameFromEnvironmentName(vals[0])
		fn := flag.CommandLine.Lookup(flagName)
		if fn == nil || fn.Changed {
			continue
		}
		if err := fn.Value.Set(vals[1]); err != nil {
			return err
		}
	}
	flag.Parse()
	return nil
}

func extraServiceInfoFactory(_ *internal.Server) profiler.ExtraServiceInfoRetriever {
	return func() map[string]interface{} {
		extraInfo := make(map[string]interface{})

		expvar.Get("stats").(*expvar.Map).Do(func(kv expvar.KeyValue) {
			extraInfo[kv.Key] = kv.Value.String()
		})

		return extraInfo
	}
}

func main() {
	initFlags()
	parseArgs()

	if version {
		fmt.Printf("yarnd %s\n", yarn.FullVersion())
		os.Exit(0)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)

		// Disable deadlock detection in production mode
		sync.Opts.Disable = true
	}

	// https://git.mills.io/yarnsocial/yarn/issues/1002
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
		if err == nil {
			defer f.Close()
			log.StandardLogger().SetOutput(f)
		} else {
			log.WithError(err).Warnf("error opening logfile %q for writing", logFile)
			log.StandardLogger().SetOutput(os.Stderr)
		}
	}

	svr, err := internal.NewServer(bind,
		// Debug mode
		internal.WithDebug(debug),

		// TLS options
		internal.WithTLS(tls),
		internal.WithTLSKey(tlsKey),
		internal.WithTLSCert(tlsCert),

		// Basic options
		internal.WithName(name),
		internal.WithDescription(description),
		internal.WithData(data),
		internal.WithStore(store),
		internal.WithTheme(theme),
		internal.WithBaseURL(baseURL),

		// Auth options
		internal.WithAuth(auth),
		internal.WithAuthHeader(authHeader),

		// Pod Operator
		internal.WithAdminUser(adminUser),
		internal.WithAdminName(adminName),
		internal.WithAdminEmail(adminEmail),

		// Pod Settings
		internal.WithFrontPage(frontPage),
		internal.WithOpenProfiles(openProfiles),
		internal.WithOpenRegistrations(openRegistrations),
		internal.WithDisableSupport(disableSupport),

		internal.WithDisableGzip(disableGzip),
		internal.WithDisableLogger(disableLogger),
		internal.WithDisableMedia(disableMedia),
		internal.WithDisableFfmpeg(disableFfmpeg),

		// Pod Limits
		internal.WithTwtsPerPage(twtsPerPage),
		internal.WithMaxTwtLength(maxTwtLength),
		internal.WithMaxUploadSize(maxUploadSize),
		internal.WithMaxFetchLimit(maxFetchLimit),
		internal.WithMaxFeedSize(maxFeedSize),
		internal.WithMaxCacheFetchers(maxCacheFetchers),
		internal.WithQueueBufferSize(queueBufferSize),
		internal.WithMaxAgeDays(maxAgeDays),
		internal.WithFetchInterval(fetchInterval),

		// Pod Secrets
		internal.WithAPISigningKey(apiSigningKey),
		internal.WithCookieSecret(cookieSecret),
		internal.WithMagicLinkSecret(magiclinkSecret),

		// Email Settings
		internal.WithSMTPHost(smtpHost),
		internal.WithSMTPPort(smtpPort),
		internal.WithSMTPUser(smtpUser),
		internal.WithSMTPPass(smtpPass),
		internal.WithSMTPFrom(smtpFrom),

		// Timeouts
		internal.WithSessionExpiry(sessionExpiry),
		internal.WithSessionCacheTTL(sessionCacheTTL),
		internal.WithAPISessionTime(apiSessionTime),
		internal.WithTranscoderTimeout(transcoderTimeout),

		// PermittedImages, Blocklists
		internal.WithPermittedImages(permittedImages),
		internal.WithBlockedFeeds(blockedFeeds),

		// Optional Features
		internal.WithEnabledFeatures(enabledFeatures),
	)
	if err != nil {
		log.WithError(err).Fatal("error creating server")
	}

	if debug {
		log.Info("starting memory profiler (debug mode) ...")

		runtime.SetBlockProfileRate(10)
		runtime.SetCPUProfileRate(10)

		go func() {
			// add the profiler handler endpoints
			profiler.AddMemoryProfilingHandlers()

			// add realtime extra key/value diagnostic info (optional)
			profiler.RegisterExtraServiceInfoRetriever(extraServiceInfoFactory(svr))

			// start the profiler on service start (optional)
			profiler.StartProfiling()

			// Add pprof handlers
			http.Handle("/block", pprof.Handler("block"))
			http.Handle("/goroutine", pprof.Handler("goroutine"))
			http.Handle("/heap", pprof.Handler("heap"))
			http.Handle("/threadcreate", pprof.Handler("threadcreate"))

			// listen on port 6060 (pick a port)
			http.ListenAndServe(":6060", nil)
		}()
	}

	log.Infof("%s v%s listening on %s", path.Base(os.Args[0]), yarn.FullVersion(), bind)
	if err := svr.Run(); err != nil {
		log.WithError(err).Fatal("error running or shutting down server")
	}
}
