// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"time"

	"github.com/marksalpeter/token/v2"
)

var tokenCache *TTLCache

func init() {
	// #244: How to make discoverability via user agents work again?
	// TODO: Make the token cache expiry configurable?
	tokenCache = NewTTLCache(30 * time.Minute)
}

func GenerateWhoFollowsToken(feedurl string) string {
	t := token.New().Encode()
	for {
		if tokenCache.GetString(t) == "" {
			tokenCache.SetString(t, feedurl)
			return t
		}
		t = token.New().Encode()
	}
}
