// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

// Job is an interface that represents a job that can be executed in the background.
// It has a String method that returns a string representation of the job
// and a Run method that executes the job.
// The JobSpec struct is used to specify the schedule and factory for a job.
// The NewJobSpec function creates a new JobSpec instance.
// The SqliteCache struct manages the SQLite database
type Job interface {
	fmt.Stringer
	Run()
}

// JobSpec is a struct that specifies the schedule and factory for a job.
// It has a Schedule field that specifies the schedule for the job and a Factory field that specifies the factory for the job.
// The NewJobSpec function creates a new JobSpec instance. The SqliteCache struct manages the SQLite database
type JobSpec struct {
	Schedule string
	Factory  JobFactory
}

// NewJobSpec creates a new JobSpec instance.
// The Schedule field specifies the schedule for the job and a Factory field that specifies the factory for the job.
// The NewJobSpec function creates a new JobSpec instance. The SqliteCache struct manages the SQLite database
func NewJobSpec(schedule string, factory JobFactory) JobSpec {
	return JobSpec{schedule, factory}
}

var (
	// Jobs is a registry of defined jobs.
	Jobs map[string]JobSpec

	// StartupJobs is a registry of startup jobs.
	StartupJobs map[string]JobSpec
)

// InitJobs initializes the scheduled and startup jobs with their respective
// specifications. It sets up recurring jobs such as syncing the store,
// updating feeds, and managing user sessions, as well as startup jobs
// that need to run immediately upon initialization. The job specifications
// include the schedule and the function to execute.
func InitJobs(conf *Config) {
	Jobs = map[string]JobSpec{
		"SyncStore":        NewJobSpec("@every 1m", NewSyncStoreJob),
		"UpdateFeeds":      NewJobSpec(conf.FetchInterval, NewUpdateFeedsJob),
		"ConvergeFeeds":    NewJobSpec(conf.FetchInterval, NewConvergeFeedsJob),
		"CleanupDeadPeers": NewJobSpec("@daily", NewCleanupDeadPeersJob),

		"ActiveUsers":       NewJobSpec("@hourly", NewActiveUsersJob),
		"DeleteOldSessions": NewJobSpec("@hourly", NewDeleteOldSessionsJob),

		"RotateFeeds": NewJobSpec("0 0 1 * *", NewRotateFeedsJob),
	}

	StartupJobs = map[string]JobSpec{
		"RotateFeeds":       Jobs["RotateFeeds"],
		"UpdateFeeds":       Jobs["UpdateFeeds"],
		"DeleteOldSessions": Jobs["DeleteOldSessions"],
	}
}

type JobFactory func(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, store Store) Job

type SyncStoreJob struct {
	conf    *Config
	cache   Cacher
	fetcher *FeedFetcher
	db      Store
}

func NewSyncStoreJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, db Store) Job {
	return &SyncStoreJob{conf, cache, fetcher, db}
}

func (job *SyncStoreJob) String() string { return "SyncStore" }

func (job *SyncStoreJob) Run() {
	if err := job.db.Sync(); err != nil {
		log.WithError(err).Warn("error sycning store")
	}
	log.Info("synced store")
}

// ConvergeFeeds is a background job that checks for missing root tweets.
// It looks up subject keys in the cache (such as "(#abc123)") whose referenced root tweet,
// after stripping the surrounding characters, is not present in the cache.
// It then bumps a missing twts metric and (optionally) attempts to fetch these missing tweets from peers.
type ConvergeFeeds struct {
	conf    *Config
	cache   Cacher
	peering *Peering
}

// NewConvergeFeedsJob creates a new ConvergeMissingTwtsJob.
func NewConvergeFeedsJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, store Store) Job {
	return &ConvergeFeeds{
		conf:    conf,
		cache:   cache,
		peering: peering,
	}
}

// String implements the Job interface.
func (job *ConvergeFeeds) String() string {
	return "ConvergeMissingTwtsJob"
}

