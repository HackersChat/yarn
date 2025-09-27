// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.mills.io/tasks"
	"go.sour.is/passwd"

	"go.yarn.social/types"
)

// ContextKey is the type of the context key
type ContextKey int

const (
	// TokenContextKey is the context key
	TokenContextKey ContextKey = iota

	// UserContextKey is the context key
	UserContextKey
)

var (
	// ErrInvalidCredentials is returned for invalid credentials against /auth
	ErrInvalidCredentials = errors.New("error: invalid credentials")

	// ErrInvalidToken is returned for expired or invalid tokens used in Authorization headers
	ErrInvalidToken = errors.New("error: invalid token")
)

// Token ...
type Token struct {
	Signature string
	Value     string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// API ...
type API struct {
	router    *Router
	config    *Config
	cache     Cacher
	fetcher   *FeedFetcher
	followers *Followers
	db        Store
	pm        *passwd.Passwd
	tasks     *tasks.Dispatcher
}

// NewAPI ...
func NewAPI(router *Router, config *Config, cache Cacher, fetcher *FeedFetcher, followers *Followers, db Store, pm *passwd.Passwd, tasks *tasks.Dispatcher) *API {
	api := &API{router, config, cache, fetcher, followers, db, pm, tasks}

	api.initRoutes()

	return api
}

func (a *API) initRoutes() {
	router := a.router.Group("/api/v1")
	router.Use(CORSMiddleware)

	router.GET("/ping", a.PingEndpoint())
	router.POST("/auth", a.AuthEndpoint())
	router.POST("/register", a.RegisterEndpoint())
	router.GET("/config", a.PodConfigEndpoint())

	router.GET("/whoami", a.isAuthorized(a.WhoAmIEndpoint()))

	router.POST("/post", a.isAuthorized(a.PostEndpoint()))
	router.POST("/upload", a.isAuthorized(a.UploadMediaEndpoint()))
	router.POST("/sync", a.isAuthorized(a.SyncEndpoint()))

	router.GET("/settings", a.isAuthorized(a.SettingsEndpoint()))
	router.POST("/settings", a.isAuthorized(a.SettingsEndpoint()))

	router.POST("/follow", a.isAuthorized(a.FollowEndpoint()))
	router.POST("/unfollow", a.isAuthorized(a.UnfollowEndpoint()))

	router.POST("/mute", a.isAuthorized(a.MuteEndpoint()))
	router.POST("/unmute", a.isAuthorized(a.UnmuteEndpoint()))

	router.POST("/timeline", a.isAuthorized(a.TimelineEndpoint()))
	router.POST("/discover", a.DiscoverEndpoint())

	router.GET("/profile", a.ProfileEndpoint())
	router.GET("/profile/:username", a.ProfileEndpoint())
	router.POST("/fetch-twts", a.FetchTwtsEndpoint())
	router.POST("/conv", a.ConversationEndpoint())

	router.POST("/external", a.ExternalProfileEndpoint())

	router.POST("/mentions", a.isAuthorized(a.MentionsEndpoint()))

	// Debugging Endpoints
	debug := router.Group("/debug", a.isAdminUser, a.isAuthorized)
	debug.GET("/websub", a.DebugWebSubEndpoint())
	debug.GET("/heap", a.DebugHeapEndpoint())
	debug.GET("/db", a.DebugDBEndpoint())
	debug.POST("/restore", a.DebugRestoreEndpoint())

	// Admin Endpoints
	admin := router.Group("/admin", a.isAdminUser, a.isAuthorized)
	admin.POST("/deluser/:username", a.AdminDeleteUserEndpoint())

	// Cache Endpoints
	cache := router.Group("/cache", a.isAdminUser, a.isAuthorized)
	cache.POST("/delete", a.CacheDelete())

	// Support / Report endpoints
	router.POST("/support", a.isAuthorized(a.SupportEndpoint()))
	router.POST("/report", a.isAuthorized(a.ReportEndpoint()))
}

// CreateToken ...
func (a *API) CreateToken(user *User, r *http.Request) (*Token, error) {
	claims := jwt.MapClaims{}
	claims["username"] = user.Username
	createdAt := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.config.APISigningKey))
	if err != nil {
		log.WithError(err).Error("error creating signed token")
		return nil, err
	}

	signedToken, err := jwt.Parse(tokenString, a.jwtKeyFunc)
	if err != nil {
		log.WithError(err).Error("error creating signed token")
		return nil, err
	}

	tkn := &Token{
		Signature: signedToken.Signature,
		Value:     tokenString,
		UserAgent: r.UserAgent(),
		CreatedAt: createdAt,
	}

	return tkn, nil
}

