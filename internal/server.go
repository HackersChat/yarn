// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"git.mills.io/prologic/observe"
	"github.com/NYTimes/gziphandler"
	"github.com/angelofallars/htmx-go"
	humanize "github.com/dustin/go-humanize"
	"github.com/gabstv/merger"
	"github.com/justinas/nosurf"
	cron "github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	metricsMiddlewarePrometheus "github.com/slok/go-http-metrics/metrics/prometheus"
	metricsMiddleware "github.com/slok/go-http-metrics/middleware"
	httproutermiddleware "github.com/slok/go-http-metrics/middleware/httprouter"
	"github.com/unrolled/logger"
	"go.sour.is/passwd"
	"go.sour.is/passwd/pkg/argon2"
	"go.sour.is/passwd/pkg/scrypt"
	"golang.org/x/crypto/acme/autocert"
	"willnorris.com/go/microformats"

	"go.mills.io/sessions"
	"go.mills.io/tasks"
	"go.mills.io/webfinger"
	"go.mills.io/webmention"
	"go.yarn.social/types"

	"git.mills.io/yarnsocial/yarn"
	"git.mills.io/yarnsocial/yarn/internal/auth"
	"git.mills.io/yarnsocial/yarn/internal/indieweb"
)

const (
	acmeDir = "acme"
)

var (
	metrics     *observe.Metrics
	webmentions webmention.WebMention
	websub      *indieweb.WebSub

	//go:embed theme
	builtinThemeFS embed.FS
)

func init() {
	metrics = observe.NewMetrics("yarnd")
}

// Server ...
type Server struct {
	bind    string
	config  *Config
	tmplman *TemplateManager
	router  *Router
	server  *http.Server

	// Feed Cache
	cache Cacher

	// Feed Fetcher
	fetcher *FeedFetcher

	// Peering
	peering *Peering

	// Data Store
	db Store

	// Followers
	followers *Followers

	// Scheduler
	cron *cron.Cron

	// Dispatcher
	tasks *tasks.Dispatcher

	// Auther
	auther auth.Auther

	// Sessions
	sc sessions.Store
	sm *sessions.Manager

	// API
	api *API

	// Passwords
	pm *passwd.Passwd

	// Translator
	translator *Translator

	// Factory Functions
	AppendTwt AppendTwtFunc
}

