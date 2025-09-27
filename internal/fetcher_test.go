// internal/fetcher_test.go
package internal

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yarn.social/types"
)

// --- fake ProtocolFetcher ---------------------------------------------------

type fakeProto struct {
	resp        *http.Response
	err         error
	lastMethod  string
	lastURI     string
	lastHeaders http.Header
}

func (f *fakeProto) FetchHTTP(_ *Config, method, uri string, headers http.Header) (*http.Response, error) {
	f.lastMethod = method
	f.lastURI = uri
	f.lastHeaders = headers.Clone()
	return f.resp, f.err
}
func (f *fakeProto) FetchGopher(_ *Config, _ string) (responseWrapper, error) {
	return responseWrapper{Body: io.NopCloser(bytes.NewBuffer(nil))}, nil
}
func (f *fakeProto) FetchGemini(_ *Config, _ string) (responseWrapper, error) {
	return responseWrapper{Body: io.NopCloser(bytes.NewBuffer(nil))}, nil
}

// --- fake Cacher ------------------------------------------------------------

type fakeCache struct {
	// for HTTP 200 path
	updatedFeedURI string
	updatedLastMod string
	updatedTwts    types.Twts

	// for deferred UpdateCachedFeed
	updatedCachedFeed *Feed

	// the in-memory feed object
	feed *Feed
}

func newFakeCache(seed *Feed) *fakeCache {
	return &fakeCache{feed: seed}
}

func (f *fakeCache) FeedCount() int                   { return 0 }
func (f *fakeCache) TwtCount() int                    { return 0 }
func (f *fakeCache) HasFeed(_ string) bool            { return false }
func (f *fakeCache) FindTwter(_ string) *types.Twter  { return nil }
func (f *fakeCache) GetTwter(_ string) *types.Twter   { return nil }
func (f *fakeCache) SetTwter(_ string, _ types.Twter) {}

func (f *fakeCache) GetCachedFeed(_ string) *Feed {
	return f.feed
}
func (f *fakeCache) GetAllCachedFeeds() (map[string]*Feed, error) {
	return map[string]*Feed{}, nil
}

func (f *fakeCache) GetOrSetCachedFeed(_ string) (*Feed, bool) {
	// always treat as “new” so deferred UpdateCachedFeed fires
	return f.feed, true
}

func (f *fakeCache) UpdateFeed(uri, lastModified string, twts types.Twts) error {
	f.updatedFeedURI = uri
	f.updatedLastMod = lastModified
	f.updatedTwts = twts
	return nil
}

func (f *fakeCache) UpdateCachedFeed(_ string, cached *Feed) error {
	// capture a copy
	f.updatedCachedFeed = &Feed{
		Errors:       cached.Errors,
		Fetches:      cached.Fetches,
		LastError:    cached.LastError,
		LastFetched:  cached.LastFetched,
		LastModified: cached.LastModified,
	}
	return nil
}

func (f *fakeCache) DeleteFeeds(_ ...string)            {}
func (f *fakeCache) RenameFeed(_, _ string)             {}
func (f *fakeCache) ShouldRefreshFeed(_, _ string) bool { return true }
func (f *fakeCache) Delete(...string)                   {}
func (f *fakeCache) Lookup(_ string, _ *QueryOptions) (types.Twt, bool) {
	return types.NilTwt, false
}
func (f *fakeCache) Search(_ string, _ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) GetAll(_ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) GetMentions(_ string, _ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) GetBySubject(_ string, _ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) GetByFeeds(_ []string, _ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) GetByURL(_ string, _ *QueryOptions) (types.Twts, int, error) {
	return nil, 0, nil
}
func (f *fakeCache) MissingRootSubjects() ([]string, error) { return nil, nil }
func (f *fakeCache) FilterByMissingSubjects(_ types.Twts) (types.Twts, error) {
	return nil, nil
}

// --- Helpers ---------------------------------------------------------------

func makeConf() *Config {
	return &Config{
		BaseURL:          "http://example.test",
		MaxFetchLimit:    1024,
		QueueBufferSize:  1,
		MaxCacheFetchers: 1,
	}
}

func makeFetcher(proto ProtocolFetcher) *FeedFetcher {
	return &FeedFetcher{Proto: proto}
}

func mustURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// --- TESTS ------------------------------------------------------------------

func TestFetchFeed_HTTP200(t *testing.T) {
	seed := &Feed{}
	cache := newFakeCache(seed)

	fake := &fakeProto{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"text/plain"}, "Last-Modified": {"Wed, 05 May 2025 15:00:00 GMT"}},
			Body:       io.NopCloser(bytes.NewBufferString("2022-10-30T23:20:41+10:00	Hello World\n")),
			Request:    &http.Request{URL: mustURL("http://f/test")},
		},
	}

	FetchFeed(makeConf(), cache, makeFetcher(fake), FetchFeedRequest{
		URI:   "http://f/test",
		Force: true,
	})

	// 1) UpdateFeed was called with the new Last-Modified :contentReference[oaicite:0]{index=0}:contentReference[oaicite:1]{index=1}
	assert.Equal(t, "http://f/test", cache.updatedFeedURI)
	assert.Equal(t, "Wed, 05 May 2025 15:00:00 GMT", cache.updatedLastMod)
	assert.Len(t, cache.updatedTwts, 1)

	// 2) Deferred UpdateCachedFeed saw one fetch and the new timestamp
	uc := cache.updatedCachedFeed
	require.NotNil(t, uc)
	assert.EqualValues(t, 1, uc.Fetches)
	assert.Equal(t, "Wed, 05 May 2025 15:00:00 GMT", uc.LastModified)
}

func TestFetchFeed_HTTP304_NotModified(t *testing.T) {
	seed := &Feed{LastModified: "Tue, 04 May 2025 10:00:00 GMT"}
	cache := newFakeCache(seed)

	fake := &fakeProto{
		resp: &http.Response{
			StatusCode: http.StatusNotModified,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Request:    &http.Request{URL: mustURL("http://u/test")},
		},
	}

	FetchFeed(makeConf(), cache, makeFetcher(fake), FetchFeedRequest{
		URI:   "http://u/test",
		Force: true,
	})

	// header
	require.NotNil(t, fake.lastHeaders)
	assert.Equal(t, seed.LastModified, fake.lastHeaders.Get("If-Modified-Since"))

	uc := cache.updatedCachedFeed
	require.NotNil(t, uc)
	assert.EqualValues(t, 1, uc.Fetches)
	assert.EqualValues(t, 0, uc.Errors)
	assert.Equal(t, seed.LastModified, uc.LastModified)
}

func TestFetchFeed_DeadFeed404(t *testing.T) {
	seed := &Feed{}
	cache := newFakeCache(seed)

	fake := &fakeProto{
		resp: &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Request:    &http.Request{URL: mustURL("http://d/test")},
		},
	}

	FetchFeed(makeConf(), cache, makeFetcher(fake), FetchFeedRequest{
		URI:   "http://d/test",
		Force: true,
	})

	uc := cache.updatedCachedFeed
	require.NotNil(t, uc)
	assert.EqualValues(t, 0, uc.Fetches)
	assert.EqualValues(t, 1, uc.Errors)
	assert.NotEmpty(t, uc.LastError)
}