func (a *API) jwtKeyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("there was an error")
	}
	return []byte(a.config.APISigningKey), nil
}

func (a *API) getLoggedInUser(r *http.Request) *User {
	token, err := jwt.Parse(r.Header.Get("Token"), a.jwtKeyFunc)
	if err != nil {
		return nil
	}

	if !token.Valid {
		return nil
	}

	claims := token.Claims.(jwt.MapClaims)

	username := claims["username"].(string)

	user, err := a.db.GetUser(username)
	if err != nil {
		log.WithError(err).Error("error loading user object")
		return nil
	}

	// Every registered new user follows themselves
	// TODO: Make  this configurable server behaviour?
	if user.Following == nil {
		user.Following = make(map[string]string)
	}

	user.Follow(user.Username, user.URL)

	return user
}

func (a *API) isAuthorized(endpoint httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if r.Header.Get("Token") == "" {
			http.Error(w, "No Token Provided", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(r.Header.Get("Token"), a.jwtKeyFunc)
		if err != nil {
			log.WithError(err).Error("error parsing token")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if token.Valid {
			claims := token.Claims.(jwt.MapClaims)

			username := claims["username"].(string)

			user, err := a.db.GetUser(username)
			if err != nil {
				log.WithError(err).Error("error loading user object")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Every registered new user follows themselves
			// TODO: Make  this configurable server behaviour?
			if user.Following == nil {
				user.Following = make(map[string]string)
			}
			user.Follow(user.Username, user.URL)

			ctx := context.WithValue(r.Context(), TokenContextKey, token)
			ctx = context.WithValue(ctx, UserContextKey, user)

			// TODO: Use event sourcing for this?
			user.LastSeenAt = time.Now().Round(24 * time.Hour)
			if err := a.db.SetUser(user.Username, user); err != nil {
				log.WithError(err).Warnf("error updating user.LastSeenAt for %s", user.Username)
			}

			endpoint(w, r.WithContext(ctx), p)
		} else {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}
	}
}

func (a *API) isAdminUser(endpoint httprouter.Handle) httprouter.Handle {
	isAdminUser := IsAdminUserFactory(a.config)

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		if !isAdminUser(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		endpoint(w, r, p)
	}
}

// PingEndpoint ...
func (a *API) PingEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// RegisterEndpoint ...
func (a *API) RegisterEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if !a.config.OpenRegistrations {
			http.Error(w, "ErrorRegisterDisabled", http.StatusForbidden)
			return
		}

		req, err := types.NewRegisterRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing register request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := NormalizeUsername(req.Username)
		password := req.Password
		// XXX: We DO NOT store this! (EVER)
		email := strings.TrimSpace(req.Email)

		if err := ValidateUsername(username); err != nil {
			http.Error(w, "Bad Username", http.StatusBadRequest)
			return
		}

		if a.db.HasUser(username) {
			http.Error(w, "Bad Username", http.StatusBadRequest)
			return
		}

		hash, err := a.pm.Passwd([]byte(password), nil)
		if err != nil {
			log.WithError(err).Error("error creating password hash")
			http.Error(w, "Password Creation Failed", http.StatusInternalServerError)
			return
		}

		recoveryHash := fmt.Sprintf("email:%s", FastHashString(email))

		user := &User{
			Username:     username,
			PasswordHash: hash,
			Recovery:     recoveryHash,
			URL:          URLForUser(a.config.BaseURL, username),
			CreatedAt:    time.Now(),
		}

		if err := a.db.SetUser(username, user); err != nil {
			log.WithError(err).Error("error saving user object for new user")
			http.Error(w, "User Creation Failed", http.StatusInternalServerError)
			return
		}
	}
}

// AuthEndpoint ...
func (a *API) AuthEndpoint() httprouter.Handle {
	// #239: Throttle failed login attempts and lock user  account.
	failures := NewTTLCache(5 * time.Minute)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewAuthRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing auth request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := NormalizeUsername(req.Username)
		password := req.Password

		// Error: no username or password provided
		if username == "" || password == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Lookup user
		user, err := a.db.GetUser(username)
		if err != nil {
			log.WithField("username", username).Warn("login attempt from non-existent user")
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}

		// #239: Throttle failed login attempts and lock user  account.
		if failures.Get(user.Username) > MaxFailedLogins {
			http.Error(w, "Account Locked", http.StatusTooManyRequests)
			return
		}

		// Validate cleartext password against KDF hash
		_, err = a.pm.Passwd([]byte(password), user.PasswordHash)
		if err != nil {
			// #239: Throttle failed login attempts and lock user  account.
			failed := failures.Inc(user.Username)
			time.Sleep(time.Duration(IntPow(2, failed)) * time.Second)

			log.WithField("username", username).Warn("login attempt with invalid credentials")
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}

		// #239: Throttle failed login attempts and lock user  account.
		failures.Reset(user.Username)

		// Login successful
		log.WithField("username", username).Info("login successful")

		token, err := a.CreateToken(user, r)
		if err != nil {
			log.WithError(err).Error("error creating token")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		res := types.AuthResponse{Token: token.Value}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// PostEndpoint ...
func (a *API) PostEndpoint() httprouter.Handle {
	appendTwt := AppendTwtFactory(a.config, a.cache, a.db)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewPostRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing post request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		originalText := req.Text
		text := CleanTwt(originalText)

		if text == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		twt, err := appendTwt(user, text)

		if err != nil {
			log.WithError(err).Error("error posting twt")
			if err == ErrFeedImposter {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		feedURL := a.config.URLForUser(user.Username)

		// Update user's own timeline with their own new post.
		a.cache.UpdateFeed(feedURL, "", types.Twts{twt})

		// PostResponse
		w.Header().Set("Content-Type", "application/json")
		if twt != nil {
			p := types.PostResponse{
				Text:    originalText,
				Created: twt.Created().Format(time.RFC3339),
				Hash:    twt.Hash(),
			}
			body, err := p.Bytes()
			if err != nil {
				log.WithError(err).Error("error serializing response")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(body)
		} else {
			_, _ = w.Write([]byte(`{}`))
		}
	}
}

// SyncEndpoint ...
func (a *API) SyncEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewSyncRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing sync request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		res := types.SyncResponse{}

		merged, err := MergeFeed(a.config, req.Feed, req.Body, req.Delete)
		if err != nil {
			res.Success = false
			res.Error = fmt.Errorf("error merging feed: %w", err).Error()
		} else {
			res.Success = true
			res.Merged = merged
		}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		sources := user.Source()

		// Update user's own timeline with their newly megedd feed.
		a.fetcher.EnqueueFeeds(sources, a.config.fetchInterval)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// TimelineEndpoint ...
func (a *API) TimelineEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewPagedRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing paged request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Compute page, limit and offset values.
		page := req.Page
		if page < 1 {
			page = 1
		}
		limit := a.config.TwtsPerPage
		offset := (page - 1) * limit

		// Get the timeline tweets by user with database paging.
		var feeds []string
		for _, source := range user.Sources() {
			feeds = append(feeds, source.URI)
		}
		twts, total, err := a.cache.GetByFeeds(feeds, &QueryOptions{Limit: limit, Offset: offset})
		if err != nil {
			log.WithError(err).Error("error loading timeline")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if there are no tweets (possibly due to muted feeds).
		if len(twts) == 0 {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Calculate the maximum number of pages.
		maxPages := (total + limit - 1) / limit

		// Build the paged response.
		res := types.PagedResponse{
			Twts: twts,
			Pager: types.PagerResponse{
				Current:   page,
				MaxPages:  maxPages,
				TotalTwts: total,
			},
		}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// DiscoverEndpoint ...
func (a *API) DiscoverEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewPagedRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing paged request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Compute page, limit, and offset.
		page := req.Page
		if page < 1 {
			page = 1
		}
		limit := a.config.TwtsPerPage
		offset := (page - 1) * limit

		// Get the discover tweets from the database.
		twts, total, err := a.cache.GetAll(&QueryOptions{Limit: limit, Offset: offset})
		if err != nil {
			log.WithError(err).Error("error loading discover")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// If no tweets are found, return a 404.
		if len(twts) == 0 {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Compute maximum number of pages.
		maxPages := (total + limit - 1) / limit

		res := types.PagedResponse{
			Twts: twts,
			Pager: types.PagerResponse{
				Current:   page,
				MaxPages:  maxPages,
				TotalTwts: total,
			},
		}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// MentionsEndpoint ...
func (a *API) MentionsEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewPagedRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing paged request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		user := r.Context().Value(UserContextKey).(*User)

		// Compute page, limit and offset.
		page := req.Page
		if page < 1 {
			page = 1
		}
		limit := a.config.TwtsPerPage
		offset := (page - 1) * limit

		// Get mentions with database-driven paging.
		twts, total, err := a.cache.GetMentions(user.URL, &QueryOptions{Limit: limit, Offset: offset})
		if err != nil {
			log.WithError(err).Error("error loading mentions")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Compute the maximum number of pages.
		maxPages := (total + limit - 1) / limit

		res := types.PagedResponse{
			Twts: twts,
			Pager: types.PagerResponse{
				Current:   page,
				MaxPages:  maxPages,
				TotalTwts: total,
			},
		}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// FollowEndpoint ...
func (a *API) FollowEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewFollowRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing follow request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := NormalizeFeedName(req.Nick)
		url := NormalizeURL(req.URL)

		if nick == "" || url == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if err := user.FollowAndValidate(a.config, url); err != nil {
			log.WithError(err).Errorf("error validating new feed @<%s %s>", nick, url)
			http.Error(w, "Invalid Feed", http.StatusBadRequest)
			return
		}

		if err := a.db.SetUser(user.Username, user); err != nil {
			log.WithError(err).Error("error saving user object")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// UnfollowEndpoint ...
func (a *API) UnfollowEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewUnfollowRequest(r.Body)

		if err != nil {
			log.WithError(err).Error("error parsing follow request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := req.Nick

		if nick == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		user := r.Context().Value(UserContextKey).(*User)

		if user == nil {
			log.Fatalf("user not found in context")
		}

		if _, ok := user.Following[nick]; !ok {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if err := user.Unfollow(a.config, nick); err != nil {
			log.WithError(err).Errorf("error unfollowing feed %s", nick)
			http.Error(w, "Invalid Feed", http.StatusBadRequest)
			return
		}

		if err := a.db.SetUser(user.Username, user); err != nil {
			log.WithError(err).Warnf("error updating user object for user  %s", user.Username)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// SettingsEndpoint ...
func (a *API) SettingsEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Limit request body to to abuse
		r.Body = http.MaxBytesReader(w, r.Body, a.config.MaxUploadSize)
		defer r.Body.Close()

		user := r.Context().Value(UserContextKey).(*User)

		if r.Method == http.MethodGet {
			data, err := json.Marshal(user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
			return
		}

		// XXX: We DO NOT store this! (EVER)
		email := strings.TrimSpace(r.FormValue("email"))
		tagline := strings.TrimSpace(r.FormValue("tagline"))
		password := r.FormValue("password")

		isFollowingPubliclyVisible := r.FormValue("isFollowingPubliclyVisible") == "on"

		avatarFile, _, err := r.FormFile("avatar_file")
		if err != nil && err != http.ErrMissingFile {
			log.WithError(err).Error("error parsing form file")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if password != "" {
			hash, err := a.pm.Passwd([]byte(password), nil)
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
				Width:  a.config.AvatarResolution,
				Height: a.config.AvatarResolution,
			}
			_, err = StoreUploadedImage(
				a.config, avatarFile,
				avatarsDir, user.Username,
				opts,
			)
			if err != nil {
				log.WithError(err).Error("error updating user avatar")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			avatarFn := filepath.Join(a.config.Data, avatarsDir, fmt.Sprintf("%s.png", user.Username))
			if avatarHash, err := FastHashFile(avatarFn); err == nil {
				user.AvatarHash = avatarHash
			} else {
				log.WithError(err).Warnf("error updating avatar hash for %s", user.Username)
			}
		}

		recoveryHash := fmt.Sprintf("email:%s", FastHashString(email))

		user.Recovery = recoveryHash
		user.Tagline = tagline

		// XXX: Commented out as these are more specific to the Web App currently.
		// API clients such as Goryon (the Flutter iOS/Android app) have their own mechanisms.
		// user.Theme = theme
		// user.DisplayDatesInTimezone = displayDatesInTimezone
		// user.DisplayImagesPreferences = displayImagesPreference

		user.IsFollowingPubliclyVisible = isFollowingPubliclyVisible

		if err := a.db.SetUser(user.Username, user); err != nil {
			log.WithError(err).Error("error updating user object")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// UploadMediaEndpoint handles the uploading of media files to the server.
// It supports image, audio, and video files by determining the content type
// of the uploaded file. The function processes the file accordingly and
// dispatches a task for further processing. For older clients (pre v1.0.3),
// it redirects to the OldUploadMediaEndpoint. The function also ensures
// the request body does not exceed the maximum upload size defined in the
// configuration, returning appropriate HTTP responses for errors encountered
// during file handling or task dispatching.
func (a *API) UploadMediaEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Limit request body to to abuse
		r.Body = http.MaxBytesReader(w, r.Body, a.config.MaxUploadSize)

		mfile, headers, err := r.FormFile("media_file")
		if err != nil && err != http.ErrMissingFile {
			if err.Error() == "http: request body too large" {
				http.Error(w, "Media Upload Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			log.WithError(err).Error("error parsing form file")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if mfile == nil || headers == nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		ctype := headers.Header.Get("Content-Type")

		var uri URI

		if strings.HasPrefix(ctype, "image/") {
			fn, err := ReceiveImage(mfile, a.config.MaxUploadSize)
			if err != nil {
				log.WithError(err).Error("error writing uploaded image")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uuid, err := a.tasks.Dispatch(NewImageTask(a.config, fn))
			if err != nil {
				log.WithError(err).Error("error dispatching image processing task")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			uri.Type = "taskURI"
			uri.Path = URLForTask(a.config.BaseURL, uuid)
		}

		if strings.HasPrefix(ctype, "audio/") {
			fn, err := ReceiveAudio(mfile, a.config.MaxUploadSize)
			if err != nil {
				log.WithError(err).Error("error writing uploaded audio")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uuid, err := a.tasks.Dispatch(NewAudioTask(a.config, fn))
			if err != nil {
				log.WithError(err).Error("error dispatching audio transcoding task")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			uri.Type = "taskURI"
			uri.Path = URLForTask(a.config.BaseURL, uuid)
		}

		if strings.HasPrefix(ctype, "video/") {
			fn, err := ReceiveVideo(mfile, a.config.MaxUploadSize)
			if err != nil {
				log.WithError(err).Error("error writing uploaded video")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uuid, err := a.tasks.Dispatch(NewVideoTask(a.config, fn))
			if err != nil {
				log.WithError(err).Error("error dispatching vodeo transcode task")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			uri.Type = "taskURI"
			uri.Path = URLForTask(a.config.BaseURL, uuid)
		}

		if uri.IsZero() {
			log.Warnf("unable to detect media type: %s", uri)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		data, err := json.Marshal(uri)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if uri.Type == "taskURI" {
			w.WriteHeader(http.StatusAccepted)
		}
		_, _ = w.Write(data)
	}
}

// WhoAmIEndpoint returns an HTTP handler that responds with the username
// of the currently logged-in user in a JSON format. If the user is not
// logged in, it returns an HTTP 401 Unauthorized error. This endpoint
// helps clients determine the identity of the authenticated user.
func (a *API) WhoAmIEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		loggedInUser := a.getLoggedInUser(r)
		if loggedInUser == nil {
			http.Error(w, "You are not logged in", http.StatusUnauthorized)
			return
		}

		res := types.WhoAmIResponse{Username: loggedInUser.Username}

		body, err := res.Bytes()
		if err != nil {
			log.WithError(err).Error("error serializing response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// ProfileEndpoint ...
func (a *API) ProfileEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		loggedInUser := a.getLoggedInUser(r)

		username := NormalizeUsername(p.ByName("username"))
		if username == "" {
			if loggedInUser != nil {
				username = loggedInUser.Username
			}
		}

		if username == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var profile types.Profile

		if a.db.HasUser(username) {
			user, err := a.db.GetUser(username)
			if err != nil {
				log.WithError(err).Errorf("error loading user object for %s", username)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			profile = user.Profile(a.config.BaseURL, loggedInUser)
			profile.FollowedBy = loggedInUser.Follows(profile.URI)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		if !a.cache.HasFeed(profile.URI) {
			a.fetcher.EnqueueFeeds(
				FetchFeedRequests{
					{URI: profile.URI, Created: time.Now()},
				},
				a.config.fetchInterval,
			)
		}

		var twter types.Twter

		if cachedTwter := a.cache.GetTwter(profile.URI); cachedTwter != nil {
			twter = *cachedTwter
		} else {
			twter = types.Twter{Nick: profile.Nick, URI: profile.URI}
		}

		// FIXME: Fix this.
		var followers types.Followers
		profile.Followers = followers
		profile.NFollowers = len(followers)

		profileResponse := types.ProfileResponse{
			Profile: profile,
			Twter:   twter,
		}

		data, err := json.Marshal(profileResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// ConversationEndpoint ...
func (a *API) ConversationEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		loggedInUser := a.getLoggedInUser(r)

		req, err := types.NewConversationRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing conversation request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		hash := req.Hash
		if hash == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		opts := &QueryOptions{
			Exclude: loggedInUser.MutedList(),
		}

		twt, inCache := a.cache.Lookup(hash, opts)
		if twt.IsZero() {
			http.Error(w, "Conversation Not Found", http.StatusNotFound)
			return
		}

		page := req.Page
		if page < 1 {
			page = 1
		}
		opts.Limit = a.config.TwtsPerPage
		opts.Offset = (page - 1) * opts.Limit

		// For conversation, we load all the conversation twts
		// by using a high limit since conversation threads tend to be small.
		convTwts, _, err := a.cache.GetBySubject("#"+hash, opts)
		if err != nil {
			log.WithError(err).Error("error loading conversation twts")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// If the original twt wasn't in cache then add it.
		if !inCache {
			convTwts = append(convTwts, twt)
		}

		// Sort the conversation in reverse order (most recent first).
		sort.Sort(sort.Reverse(convTwts))

		// Now do in-memory pagination on the resulting slice.
		perPage := a.config.TwtsPerPage
		total := len(convTwts)

		start := (page - 1) * perPage
		if start > total {
			start = total
		}
		end := start + perPage
		if end > total {
			end = total
		}
		pagedTwts := convTwts[start:end]

		// Compute maximum pages.
		maxPages := (total + perPage - 1) / perPage

		res := types.PagedResponse{
			Twts: pagedTwts,
			Pager: types.PagerResponse{
				Current:   page,
				MaxPages:  maxPages,
				TotalTwts: total,
			},
		}

		data, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// FetchTwtsEndpoint ...
func (a *API) FetchTwtsEndpoint() httprouter.Handle {
	isLocal := IsLocalURLFactory(a.config)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		loggedInUser := a.getLoggedInUser(r)

		req, err := types.NewFetchTwtsRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing fetch twts request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := NormalizeUsername(req.Nick)
		if nick == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var profile types.Profile
		var twts types.Twts
		var total int

		// Compute page, limit, and offset.
		page := req.Page
		if page < 1 {
			page = 1
		}
		limit := a.config.TwtsPerPage
		offset := (page - 1) * limit

		// Choose which branch to run based on request values.
		if req.URL != "" && !isLocal(req.URL) {
			// If the feed is not local and not yet in our cache,
			// enqueue it for fetching.
			if !a.cache.HasFeed(req.URL) {
				a.fetcher.EnqueueFeeds(
					FetchFeedRequests{
						{URI: req.URL, Created: time.Now()},
					},
					a.config.fetchInterval,
				)
			}

			twts, total, err = a.cache.GetByURL(req.URL, &QueryOptions{Limit: limit, Offset: offset})
			if err != nil {
				log.WithError(err).Error("error loading twts by URL")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else if a.db.HasUser(nick) {
			user, err := a.db.GetUser(nick)
			if err != nil {
				log.WithError(err).Errorf("error loading user object for %s", nick)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			profile = user.Profile(a.config.BaseURL, loggedInUser)
			twts, total, err = a.cache.GetByURL(profile.URI, &QueryOptions{Limit: limit, Offset: offset})
			if err != nil {
				log.WithError(err).Error("error loading twts by URL")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Compute maximum number of pages.
		maxPages := (total + limit - 1) / limit

		res := types.PagedResponse{
			Twts: twts,
			Pager: types.PagerResponse{
				Current:   page,
				MaxPages:  maxPages,
				TotalTwts: total,
			},
		}

		data, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// ExternalProfileEndpoint ...
func (a *API) ExternalProfileEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		loggedInUser := a.getLoggedInUser(r)
		req, err := types.NewExternalProfileRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing external profile request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		uri := req.URL
		nick := req.Nick

		if uri == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if !a.cache.HasFeed(uri) {
			a.tasks.DispatchFunc(func() error {
				a.fetcher.EnqueueFeeds(
					FetchFeedRequests{
						{URI: uri, Created: time.Now()},
					},
					a.config.fetchInterval,
				)
				return nil
			})
		}

		var twter types.Twter

		if cachedTwter := a.cache.GetTwter(uri); cachedTwter != nil {
			twter = *cachedTwter
		} else {
			twter = types.Twter{Nick: nick, URI: uri}
		}

		// Set nick to what the user follows as (if any)
		nick = loggedInUser.FollowsAs(uri)

		// If no nick provided try to guess a suitable nick
		// from the feed or some heuristics from the feed's URI
		// (borrowed from Yarns)
		if nick == "" {
			if twter.Nick != "" {
				nick = twter.Nick
			} else {
				// TODO: Move this logic into types/lextwt and types/retwt
				if u, err := url.Parse(uri); err == nil {
					if strings.HasSuffix(u.Path, "/twtxt.txt") {
						if rest := strings.TrimSuffix(u.Path, "/twtxt.txt"); rest != "" {
							nick = strings.Trim(rest, "/")
						} else {
							nick = u.Hostname()
						}
					} else if strings.HasSuffix(u.Path, ".txt") {
						base := filepath.Base(u.Path)
						if name := strings.TrimSuffix(base, filepath.Ext(base)); name != "" {
							nick = name
						} else {
							nick = u.Hostname()
						}
					} else {
						nick = Slugify(uri)
					}
				}
			}
		}

		var follows types.Follows
		for nick, twter := range twter.Follow {
			follows = append(follows, types.Follow{Nick: nick, URI: twter.URI})
		}

		profile := types.Profile{
			Type: "External",

			Nick:        nick,
			Description: twter.Tagline,
			Avatar:      URLForExternalAvatar(a.config, uri),
			URI:         uri,

			Following:  follows,
			NFollowing: twter.Following,
			NFollowers: twter.Followers,

			ShowFollowing: true,
			ShowFollowers: true,

			Follows:    loggedInUser.Follows(uri),
			FollowedBy: a.followers.IsFollowedBy(uri, loggedInUser.Username),
			Muted:      loggedInUser.HasMuted(uri),
		}

		profileResponse := types.ProfileResponse{
			Profile: profile,
			Twter:   twter,
		}

		data, err := json.Marshal(profileResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// MuteEndpoint ...
func (a *API) MuteEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewMuteRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing mute request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := req.Nick
		url := req.URL

		if nick == "" || url == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		user.Mute(nick, url)

		if err := a.db.SetUser(user.Username, user); err != nil {
			log.WithError(err).Error("error updating user object")
			http.Error(w, "User Update Failed", http.StatusInternalServerError)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))

	}
}

// UnmuteEndpoint ...
func (a *API) UnmuteEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user := r.Context().Value(UserContextKey).(*User)

		req, err := types.NewUnmuteRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing unmute request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := req.Nick

		if nick == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		user.Unmute(nick)

		if err := a.db.SetUser(user.Username, user); err != nil {
			log.WithError(err).Error("error updating user object")
			http.Error(w, "User Update Failed", http.StatusInternalServerError)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// SupportEndpoint ...
func (a *API) SupportEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewSupportRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing support request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name := req.Name
		email := req.Email
		subject := req.Subject
		message := req.Message

		if err := SendSupportRequestEmail(a.config, name, email, subject, message); err != nil {
			log.WithError(err).Errorf("unable to send support email for %s", email)
			log.WithError(err).Error("error sending support request")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Infof("support message email sent for %s", email)

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))

	}
}

// ReportEndpoint ...
func (a *API) ReportEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		req, err := types.NewReportRequest(r.Body)
		if err != nil {
			log.WithError(err).Error("error parsing report request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		nick := req.Nick
		url := req.URL

		if nick == "" || url == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name := req.Name
		email := req.Email
		category := req.Category
		message := req.Message

		if err := SendReportAbuseEmail(a.config, nick, url, name, email, category, message); err != nil {
			log.WithError(err).Errorf("unable to send report email for %s", email)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// No real response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

// PodConfigEndpoint ...
func (a *API) PodConfigEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		data, err := json.Marshal(a.config.Settings())
		if err != nil {
			log.WithError(err).Error("error serializing pod config response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

//
// Debug Endpoints
//

// DebugWebSubEndpoint ...
func (a *API) DebugWebSubEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		websub.DebugEndpoint(w, r)
	}
}

// DebugHeapEndpoint ...
func (a *API) DebugHeapEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		p := pprof.Lookup("heap")
		if p == nil {
			http.Error(w, "Unknown Profile", http.StatusNotFound)
			return
		}

		if gc, _ := strconv.Atoi(r.FormValue("gc")); gc > 0 {
			runtime.GC()
		}

		debug, _ := strconv.Atoi(r.FormValue("debug"))
		if debug != 0 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="heap"`)
		}
		if err := p.WriteTo(w, debug); err != nil {
			log.WithError(err).Error("error serializing pod cache")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// DebugDBEndpoint ...
func (a *API) DebugDBEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		if err := a.db.DB().ForEach(exportKey(a.db.DB(), w)); err != nil {
			log.WithError(err).Error("error dumping database")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// DebugRestoreEndpoint ...
func (a *API) DebugRestoreEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		tx := a.db.DB().Transaction()
		defer tx.Commit()

		defer req.Body.Close()
		scanner := bufio.NewScanner(req.Body)
		for scanner.Scan() {
			var kv kvPair

			if err := json.Unmarshal(scanner.Bytes(), &kv); err != nil {
				tx.Discard()
				log.WithError(err).Error("error parsing input: %w", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			key, err := base64.StdEncoding.DecodeString(kv.Key)
			if err != nil {
				tx.Discard()
				log.WithError(err).Errorf("error decoding key for %q", kv.Key)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			value, err := base64.StdEncoding.DecodeString(kv.Value)
			if err != nil {
				tx.Discard()
				log.WithError(err).Errorf("error decoding value for %q", kv.Value)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if err := tx.Put(key, value); err != nil {
				tx.Discard()
				log.WithError(err).Error("error storing key-value pair: %w", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			tx.Discard()
			log.WithError(err).Error("error reading input: %w", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// AdminDeleteUserEndpoint ...
func (a *API) AdminDeleteUserEndpoint() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		username := NormalizeUsername(p.ByName("username"))
		if username == "" {
			log.Warn("no username specified")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if err := DeleteUser(a.config, a.cache, a.db, username); err != nil {
			log.WithError(err).WithFields(log.Fields{"Username": username}).Error("error deleting user")
			if errors.Is(err, ErrUserNotFound) {
				http.Error(w, "User Not Found", http.StatusNotAcceptable)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		actor := r.Context().Value(UserContextKey).(*User)
		log.Infof("user %s deleted by %s from %s", username, actor.Username, r.RemoteAddr)
	}
}

// CacheDelete ...
func (a *API) CacheDelete() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		uri := NormalizeURL(r.FormValue("uri"))
		if uri == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		a.cache.DeleteFeeds(uri)
		actor := r.Context().Value(UserContextKey).(*User)
		log.Infof("cached feed %s deleted by %s from %s", uri, actor.Username, r.RemoteAddr)
	}
}
