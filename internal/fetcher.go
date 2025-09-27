package internal

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"git.mills.io/yarnsocial/yarn"
	"github.com/dustin/go-humanize"
	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
	"golang.org/x/time/rate"
)

var (
	// XXX: Remove this once feeds grows support for websub
	alwaysRefreshDomains = []string{
		"feeds.twtxt.net",
	}
)

// FetchFeedRequest is an single request for a twtxt.txt feed with a canonical Nickname and URL for the feed
// and optional request parameters that affect how the Cache fetches the feed.
type FetchFeedRequest struct {
	// URI of the feed
	URI string

	// Followers is a list of local usernames following this feed publicly
	Followers []string

	// Created Time when the request was first generated
	Created time.Time

	// Enqueued time when the request was enqueued into the work queue
	Enqueued time.Time

	// Started time when a worker began processing this request
	Started time.Time

	// Force whether or not to immediately fetch the feed and bypass Cache.ShouldRefreshFeed().
	Force bool
}

// String implements the Stringer interface and returns the Feed represented
// as a twtxt.txt URI in the form @<nick url>.
func (f FetchFeedRequest) String() string {
	return fmt.Sprintf("FetchFeedRequest: @<%s>", f.URI)
}

// FetchFeedRequests is a list of unique FetchFeedRequest items.
type FetchFeedRequests []FetchFeedRequest

// Feed represents a cached feed
type Feed struct {
	Errors       int64
	Fetches      int64
	LastError    string
	LastFetched  time.Time
	LastModified string
}

// Availability returns the feed's availability (success rate) in terms of
// successful fetches / total attempted fetches as a percentage (0-100).
func (f *Feed) Availability() float64 {
	total := float64(f.Fetches + f.Errors)
	if total == 0 {
		return 100
	}
	good := float64(f.Fetches)
	return good / total * 100
}

// Errored sets the error for the feed and resets the last error time.
func (f *Feed) Errored(err error) {
	f.Errors++
	f.LastError = err.Error()
}

// Fetched marks the feed as fetched and resets the last error time.
func (f *Feed) Fetched() {
	f.Fetches++
	f.LastError = ""
	f.LastFetched = time.Now()
}

// ProtocolFetcher is an interface for fetching data from a protocol.
// It is used by the FetchFeed function to fetch data from various protocols such as HTTP, Gopher, and Gemini.
// The FetchHTTP method is used to fetch data from HTTP URLs, the FetchGopher method is used to fetch data from Gopher URLs,
// and the FetchGemini method is used to fetch data from Gemini
type ProtocolFetcher interface {
	// for HTTP(S)
	FetchHTTP(conf *Config, method, uri string, headers http.Header) (*http.Response, error)
	// for gopher://
	FetchGopher(conf *Config, uri string) (responseWrapper, error)
	// for gemini:// (if you ever support it)
	FetchGemini(conf *Config, uri string) (responseWrapper, error)
}

// defaultFetcher bridges to your existing globals:
type defaultFetcher struct{}

func (d defaultFetcher) FetchHTTP(conf *Config, method, uri string, headers http.Header) (*http.Response, error) {
	return RequestHTTP(conf, method, uri, headers)
}

func (d defaultFetcher) FetchGopher(conf *Config, uri string) (responseWrapper, error) {
	// RequestGopher is assumed to return (*gopher.Response, error)
	resp, err := RequestGopher(conf, uri)
	if err != nil {
		return responseWrapper{}, err
	}
	return responseWrapper{Body: resp.Body}, nil
}

func (d defaultFetcher) FetchGemini(conf *Config, uri string) (responseWrapper, error) {
	// RequestGemini is assumed to return (*gemini.Response, error)
	resp, err := RequestGemini(conf, uri)
	if err != nil {
		return responseWrapper{}, err
	}
	return responseWrapper{Body: resp.Body}, nil
}

func safeCloseBody(body io.ReadCloser) {
	if body != nil {
		_, _ = io.Copy(io.Discard, body)
		_ = body.Close()
	}
}