func (s *Server) render(name string, r *http.Request, w http.ResponseWriter, ctx *Context) {
	ctx.View = name

	var (
		err error
		buf io.WriterTo
	)

	if htmx.IsHTMX(r) {
		buf, err = s.tmplman.ExecPartial(name, ctx)
		// Set Vary header to "HX-Request" to bust the cache for HTMX requests
		// See: https://htmx.org/docs/#caching
		w.Header().Set("Vary", "HX-Request")
	} else {
		buf, err = s.tmplman.Exec(name, ctx)
	}
	if err != nil {
		log.WithError(err).Error("error executing template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		log.WithError(err).Error("error writing response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderMessage(req *http.Request, w http.ResponseWriter, ctx *Context, statusCode int, message string, args []any) {
	ctx.Error = statusCode >= http.StatusBadRequest
	if len(args) == 0 {
		ctx.Message = message
	} else {
		ctx.Message = fmt.Sprintf(message, args...)
	}
	w.WriteHeader(statusCode)
	s.render("error", req, w, ctx)
}

func (s *Server) renderError(req *http.Request, w http.ResponseWriter, statusCode int, errorMessage string, args ...any) {
	s.renderMessage(req, w, NewContext(s, req), statusCode, errorMessage, args)
}

func (s *Server) renderSuccessCtx(req *http.Request, w http.ResponseWriter, ctx *Context, successMessage string, args ...any) {
	s.renderMessage(req, w, ctx, http.StatusOK, successMessage, args)
}

func (s *Server) renderSuccess(req *http.Request, w http.ResponseWriter, successMessage string, args ...any) {
	s.renderSuccessCtx(req, w, NewContext(s, req), successMessage, args...)
}

// AddRoute ...
func (s *Server) AddRoute(method, path string, handler http.Handler) {
	s.router.Handler(method, path, handler)
}

// AddShutdownHook ...
func (s *Server) AddShutdownHook(f func()) {
	s.server.RegisterOnShutdown(f)
}

// Shutdown ...
func (s *Server) Shutdown(ctx context.Context) error {
	websub.Save()
	s.cron.Stop()
	s.tasks.Stop()
	s.fetcher.Stop()
	s.peering.Stop()
	s.followers.Close()

	if err := s.server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("error shutting down server")
		return err
	}

	if err := s.db.Close(); err != nil {
		log.WithError(err).Error("error closing store")
		return err
	}

	if err := s.peering.SaveToFile(filepath.Join(s.config.Data, "peers.json")); err != nil {
		log.WithError(err).Warn("error saving peering state")
	}

	return nil
}

// Run ...
func (s *Server) Run() (err error) {
	idleConnsClosed := make(chan struct{})

	go func() {
		if err = s.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.WithError(err).Fatal("HTTP server ListenAndServe")
		}
	}()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigch
	log.Infof("Received signal %s", sig)

	log.Info("Shutting down...")

	// We received an interrupt signal, shut down.
	if err = s.Shutdown(context.Background()); err != nil {
		// Error from closing listeners, or context timeout:
		log.WithError(err).Fatal("Error shutting down HTTP server")
	}
	close(idleConnsClosed)

	<-idleConnsClosed

	return
}

// ListenAndServe ...
func (s *Server) ListenAndServe() error {
	// If bind starts with unix:// then use unix socket
	if strings.HasPrefix(s.bind, "unix://") {
		ln, err := net.Listen("unix", s.bind[7:])
		if err != nil {
			log.WithError(err).Error("error listening on unix socket")
			return err
		}
		return s.server.Serve(ln)
	}

	_, port, err := net.SplitHostPort(s.bind)
	if err != nil {
		log.WithError(err).Errorf("error parsing bind hostport %s", s.bind)
		return err
	}

	useLetsEncrypt := s.config.TLSKey == "" && s.config.TLSCert == ""

	if s.config.TLS {
		if useLetsEncrypt && (port == "443" || port == "https") {
			log.Info("Setting up Lets Encrypt ...")

			m := &autocert.Manager{
				Cache:      autocert.DirCache(filepath.Join(s.config.Data, acmeDir)),
				Prompt:     autocert.AcceptTOS,
				Email:      s.config.AdminEmail,
				HostPolicy: autocert.HostWhitelist(s.config.baseURL.Hostname()),
			}
			s.server.TLSConfig = m.TLSConfig()

			httpServer := &http.Server{
				Addr: ":http",
				Handler: logger.New(logger.Options{
					Prefix:               "yarnd-http",
					RemoteAddressHeaders: []string{"X-Forwarded-For"},
				}).Handler(m.HTTPHandler(nil)),
			}

			go func() {
				if err := httpServer.ListenAndServe(); err != nil {
					log.WithError(err).Fatalf("error running http server")
				}
			}()

			return s.server.ListenAndServeTLS("", "")
		}
		log.Infof("Setting up TLS (key=%s cert=%s)", s.config.TLSKey, s.config.TLSCert)
		return s.server.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
	}
	log.Warn("No TLS configured")
	return s.server.ListenAndServe()
}

// AddCronJob ...
func (s *Server) AddCronJob(spec string, job cron.Job) error {
	_, err := s.cron.AddJob(spec, job)
	return err
}

func (s *Server) setupMetrics() {
	ctime := time.Now()

	// server uptime counter
	metrics.NewCounterFunc(
		"server", "uptime",
		"Number of nanoseconds the server has been running",
		func() float64 {
			return float64(time.Since(ctime).Nanoseconds())
		},
	)

	// sessions
	metrics.NewGaugeFunc(
		"server", "sessions",
		"Number of active sessions",
		func() float64 {
			return float64(s.sc.LenSessions())
		},
	)

	// dau (daily active users)
	metrics.NewGauge(
		"server", "dau",
		"Number of daily active users",
	)

	// mau (monthly active users)
	metrics.NewGauge(
		"server", "mau",
		"Number of monthly active users",
	)

	// database keys
	metrics.NewGaugeFunc(
		"db", "users",
		"Number of database /users keys",
		func() float64 {
			return float64(s.db.LenUsers())
		},
	)

	// feed cache size
	metrics.NewGaugeFunc(
		"cache", "feeds",
		"Number of unique feeds in the global feed cache",
		func() float64 {
			return float64(s.cache.FeedCount())
		},
	)

	// feed cache size
	metrics.NewGaugeFunc(
		"cache", "twts",
		"Number of active twts in the global feed cache",
		func() float64 {
			return float64(s.cache.TwtCount())
		},
	)

	// time in seconds between requesting a feed and enqueueing it
	metrics.NewSummary(
		"cache", "feed_schedule_latency_seconds",
		"Latency in seconds from when a feed fetch request is created until it is enqueued",
	)

	// time in seconds a feed request spent waiting in the queue
	metrics.NewSummary(
		"cache", "feed_queue_latency_seconds",
		"Latency in seconds that a feed fetch request spent waiting in the queue before execution",
	)

	// time in seconds to fetch & process a single feed request
	metrics.NewSummary(
		"cache", "feed_fetch_duration_seconds",
		"Duration in seconds to fetch and process a feed request",
	)

	// feed cache convergence time
	metrics.NewSummary(
		"cache", "last_convergence_seconds",
		"Number of seconds for cache convergence",
	)

	// feed cache queue_full counter
	metrics.NewCounter(
		"cache", "queue_full",
		"Number of times the queue was full",
	)

	// feed cache limited fetch (feed exceeded MaxFetchLImit or unknown size)
	metrics.NewCounter(
		"cache", "limited",
		"Number of feed cache fetches affected by MaxFetchLimit",
	)

	// no. of missing twts found in feed cache
	metrics.NewGauge(
		"cache", "missing_twts",
		"Number of missing twts found in the feed cache",
	)

	// rate of twts being corrected during convergence with peers
	metrics.NewCounter(
		"cache", "corrected_twts",
		"Number of corrected twts during convergence with peers",
	)

	// server info
	metrics.NewGaugeVec(
		"server", "info",
		"Server information",
		[]string{"full_version", "version", "commit"},
	)
	metrics.GaugeVec("server", "info").
		With(map[string]string{
			"full_version": yarn.FullVersion(),
			"version":      yarn.Version,
			"commit":       yarn.Commit,
		}).Set(1)

	// websub topics
	metrics.NewGaugeFunc(
		"websub", "topics",
		"Number of active topics subscribed",
		func() float64 {
			if websub != nil {
				return float64(websub.Stats().Topics)
			}
			return 0
		},
	)
	// websub subscribers
	metrics.NewGaugeFunc(
		"websub", "subscribers",
		"Number of subscribers created",
		func() float64 {
			if websub != nil {
				return float64(websub.Stats().Subscribers)
			}
			return 0
		},
	)
	// websub verified subscribers
	metrics.NewGaugeFunc(
		"websub", "verified",
		"Number of verified subscribers",
		func() float64 {
			if websub != nil {
				return float64(websub.Stats().Verified)
			}
			return 0
		},
	)
	// websub subscriptions
	metrics.NewGaugeFunc(
		"websub", "subscriptions",
		"Number of subscriptions created",
		func() float64 {
			if websub != nil {
				return float64(websub.Stats().Subscriptions)
			}
			return 0
		},
	)
	// websub confirmed subscriptions
	metrics.NewGaugeFunc(
		"websub", "confirmed",
		"Number of confirmed subscriptions",
		func() float64 {
			if websub != nil {
				return float64(websub.Stats().Confirmed)
			}
			return 0
		},
	)

	s.AddRoute("GET", "/metrics", metrics.Handler())
}

func (s *Server) processWebMention(source, target *url.URL, data *microformats.Data) {
	log.
		WithField("source", source).
		WithField("target", target).
		Debugf("received webmention from %s to %s", source.String(), target.String())

	getEntry := func(data *microformats.Data) (*microformats.Microformat, error) {
		if data != nil {
			for _, item := range data.Items {
				if HasString(item.Type, "h-entry") {
					return item, nil
				}
			}
		}
		return nil, errors.New("error: no entry found")
	}

	getAuthor := func(entry *microformats.Microformat) (*microformats.Microformat, error) {
		if entry != nil {
			authors := entry.Properties["author"]
			if len(authors) > 0 {
				if v, ok := authors[0].(*microformats.Microformat); ok {
					return v, nil
				}
			}
		}
		return nil, errors.New("error: no author found")
	}

	processData := func(data *microformats.Data) (name, summary, feed string, err error) {
		if data == nil {
			return
		}

		entry, err := getEntry(data)
		if err != nil {
			log.WithError(err).Error("error getting entry")
			return
		}

		if summaries := entry.Properties["summary"]; len(summaries) > 0 {
			if v, ok := summaries[0].(string); ok {
				summary = strings.TrimSpace(v)
			}
		}

		author, err := getAuthor(entry)
		if err != nil {
			log.WithError(err).Error("error getting author")
			return
		}

		if names := author.Properties["name"]; len(names) > 0 {
			if v, ok := names[0].(string); ok {
				name = strings.TrimSpace(v)
			}
		}

		for url, rel := range data.RelURLs {
			if rel.Type == "text/plain" {
				feed = url
				break
			}
		}

		return
	}

	var (
		user      *User
		userError error
	)

	if strings.HasPrefix(target.Path, "/twt/") {
		hash := strings.TrimPrefix(target.Path, "/twt/")
		if len(hash) < types.TwtHashLength {
			log.Errorf("invalid twt %s from webmention target %s", hash, target.String())
			return
		}

		bs, err := DecodeHash(hash)
		if err != nil || len(bs) < 2 {
			log.WithError(err).Errorf("error decoding twt %s from webmention target %s", hash, target.String())
			return
		}

		twt, _ := s.cache.Lookup(hash, nil)
		if twt == nil || twt.IsZero() {
			log.Errorf("invalid twt %s processing webmention from %s", hash, target.String())
			return
		}

		user, userError = GetUserFromTwter(s.config, s.db, twt.Twter())
	} else if strings.HasPrefix(target.Path, "/user/") {
		user, userError = GetUserFromURL(s.config, s.db, target.String())
	} else {
		log.Errorf("unable to process webmention from %s", target.String())
		return
	}

	if userError != nil {
		log.WithError(userError).Errorf("error loading user while processing webmention from %s", target.String())
		return
	}

	name, summary, feed, err := processData(data)
	if err != nil {
		log.WithError(err).Warnf("error processing mf2 data from %s", source)
		return
	}
	log.Debugf("name: %q", name)
	log.Debugf("summary: %q", summary)
	log.Debugf("feed: %q", feed)

	// If the mention is an ordinary WebMention (no Source Feed)
	// then inject a message notifying the user of the mention via
	// @support feed.
	// Otherwise if the mention came from a Yarn.social Pod (valid Source Feed)
	// AND if the user doesn't already follow the feed and would see the
	// mention normally, then fetch the feed as a once-off (on demand).
	if !user.Follows(feed) {
		s.fetcher.EnqueueFeeds(
			FetchFeedRequests{
				{URI: feed, Created: time.Now()},
			},
			s.config.fetchInterval,
		)
	}
}

func (s *Server) setupWebMentions() {
	webmentions = webmention.New()
	webmentions.SetCallback(s.processWebMention)
}

func (s *Server) processNotification(topic string) error {
	log.Debugf("received notification for %s", topic)

	s.fetcher.EnqueueFeeds(
		FetchFeedRequests{
			{URI: topic, Force: true, Created: time.Now()},
		},
		s.config.fetchInterval,
	)

	return nil
}

func (s *Server) setupWebSub() error {
	fn := filepath.Join(s.config.Data, "websub.json")
	endpoint := fmt.Sprintf("%s/websub", s.config.BaseURL)

	websub = indieweb.NewWebSub(fn, endpoint)
	if err := websub.Load(); err != nil {
		log.WithError(err).Warnf("error loading websub state")
	}

	websub.Notify = s.processNotification
	websub.ValidateTopic = func(topic string) bool {
		u, err := url.Parse(topic)
		if err != nil {
			log.WithError(err).Errorf("error parsing topic %q", topic)
			return false
		}

		if !s.config.IsLocalURL(topic) {
			log.Debugf("invalid topic %q (not a local url)", topic)
			return false
		}

		if !validFeedPath.MatchString(u.Path) {
			log.Debugf("invalid topic %q (not a valid feed path)", topic)
			return false
		}

		userURL := UserURL(topic)
		feedName := filepath.Base(userURL)

		if s.db.HasUser(feedName) {
			return true
		}

		log.Debugf("invalid topic %q (neither a special feed, local feed or user)", topic)

		return false
	}

	return nil
}

func (s *Server) setupJobs() error {
	InitJobs(s.config)
	for name, jobSpec := range Jobs {
		if jobSpec.Schedule == "" {
			continue
		}

		job := jobSpec.Factory(s.config, s.cache, s.fetcher, s.peering, s.db)
		if _, err := s.cron.AddJob(jobSpec.Schedule, job); err != nil {
			return fmt.Errorf("invalid cron schedule for job %s: %v (see https://pkg.go.dev/github.com/robfig/cron)", name, err)
		}
		log.Infof("Started background job %s (%s)", name, jobSpec.Schedule)
	}

	// TODO: Make this a proper job?
	s.cron.AddJob("@daily", cron.FuncJob(func() {
		cutoff := time.Now().Add(-30 * 24 * time.Hour) // prune 30‑day stale
		s.followers.PruneOlderThan(cutoff)
	}))

	return nil
}

func (s *Server) runStartupJobs() {
	Jobs["ActiveUsers"].Factory(s.config, s.cache, s.fetcher, s.peering, s.db).Run()

	time.Sleep(time.Second * 5)

	log.Info("running startup jobs")
	for name, jobSpec := range StartupJobs {
		job := jobSpec.Factory(s.config, s.cache, s.fetcher, s.peering, s.db)
		log.Infof("running %s now...", name)
		job.Run()
	}

	// Merge store
	if err := s.db.Merge(); err != nil {
		log.WithError(err).Error("error merging store")
	}
}

func (s *Server) initRoutes() {
	var (
		staticDir string
		staticFS  fs.FS
		err       error
	)

	if s.config.Theme == "" {
		staticDir = "./internal/theme/static"
		staticFS, err = fs.Sub(builtinThemeFS, "theme/static")
		if err != nil {
			log.WithError(err).Fatalf("error loading builtin theme static assets")
		}
	} else {
		staticDir = filepath.Join(s.config.Theme, "static")
		staticFS = os.DirFS(staticDir)
	}

	// To serve up arbitrary static assets in /path/to/theme/static/custom/...
	s.router.Static("/custom/*filepath", filepath.Join(staticDir, "custom"))

	if s.config.Debug {
		s.router.ServeFiles("/css/*filepath", http.Dir(filepath.Join(staticDir, "css")))
		s.router.ServeFiles("/img/*filepath", http.Dir(filepath.Join(staticDir, "img")))
		s.router.ServeFiles("/js/*filepath", http.Dir(filepath.Join(staticDir, "js")))
	} else {
		cssFS, err := fs.Sub(staticFS, "css")
		if err != nil {
			log.Fatal("error getting SubFS for static/css")
		}

		jsFS, err := fs.Sub(staticFS, "js")
		if err != nil {
			log.Fatal("error getting SubFS for static/js")
		}

		imgFS, err := fs.Sub(staticFS, "img")
		if err != nil {
			log.Fatal("error getting SubFS for static/img")
		}

		s.router.ServeFilesWithCacheControl("/css/:commit/*filepath", cssFS)
		s.router.ServeFilesWithCacheControl("/img/:commit/*filepath", imgFS)
		s.router.ServeFilesWithCacheControl("/js/:commit/*filepath", jsFS)
	}

	mdlw := metricsMiddleware.New(metricsMiddleware.Config{
		Recorder: metricsMiddlewarePrometheus.NewRecorder(
			metricsMiddlewarePrometheus.Config{
				Prefix: "yarnd",
			},
		),
		Service:       "yarnd",
		GroupedStatus: true,
	})

	s.router.NotFound = http.HandlerFunc(s.NotFoundHandler)

	s.router.GET("/about", httproutermiddleware.Handler("page", s.PageHandler("about"), mdlw))
	s.router.GET("/help", httproutermiddleware.Handler("page", s.PageHandler("help"), mdlw))
	s.router.GET("/privacy", httproutermiddleware.Handler("page", s.PageHandler("privacy"), mdlw))
	s.router.GET("/abuse", httproutermiddleware.Handler("page", s.PageHandler("abuse"), mdlw))

	s.router.GET("/", httproutermiddleware.Handler("timeline", s.TimelineHandler(), mdlw))
	s.router.HEAD("/", httproutermiddleware.Handler("timeline", s.TimelineHandler(), mdlw))
	s.router.GET("/search", httproutermiddleware.Handler("search", s.SearchHandler(), mdlw))

	s.router.GET("/robots.txt", httproutermiddleware.Handler("robots", s.RobotsHandler(), mdlw))
	s.router.HEAD("/robots.txt", httproutermiddleware.Handler("robots", s.RobotsHandler(), mdlw))

	s.router.GET("/discover", httproutermiddleware.Handler("discover", s.auther.MustAuth(s.DiscoverHandler()), mdlw))
	s.router.GET("/mentions", httproutermiddleware.Handler("mentions", s.auther.MustAuth(s.MentionsHandler()), mdlw))

	s.router.HEAD("/twt/:hash", httproutermiddleware.Handler("twt", s.PermalinkHandler(), mdlw))
	s.router.GET("/twt/:hash", httproutermiddleware.Handler("twt", s.PermalinkHandler(), mdlw))

	s.router.POST("/bookmark/:hash", httproutermiddleware.Handler("bookmark", s.auther.MustAuth(s.BookmarkHandler()), mdlw))

	s.router.HEAD("/conv/:hash", httproutermiddleware.Handler("conv", s.ConversationHandler(), mdlw))
	s.router.GET("/conv/:hash", httproutermiddleware.Handler("conv", s.ConversationHandler(), mdlw))

	// Posting, Editing and Deleting
	s.router.POST("/post", httproutermiddleware.Handler("post", s.auther.MustAuth(s.PostHandler()), mdlw))
	s.router.PATCH("/post", httproutermiddleware.Handler("post", s.auther.MustAuth(s.PostHandler()), mdlw))
	s.router.DELETE("/post", httproutermiddleware.Handler("post", s.auther.MustAuth(s.PostHandler()), mdlw))
	s.router.DELETE("/twt/:hash", httproutermiddleware.Handler("deleteTwt", s.auther.MustAuth(s.DeleteTwtHandler()), mdlw))

	// TODO: Figure out how to internally rewrite/proxy /~:nick -> /user/:nick

	// XXX: HEAD is always exposed for IndieAuth Authorization Discovery
	s.router.HEAD("/user/:nick", s.ProfileHandler())
	s.router.HEAD("/user/:nick/", s.ProfileHandler())

	if s.config.OpenProfiles {
		s.router.GET("/user/:nick/", httproutermiddleware.Handler("user", s.ProfileHandler(), mdlw))
		s.router.GET("/user/:nick/config.yaml", httproutermiddleware.Handler("user_config", s.UserConfigHandler(), mdlw))
	} else {
		s.router.GET("/user/:nick/", httproutermiddleware.Handler("user", s.auther.MustAuth(s.ProfileHandler()), mdlw))
		s.router.GET("/user/:nick/config.yaml", httproutermiddleware.Handler("user_config", s.auther.MustAuth(s.UserConfigHandler()), mdlw))
	}
	s.router.GET("/user/:nick/avatar", httproutermiddleware.Handler("avatar", CORSMiddleware(s.AvatarHandler()), mdlw))
	s.router.HEAD("/user/:nick/avatar", httproutermiddleware.Handler("avatar", CORSMiddleware(s.AvatarHandler()), mdlw))

	s.router.HEAD("/user/:nick/twtxt.txt", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.GET("/user/:nick/twtxt.txt", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.HEAD("/user/:nick/twtxt.txt/:n", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.GET("/user/:nick/twtxt.txt/:n", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))

	s.router.GET("/followers", httproutermiddleware.Handler("followers", s.FollowersHandler(), mdlw))
	s.router.GET("/user/:nick/bookmarks", httproutermiddleware.Handler("bookmarks", s.BookmarksHandler(), mdlw))

	// WebMentions
	s.router.POST("/webmention", httproutermiddleware.Handler("webmentions", s.WebMentionHandler(), mdlw))

	// WebSub
	s.router.GET("/websub", httproutermiddleware.Handler("websub", s.WebSubHandler(), mdlw))
	s.router.POST("/websub", httproutermiddleware.Handler("websub", s.WebSubHandler(), mdlw))
	s.router.GET("/notify", httproutermiddleware.Handler("notify", s.NotifyHandler(), mdlw))
	s.router.POST("/notify", httproutermiddleware.Handler("notify", s.NotifyHandler(), mdlw))

	if s.config.OpenProfiles {
		s.router.GET("/~:nick/", httproutermiddleware.Handler("user", s.ProfileHandler(), mdlw))
		s.router.GET("/~:nick/config.yaml", httproutermiddleware.Handler("user_config", s.UserConfigHandler(), mdlw))
	} else {
		s.router.GET("/~:nick/", httproutermiddleware.Handler("user", s.auther.MustAuth(s.ProfileHandler()), mdlw))
		s.router.GET("/~:nick/config.yaml", httproutermiddleware.Handler("user_config", s.auther.MustAuth(s.UserConfigHandler()), mdlw))
	}
	s.router.GET("/~:nick/avatar", httproutermiddleware.Handler("avatar", CORSMiddleware(s.AvatarHandler()), mdlw))
	s.router.HEAD("/~:nick/avatar", httproutermiddleware.Handler("avatar", CORSMiddleware(s.AvatarHandler()), mdlw))

	s.router.HEAD("/~:nick/twtxt.txt", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.GET("/~:nick/twtxt.txt", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.HEAD("/~:nick/twtxt.txt/:n", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))
	s.router.GET("/~:nick/twtxt.txt/:n", httproutermiddleware.Handler("twtxt", CORSMiddleware(s.TwtxtHandler()), mdlw))

	s.router.GET("/~:nick/following", httproutermiddleware.Handler("following", s.FollowingHandler(), mdlw))
	s.router.GET("/~:nick/bookmarks", httproutermiddleware.Handler("bookmarks", s.BookmarksHandler(), mdlw))

	// IndieAuth  Authorization Endpoint
	s.router.GET("/indieauth/auth", httproutermiddleware.Handler("indieauth_auth", s.auther.MustAuth(s.IndieAuthHandler()), mdlw))
	s.router.POST("/indieauth/auth", httproutermiddleware.Handler("indieauth_verify", s.IndieAuthVerifyHandler(), mdlw))
	s.router.GET("/indieauth/callback", httproutermiddleware.Handler("indieauth_callback", s.auther.MustAuth(s.IndieAuthCallbackHandler()), mdlw))

	// External Feeds
	s.router.GET("/external", httproutermiddleware.Handler("external", s.ExternalHandler(), mdlw))
	s.router.GET("/externalFollowing", httproutermiddleware.Handler("external_following", s.ExternalFollowingHandler(), mdlw))
	s.router.GET("/externalAvatar", httproutermiddleware.Handler("external_avatar", s.ExternalAvatarHandler(), mdlw))
	s.router.HEAD("/externalAvatar", httproutermiddleware.Handler("external_avatar", s.ExternalAvatarHandler(), mdlw))

	// External Queries (protected by a short-lived token)
	s.router.GET("/whoFollows", httproutermiddleware.Handler("whoFollows", s.WhoFollowsHandler(), mdlw))

	s.router.GET("/login", httproutermiddleware.Handler("login", s.LoginHandler(), mdlw))
	s.router.POST("/login", httproutermiddleware.Handler("login", s.LoginHandler(), mdlw))

	s.router.GET("/login/email", httproutermiddleware.Handler("login_email", s.LoginEmailHandler(), mdlw))
	s.router.POST("/login/email", httproutermiddleware.Handler("login_email", s.LoginEmailHandler(), mdlw))
	s.router.GET("/magiclinkauth", httproutermiddleware.Handler("magiclinkauth", s.MagicLinkAuthHandler(), mdlw))

	s.router.GET("/logout", httproutermiddleware.Handler("logout", s.LogoutHandler(), mdlw))
	s.router.POST("/logout", httproutermiddleware.Handler("logout", s.LogoutHandler(), mdlw))

	s.router.GET("/register", httproutermiddleware.Handler("register", s.RegisterHandler(), mdlw))
	s.router.POST("/register", httproutermiddleware.Handler("register", s.RegisterHandler(), mdlw))
	s.router.GET("/join", httproutermiddleware.Handler("join", s.PageHandler("join"), mdlw))

	// Reset Password
	s.router.GET("/resetPassword", httproutermiddleware.Handler("resetPassword", s.ResetPasswordHandler(), mdlw))
	s.router.POST("/resetPassword", httproutermiddleware.Handler("resetPassword", s.ResetPasswordHandler(), mdlw))
	s.router.GET("/newPassword", httproutermiddleware.Handler("resetPassword", s.ResetPasswordMagicLinkHandler(), mdlw))
	s.router.POST("/newPassword", httproutermiddleware.Handler("newPassword", s.NewPasswordHandler(), mdlw))

	// Media Handling
	s.router.GET("/media/:name", httproutermiddleware.Handler("media", s.MediaHandler(), mdlw))
	s.router.HEAD("/media/:name", httproutermiddleware.Handler("media", s.MediaHandler(), mdlw))
	s.router.POST("/upload", httproutermiddleware.Handler("upload", s.auther.MustAuth(s.UploadMediaHandler()), mdlw))

	// Task State
	s.router.GET("/task/:uuid", httproutermiddleware.Handler("task", s.TaskHandler(), mdlw))

	// User/Feed Lookups
	s.router.GET("/lookup", httproutermiddleware.Handler("lookup", s.auther.MustAuth(s.LookupHandler()), mdlw))

	s.router.GET("/follow", httproutermiddleware.Handler("follow", s.auther.MustAuth(s.FollowHandler()), mdlw))
	s.router.POST("/follow", httproutermiddleware.Handler("follow", s.auther.MustAuth(s.FollowHandler()), mdlw))

	s.router.GET("/unfollow", httproutermiddleware.Handler("unfollow", s.auther.MustAuth(s.UnfollowHandler()), mdlw))
	s.router.POST("/unfollow", httproutermiddleware.Handler("unfollow", s.auther.MustAuth(s.UnfollowHandler()), mdlw))

	s.router.GET("/mute", httproutermiddleware.Handler("mute", s.auther.MustAuth(s.MuteHandler()), mdlw))
	s.router.POST("/mute", httproutermiddleware.Handler("mute", s.auther.MustAuth(s.MuteHandler()), mdlw))
	s.router.GET("/muted", httproutermiddleware.Handler("muted", s.auther.MustAuth(s.MutedHandler()), mdlw))
	s.router.GET("/unmute", httproutermiddleware.Handler("unmute", s.auther.MustAuth(s.UnmuteHandler()), mdlw))
	s.router.POST("/unmute", httproutermiddleware.Handler("unmute", s.auther.MustAuth(s.UnmuteHandler()), mdlw))

	s.router.GET("/user/:nick/following", httproutermiddleware.Handler("following", s.FollowingHandler(), mdlw))

	s.router.POST("/mute/:hash", httproutermiddleware.Handler("mute", s.auther.MustAuth(s.MuteHandler()), mdlw))
	s.router.GET("/unmute/:hash", httproutermiddleware.Handler("unmute", s.auther.MustAuth(s.UnmuteHandler()), mdlw))
	s.router.POST("/unmute/:hash", httproutermiddleware.Handler("unmute", s.auther.MustAuth(s.UnmuteHandler()), mdlw))

	s.router.GET("/settings", httproutermiddleware.Handler("settings", s.auther.MustAuth(s.SettingsHandler()), mdlw))
	s.router.POST("/settings", httproutermiddleware.Handler("settings", s.auther.MustAuth(s.SettingsHandler()), mdlw))
	s.router.POST("/settings/addlink", httproutermiddleware.Handler("settings_addlink", s.auther.MustAuth(s.SettingsAddLinkHandler()), mdlw))
	s.router.POST("/settings/removelink", httproutermiddleware.Handler("settings_removelink", s.auther.MustAuth(s.SettingsRemoveLinkHandler()), mdlw))

	s.router.GET("/info", httproutermiddleware.Handler("info", s.PodInfoHandler(), mdlw))
	s.router.GET("/logo", httproutermiddleware.Handler("logo", s.PodLogoHandler(), mdlw))
	s.router.GET("/config", httproutermiddleware.Handler("config", s.auther.MustAuth(s.PodConfigHandler()), mdlw))
	s.router.GET("/manage/pod", httproutermiddleware.Handler("manage_pod", s.auther.MustAuth(s.ManagePodHandler()), mdlw))
	s.router.GET("/manage/jobs", httproutermiddleware.Handler("manage_jobs", s.auther.MustAuth(s.ManageJobsHandler()), mdlw))
	s.router.POST("/manage/jobs", httproutermiddleware.Handler("manage_jobs", s.auther.MustAuth(s.ManageJobsHandler()), mdlw))
	s.router.GET("/manage/peers", httproutermiddleware.Handler("manage_peers", s.auther.MustAuth(s.ManagePeersHandler()), mdlw))
	s.router.GET("/manage/delpeer", httproutermiddleware.Handler("delpeer", s.auther.MustAuth(s.DeletePeerHandler()), mdlw))
	s.router.POST("/manage/delpeer", httproutermiddleware.Handler("delpeer", s.auther.MustAuth(s.DeletePeerHandler()), mdlw))
	s.router.POST("/manage/trspeer", httproutermiddleware.Handler("trspeer", s.auther.MustAuth(s.TrustPeerHandler()), mdlw))
	s.router.POST("/manage/pod", httproutermiddleware.Handler("manage_pod", s.auther.MustAuth(s.ManagePodHandler()), mdlw))

	s.router.GET("/manage/users", httproutermiddleware.Handler("manager_users", s.auther.MustAuth(s.ManageUsersHandler()), mdlw))
	s.router.POST("/manage/adduser", httproutermiddleware.Handler("adduser", s.auther.MustAuth(s.AddUserHandler()), mdlw))
	s.router.POST("/manage/deluser", httproutermiddleware.Handler("deluser", s.auther.MustAuth(s.DelUserHandler()), mdlw))
	s.router.POST("/manage/rstuser", httproutermiddleware.Handler("rstuser", s.auther.MustAuth(s.RstUserHandler()), mdlw))

	s.router.POST("/delete", httproutermiddleware.Handler("delete", s.auther.MustAuth(s.DeleteHandler()), mdlw))

	// Support / Report Abuse handlers
	s.router.GET("/support", httproutermiddleware.Handler("support", s.SupportHandler(), mdlw))
	s.router.POST("/support", httproutermiddleware.Handler("support", s.SupportHandler(), mdlw))
	s.router.GET("/_captcha", httproutermiddleware.Handler("captcha", s.CaptchaHandler(), mdlw))
	s.router.GET("/report", httproutermiddleware.Handler("report", s.ReportHandler(), mdlw))
	s.router.POST("/report", httproutermiddleware.Handler("report", s.ReportHandler(), mdlw))
	s.router.GET("/contact", httproutermiddleware.Handler("contact", s.PageHandler("contact"), mdlw))

	// WebFinger support
	s.router.GET(webfinger.WebFingerPath, httproutermiddleware.Handler("webfinger", s.WebFingerHandler(), mdlw))
}

// NewServer ...
func NewServer(bind string, options ...Option) (*Server, error) {
	config := NewConfig()

	for _, opt := range options {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	settingsFn := filepath.Join(config.Data, "settings.yaml")
	if FileExists(settingsFn) {
		if settings, err := LoadSettings(settingsFn); err != nil {
			log.Warnf("error loading pod settings from %s: %s", settingsFn, err)
		} else {
			if err := merger.MergeOverwrite(config, settings); err != nil {
				log.WithError(err).Error("error merging pod settings")
				return nil, err
			}
		}
	}

	if err := config.Validate(); err != nil {
		log.WithError(err).Error("error validating config")
		return nil, fmt.Errorf("error validating config: %w", err)
	}

	cache, err := NewSqliteCache(filepath.Join(config.Data, "cache.db"))
	if err != nil {
		log.WithError(err).Error("error creating new feed cache")
		return nil, err
	}

	fetcher := NewFeedFetcher(config, cache)

	peering := NewPeerManager(config)
	if err := peering.LoadFromFile(filepath.Join(config.Data, "peers.json")); err != nil {
		log.WithError(err).Warn("error loading peering state")
	}

	db, err := NewStore(config.Store)
	if err != nil {
		log.WithError(err).Error("error creating store")
		return nil, err
	}
	restoreFn := filepath.Join(config.Data, restoreFilename)
	if FileExists(restoreFn) {
		log.Info("restoring database from backup...")
		if err := db.Restore(restoreFn); err != nil {
			log.WithError(err).Errorf("error restoring database from backup from %s", restoreFn)
			return nil, err
		}
	}

	followers := NewFollowers(config)

	// translator
	translator, err := NewTranslator()
	if err != nil {
		log.WithError(err).Error("error loading translator")
		return nil, err
	}

	tmplman, err := NewTemplateManager(config, translator, cache)
	if err != nil {
		log.WithError(err).Error("error creating template manager")
		return nil, err
	}

	router := NewRouter()

	var auther auth.Auther

	switch config.Auth {
	case "proxy":
		auther = auth.NewProxyAuth(config.AuthHeader, NewUserCreator(config, db))
		router.Use(auther.MustAuth)
	case "session":
		auther = auth.NewSessionAuth("/login")
	default:
		log.Fatalf("error: unsupported auth type %q", config.Auth)
	}

	tasks := tasks.NewDispatcher(10, 100) // TODO: Make this configurable?

	pm := passwd.New(
		argon2.Argon2id, // preferred.
		argon2.Argon2i,
		scrypt.Simple,
	)

	sc := NewSessionStore(db, config.SessionCacheTTL)

	sm := sessions.NewManager(
		sessions.NewOptions(
			"yarnd_token",
			config.CookieSecret,
			config.LocalURL().Scheme == "https",
			config.SessionExpiry,
		),
		sc,
	)
	sm.ExemptPath("/robots.txt")
	sm.ExemptGlob("/css/*")
	sm.ExemptGlob("/img/*")
	sm.ExemptGlob("/js/*")
	sm.ExemptGlob("/user/*/twtxt.txt")
	sm.ExemptGlob("/~*/twtxt.txt")
	sm.ExemptGlob("/user/*/avatar")
	sm.ExemptGlob("/~*/avatar")

	api := NewAPI(router, config, cache, fetcher, followers, db, pm, tasks)

	var handler http.Handler

	csrfHandler := nosurf.New(router)
	csrfHandler.ExemptRegexp(regexp.MustCompile(`\/api\/v1\/.*`))
	csrfHandler.ExemptGlob("/indieauth/*")
	csrfHandler.ExemptPath("/webmention")
	csrfHandler.ExemptPath("/websub")
	csrfHandler.ExemptPath("/notify")

	// Useful for Safari / Mobile Safari when behind Cloudflare to streaming
	// videos _actually_ works :O
	if config.DisableGzip {
		handler = sm.Handler(csrfHandler)
	} else {
		handler = gziphandler.GzipHandler(sm.Handler(csrfHandler))
	}

	// Create a PeerDetector instance (using your configuration and peering manager):
	peerDetector := NewPeerDetector(config, peering)
	// Wrap the existing handler chain with the peer detection middleware.
	handler = PeerDetectionHandler(peerDetector, handler)

	if !config.DisableLogger {
		handler = logger.New(logger.Options{
			Prefix:               "yarnd",
			RemoteAddressHeaders: []string{"X-Forwarded-For"},
		}).Handler(handler)
	}

	server := &Server{
		bind:    bind,
		config:  config,
		router:  router,
		tmplman: tmplman,

		server: &http.Server{Addr: bind, Handler: handler},

		// API
		api: api,

		// Feed cache
		cache: cache,

		// Feed Fetcher
		fetcher: fetcher,

		// Peering
		peering: peering,

		// Data Store
		db: db,

		// Followers
		followers: followers,

		// Schedular
		cron: cron.New(),

		// Dispatcher
		tasks: tasks,

		// Auther
		auther: auther,

		// Session Manager
		sc: sc,
		sm: sm,

		// Password Manager
		pm: pm,

		// Translator
		translator: translator,
	}

	// Factory functions that require access to the Pod Config and Store
	server.AppendTwt = AppendTwtFactory(config, cache, db)

	if err := server.setupWebSub(); err != nil {
		log.WithError(err).Error("error setting up websub")
		return nil, err
	}
	log.Infof("started websub processor")

	if err := server.setupJobs(); err != nil {
		log.WithError(err).Error("error setting up background jobs")
		return nil, err
	}
	server.cron.Start()
	log.Info("started background jobs")

	server.tasks.Start()
	log.Info("started task dispatcher")

	server.fetcher.Start()
	log.Info("started feed fetchaer")

	server.peering.Start()
	log.Info("started peering")

	server.setupWebMentions()
	log.Infof("started webmentions processor")

	server.setupMetrics()
	log.Infof("serving metrics endpoint at %s/metrics", server.config.BaseURL)

	// Log interesting configuration options
	log.Infof("Debug: %t", server.config.Debug)
	log.Infof("Instance Name: %s", server.config.Name)
	log.Infof("Base URL: %s", server.config.BaseURL)
	log.Infof("Using Theme: %s", server.config.Theme)
	log.Infof("Using %s Auth (Header: %s)", server.config.Auth, server.config.AuthHeader)
	log.Infof("Admin User: %s", server.config.AdminUser)
	log.Infof("Admin Name: %s", server.config.AdminName)
	log.Infof("Admin Email: %s", server.config.AdminEmail)
	log.Infof("Max Twts per Page: %d", server.config.TwtsPerPage)
	log.Infof("Fetch Interval: %s", server.config.FetchInterval)
	log.Infof("Max Age Days : %d", server.config.MaxAgeDays)
	log.Infof("Maximum length of Posts: %d", server.config.MaxTwtLength)
	log.Infof("Open User Profiles: %t", server.config.OpenProfiles)
	log.Infof("Open Registrations: %t", server.config.OpenRegistrations)
	log.Infof("Support Enabled: %t", !server.config.DisableSupport)
	log.Infof("Disable Gzip: %t", server.config.DisableGzip)
	log.Infof("Disable Logger: %t", server.config.DisableLogger)
	log.Infof("Disable Media: %t", server.config.DisableMedia)
	log.Infof("Disable FFMpeg: %t", server.config.DisableFfmpeg)
	log.Infof("SMTP Host: %s", server.config.SMTPHost)
	log.Infof("SMTP Port: %d", server.config.SMTPPort)
	log.Infof("SMTP User: %s", server.config.SMTPUser)
	log.Infof("SMTP From: %s", server.config.SMTPFrom)
	log.Infof("Max Fetch Limit: %s", humanize.Bytes(uint64(server.config.MaxFetchLimit)))
	log.Infof("Max Feed Size: %s", humanize.Bytes(uint64(server.config.MaxFeedSize)))
	log.Infof("Max Upload Size: %s", humanize.Bytes(uint64(server.config.MaxUploadSize)))
	log.Infof("API Session Time: %s", server.config.APISessionTime)
	log.Infof("Enabled Features: %s", server.config.Features)

	// Warn about user registration being disabled.
	if !server.config.OpenRegistrations {
		log.Warn("registrations are disabled as per configuration (no -R/--open-registrations)")
	}

	// Warn about `ffmpeg` not installed or available
	if !CmdExists("ffmpeg") {
		log.Warn("ffmpeg not found, audio and video support will be disabled")
		server.config.DisableFfmpeg = true
	}

	server.initRoutes()
	api.initRoutes()

	go server.runStartupJobs()

	return server, nil
}

func (s *Server) tr(ctx *Context, msgID string, data ...interface{}) string {
	return s.translator.Translate(ctx, msgID, data...)
}
