package internal

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	// sqlite database/sql driver
	_ "github.com/glebarez/sqlite"
	lru "github.com/hashicorp/golang-lru"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/logrusadapter"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

const (
	maxLookupCache = 1024 // maximum tweet objects to cache
	maxTwterCache  = 1024 // maximum twter objects to cache
)

//go:embed sqlite_cache.sql
var schemaSQL string

func convertToAny[T any](items []T) []any {
	result := make([]any, len(items))
	for i, v := range items {
		result[i] = v
	}
	return result
}

// isExcluded checks if any of the provided identifiers match an entry in the exclusion list.
func isExcluded(feedURL, subject, hash string, exclude []string) bool {
	for _, ex := range exclude {
		if ex == feedURL || ex == subject || ex == hash {
			return true
		}
	}
	return false
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.WithError(err).Warnf("Invalid time format: %s", s)
		return time.Time{}
	}
	return t
}

// SlowLogger is a logger that logs queries with a specified threshold. It wraps another logger and logs queries that take longer than the threshold. The logger is passed as a parameter. The method returns the logger.
type SlowLogger struct {
	inner         sqldblogger.Logger
	slowThreshold time.Duration
}

// NewSlowLogger creates a new SlowLogger instance with the specified database path and slow threshold. The logger is passed as a parameter. The method returns the logger.
func NewSlowLogger(inner sqldblogger.Logger, threshold time.Duration) *SlowLogger {
	return &SlowLogger{
		inner:         inner,
		slowThreshold: threshold,
	}
}

func durationFromMilliseconds(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

// Log logs a message with optional data and a start time. If the elapsed time exceeds the slowThreshold, it logs the message with a duration field. The log level is determined by the level parameter. The log adapter is set to logrusadapter. The logger is passed as a parameter. The method returns the logger.
func (l *SlowLogger) Log(ctx context.Context, level sqldblogger.Level, msg string, data map[string]interface{}) {
	if durationFloat, ok := data["duration"].(float64); ok {
		duration := durationFromMilliseconds(durationFloat)
		if duration > l.slowThreshold {
			level = sqldblogger.LevelError
			msg = fmt.Sprintf("[SLOW QUERY] %s: %s", duration, msg)
		}
	}
	l.inner.Log(ctx, level, msg, data)
}

// SqliteCache is a struct that manages the SQLite database for caching feeds and their associated Twts. It uses the zombiezen driver to interact with the database. The struct is safe for concurrent use. The `UpdateFeed` method inserts a feed and its associated Twts into the database. The `Close` method closes the SQLite connection. The `DB` method returns
type SqliteCache struct {
	db *sql.DB

	lookupCache *lru.Cache // key: tweet hash (string), value: types.Twt
	twterCache  *lru.Cache // key: twter URI (string), value: *types.Twter
}

// NewSqliteCache creates a new SqliteCache instance with the specified database path.
// It also initializes the SQLite database with the provided schema. The `UpdateFeed` method inserts a feed and its associated Twts into the database. The `Close` method closes the SQLite connection. The `DB` method returns the SQLite connection used by the cache. The `Update
func NewSqliteCache(dbPath string) (*SqliteCache, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	logger := NewSlowLogger(logrusadapter.New(log.StandardLogger()), 100*time.Millisecond)

	db = sqldblogger.OpenDriver(
		dbPath,
		db.Driver(),
		logger,
		sqldblogger.WithPreparerLevel(sqldblogger.LevelDebug), // default: LevelInfo
		sqldblogger.WithQueryerLevel(sqldblogger.LevelDebug),  // default: LevelInfo
		sqldblogger.WithExecerLevel(sqldblogger.LevelDebug),   // default: LevelInfo
	)

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// Create LRU caches with fixed capacity.
	lookupLRU, err := lru.New(maxLookupCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup LRU cache: %w", err)
	}
	twterLRU, err := lru.New(maxTwterCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create twter LRU cache: %w", err)
	}

	return &SqliteCache{
		db:          db,
		lookupCache: lookupLRU,
		twterCache:  twterLRU,
	}, nil
}

// deleteOldTwts deletes excess Twts for the specified feed URL, retaining the last 'retentionCount' Twts.
// If retentionCount <= 0, it deletes all Twts for the feed.
func (cache *SqliteCache) deleteOldTwts(feedURL string, retentionCount int) error {
	// If no retention policy is set or invalid, delete all Twts for the feed
	if retentionCount <= 0 {
		_, err := cache.db.Exec(`DELETE FROM twts WHERE feed_url = ?`, feedURL)
		if err != nil {
			log.WithError(err).Errorf("[sqlite] deleteOldTwts: failed to delete Twts for feed %s", feedURL)
			return fmt.Errorf("[sqlite] deleteOldTwts: failed to delete Twts for feed %s: %w", feedURL, err)
		}
		log.Infof("[sqlite] deleted all Twts for feed %s", feedURL)
		return nil
	}

	// Get the total count of Twts for this feed
	var totalCount int
	err := cache.db.QueryRow(`SELECT COUNT(*) FROM twts WHERE feed_url = ?`, feedURL).Scan(&totalCount)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] deleteOldTwts: failed to count Twts for feed %s", feedURL)
		return fmt.Errorf("[sqlite] deleteOldTwts: failed to count Twts for feed %s: %w", feedURL, err)
	}

	// If the number of Twts is less than or equal to the retention count, no deletion needed
	if totalCount <= retentionCount {
		log.Infof("[sqlite] Feed %s has %d Twts, no deletion needed", feedURL, totalCount)
		return nil
	}

	// Delete the excess Twts (oldest first) beyond the last N
	_, err = cache.db.Exec(`
		DELETE FROM twts WHERE feed_url = ? AND hash NOT IN (
			SELECT hash FROM twts WHERE feed_url = ? ORDER BY created_dt DESC LIMIT ?
		)
	`, feedURL, feedURL, retentionCount)

	if err != nil {
		log.WithError(err).Errorf("[sqlite] deleteOldTwts: failed to delete excess Twts for feed %s", feedURL)
		return fmt.Errorf("[sqlite] deleteOldTwts: failed to delete excess Twts for feed %s: %w", feedURL, err)
	}

	log.Infof("[sqlite] deleted excess Twts for feed %s, keeping last %d", feedURL, retentionCount)
	return nil
}

