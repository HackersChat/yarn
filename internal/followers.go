package internal

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

const followersFilename = "followers.json"

// Followers handles User-Agent–based follower discovery
// and stores all followers in an in-memory cache persisted to disk.
type Followers struct {
	conf  *Config
	mu    sync.RWMutex
	cache map[string]types.Followers
}

// NewFollowers constructs a Followers, loading existing data if present.
func NewFollowers(conf *Config) *Followers {
	f := &Followers{
		conf:  conf,
		cache: make(map[string]types.Followers),
	}
	if err := f.loadCache(); err != nil {
		log.WithError(err).Warn("error loading followers")
	}
	return f
}

// Close locks the Followers instance, ensuring thread-safe access,
// and saves the current state of the followers cache to persistent storage.
// Returns an error if the cache saving process fails.
func (f *Followers) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saveCache()
}

// DetectFromRequest inspects the request for UA-based followers,
// updating the in-memory cache (never writes to any other store).
func (f *Followers) DetectFromRequest(req *http.Request, username string) error {
	if req == nil {
		return nil
	}
	ua, err := ParseUserAgent(req.UserAgent())
	if err != nil {
		log.WithError(err).Debugf("error parsing user agent from request")
		return nil
	}
	log.Debugf("ua: %T", ua)

	now := time.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	existing := f.cache[username]
	seen := make(map[string]bool, len(existing))
	for _, fo := range ua.Followers(f.conf) {
		updated := false
		for _, ex := range existing {
			log.Debugf("fo: %+v", fo)
			log.Debugf("ex: %+v", ex)
			if ex.URI == fo.URI && ex.Nick == fo.Nick {
				ex.LastSeenAt = now
				updated = true
				break
			}
		}
		if !updated {
			existing = append(existing, &types.Follower{
				URI:        fo.URI,
				Nick:       fo.Nick,
				LastSeenAt: now,
			})
		}
		seen[fo.URI] = true
	}
	f.cache[username] = existing

	return nil
}

// GetFor returns the slice of followers for a user, sorted by LastSeenAt.
func (f *Followers) GetFor(username string) types.Followers {
	f.mu.RLock()
	defer f.mu.RUnlock()

	slice := f.cache[username]
	slice.SortBy("LastSeenAt")
	return slice
}

// IsFollowedBy reports whether the specified user is followed by the given follower URI.
func (f *Followers) IsFollowedBy(followerURI, username string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, fo := range f.cache[username] {
		if hostPath(NormalizeURL(fo.URI)) == hostPath(NormalizeURL(followerURI)) {
			return true
		}
	}
	return false
}

// LastSeenFor returns the LastSeenAt timestamp for a follower URI, or time.Time{} if not found.
func (f *Followers) LastSeenFor(followerURI, username string) time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, fo := range f.cache[username] {
		if hostPath(NormalizeURL(fo.URI)) == hostPath(NormalizeURL(followerURI)) {
			return fo.LastSeenAt
		}
	}
	return time.Time{}
}

// PruneOlderThan removes any follower entries not seen since cutoff.
func (f *Followers) PruneOlderThan(cutoff time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for user, slice := range f.cache {
		out := slice[:0]
		for _, fo := range slice {
			if !fo.LastSeenAt.Before(cutoff) {
				out = append(out, fo)
			}
		}
		if len(out) > 0 {
			f.cache[user] = out
		} else {
			delete(f.cache, user)
		}
	}
}

// loadCache loads followers.json into memory.
func (f *Followers) loadCache() error {
	fn := filepath.Join(f.conf.Data, followersFilename)
	file, err := os.Open(fn)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()

	raw := make(map[string]types.Followers)
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return err
	}

	f.cache = raw
	return nil
}

// saveCache writes the in-memory cache back to followers.json.
func (f *Followers) saveCache() error {
	fn := filepath.Join(f.conf.Data, followersFilename)
	tmp := fn + ".tmp"

	serial := make(map[string]types.Followers, len(f.cache))
	for user, slice := range f.cache {
		serial[user] = slice
	}

	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&serial); err != nil {
		file.Close()
		return err
	}
	file.Close()
	return os.Rename(tmp, fn)
}
