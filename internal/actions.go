// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package internal contains code that is not exposed to the outside world.
//
// It contains functions for deleting a feed, and for creating a new feed.
package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

// DeleteUser ...
func DeleteUser(conf *Config, cache Cacher, db Store, username string) error {
	user, err := db.GetUser(username)
	if err != nil {
		return fmt.Errorf("error loading user object %s: %w", username, err)
	}

	// Get user's primary feed twts
	twts, err := GetAllTwts(conf, user.Username)
	if err != nil {
		return fmt.Errorf("error loading tsts for %s: %w", username, err)
	}

	// Parse twts to search and remove primary feed uploaded media
	for _, twt := range twts {
		mediaPaths := GetMediaNamesFromText(fmt.Sprintf("%t", twt))

		// Remove all uploaded media in a twt
		for _, mediaPath := range mediaPaths {
			// Delete .png
			fn := filepath.Join(conf.Data, mediaDir, fmt.Sprintf("%s.png", mediaPath))
			if FileExists(fn) {
				if err := os.Remove(fn); err != nil {
					return fmt.Errorf("error deleting media %s: %w", fn, err)
				}
			}
		}
	}

	// Delete user's avatar
	if fns, err := filepath.Glob(filepath.Join(conf.Data, avatarsDir, fmt.Sprintf("%s.*", user.Username))); err == nil {
		for _, fn := range fns {
			if FileExists(fn) {
				if err := os.Remove(fn); err != nil {
					return fmt.Errorf("error deleting avatar %s: %w", fn, err)
				}
			}
		}
	}

	// Delete user's twtxt.txt
	fn := filepath.Join(conf.Data, feedsDir, user.Username)
	if FileExists(fn) {
		if err := os.Remove(fn); err != nil {
			return fmt.Errorf("error deleting feed %s: %w", fn, err)
		}
	}

	// Delete user
	if err := db.DelUser(user.Username); err != nil {
		return fmt.Errorf("error deleting user object %s: %w", user.Username, err)
	}

	// Delete user's feed from cache
	var urls []string
	for _, feed := range user.Source() {
		urls = append(urls, feed.URI)
	}
	cache.DeleteFeeds(urls...)

	return nil
}