// Close closes the SQLite connection
func (cache *SqliteCache) Close() error {
	return cache.db.Close()
}

// FeedCount returns the number of feeds stored in the database.
func (cache *SqliteCache) FeedCount() int {
	var count int
	err := cache.db.QueryRow(`SELECT COUNT(*) FROM feeds`).Scan(&count)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] FeedCount error: %v", err)
		return 0
	}
	return count
}

// TwtCount returns the number of Twts stored in the database.
func (cache *SqliteCache) TwtCount() int {
	var count int
	err := cache.db.QueryRow(`SELECT COUNT(*) FROM twts`).Scan(&count)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] TwtCount error: %v", err)
		return 0
	}
	return count
}

// HasFeed checks if a feed exists in the database by URL
func (cache *SqliteCache) HasFeed(uri string) bool {
	const query = `SELECT 1 FROM feeds WHERE url LIKE ? LIMIT 1`
	var exists int
	err := cache.db.QueryRow(query, "%"+hostPath(NormalizeURL(uri))).Scan(&exists)
	if err != nil {
		if err != sql.ErrNoRows {
			log.WithError(err).Errorf("[sqlite] IsCached error: %v", err)
		}
		return false
	}
	return true
}

// UpdateFeed inserts or updates Twts for a given feed in the SQLite cache
func (cache *SqliteCache) UpdateFeed(url, lastModified string, twts types.Twts) error {
	if len(twts) == 0 {
		return nil
	}

	// Check if the Twter is a bot and delete old Twts if necessary
	twter := twts[0].Twter()
	if twter.Metadata.Get("type") == string(types.FeedTypeBot) {
		retention := SafeParseInt(twter.Metadata.Get("retention"), 0)
		if err := cache.deleteOldTwts(url, retention); err != nil {
			return err
		}
	}

	// Proceed with the usual feed update logic
	tx, err := cache.db.Begin()
	if err != nil {
		log.WithError(err).Error("[sqlite]: UpdateFeed: begin transaction failed")
		return fmt.Errorf("[sqlite] UpdateFeed: begin transaction failed: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Upsert feed metadata
	_, err = tx.Exec(`
		INSERT INTO feeds (url, fetches, last_fetched, last_modified)
		VALUES (?, 1, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			fetches = fetches + 1,
			last_fetched = excluded.last_fetched,
			last_modified = excluded.last_modified
	`, url, time.Now().Unix(), lastModified)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] UpdateFeed: insert feed failed")
		return fmt.Errorf("[sqlite] UpdateFeed: insert feed failed: %v", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO twts (
			hash, feed_url, content, created, subject, mentions, tags, links
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] UpdateFeed: insert twt failed")
		return fmt.Errorf("[sqlite] UpdateFeed: insert twt failed: %w", err)
	}
	defer stmt.Close()

	for _, twt := range twts {
		// Ignore Direct Message (DM) Twts
		// XXX: This is a highly experimental extension that uses mentions of the form:
		// !<nick url> <ciphertext>
		if lextwt.HasElement[*lextwt.BangMention](twt) {
			log.Debugf("ignoring DM Twt: %s", twt.Hash())
			continue
		}

		// Serialize fields
		mentions := []string{}
		for _, mention := range twt.Mentions() {
			mentions = append(mentions, fmt.Sprintf("%l", mention))
		}
		tags := []string{}
		for _, tag := range twt.Tags() {
			tags = append(tags, tag.Text())
		}
		links := []string{}
		for _, link := range twt.Links() {
			links = append(links, fmt.Sprintf("%s", link))
		}

		mentionsJSON := "[]"
		if m, err := json.Marshal(mentions); err == nil {
			mentionsJSON = string(m)
		}
		tagsJSON := "[]"
		if t, err := json.Marshal(tags); err == nil {
			tagsJSON = string(t)
		}
		linksJSON := "[]"
		if l, err := json.Marshal(links); err == nil {
			linksJSON = string(l)
		}

		_, err = stmt.Exec(
			twt.Hash(),
			twt.Twter().URI,
			fmt.Sprintf("%l", twt),
			twt.Created().Format(time.RFC3339),
			twt.Subject().Text(),
			mentionsJSON,
			tagsJSON,
			linksJSON,
		)
		if err != nil {
			log.WithError(err).Errorf("[sqlite] UpdateFeed: insert twt failed")
			return fmt.Errorf("[sqlite] UpdateFeed: insert twt failed: %w", err)
		}
		log.Debugf("[sqlite]: inserted twt %s", twt.Hash())
	}

	return nil
}

// FindTwter attempts to locate a Twter by nick or domainNick using case-insensitive match
func (cache *SqliteCache) FindTwter(nick string) *types.Twter {
	query := `
		SELECT uri, nick, hashing_uri, metadata
		FROM twters
		WHERE LOWER(nick) = LOWER(?) OR LOWER(uri) LIKE ?
		LIMIT 1
	`

	// The domainNick is usually of the form "nick@example.com"
	likePattern := "%" + strings.ToLower(nick) + "%"

	row := cache.db.QueryRow(query, nick, likePattern)

	var uri, dbNick, hashingURI, metadataStr string
	err := row.Scan(&uri, &dbNick, &hashingURI, &metadataStr)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("[sqlite] FindTwter: query error: %v", err)
		}
		return &types.Twter{}
	}

	twter := types.NewTwter(dbNick, uri)
	twter.HashingURI = hashingURI
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &twter.Metadata); err == nil {
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}
	}

	return &twter
}

