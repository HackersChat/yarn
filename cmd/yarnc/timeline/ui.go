// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package timeline

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"git.mills.io/yarnsocial/yarn/cmd/yarnc/timeline/renderer"
	"github.com/dustin/go-humanize"
	"github.com/russross/blackfriday"
	"go.yarn.social/types"
)

func red(s string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", s)
}
func green(s string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", s)
}
func yellow(s string) string {
	return fmt.Sprintf("\033[33m%s\033[0m", s)
}
func blue(s string) string {
	return fmt.Sprintf("\033[34m%s\033[0m", s)
}
func purple(s string) string {
	return fmt.Sprintf("\033[35m%s\033[0m", s)
}
func cyan(s string) string {
	return fmt.Sprintf("\033[36m%s\033[0m", s)
}
func white(s string) string {
	return fmt.Sprintf("\033[0;37m%s\033[0m", s)
}
func redBold(s string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", s)
}
func boldgreen(s string) string {
	return fmt.Sprintf("\033[32;1m%s\033[0m", s)
}

var colorArray = []func(string) string{red, green, yellow, blue, purple, cyan}

func HashColor(hash string) func(string) string {
	//Hash
	h := fnv.New32a()
	h.Write([]byte(hash))
	n := h.Sum32() % uint32(len(colorArray))

	//Get func
	f := colorArray[n]
	return f
}

func PrintFollowee(nick, url string) {
	fmt.Printf("> %s @ %s",
		yellow(nick),
		url,
	)
}

func PrintFolloweeRaw(nick, url string) {
	fmt.Printf("%s: %s\n", nick, url)
}

func PrintTwt(twt types.Twt, now time.Time, me types.Twter) {
	time := humanize.Time(twt.Created())
	nick := green(twt.Twter().DomainNick())
	hash := blue(twt.Hash())

	if twt.Mentions().IsMentioned(me) {
		nick = boldgreen(twt.Twter().DomainNick())
	}

	renderer := &renderer.Console{}
	extensions := 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS

	input := []byte(twt.FormatText(types.MarkdownFmt, nil))
	output := blackfriday.Markdown(input, renderer, extensions)

	fmt.Printf("> %s (%s) [%s]\n%s", nick, time, hash, string(output))
}

func PrintTwtRaw(twt types.Twt) {
	fmt.Printf(
		"%s\t%s\t%s\n",
		twt.Twter().URI,
		twt.Created().Format(time.RFC3339),
		strings.ReplaceAll(fmt.Sprintf("%t", twt), "\n", "\u2028"),
	)
}
