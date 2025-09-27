// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"go.yarn.social/types"
)

// QueryOptions is a struct that contains options for querying the database.
// It includes fields for flat, compact, limit, offset, and sortBy.
type QueryOptions struct {
	// Flat if true shows only the latest Twt grouped by Subject
	Flat bool

	// Compact if true shows only the latest Twt grouped by Feed
	Compact bool

	// Limit the number of results returned
	Limit int

	// Offset is the number of results to skip before returning results
	Offset int

	// SortBy the order of the results
	SortBy string

	// MaxAgeDays filters results to only include Twts created in the last N days.
	// If zero or negative, no filtering is applied.
	MaxAgeDays int

	// Exclude the specified Feed URIs or Twt hashes from the resutls
	Exclude []string
}

// Cacher is an interface for caching feeds and their associated Twts
type Cacher interface {
	FeedCount() int
	TwtCount() int
	HasFeed(uri string) bool

	FindTwter(nick string) *types.Twter
	GetTwter(uri string) *types.Twter
	SetTwter(uri string, twter types.Twter)

	GetCachedFeed(uri string) *Feed
	GetAllCachedFeeds() (map[string]*Feed, error)
	GetOrSetCachedFeed(uri string) (*Feed, bool)
	UpdateCachedFeed(uri string, cached *Feed) error

	DeleteFeeds(uris ...string)
	RenameFeed(oldURI, newURI string)
	UpdateFeed(uri, lastModified string, twts types.Twts) error

	ShouldRefreshFeed(localBaseURL string, uri string) bool

	Delete(hashes ...string)
	Lookup(hash string, opts *QueryOptions) (types.Twt, bool)
	Search(query string, opts *QueryOptions) (types.Twts, int, error)

	GetAll(opts *QueryOptions) (types.Twts, int, error)
	GetMentions(mention string, opts *QueryOptions) (types.Twts, int, error)
	GetBySubject(subject string, opts *QueryOptions) (types.Twts, int, error)
	GetByFeeds(feeds []string, opts *QueryOptions) (types.Twts, int, error)
	GetByURL(url string, opts *QueryOptions) (types.Twts, int, error)
	MissingRootSubjects() ([]string, error)
	FilterByMissingSubjects(twts types.Twts) (types.Twts, error)
}
