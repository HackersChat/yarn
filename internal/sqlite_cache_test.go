// sqlite_cache_test.go
package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yarn.social/types"
)

// newTestCache creates a new in‑memory SqliteCache for testing.
func newTestCache(t *testing.T) *SqliteCache {
	t.Helper()

	cache, err := NewSqliteCache(":memory:")
	require.NoError(t, err)
	require.NotNil(t, cache)
	return cache
}

// newTestTwt creates a new test Twt for testing.
func newTestTwt(t *testing.T, nick, uri string, created time.Time, content string) types.Twt {
	t.Helper()

	twter := types.Twter{Nick: nick, URI: uri}
	twt := types.MakeTwt(twter, created, content)

	return twt
}

func TestNewSqliteCache(t *testing.T) {
	cache := newTestCache(t)
	defer cache.Close()

	// A new cache should start with no feeds.
	assert.Equal(t, 0, cache.FeedCount(), "new cache should have 0 feeds")
}

func TestUpdateFeedAndCounts(t *testing.T) {
	cache := newTestCache(t)
	defer cache.Close()

	// Create a new twt and add it to the feed.
	twt := newTestTwt(t, "tester", "http://example.com", time.Now(), "Hello World!")
	twts := types.Twts{twt}

	// Ensure a Twter exists before adding it to the feed.
	cache.SetTwter(twt.Twter().URI, twt.Twter())

	// Update the feed with the twts.
	err := cache.UpdateFeed(twt.Twter().URI, "", twts)
	require.NoError(t, err)

	// After update there should be one feed and one twt.
	assert.Equal(t, 1, cache.FeedCount(), "feed count should be 1 after update")
	assert.Equal(t, 1, cache.TwtCount(), "twt count should be 1 after update")
	assert.True(t, cache.HasFeed(twt.Twter().URI), "feed should be cached")
}