// GetTwter retrieves a Twter object from the SQLite database using the provided URI.
// It first checks the LRU cache before querying the database, and caches the result.
func (cache *SqliteCache) GetTwter(uri string) *types.Twter {
	if value, ok := cache.twterCache.Get(uri); ok {
		return value.(*types.Twter)
	}

	row := cache.db.QueryRow(`SELECT nick, hashing_uri, metadata FROM twters WHERE uri LIKE ?`, "%"+hostPath(NormalizeURL(uri)))

	var nick, hashingURI, metadataStr string
	err := row.Scan(&nick, &hashingURI, &metadataStr)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetTwter: failed to unmarshal metadata")
		return nil
	}

	twter := types.NewTwter(nick, uri)
	twter.HashingURI = hashingURI
	if metadataStr != "" {
		_ = json.Unmarshal([]byte(metadataStr), &twter.Metadata)
		twter.Avatar = twter.Metadata.Get("avatar")
		twter.Tagline = twter.Metadata.Get("description")
	}

	cache.twterCache.Add(uri, &twter)

	return &twter
}

// SetTwter inserts or updates a Twter record in the SQLite database.
// It locks the cache, marshals the Twter metadata to JSON, and executes
// an SQL statement to insert or update the record based on the URI.
// Logs errors if JSON marshaling or SQL execution fails.
func (cache *SqliteCache) SetTwter(uri string, twter types.Twter) {
	metadataJSON, err := json.Marshal(twter.Metadata)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] SetTwter: failed to marshal metadata: %v", err)
		return
	}

	query := `
		INSERT INTO twters (uri, nick, hashing_uri, metadata)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(uri) DO UPDATE SET
		  nick = excluded.nick,
		  metadata = excluded.metadata
	`

	_, err = cache.db.Exec(query, uri, twter.Nick, twter.HashingURI, string(metadataJSON))
	if err != nil {
		log.WithError(err).Errorf("[sqlite] SetTwter error: %v", err)
	}

}

// GetCachedFeed retrieves a cached feed from the database by URL. If it does not exist, it inserts a new row with default values.
func (cache *SqliteCache) GetCachedFeed(url string) *Feed {
	query := `
		SELECT fetches, errors, last_error, last_fetched, last_modified
		FROM feeds WHERE url LIKE ?
	`

	row := cache.db.QueryRow(query, "%"+hostPath(NormalizeURL(url)))

	var fetches, errors int64
	var lastError, lastModified string
	var lastFetchedUnix int64

	err := row.Scan(&fetches, &errors, &lastError, &lastFetchedUnix, &lastModified)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.WithError(err).Errorf("[sqlite] GetCachedFeed error: %v", err)
		return nil
	}

	return &Feed{
		Fetches:      fetches,
		Errors:       errors,
		LastError:    lastError,
		LastFetched:  time.Unix(lastFetchedUnix, 0),
		LastModified: lastModified,
	}
}