// Run executes the convergence job.
func (job *ConvergeFeeds) Run() {
	start := time.Now()

	// Query the cache for subject keys with missing roots.
	missingSubjects, err := job.cache.MissingRootSubjects()
	if err != nil {
		log.WithError(err).Warn("error querying missing root subjects")
		return
	}

	var validMissingRootSubjectHashes []string

	for _, subject := range missingSubjects {
		hash := ExtractHashFromSubject(subject)
		if len(hash) != types.TwtHashLength {
			continue
		}
		validMissingRootSubjectHashes = append(validMissingRootSubjectHashes, hash)
	}

	validMissingSubjectHashesCount := len(validMissingRootSubjectHashes)
	metrics.Gauge("cache", "missing_twts").Set(float64(validMissingSubjectHashesCount))
	log.Debugf("Convergence job: found %d valid missing root subject hashes", validMissingSubjectHashesCount)

	for _, hash := range validMissingRootSubjectHashes {
		log.Debugf("Attempting to resolve missing root twt for hash: %s", hash)

		var missingTwt types.Twt
		// Iterate over all available peers (from the Peering manager).
		candidates := job.peering.GetCandidatePeers()
		log.Debugf("converging with %d candidate peers: %v", len(candidates), candidates)
		for _, peer := range candidates {
			// Skip local peers.
			if job.conf.IsLocalURL(peer.URI) {
				continue
			}
			// Try to fetch the missing tweet from this peer.
			twt, err := peer.GetTwt(job.conf, hash)
			if err != nil {
				log.Debugf("Peer %s did not return twt for %s: %v", peer.URI, hash, err)
				continue
			}
			// If we got a non-nil tweet, then we assume success.
			missingTwt = twt
			break
		}

		if missingTwt != nil {
			log.Debugf("Resolved missing twt with hash %s from peers", hash)

			correctedTwt, err := validateOrCorrectTwt(missingTwt)
			if err != nil {
				log.WithError(err).Warnf("error validating or correcting twt %s from %s", missingTwt.Hash(), missingTwt.Twter().URI)
				continue
			}
			missingTwt = correctedTwt

			// Update the Twter's metadata with the avatar URL.
			twter := missingTwt.Twter()
			twter.Metadata = make(url.Values)
			twter.Metadata.Add("avatar", twter.Avatar)

			// Insert or Update the Twter
			job.cache.SetTwter(twter.URI, twter)

			// Ensure we have an external Avatar for this Twter
			GetExternalAvatar(job.conf, missingTwt.Twter())

			if err := job.cache.UpdateFeed(twter.URI, "", types.Twts{missingTwt}); err != nil {
				log.WithError(err).Warnf("error updating cache with missing twt %s from %s", missingTwt.Hash(), missingTwt.Twter().URI)
			}
		} else {
			log.Debugf("Could not resolve missing twt for hash %s", hash)
		}

		// We use a random number between 0 and 1000 milliseconds and 10ms jitter.
		// This is to avoid overwhelming the network.
		time.Sleep(time.Duration(rand.Intn(1000))*time.Millisecond + 10*time.Millisecond)
	}

	duration := time.Since(start)
	log.Debugf("Convergence job completed in %s", duration)
	metrics.Summary("cache", "last_convergence_seconds").Observe(float64(duration.Seconds()))
}

type UpdateFeedsJob struct {
	staleAfter  time.Duration
	maxRetries  int
	baseBackoff time.Duration

	db      Store
	cache   Cacher
	fetcher *FeedFetcher
}

func NewUpdateFeedsJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, db Store) Job {
	return &UpdateFeedsJob{
		staleAfter:  conf.fetchInterval,
		maxRetries:  3,
		baseBackoff: 100 * time.Millisecond,

		db:      db,
		cache:   cache,
		fetcher: fetcher,
	}
}

func (job *UpdateFeedsJob) String() string { return "UpdateFeeds" }

func (job *UpdateFeedsJob) Run() {
	start := time.Now()

	users, err := job.db.GetAllUsers()
	if err != nil {
		log.WithError(err).Warn("unable to get all users from database")
		return
	}
	log.Infof("updating feeds for %d users", len(users))

	// Collect unique feeds and public followers.
	byURI := make(map[string]FetchFeedRequest)
	publicByURI := make(map[string][]string)
	inactiveUsers := 0

	for _, user := range users {
		if time.Since(user.LastSeenAt) > 90*24*time.Hour {
			inactiveUsers++
			continue
		}
		for _, feed := range user.Sources() {
			if _, seen := byURI[feed.URI]; !seen {
				byURI[feed.URI] = feed
			}
			if user.IsFollowingPubliclyVisible {
				publicByURI[feed.URI] = append(publicByURI[feed.URI], user.Username)
			}
		}
	}
	log.Infof("skipping %d inactive users", inactiveUsers)

	// Build fetch requests with embedded followers.
	var sources FetchFeedRequests
	for uri, feed := range byURI {
		sources = append(sources, FetchFeedRequest{
			URI:       feed.URI,
			Created:   time.Now(),
			Force:     feed.Force,
			Followers: publicByURI[uri],
		})
	}

	// Load cached feeds.
	cachedFeeds, err := job.cache.GetAllCachedFeeds()
	if err != nil {
		log.WithError(err).Warn("unable to get cached feeds")
		cachedFeeds = make(map[string]*Feed)
	}

	// Exclude WebSub-subscribed feeds.
	var filtered FetchFeedRequests
	subscribed := 0
	for _, req := range sources {
		if _, ok := cachedFeeds[req.URI]; ok {
			sub := websub.GetSubscription(req.URI)
			if sub != nil && sub.Confirmed() && !sub.Expired() {
				subscribed++
				continue
			}
		}
		filtered = append(filtered, req)
	}
	log.Infof("skipping %d subscribed feeds", subscribed)

	// Select stale feeds.
	var staleSources FetchFeedRequests
	for _, req := range filtered {
		if cf, ok := cachedFeeds[req.URI]; ok {
			if time.Since(cf.LastFetched) < job.staleAfter {
				continue
			}
		}
		staleSources = append(staleSources, req)
	}
	log.Infof("updating %d sources (stale feeds)", len(staleSources))

	// Enqueue and retry if necessary.
	dropped := job.fetcher.EnqueueFeeds(staleSources, job.staleAfter)
	if len(dropped) > 0 {
		log.Infof("dropped %d feeds on first pass", len(dropped))
		delay := job.baseBackoff
		for attempt := 1; attempt <= job.maxRetries && len(dropped) > 0; attempt++ {
			time.Sleep(delay + time.Duration(rand.Intn(100)-50)*time.Millisecond)
			dropped = job.fetcher.EnqueueFeeds(dropped, job.staleAfter)
			log.Infof("dropped %d feeds after retry #%d", len(dropped), attempt)
			delay *= 2
		}
	}

	log.Infof("UpdateFeeds job took %s", time.Since(start))
}