// ensureAvatar fetches an external avatar if the twter's Avatar is not local.
func ensureAvatar(conf *Config, twter *types.Twter) {
	if !conf.IsLocalURL(twter.Avatar) {
		GetExternalAvatar(conf, *twter)
	}
}

// processFeedBody reads from the body, parses the feed, and returns the parsed twts.
func processFeedBody(body io.ReadCloser, conf *Config, twter *types.Twter, feed FetchFeedRequest) ([]types.Twt, error) {
	limitedReader := &io.LimitedReader{R: body, N: conf.MaxFetchLimit}
	tf, err := types.ParseFile(limitedReader, twter)
	// Always ensure the body is closed after reading.
	safeCloseBody(body)
	if err != nil {
		return nil, err
	}
	// If the limited reader's N is 0, we may have exceeded the fetch limit.
	if limitedReader.N <= 0 {
		log.Warnf("feed size possibly exceeds MaxFetchLimit of %s for %s", humanize.Bytes(uint64(conf.MaxFetchLimit)), feed)
		metrics.Counter("cache", "limited").Inc()
	}
	_, _, twts := types.SplitTwts(tf.Twts(), 0, 0)
	return twts, nil
}

// processHTTPFeed processes an HTTP response's body by delegating to processFeedBody.
func processHTTPFeed(res *http.Response, conf *Config, twter *types.Twter, feed FetchFeedRequest) ([]types.Twt, error) {
	return processFeedBody(res.Body, conf, twter, feed)
}

// responseWrapper is a common wrapper for responses that provide an io.ReadCloser.
type responseWrapper struct {
	Body io.ReadCloser
}

// processNonHTTPFeed consolidates common processing for non-HTTP feed responses.
// After inserting the feed’s Twts, we also opportunistically
// pull in any missing-subject tweets by following @mentions.
func processNonHTTPFeed(
	requestFunc func(conf *Config, uri string) (responseWrapper, error),
	conf *Config,
	feed FetchFeedRequest,
	twter *types.Twter,
	cachedFeed *Feed,
	cache Cacher,
) {
	resp, err := requestFunc(conf, feed.URI)
	if err != nil {
		cachedFeed.Errored(err)
		return
	}
	twts, err := processFeedBody(resp.Body, conf, twter, feed)
	if err != nil {
		cachedFeed.Errored(err)
		return
	}
	cachedFeed.Fetched()
	ensureAvatar(conf, twter)
	cache.SetTwter(feed.URI, *twter)
	cache.UpdateFeed(feed.URI, "", twts)
}