// GetAllCachedFeeds retrieves all cached feed entries from the database.
// It returns a map where the keys are feed URLs and the values are pointers
// to Cached objects containing details about fetches, errors, last error,
// last fetched time, and last modified time. If an error occurs during the
// query or row scanning, it returns the error.
func (cache *SqliteCache) GetAllCachedFeeds() (map[string]*Feed, error) {
	rows, err := cache.db.Query(`
        SELECT url, fetches, errors, last_error, last_fetched, last_modified FROM feeds
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make(map[string]*Feed)
	for rows.Next() {
		var url, lastError, lastModified string
		var fetches, errors, lastFetchedUnix int64

		if err := rows.Scan(&url, &fetches, &errors, &lastError, &lastFetchedUnix, &lastModified); err != nil {
			log.WithError(err).Error("error scanning feed row")
			continue
		}
		feeds[url] = &Feed{
			Fetches:      fetches,
			Errors:       errors,
			LastError:    lastError,
			LastFetched:  time.Unix(lastFetchedUnix, 0),
			LastModified: lastModified,
		}
	}
	return feeds, nil
}

// GetOrSetCachedFeed retrieves a cached feed from the database by URL.
// If it does not exist, it inserts a new row with default values.
func (cache *SqliteCache) GetOrSetCachedFeed(url string) (*Feed, bool) {
	var (
		fetches, errors int64
		lastError       string
		lastFetchedUnix int64
		lastModified    string
	)

	query := `
		SELECT fetches, errors, last_error, last_fetched, last_modified
		FROM feeds WHERE url LIKE ?
	`
	err := cache.db.QueryRow(query, "%"+hostPath(NormalizeURL(url))).Scan(&fetches, &errors, &lastError, &lastFetchedUnix, &lastModified)
	if err == nil {
		return &Feed{
			Fetches:      fetches,
			Errors:       errors,
			LastError:    lastError,
			LastFetched:  time.Unix(lastFetchedUnix, 0),
			LastModified: lastModified,
		}, false
	}

	if err != sql.ErrNoRows {
		log.WithError(err).Errorf("[sqlite] GetOrSetCachedFeed error (select): %v", err)
		return nil, false
	}

	insert := `
		INSERT INTO feeds (url, fetches, errors, last_error, last_fetched, last_modified)
		VALUES (?, 0, 0, '', 0, '')
	`
	if _, err := cache.db.Exec(insert, url); err != nil {
		log.WithError(err).Errorf("[sqlite] GetOrSetCachedFeed error (insert): %v", err)
		return nil, false
	}

	return &Feed{
		Fetches:      0,
		Errors:       0,
		LastError:    "",
		LastFetched:  time.Unix(0, 0),
		LastModified: "",
	}, true
}

// UpdateCachedFeed updates the 'feeds' table for the given URL with the data
// from the provided *Cached object. It sets the fetch count, error count,
// last error message, last fetched time (as a Unix timestamp), and the last modified value.
func (cache *SqliteCache) UpdateCachedFeed(url string, cached *Feed) error {
	// Execute an UPDATE statement with proper bound parameters.
	_, err := cache.db.Exec(
		`UPDATE feeds
		 SET fetches = ?,
		     errors = ?,
		     last_error = ?,
		     last_fetched = ?,
		     last_modified = ?
		 WHERE url = ?`,
		cached.Fetches,
		cached.Errors,
		cached.LastError,
		cached.LastFetched.Unix(), // Convert time.Time to Unix timestamp
		cached.LastModified,
		url,
	)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] UpdateCachedFeed: update failed for feed %s", url)
		return err
	}
	return nil
}

// DeleteFeeds removes one or more feeds from the feeds table
func (cache *SqliteCache) DeleteFeeds(urls ...string) {
	if len(urls) == 0 {
		return
	}

	query := `DELETE FROM feeds WHERE url IN (` + strings.Repeat("?,", len(urls)-1) + `?)`
	args := make([]any, len(urls))
	for i, url := range urls {
		args[i] = url
	}

	if _, err := cache.db.Exec(query, args...); err != nil {
		log.WithError(err).Errorf("[sqlite] DeleteFeeds error for urls %v: %v", urls, err)
	}
}

// RenameFeed updates the feed URL in the feeds table
func (cache *SqliteCache) RenameFeed(oldURL, newURL string) {
	stmt := `
		UPDATE feeds
		SET url = ?
		WHERE url = ?
	`
	if _, err := cache.db.Exec(stmt, newURL, oldURL); err != nil {
		log.WithError(err).Errorf("[sqlite] RenameFeed error: %v", err)
	}
}

// ShouldRefreshFeed determines whether a feed should be refreshed based on metadata and errors.
func (cache *SqliteCache) ShouldRefreshFeed(localBaseURL string, uri string) bool {
	cachedFeed, ok := cache.GetOrSetCachedFeed(uri)
	if !ok {
		return true
	}

	// Handle special error cases
	if cachedFeed.LastError != "" {
		if strings.HasPrefix(cachedFeed.LastError, types.ErrDeadFeed{}.Error()) {
			if strings.HasPrefix(cachedFeed.LastError, types.ErrDeadFeed{Reason: "410"}.Error()) {
				log.Warnf("feed %s is absolutely dead, skipping forever: %s", uri, cachedFeed.LastError)
				return false
			}

			if time.Since(cachedFeed.LastFetched) <= 24*time.Hour {
				log.Warnf("feed %s is probably dead, skipping now, trying only once a day: %s", uri, cachedFeed.LastError)
				return false
			}
		}
	}

	u, err := url.Parse(uri)
	if err == nil && HasString(alwaysRefreshDomains, u.Hostname()) {
		return true
	}

	if strings.HasPrefix(NormalizeURL(uri), NormalizeURL(localBaseURL)) {
		return true
	}

	twter := cache.GetTwter(uri)
	if twter == nil {
		return true
	}

	refresh := twter.Metadata.Get("refresh")
	if refresh != "" {
		if n, err := strconv.Atoi(refresh); err == nil {
			return int(time.Since(cachedFeed.LastFetched).Seconds()) >= n
		}
	}

	return true
}

// Delete removes one or more Twts from the database by their hashes
func (cache *SqliteCache) Delete(hashes ...string) {
	if len(hashes) == 0 {
		return
	}

	query := `DELETE FROM twts WHERE hash IN (` + strings.Repeat("?,", len(hashes)-1) + `?)`
	stmt, err := cache.db.Prepare(query)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] Delete: prepare error for hashes %v: %v", hashes, err)
		return
	}
	defer stmt.Close()

	args := make([]any, len(hashes))
	for i, h := range hashes {
		args[i] = h
	}

	if _, err := stmt.Exec(args...); err != nil {
		log.WithError(err).Errorf("[sqlite] Delete: exec error for hashes %v: %v", hashes, err)
	}
}

// Lookup retrieves a Twt by its hash.
// It first checks our internal LRU cache, and if missing, queries the database.
// The result is then added to the cache.
func (cache *SqliteCache) Lookup(hash string, opts *QueryOptions) (types.Twt, bool) {
	// Fast path: check in-memory LRU cache
	if value, ok := cache.lookupCache.Get(hash); ok {
		twt := value.(types.Twt)

		// Check exclusion if present
		if opts != nil && isExcluded(twt.Twter().URI, twt.Subject().Text(), twt.Hash(), opts.Exclude) {
			return types.NilTwt, false
		}

		return twt, true
	}

	// Build WHERE clause with optional exclusions
	whereClauses := []string{"t.hash = ?"}
	args := []any{hash}

	if opts != nil && len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		whereClauses = append(whereClauses, fmt.Sprintf("t.feed_url NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.subject NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.hash NOT IN (%s)", placeholderList))

		for i := 0; i < 3; i++ {
			args = append(args, convertToAny(opts.Exclude)...)
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT t.feed_url, t.content, t.created,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		WHERE %s
		LIMIT 1
	`, whereClause)

	row := cache.db.QueryRow(query, args...)

	var (
		uri, content, created, nick, hashingURI, metadataStr sql.NullString
	)

	if err := row.Scan(&uri, &content, &created, &nick, &hashingURI, &metadataStr); err != nil {
		if err != sql.ErrNoRows {
			log.WithError(err).Errorf("[sqlite] Lookup: scan error for hash %s: %v", hash, err)
		}
		return types.NilTwt, false
	}

	twter := types.NewTwter(nick.String, uri.String)
	twter.HashingURI = hashingURI.String
	if metadataStr.Valid {
		_ = json.Unmarshal([]byte(metadataStr.String), &twter.Metadata)
		twter.Avatar = twter.Metadata.Get("avatar")
		twter.Tagline = twter.Metadata.Get("description")
	}

	ts := parseTime(created.String)
	twt := types.MakeTwt(twter, ts, content.String)
	cache.lookupCache.Add(hash, twt)

	return twt, true
}