type ActiveUsersJob struct {
	conf  *Config
	cache Cacher
	db    Store
}

func NewActiveUsersJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, db Store) Job {
	return &ActiveUsersJob{conf, cache, db}
}

func (job *ActiveUsersJob) String() string { return "ActiveUsers" }

func (job *ActiveUsersJob) Run() {
	log.Info("updating active user stats")

	users, err := job.db.GetAllUsers()
	if err != nil {
		log.WithError(err).Warn("unable to get all users from database")
		return
	}

	dau := 0
	mau := 0
	for _, user := range users {
		if time.Since(user.LastSeenAt) <= (24 * time.Hour) {
			dau++
		}
		if time.Since(user.LastSeenAt) <= (28 * 24 * time.Hour) {
			mau++
		}
	}

	metrics.Gauge("server", "dau").Set(float64(dau))
	metrics.Gauge("server", "mau").Set(float64(mau))
}

type DeleteOldSessionsJob struct {
	conf  *Config
	cache Cacher
	db    Store
}

func NewDeleteOldSessionsJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, db Store) Job {
	return &DeleteOldSessionsJob{conf, cache, db}
}

func (job *DeleteOldSessionsJob) String() string { return "DeleteOldSessions" }

func (job *DeleteOldSessionsJob) Run() {
	log.Info("deleting old sessions")

	sessions, err := job.db.GetAllSessions()
	if err != nil {
		log.WithError(err).Error("error loading seessions")
		return
	}

	for _, session := range sessions {
		if session.Expired() {
			log.Infof("deleting expired session %s", session.ID)
			if err := job.db.DelSession(session.ID); err != nil {
				log.WithError(err).Error("error deleting session object")
			}
		}
	}
}

type RotateFeedsJob struct {
	conf  *Config
	cache Cacher
	db    Store
}

func NewRotateFeedsJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, db Store) Job {
	return &RotateFeedsJob{conf, cache, db}
}

func (job *RotateFeedsJob) String() string { return "RotateFeeds" }

func (job *RotateFeedsJob) Run() {
	feeds, err := GetAllFeeds(job.conf)
	if err != nil {
		log.WithError(err).Warn("unable to get all local feeds")
		return
	}

	for _, feed := range feeds {
		fn := filepath.Join(job.conf.Data, feedsDir, feed)
		stat, err := os.Stat(fn)
		if err != nil {
			log.WithError(err).Error("error getting feed size")
			continue
		}

		if stat.Size() > job.conf.MaxFeedSize {
			log.Infof("rotating %s with size %s > %s", feed, humanize.Bytes(uint64(stat.Size())), humanize.Bytes(uint64(job.conf.MaxFeedSize)))

			if err := RotateFeed(job.conf, feed); err != nil {
				log.WithError(err).Error("error rotating feed")
			} else {
				log.Infof("rotated feed %s", feed)
			}
		}
	}
}

// CleanupDeadPeersJob is a job that purges peers that have not been seen recently.
type CleanupDeadPeersJob struct {
	conf    *Config
	peering *Peering
}

// NewCleanupDeadPeersJob is the factory function for CleanupDeadPeersJob.
// Note: The JobFactory now needs to accept a peering argument.
func NewCleanupDeadPeersJob(conf *Config, cache Cacher, fetcher *FeedFetcher, peering *Peering, store Store) Job {
	return &CleanupDeadPeersJob{
		conf:    conf,
		peering: peering,
	}
}

// String returns the name of the job.
func (job *CleanupDeadPeersJob) String() string {
	return "CleanupDeadPeers"
}

// Run iterates over the list of peers and removes those that are considered dead.
func (job *CleanupDeadPeersJob) Run() {
	// Define what "dead" means – for example, if a peer has not been seen for more
	// than three times your normal refresh interval.
	deadTimeout := defaultPeeringDeadTimeout // e.g., defaultPeeringDeadTimeout is defined as (defaultPeeringRefreshInterval * 3)
	peers := job.peering.ListPeers()
	var removedCount int

	for _, peer := range peers {
		if time.Since(peer.LastSeen) > deadTimeout {
			job.peering.DeletePeer(peer.URI)
			log.Infof("removed dead peer %s (last seen %s)", peer.URI, time.Since(peer.LastSeen))
			removedCount++
		}
	}
}
