// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

// -*- tab-width: 4; -*-

package internal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	read_file_last_line "git.mills.io/prologic/read-file-last-line"
	log "github.com/sirupsen/logrus"

	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

const (
	feedsDir = "feeds"
)

func ReplaceTwt(conf *Config, user *User, toReplace, replaceWith types.Twt) error {
	p := filepath.Join(conf.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return err
	}

	fn := filepath.Join(p, user.Username)

	var twts types.Twts

	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	twter := user.Twter(conf)
	t, err := types.ParseFile(f, &twter)
	if err != nil {
		return err
	}
	twts = append(twts, t.Twts()...)

	if err := RewriteFeed(fn, twts, func(twt types.Twt) types.Twt {
		if twt.Hash() == toReplace.Hash() {
			return replaceWith
		}
		return twt
	}); err != nil {
		return err
	}

	return nil
}

func DeleteLastTwt(conf *Config, user *User) error {
	p := filepath.Join(conf.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return err
	}

	fn := filepath.Join(p, user.Username)

	_, n, err := GetLastTwt(conf, user.Twter(conf))
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Truncate(int64(n))
}

type AppendTwtFunc func(user *User, text string, args ...interface{}) (types.Twt, error)

func AppendTwtFactory(conf *Config, cache Cacher, db Store) AppendTwtFunc {
	return func(user *User, text string, args ...interface{}) (types.Twt, error) {
		text = strings.TrimSpace(text)
		if text == "" {
			return types.NilTwt, fmt.Errorf("cowardly refusing to twt empty text, or only spaces")
		}

		p := filepath.Join(conf.Data, feedsDir)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.WithError(err).Error("error creating feeds directory")
			return types.NilTwt, err
		}

		fn := filepath.Join(p, user.Username)

		f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return types.NilTwt, err
		}
		defer f.Close()

		// Support replacing/editing an existing Twt whilst preserving Created Timestamp
		now := time.Now()
		if len(args) > 0 {
			if t, ok := args[0].(time.Time); ok {
				now = t
			}
		}

		twter := user.Twter(conf)

		feedLookupFn := NewMultiFeedLookup(
			NewUserFollowedAsFeedLookup(user),
			NewCachedFeedLookup(cache),
			NewLocalFeedLookup(conf, db),
			NewRemoteFeedLookup(conf),
		)

		// XXX: This is a bit convoluted @xuu can we improve this somehow?
		tmpTwt := types.MakeTwt(twter, now, strings.TrimSpace(text))
		tmpTwt.ExpandMentions(&nickForURLFmtOpts{conf, cache}, feedLookupFn)
		newText := tmpTwt.FormatText(types.LiteralFmt, nil)
		twt := types.MakeTwt(twter, now, newText)

		if _, err = fmt.Fprintf(f, "%+l\n", twt); err != nil {
			return types.NilTwt, err
		}

		websub.SendNotification(conf.URLForUser(twter.Nick))

		return twt, nil
	}
}

func GetLastTwtFrom(conf *Config, twter types.Twter, fn string) (twt types.Twt, offset int, err error) {
	if !FileExists(fn) {
		return
	}

	var data []byte
	data, offset, err = read_file_last_line.ReadLastLine(fn)
	if err != nil {
		return
	}

	twt, err = types.ParseLine(string(data), &twter)

	return
}

func GetLastTwt(conf *Config, twter types.Twter) (twt types.Twt, offset int, err error) {
	twt = types.NilTwt

	p := filepath.Join(conf.Data, feedsDir)
	if err = os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return
	}

	fn := filepath.Join(p, twter.Nick)

	return GetLastTwtFrom(conf, twter, fn)
}

func GetAllFeeds(conf *Config) ([]string, error) {
	p := filepath.Join(conf.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return nil, err
	}

	files, err := os.ReadDir(p)
	if err != nil {
		log.WithError(err).Error("error reading feeds directory")
		return nil, err
	}

	fns := []string{}
	for _, fileInfo := range files {
		fn := filepath.Base(fileInfo.Name())
		// feeds with an extension are rotated/archived feeds
		// e.g: prologic.1 prologic.2
		if filepath.Ext(fn) != "" {
			continue
		}
		fns = append(fns, fn)
	}
	return fns, nil
}