func selectRankIfScore(sortBy string) string {
	if sortBy == "_score" {
		return ", bm25(twts_fts) AS rank"
	}
	return ""
}

// Search performs a full-text search using FTS5 with pagination support.
// It returns a slice of matching Twts, the total number of matching items,
// and an error (if any).
func (cache *SqliteCache) Search(query string, opts *QueryOptions) (types.Twts, int, error) {
	query = fmt.Sprintf(`"%s"`, query)

	var orderBy string
	switch opts.SortBy {
	case "created":
		orderBy = "t.created_dt ASC"
	case "-created":
		orderBy = "t.created_dt DESC"
	case "_score":
		orderBy = "rank ASC" // SQLite FTS5 score: lower is better
	default:
		orderBy = "t.created DESC"
	}

	// Perform a count query to determine the total number of matching rows.
	countQuery := `SELECT COUNT(*) FROM twts_fts WHERE twts_fts MATCH ?`
	var total int
	if err := cache.db.QueryRow(countQuery, query).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] Search: count query error for query %s", query)
		return nil, 0, err
	}

	// Build the main query with pagination.
	// The selectRankIfScore function will return additional columns when sorting by score.
	queryStr := fmt.Sprintf(`
		SELECT t.feed_url, t.content, t.created, tw.nick, tw.hashing_uri, tw.metadata%s
		FROM twts_fts
		JOIN twts t ON t.hash = twts_fts.hash
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		WHERE twts_fts MATCH ?
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, selectRankIfScore(opts.SortBy), orderBy)

	args := []any{
		query,
		opts.Limit,
		opts.Offset,
	}

	rows, err := cache.db.Query(queryStr, args...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] Search: query error")
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var uri, content, created, nick, hashingURI, metadataJSON sql.NullString
		if err := rows.Scan(&uri, &content, &created, &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] Search: scan error")
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

// GetMentions returns the Twts that mention the user with proper pagination.
// It returns a slice of Twts, the total count of matching rows, and an error if encountered.
func (cache *SqliteCache) GetMentions(mention string, opts *QueryOptions) (types.Twts, int, error) {
	whereClauses := []string{"json_each.value LIKE ?"}
	args := []any{fmt.Sprintf("%%%s%%", mention)}

	// Optional max age filter
	if opts.MaxAgeDays > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("t.created_dt >= datetime('now', '-%d days')", opts.MaxAgeDays))
	}

	// Optional exclusion filter (feeds, hashes, subjects)
	if len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		whereClauses = append(whereClauses, fmt.Sprintf("t.feed_url NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.subject NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.hash NOT IN (%s)", placeholderList))

		for i := 0; i < 3; i++ {
			args = append(args, convertToAny(opts.Exclude)...)
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM twts t, json_each(t.mentions)
		WHERE %s
	`, whereClause)

	var total int
	if err := cache.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] GetMentions: count query error for mention %s", mention)
		return nil, 0, err
	}

	// Main query
	mainQuery := fmt.Sprintf(`
		SELECT t.feed_url, t.content, t.created, t.mentions,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri,
		     json_each(t.mentions)
		WHERE %s
		ORDER BY t.created_dt DESC, t.hash ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := cache.db.Query(mainQuery, args...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetMentions: query error")
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var (
			uri, content, created, nick, hashingURI, metadataJSON sql.NullString
			_                                                     sql.NullString // mentions (ignored)
		)

		if err := rows.Scan(&uri, &content, &created, new(sql.NullString), &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] GetMentions: scan error")
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

// GetBySubject returns all Twts with a matching subject using pagination.
// It returns a slice of matching Twts, the total number of matching rows, and an error if any.
func (cache *SqliteCache) GetBySubject(subject string, opts *QueryOptions) (types.Twts, int, error) {
	whereClauses := []string{"subject = ?"}
	args := []any{subject}

	// Optional exclusion filter
	if len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		whereClauses = append(whereClauses, fmt.Sprintf("feed_url NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("subject NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("hash NOT IN (%s)", placeholderList))

		for i := 0; i < 3; i++ {
			args = append(args, convertToAny(opts.Exclude)...)
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM twts WHERE %s`, whereClause)
	var total int
	if err := cache.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] GetBySubject: count query error for subject %s", subject)
		return nil, 0, err
	}

	// Main query with pagination
	queryStr := fmt.Sprintf(`
		SELECT t.feed_url, t.content, t.created, t.subject,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		WHERE %s
		ORDER BY t.created_dt ASC, t.hash ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := cache.db.Query(queryStr, args...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetBySubject: query error for subject %s", subject)
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var (
			uri, content, created, nick, hashingURI, metadataJSON sql.NullString
			_                                                     sql.NullString // subject, unused here
		)
		if err := rows.Scan(&uri, &content, &created, new(sql.NullString), &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] GetBySubject: scan error")
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

// GetByURL returns all Twts posted by a specific feed URL using pagination.
// It returns a slice of matching Twts, the total number of matching rows, and an error if any.
func (cache *SqliteCache) GetByURL(url string, opts *QueryOptions) (types.Twts, int, error) {
	// WHERE clause parts and args
	whereClauses := []string{"t.feed_url LIKE ?"}
	args := []any{
		"%" + hostPath(NormalizeURL(url)),
	}

	// Optional exclusion filters
	if len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		whereClauses = append(whereClauses, fmt.Sprintf("t.feed_url NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.subject NOT IN (%s)", placeholderList))
		whereClauses = append(whereClauses, fmt.Sprintf("t.hash NOT IN (%s)", placeholderList))

		for i := 0; i < 3; i++ {
			args = append(args, convertToAny(opts.Exclude)...)
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM twts t
		WHERE %s
	`, whereClause)

	var total int
	if err := cache.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] GetByURL: count query error for url %s", url)
		return nil, 0, err
	}

	// Main query with pagination
	mainQuery := fmt.Sprintf(`
		SELECT t.feed_url, t.content, t.created,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		WHERE %s
		ORDER BY t.created_dt DESC, t.hash ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := cache.db.Query(mainQuery, args...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetByURL: query error for url %s", url)
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var (
			uri, content, created, nick, hashingURI, metadataJSON sql.NullString
		)

		if err := rows.Scan(&uri, &content, &created, &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] GetByURL: scan error for url %s", url)
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

// GetAll returns a paginated list of Twts for discovery.
// When FrontPageCompact is enabled, it returns one twt per feed (i.e. one per distinct feed_url),
// and the total count is computed as the number of distinct feed URLs.
// Otherwise, it returns all twts ordered by creation time, and the total count is computed normally.
func (cache *SqliteCache) GetAll(opts *QueryOptions) (types.Twts, int, error) {
	var countQuery, mainQuery string
	var args, countArgs []any

	dateFilter := ""
	if opts.MaxAgeDays > 0 {
		dateFilter = fmt.Sprintf(" AND created_dt >= datetime('now', '-%d days')", opts.MaxAgeDays)
	}

	excludeFilter := ""
	if len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		excludeFilter = fmt.Sprintf(`
			AND feed_url NOT IN (%s)
			AND subject NOT IN (%s)
			AND hash NOT IN (%s)
		`, placeholderList, placeholderList, placeholderList)

		// Add the Exclude slice 3x (for feed_url, subject, hash)
		for i := 0; i < 3; i++ {
			args = append(args, convertToAny(opts.Exclude)...)
			countArgs = append(countArgs, convertToAny(opts.Exclude)...)
		}
	}

	if opts.Compact {
		countQuery = `
			SELECT COUNT(*) FROM latest_twts WHERE 1=1` + dateFilter + excludeFilter

		mainQuery = `
			SELECT lt.feed_url, lt.content, lt.created, lt.subject,
			       tw.nick, tw.hashing_uri, tw.metadata
			FROM latest_twts lt
			LEFT JOIN twters tw ON lt.feed_url = tw.uri
			WHERE 1=1` + dateFilter + excludeFilter + `
			ORDER BY lt.created_dt DESC, lt.hash ASC
			LIMIT ? OFFSET ?
		`

	} else {
		countQuery = `
			SELECT COUNT(*) FROM twts WHERE 1=1` + dateFilter + excludeFilter

		mainQuery = `
			SELECT t.feed_url, t.content, t.created, t.subject,
			       tw.nick, tw.hashing_uri, tw.metadata
			FROM twts t
			LEFT JOIN twters tw ON t.feed_url = tw.uri
			WHERE 1=1` + dateFilter + excludeFilter + `
			ORDER BY t.created_dt DESC, t.hash ASC
			LIMIT ? OFFSET ?
		`
	}

	// Run the count query
	var total int
	if err := cache.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] GetAll: count query error")
		return nil, 0, err
	}

	// Add paging params
	args = append(args, opts.Limit, opts.Offset)

	rows, err := cache.db.Query(mainQuery, args...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetAll: query error")
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var (
			uri, content, created, nick, hashingURI, metadataJSON sql.NullString
			_                                                     sql.NullString // subject
		)

		if err := rows.Scan(&uri, &content, &created, new(sql.NullString), &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] GetAll: scan error")
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

const (
	flatTimelineQueryTemplate = `
		SELECT t.feed_url, t.content, t.created, t.subject,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		JOIN (
			SELECT subject, MAX(created_dt) AS latest
			FROM twts
			WHERE feed_url IN (%s) AND subject != ''%s
			GROUP BY subject
		) latest_twts ON t.subject = latest_twts.subject AND t.created_dt = latest_twts.latest
		ORDER BY t.created_dt DESC, t.hash ASC
		LIMIT ?
	`

	normalTimelineQueryTemplate = `
		SELECT t.feed_url, t.content, t.created, t.subject,
		       tw.nick, tw.hashing_uri, tw.metadata
		FROM twts t
		LEFT JOIN twters tw ON t.feed_url = tw.uri
		WHERE feed_url IN (%s)%s
		ORDER BY t.created_dt DESC, t.hash ASC
		LIMIT ?
	`
)

// GetByFeeds retrieves a paginated list of Twts for a given set of feeds a user follows from the SQLite cache.
// It returns the matching Twts, the total number of matching rows, and an error if encountered.
func (cache *SqliteCache) GetByFeeds(feeds []string, opts *QueryOptions) (types.Twts, int, error) {
	n := len(feeds)
	if n == 0 {
		return nil, 0, nil
	}

	placeholders := make([]string, n)
	inArgs := make([]any, n)
	for i, feed := range feeds {
		placeholders[i] = "?"
		inArgs[i] = feed
	}
	inClause := strings.Join(placeholders, ",")

	// Build additional filters
	var extraConditions []string
	var extraArgs []any

	if opts.MaxAgeDays > 0 {
		extraConditions = append(extraConditions,
			fmt.Sprintf("created_dt >= datetime('now', '-%d days')", opts.MaxAgeDays))
	}

	if len(opts.Exclude) > 0 {
		placeholders := strings.Repeat("?,", len(opts.Exclude))
		placeholderList := placeholders[:len(placeholders)-1]

		extraConditions = append(extraConditions, fmt.Sprintf("feed_url NOT IN (%s)", placeholderList))
		extraConditions = append(extraConditions, fmt.Sprintf("subject NOT IN (%s)", placeholderList))
		extraConditions = append(extraConditions, fmt.Sprintf("hash NOT IN (%s)", placeholderList))

		for i := 0; i < 3; i++ {
			extraArgs = append(extraArgs, convertToAny(opts.Exclude)...)
		}
	}

	extraSQL := ""
	if len(extraConditions) > 0 {
		extraSQL = " AND " + strings.Join(extraConditions, " AND ")
	}

	var mainQuery, countQuery string
	if opts.Flat {
		mainQuery = fmt.Sprintf(flatTimelineQueryTemplate, inClause, extraSQL)
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM (
				SELECT subject
				FROM twts
				WHERE feed_url IN (%s) AND subject != ''%s
				GROUP BY subject
			) AS flatCount
		`, inClause, extraSQL)
	} else {
		mainQuery = fmt.Sprintf(normalTimelineQueryTemplate, inClause, extraSQL)
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM twts
			WHERE feed_url IN (%s)%s
		`, inClause, extraSQL)
	}

	// Count
	countArgs := append([]any{}, inArgs...)
	countArgs = append(countArgs, extraArgs...)
	var total int
	if err := cache.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		log.WithError(err).Errorf("[sqlite] GetByUser: count query error")
		return nil, 0, err
	}

	// Main query
	mainArgs := append([]any{}, inArgs...)
	mainArgs = append(mainArgs, extraArgs...)
	mainArgs = append(mainArgs, opts.Limit, opts.Offset)

	queryStr := mainQuery + " OFFSET ?"

	rows, err := cache.db.Query(queryStr, mainArgs...)
	if err != nil {
		log.WithError(err).Errorf("[sqlite] GetByUser: query error")
		return nil, 0, err
	}
	defer rows.Close()

	var twts types.Twts
	for rows.Next() {
		var (
			uri, content, created, nick, hashingURI, metadataJSON sql.NullString
			_                                                     sql.NullString // subject
		)

		if err := rows.Scan(&uri, &content, &created, new(sql.NullString), &nick, &hashingURI, &metadataJSON); err != nil {
			log.WithError(err).Errorf("[sqlite] GetByUser: scan error")
			continue
		}

		twter := types.NewTwter(nick.String, uri.String)
		twter.HashingURI = hashingURI.String
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &twter.Metadata)
			twter.Avatar = twter.Metadata.Get("avatar")
			twter.Tagline = twter.Metadata.Get("description")
		}

		ts := parseTime(created.String)
		twt := types.MakeTwt(twter, ts, content.String)
		twts = append(twts, twt)
	}

	return twts, total, nil
}

// MissingRootSubjects retrieves all distinct subject keys from the feeds where a tweet
// for the extracted root hash does not exist.
// MissingRootSubjects returns a slice of subject strings (like "(#abc123)") for which there is no tweet
// whose hash matches the extracted hash (e.g. "abc123").
func (cache *SqliteCache) MissingRootSubjects() ([]string, error) {
	query := `
		SELECT DISTINCT t1.subject
		FROM twts AS t1
		WHERE t1.subject LIKE '#%' 
		AND NOT EXISTS (
			SELECT 1
			FROM twts AS t2
			WHERE t2.hash = substr(t1.subject, 2)
		)
    `
	rows, err := cache.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("MissingRootSubjects query error: %w", err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			log.WithError(err).Warn("error scanning missing subject")
			continue
		}
		subjects = append(subjects, subject)
	}
	return subjects, nil
}

// FilterByMissingSubjects returns those Twts whose subject is not yet present
// in the `twts.subject` column of the database.
func (cache *SqliteCache) FilterByMissingSubjects(twts types.Twts) (types.Twts, error) {
	if len(twts) == 0 {
		return nil, nil
	}

	// Prepare placeholders and arguments for SQL IN query
	placeholders := make([]string, len(twts))
	args := make([]any, len(twts))
	for i, twt := range twts {
		subj := twt.Subject().Text()
		placeholders[i] = "?"
		args[i] = subj
	}
	inClause := strings.Join(placeholders, ",")

	// Query existing subjects
	query := fmt.Sprintf(`
		SELECT DISTINCT t1.subject
		FROM twts AS t1
		WHERE t1.subject IN (%s)
		AND NOT EXISTS (
			SELECT 1
			FROM twts AS t2
			WHERE t2.hash = substr(t1.subject, 2)
		)
    `, inClause)

	rows, err := cache.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subjects := make(map[string]struct{}, len(twts))
	var existingSubj string
	for rows.Next() {
		if err := rows.Scan(&existingSubj); err != nil {
			return nil, err
		}
		subjects[existingSubj] = struct{}{}
	}

	// Filter in Twts with missing subjects
	var missing types.Twts
	for _, twt := range twts {
		s := twt.Subject().Text()
		if _, found := subjects[s]; found {
			missing = append(missing, twt)
		}
	}

	return missing, nil
}

// Ensure SqliteCacher satisfies the Cacher interface
var _ Cacher = (*SqliteCache)(nil)