// FetchFeed fetches a feed from the specified URL and updates the cache with the fetched Twts.
func FetchFeed(conf *Config, cache Cacher, fetcher *FeedFetcher, feed FetchFeedRequest) {
	// mark processing start
	feed.Started = time.Now()
	defer func() {
		// fetch duration: time from processing start to completion
		metrics.Summary("cache", "feed_fetch_duration_seconds").Observe(time.Since(feed.Started).Seconds())
	}()

	twter := cache.GetTwter(feed.URI)
	cachedFeed, newFeed := cache.GetOrSetCachedFeed(feed.URI)
	if cachedFeed == nil {
		log.Errorf("failed to get or create cache for %s; skipping feed", feed.URI)
		return
	}

	defer func() {
		if newFeed && cachedFeed.Errors != 0 {
			log.Warnf("deleting invalid or bad feed: %s", feed)
			cache.DeleteFeeds(feed.URI)
		}
		if err := cache.UpdateCachedFeed(feed.URI, cachedFeed); err != nil {
			log.WithError(err).Errorf("error updating cached feed %s", feed)
		}
	}()

	if twter == nil {
		twter = &types.Twter{URI: feed.URI}
		if !conf.IsLocalURL(feed.URI) {
			GetExternalAvatar(conf, *twter)
		}
	}

	// Handle Feed Refresh:
	// 1) Use the refresh interval if set by the feed author.
	// 2) Use an exponential back-off based on update frequency (TBD).
	// 3) Force refresh if FetchFeedRequest.Force is true.
	if !feed.Force && !cache.ShouldRefreshFeed(conf.BaseURL, feed.URI) {
		log.Debugf("feed %s has no refresh interval, skipping", feed)
		return
	}

	// Handle Gopher feeds using the consolidated non-HTTP helper.
	if strings.HasPrefix(feed.URI, "gopher://") {
		processNonHTTPFeed(fetcher.Proto.FetchGopher, conf, feed, twter, cachedFeed, cache)
		return
	}

	// Handle Gemini feeds using the consolidated non-HTTP helper.
	if strings.HasPrefix(feed.URI, "gemini://") {
		processNonHTTPFeed(fetcher.Proto.FetchGemini, conf, feed, twter, cachedFeed, cache)
		return
	}

	// Handle HTTP feeds.
	headers := make(http.Header)
	headers.Set("Accept", "text/plain")
	if len(feed.Followers) > 0 {
		var userAgent string
		if len(feed.Followers) == 1 {
			userAgent = fmt.Sprintf("yarnd/%s (+%s; @%s)",
				yarn.FullVersion(),
				URLForUser(conf.BaseURL, feed.Followers[0]), feed.Followers[0])
		} else {
			userAgent = fmt.Sprintf("yarnd/%s (~%s; contact=%s)",
				yarn.FullVersion(),
				URLForWhoFollows(conf.BaseURL, feed, len(feed.Followers)),
				URLForPage(conf.BaseURL, "support"))
		}
		headers.Set("User-Agent", userAgent)
	}
	if cachedFeed.LastModified != "" {
		headers.Set("If-Modified-Since", cachedFeed.LastModified)
	}

	res, err := fetcher.Proto.FetchHTTP(conf, http.MethodGet, feed.URI, headers)
	if err != nil {
		log.WithError(err).Errorf("error fetching feed %s", feed)
		cachedFeed.Errored(err)
		return
	}

	actualURL := res.Request.URL.String()
	if actualURL == "" {
		safeCloseBody(res.Body)
		log.WithField("feed", feed).Warnf("%s trying to redirect to an empty url", feed)
		return
	}
	if actualURL != feed.URI {
		log.Warnf("feed %s has moved to %s", feed, actualURL)
		cache.RenameFeed(feed.URI, actualURL)
		feed.URI = actualURL
		if twter.URI != feed.URI {
			twter.URI = feed.URI
		}
	}

	if ctype := res.Header.Get("Content-Type"); ctype != "" {
		mediaType, _, err := mime.ParseMediaType(ctype)
		if err != nil {
			safeCloseBody(res.Body)
			log.WithError(err).Error("unable to parse content-type")
			cachedFeed.Errored(err)
			return
		}
		if mediaType != "text/plain" {
			safeCloseBody(res.Body)
			err := fmt.Errorf("error: feed responded with unexpected Content-Type: %s", ctype)
			log.WithError(err).Warnf("skipping invalid feed")
			cachedFeed.Errored(err)
			return
		}
	}

	switch res.StatusCode {
	case http.StatusOK: // 200
		twts, err := processHTTPFeed(res, conf, twter, feed)
		if err != nil {
			log.WithError(err).Errorf("error processing feed %s, aborting", feed)
			cachedFeed.Errored(err)
			return
		}
		// processHTTPFeed already closes res.Body.
		cachedFeed.Fetched()
		ensureAvatar(conf, twter)
		lastModified := res.Header.Get("Last-Modified")
		cachedFeed.LastModified = lastModified
		cache.SetTwter(feed.URI, *twter)
		cache.UpdateFeed(feed.URI, lastModified, twts)
	case http.StatusNotModified: // 304
		safeCloseBody(res.Body)
		cachedFeed.Fetched()
		return
	case 401, 402, 403, 404, 407, 410, 451:
		safeCloseBody(res.Body)
		cachedFeed.Errored(types.ErrDeadFeed{Reason: res.Status})
	default:
		safeCloseBody(res.Body)
		err := fmt.Errorf("error: feed responded with unexpected status code: %d", res.StatusCode)
		log.WithError(err).Warnf("skipping invalid feed")
		cachedFeed.Errored(err)
	}
}

