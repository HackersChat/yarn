// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goware/urlx"
	log "github.com/sirupsen/logrus"
)

// FormatTwt formats a twt's text into a human-readable format, suitable for a timeline
// or a twt's details page. It replaces @-mentions and #-tags with simple @nick and #tag strings.
func FormatTwt(text string) string {
	re := regexp.MustCompile(`(@|#)<([^ ]+) *([^>]+)>`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		prefix, nick, _ := parts[1], parts[2], parts[3]

		switch prefix {
		case "@":
			return fmt.Sprintf("@%s", nick)
		default:
			return fmt.Sprintf("%s%s", prefix, nick)
		}
	})
}

// NormalizeURL ...
func NormalizeURL(url string) string {
	if url == "" {
		return ""
	}
	u, err := urlx.Parse(url)
	if err != nil {
		log.WithError(err).Errorf("error parsing url %s", url)
		return ""
	}
	if u.Scheme == "http" && strings.HasSuffix(u.Host, ":80") {
		u.Host = strings.TrimSuffix(u.Host, ":80")
	}
	if u.Scheme == "https" && strings.HasSuffix(u.Host, ":443") {
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}
	u.User = nil
	u.Path = strings.TrimSuffix(u.Path, "/")
	norm, err := urlx.Normalize(u)
	if err != nil {
		log.WithError(err).Errorf("error normalizing url %s", url)
		return ""
	}
	return norm
}
