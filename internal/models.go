// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/creasty/defaults"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

var (
	ErrFeedAlreadyExists = errors.New("error: feed already exists by that name")
	ErrAlreadyFollows    = errors.New("error: you already follow this feed")
)

// User ...
type User struct {
	Username     string
	PasswordHash []byte
	Tagline      string
	URL          string
	CreatedAt    time.Time
	LastSeenAt   time.Time

	StartPage  string `default:"#origin"`
	Theme      string `default:"auto"`
	Lang       string `default:""`
	Recovery   string `default:""`
	AvatarHash string `default:""`

	DisplayDatesInTimezone    string `default:"UTC"`
	DisplayTimePreference     string `default:"24h"`
	OpenLinksInPreference     string `default:"newwindow"`
	DisplayTimelinePreference string `default:"list"`
	DisplayImagesPreference   string `default:"inline"`
	DisplayMedia              bool   `default:"true"`
	OriginalMedia             bool   `default:"false"`

	VisibilityReadmore bool `default:"false"`
	StripTrackingParam bool `default:"false"`

	IsFollowingPubliclyVisible bool `default:"true"`
	IsBookmarksPubliclyVisible bool `default:"true"`

	Bookmarks map[string]string `default:"{}"`
	Following map[string]string `default:"{}"`
	Links     map[string]string `default:"{}"`
	Muted     map[string]string `default:"{}"`

	muted   map[string]string
	sources map[string]string
}

// NewUser ...
func NewUser() *User {
	user := &User{}
	if err := defaults.Set(user); err != nil {
		log.WithError(err).Error("error creating new user object")
	}
	user.muted = make(map[string]string)
	user.sources = make(map[string]string)
	return user
}

func LoadUser(data []byte) (user *User, err error) {
	user = &User{}
	if err := defaults.Set(user); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	if user.Bookmarks == nil {
		user.Bookmarks = make(map[string]string)
	}
	if user.Following == nil {
		user.Following = make(map[string]string)
	}

	user.muted = make(map[string]string)
	for n, u := range user.Muted {
		user.muted[u] = n
	}

	user.sources = make(map[string]string)
	for n, u := range user.Following {
		user.sources[u] = n
	}

	return
}

func (u *User) String() string {
	url, err := url.Parse(u.URL)
	if err != nil {
		log.WithError(err).Warn("error parsing user url")
		return u.Username
	}
	return fmt.Sprintf("%s@%s", u.Username, url.Hostname())
}

func (u *User) IsZero() bool {
	return u.Username == ""
}

func (u *User) Is(url string) bool {
	return u.URL == NormalizeURL(url)
}

func (u *User) Bookmark(hash string) {
	if _, ok := u.Bookmarks[hash]; !ok {
		u.Bookmarks[hash] = ""
	} else {
		delete(u.Bookmarks, hash)
	}
}

func (u *User) Bookmarked(hash string) bool {
	_, ok := u.Bookmarks[hash]
	return ok
}

func (u *User) AddLink(title, url string) {
	key := strings.TrimSpace(title)
	if _, ok := u.Links[key]; !ok {
		u.Links[key] = url
	}
}

func (u *User) RemoveLink(title string) {
	key := strings.TrimSpace(title)
	delete(u.Links, key)
}

func (u *User) Mute(key, value string) {
	if !u.HasMuted(value) {
		u.Muted[key] = value
		u.muted[value] = key
	}
}

func (u *User) Unmute(key string) {
	value, ok := u.Muted[key]
	if ok {
		delete(u.Muted, key)
		delete(u.muted, value)
	}
}

func (u *User) Follow(alias, uri string) error {
	if !u.Follows(uri) {
		if _, ok := u.Following[alias]; ok {
			if _u, err := url.Parse(uri); err == nil {
				alias = fmt.Sprintf("%s@%s", alias, _u.Hostname())
			} else {
				alias = UniqueKeyFor(u.Following, alias)
			}
		}

		u.Following[alias] = uri
		u.sources[uri] = alias
	}
	return nil
}

func (u *User) FollowAndValidate(conf *Config, uri string) error {
	profile := u.Profile(conf.BaseURL, nil)

	twter, err := ValidateFeed(conf, profile, uri)
	if err != nil {
		return err
	}

	if u.Follows(twter.URI) {
		return ErrAlreadyFollows
	}

	return u.Follow(twter.DomainNick(), twter.URI)
}