func GetFeedCount(conf *Config, name string) (int, error) {
	p := filepath.Join(conf.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return 0, err
	}

	fn := filepath.Join(p, name)

	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Error("error opening feed file")
		return 0, err
	}
	defer f.Close()

	return LineCount(f)
}

func GetAllTwts(conf *Config, name string) (types.Twts, error) {
	p := filepath.Join(conf.Data, feedsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.WithError(err).Error("error creating feeds directory")
		return nil, err
	}

	var twts types.Twts

	twter := types.NewTwter(name, URLForUser(conf.BaseURL, name))

	fn := filepath.Join(p, name)
	f, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		log.WithError(err).Warnf("error opening feed: %s", fn)
		return nil, err
	}
	defer f.Close()
	t, err := types.ParseFile(f, &twter)
	if err != nil {
		log.WithError(err).Errorf("error processing feed %s", fn)
		return nil, err
	}
	twts = append(twts, t.Twts()...)
	f.Close()

	return twts, nil
}

// GetArchivedFeeds ...
func GetArchivedFeeds(conf *Config, feed string) ([]string, error) {
	fns, err := filepath.Glob(filepath.Join(conf.Data, feedsDir, fmt.Sprintf("%s.[0-9]*", feed)))
	if err != nil {
		return nil, err
	}
	sort.Strings(fns)
	return fns, nil
}

// ParseArchivedFeedIds ...
func ParseArchivedFeedIds(fns []string) ([]int, error) {
	var ids []int
	for _, fn := range fns {
		base := filepath.Base(fn)
		// Split the archived feed's base filename into 3 parts
		// <feed>.<id>[.<rest>]
		// This is so we can in future support compressed archives
		// like prologic.1.gz
		parts := strings.SplitN(base, ".", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected number of parts in archived feed %s expected at least 2", fn)
		}
		// the <id> is always the 2nd part of the archived feed's filename
		idPart := parts[1]
		id, err := strconv.ParseInt(idPart, 10, 32)
		if err != nil {
			return nil, err
		}
		ids = append(ids, int(id))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))
	return ids, nil
}

func RotateFeed(conf *Config, feed string) error {
	// Get old archived feeds
	archivedFeeds, err := GetArchivedFeeds(conf, feed)
	if err != nil {
		log.WithError(err).Error("error getting list of archived feeds")
		return fmt.Errorf("error getting list of archived feeds for %s: %w", feed, err)
	}

	// Parse archived feed ids
	archivedFeedIds, err := ParseArchivedFeedIds(archivedFeeds)
	if err != nil {
		log.WithError(err).Error("error parsing archived feed ids")
		return fmt.Errorf("error parsing archived feed ids for %s: %w", feed, err)
	}

	// Shuffle old archived feeds
	for _, archiveFeedID := range archivedFeedIds {
		oldFn := filepath.Join(conf.Data, feedsDir, fmt.Sprintf("%s.%d", feed, archiveFeedID))
		newFn := filepath.Join(conf.Data, feedsDir, fmt.Sprintf("%s.%d", feed, (archiveFeedID+1)))

		if FileExists(newFn) {
			return fmt.Errorf("error shuffling archived feed %s would override archived feed %s", oldFn, newFn)
		}

		if err := os.Rename(oldFn, newFn); err != nil {
			log.WithError(err).Errorf("error renaming archived feed %s -> %s", oldFn, newFn)
		}
	}

	oldFn := filepath.Join(conf.Data, feedsDir, feed)
	newFn := filepath.Join(conf.Data, feedsDir, feed+".1")

	if FileExists(newFn) {
		return fmt.Errorf("error rotating feed %s would override archived feed %s", feed, newFn)
	}

	if err := os.Rename(oldFn, newFn); err != nil {
		log.WithError(err).Errorf("error renaming active feed %s -> %s", oldFn, newFn)
	}

	if err := os.WriteFile(oldFn, nil, os.FileMode(0644)); err != nil {
		log.WithError(err).Errorf("error re-creating active feed %s", oldFn)
	}

	return nil
}