// FeedFetcher is a struct that manages the fetching of feeds from a given URL.
// It uses a fixed worker pool to process feeds concurrently.
// The worker pool is responsible for fetching feeds, updating the cache,
// and handling errors.
type FeedFetcher struct {
	Proto ProtocolFetcher

	conf      *Config
	cache     Cacher
	feedQueue chan FetchFeedRequest
	wg        sync.WaitGroup
	id        string
	shutdown  chan struct{}
}

// NewFeedFetcher initializes a new FeedFetcher with a fixed worker pool.
func NewFeedFetcher(conf *Config, cache Cacher) *FeedFetcher {
	buffer := conf.QueueBufferSize
	if buffer <= 0 {
		// e.g. allow each worker ~50 queued items, but at least 100
		buffer = max(100, conf.MaxCacheFetchers*50)
	}

	return &FeedFetcher{
		Proto: defaultFetcher{},

		conf:      conf,
		cache:     cache,
		feedQueue: make(chan FetchFeedRequest, buffer),
		id:        GenerateRandomToken()[:4],
		shutdown:  make(chan struct{}),
	}
}

// Start launches the worker pool.
func (f *FeedFetcher) Start() {
	log.Infof("[cache-%s] Starting FeedFetcher worker pool with %d workers", f.id, f.conf.MaxCacheFetchers)
	for i := 0; i < f.conf.MaxCacheFetchers; i++ {
		f.wg.Add(1)
		go f.worker(i)
	}
}

// worker handles feed processing until shutdown is signaled.
func (f *FeedFetcher) worker(workerID int) {
	defer f.wg.Done()
	for {
		select {
		case <-f.shutdown:
			log.Infof("[cache-%s] Worker %d shutting down", f.id, workerID)
			return
		case feed, ok := <-f.feedQueue:
			if !ok {
				return
			}
			// queue latency: time from enqueue to processing start
			queueLatency := time.Since(feed.Enqueued).Seconds()
			metrics.Summary("cache", "feed_queue_latency_seconds").Observe(queueLatency)
			FetchFeed(f.conf, f.cache, f, feed)
		}
	}
}

// Stop signals shutdown and waits for all workers to finish.
func (f *FeedFetcher) Stop() {
	close(f.shutdown)
	close(f.feedQueue)
	f.wg.Wait()
	log.Infof("[cache-%s] FeedFetcher worker pool shutdown complete", f.id)
}

// EnqueueFeeds submits a batch of feeds to be fetched.
func (f *FeedFetcher) EnqueueFeeds(feeds FetchFeedRequests, interval time.Duration) FetchFeedRequests {
	total := len(feeds)
	if total == 0 {
		return nil
	}

	// shape to both total window and worker pool size
	workers := f.conf.MaxCacheFetchers
	limiter := rate.NewLimiter(
		rate.Every(interval/time.Duration(total)), // average = N/T
		workers, // burst = W
	)

	seen := make(map[string]bool)
	var dropped FetchFeedRequests

	for _, req := range feeds {
		req.URI = NormalizeURL(req.URI)
		if seen[req.URI] || f.conf.BlockedFeed(req.URI) {
			continue
		}
		seen[req.URI] = true

		// schedule latency: time from creation to now
		if !req.Created.IsZero() {
			scheduleLatency := time.Since(req.Created).Seconds()
			metrics.Summary("cache", "feed_schedule_latency_seconds").Observe(scheduleLatency)
		}

		// wait for next token (or immediate if in the burst)
		if err := limiter.Wait(context.Background()); err != nil {
			log.WithError(err).Errorf("[cache-%s] rate limit exceeded for %s", f.id, req)
			break // context cancelled or something else went wrong
		}

		// stamp enqueue time
		req.Enqueued = time.Now()
		select {
		case f.feedQueue <- req:
		default:
			log.Warnf("[cache-%s] dropping %s; queue full", f.id, req)
			metrics.Counter("cache", "queue_full").Inc()
			dropped = append(dropped, req)
		}
	}

	return dropped
}