// Follows reports whether the user follows the given URL.
// It first checks for an exact match in u.sources (after NormalizeURL).
// If that fails, it compares host+path (ignoring scheme, default ports, and trailing slash).
func (u *User) Follows(rawURL string) bool {
	if u.sources == nil {
		return false
	}

	// 1. exact match
	normalized := NormalizeURL(rawURL)
	if _, ok := u.sources[normalized]; ok {
		return true
	}

	// 2. compare host+path only
	target := hostPath(normalized)
	for src := range u.sources {
		if hostPath(src) == target {
			return true
		}
	}
	return false
}

func (u *User) FollowsAs(url string) string {
	if alias, ok := u.sources[NormalizeURL(url)]; ok {
		return alias
	}
	return ""
}

func (u *User) Unfollow(conf *Config, alias string) error {
	uri, found := u.Following[alias]
	if found {
		delete(u.sources, uri)
		delete(u.Following, alias)
	}

	return nil
}

func (u *User) HasMuted(value string) bool {
	_, ok := u.muted[value]
	return ok
}

func (u *User) MutedList() []string {
	var keys []string
	for k := range u.muted {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (u *User) Source() FetchFeedRequests {
	return FetchFeedRequests{
		{URI: u.URL, Created: time.Now()},
	}
}

func (u *User) Sources() FetchFeedRequests {
	// Ensure we fetch the user's own posts in the cache
	feeds := u.Source()[:]
	for uri := range u.sources {
		feeds = append(feeds, FetchFeedRequest{URI: uri, Created: time.Now()})
	}
	return feeds
}

func (u *User) Profile(baseURL string, viewer *User) types.Profile {
	var (
		feeds         []string
		viewerFollows bool
		links         types.Links
		muted         bool
		showBookmarks bool
		showFollowers bool
		showFollowing bool
	)

	for title, url := range u.Links {
		links = append(links, types.Link{Title: title, URL: url})
	}
	sort.Sort(links)

	if viewer != nil {
		if viewer.Is(u.URL) {
			viewerFollows = true
			showBookmarks = true
			showFollowers = true
			showFollowing = true
		} else {
			viewerFollows = viewer.Follows(u.URL)
			showBookmarks = u.IsBookmarksPubliclyVisible
			showFollowing = u.IsFollowingPubliclyVisible
		}

		muted = viewer.HasMuted(u.URL)
	}

	var follows types.Follows

	if showFollowing {
		for nick, uri := range u.Following {
			follows = append(follows, types.Follow{Nick: nick, URI: uri})
		}
	}
	follows.SortBy("Nick")

	return types.Profile{
		Type: "User",

		Nick:        u.Username,
		Description: u.Tagline,
		URI:         URLForUser(baseURL, u.Username),
		Avatar:      URLForAvatar(baseURL, u.Username, u.AvatarHash),

		Follows: viewerFollows,
		Muted:   muted,
		Links:   links,
		Feeds:   feeds,

		Bookmarks: u.Bookmarks,

		Following:  follows,
		NFollowing: len(follows),

		ShowBookmarks: showBookmarks,
		ShowFollowers: showFollowers,
		ShowFollowing: showFollowing,

		LastSeenAt: u.LastSeenAt,
	}
}

func (u *User) Twter(conf *Config) types.Twter {
	return types.Twter{
		Nick:   u.Username,
		URI:    conf.URLForUser(u.Username),
		Avatar: conf.URLForAvatar(u.Username, u.AvatarHash),
	}
}

// Reply generates a reply mention string for a given twt.
// If the user follows the original twt's author and it's not the user themselves,
// it adds the author's alias or URI as the first mention. If the author is not followed,
// it uses the author's domain nickname. Returns an empty string if the twt's author is the user.
func (u *User) Reply(twt types.Twt) string {
	// If we follow the original twt's Twter, add them as the first mention
	// only if the original twter isn't ourselves!
	if u.Follows(twt.Twter().URI) && !u.Is(twt.Twter().URI) {
		if alias := u.FollowsAs(twt.Twter().URI); alias != "" {
			return fmt.Sprintf("@%s ", alias)
		}
		return fmt.Sprintf("@<%s> ", twt.Twter().URI)
	} else if !u.Is(twt.Twter().URI) {
		return fmt.Sprintf("@<%s> ", twt.Twter().URI)
	}

	return ""
}

func (u *User) DisplayTimeFormat() string {
	switch strings.ToLower(u.DisplayTimePreference) {
	case "12h":
		return "3:04PM"
	case "24h":
		return "15:04"
	default:
		return ""

	}
}

func (u *User) Bytes() ([]byte, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	return data, nil
}