func GetPreviousArchivedFeed(conf *Config, twter types.Twter, feed, fn string) (string, error) {
	// Get old archived feeds
	archivedFeeds, err := GetArchivedFeeds(conf, feed)
	if err != nil {
		log.WithError(err).Error("error getting list of archived feeds")
		return "", fmt.Errorf("error getting list of archived feeds for %s: %w", feed, err)
	}
	if archivedFeeds == nil {
		return "", nil
	}

	lastHashAndPath := func(fn string, n int) (string, error) {
		lastTwt, _, err := GetLastTwtFrom(conf, twter, fn)
		if err != nil {
			return "", fmt.Errorf("error reading last twt for %s", fn)
		}
		return fmt.Sprintf("%s twtxt.txt/%d", lastTwt.Hash(), n), nil
	}

	for n, archivedFeed := range archivedFeeds {
		if filepath.Base(archivedFeed) == filepath.Base(fn) {
			if (n + 1) < len(archivedFeeds) {
				return lastHashAndPath(archivedFeeds[(n+1)], (n + 2))
			} else {
				return "", nil
			}
		}
	}

	return lastHashAndPath(archivedFeeds[0], 1)
}

// UniqTwts returns a slice of unique Twts. It filters out any duplicates in
// the given slice of Twts by checking their hashes. If a Twt has been seen
// before, it is not included in the result.
func UniqTwts(twts types.Twts) (res types.Twts) {
	seenTwts := make(map[string]struct{})
	for _, twt := range twts {
		if _, seenTwt := seenTwts[twt.Hash()]; !seenTwt {
			res = append(res, twt)
			seenTwts[twt.Hash()] = struct{}{}
		}
	}
	return
}

func MergeFeed(conf *Config, feed string, body []byte, delete bool) (types.Twts, error) {
	ours, err := GetAllTwts(conf, feed)
	if err != nil {
		return nil, fmt.Errorf("error getting existing twts for %s: %w", feed, err)
	}

	twter := types.NewTwter(feed, conf.URLForUser(feed))
	self := types.NewTwter(feed, conf.URLForUser(feed))

	theirs, err := lextwt.ParseFile(bytes.NewBuffer(body), &twter)
	if err != nil {
		return nil, fmt.Errorf("error parsing feed to sync from: %w", err)
	}
	if theirs.Twter().HashingURI != self.HashingURI {
		return nil, fmt.Errorf("error hashing uri differ, refusing to sync")
	}

	inTheirs := make(map[string]bool)
	for _, twt := range theirs.Twts() {
		inTheirs[twt.Hash()] = true
	}

	inOurs := make(map[string]bool)
	for _, twt := range ours {
		inOurs[twt.Hash()] = true
	}

	log.Infof("ours: %#v", inOurs)
	log.Infof("theirs: %#v", inTheirs)

	var merged types.Twts

	for _, twt := range ours {
		if inTheirs[twt.Hash()] || (!inTheirs[twt.Hash()] && !delete) {
			merged = append(merged, twt)
		}
	}

	for _, twt := range theirs.Twts() {
		if !inOurs[twt.Hash()] {
			merged = append(merged, twt)
		}
	}
	merged = UniqTwts(merged)
	sort.Sort(sort.Reverse(merged))

	fn := filepath.Join(conf.Data, feedsDir, feed)
	if err := RewriteFeed(fn, merged, nil); err != nil {
		return nil, fmt.Errorf("error rewriting feed: %w", err)
	}

	return merged, nil
}

func ReadFeedPreamble(fn string) string {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Error("error opening feed")
		return ""
	}
	defer f.Close()

	tf, err := lextwt.ParseFile(f, nil)
	if err != nil {
		log.WithError(err).Error("erro rreading preamble")
		return ""
	}

	return tf.Info().String()
}

type FilterTwt func(types.Twt) types.Twt

func RewriteFeed(fn string, twts types.Twts, f FilterTwt) error {
	mergedFn := fmt.Sprintf("%s.merged", fn)
	w, err := os.OpenFile(mergedFn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error creating merged feed: %w", err)
	}
	defer w.Close()

	preamble := ReadFeedPreamble(fn)
	preamble = fmt.Sprintf("%s\n\n", strings.TrimSpace(preamble))

	if _, err := fmt.Fprint(w, preamble); err != nil {
		return fmt.Errorf("error writing preamble: %w", err)
	}

	for _, twt := range twts {
		if f != nil {
			twt = f(twt)
		}
		if !twt.IsZero() {
			if _, err := fmt.Fprintf(w, "%+l\n", twt); err != nil {
				return fmt.Errorf("error writing twt: %w", err)
			}
		}
	}

	if err := os.Rename(mergedFn, fn); err != nil {
		return fmt.Errorf("error renaming merged feed: %w", err)
	}

	return nil
}
