// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"git.mills.io/prologic/go-gopher"
	"github.com/makeworld-the-better-one/go-gemini"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:     "stats [flags] <url|file>",
	Aliases: []string{},
	Short:   "Parses and performs statistical analytis on a Twtxt feed given a URL or local file",
	Long:    `...`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runStats(args)
	},
}

func init() {
	RootCmd.AddCommand(statsCmd)
}

func runStats(args []string) {
	url, err := url.Parse(args[0])
	if err != nil {
		log.WithError(err).Error("error parsing url")
		os.Exit(2)
	}

	switch url.Scheme {
	case "", "file":
		f, err := os.Open(url.Path)
		if err != nil {
			log.WithError(err).Error("error reading file feed")
			os.Exit(2)
		}
		defer f.Close()

		doStats(f)
	case "http", "https":
		f, err := http.Get(url.String())
		if err != nil {
			log.WithError(err).Error("error reading HTTP feed")
			os.Exit(2)
		}
		defer f.Body.Close()

		doStats(f.Body)
	case "gopher":
		res, err := gopher.Get(url.String())
		if err != nil {
			log.WithError(err).Error("error reading Gopher feed")
			os.Exit(2)
		}
		defer res.Body.Close()

		doStats(res.Body)
	case "gemini":
		res, err := gemini.Fetch(url.String())
		if err != nil {
			log.WithError(err).Error("error reading Gemini feed")
			os.Exit(2)
		}
		defer res.Body.Close()

		doStats(res.Body)
	default:
		log.WithError(err).Errorf("unsupported url scheme: %s", url.Scheme)
		os.Exit(2)
	}
}

func doStats(r io.Reader) {
	twter := types.NilTwt.Twter()
	tf, err := lextwt.ParseFile(r, &twter)
	if err != nil {
		log.WithError(err).Error("error parsing feed")
		os.Exit(2)
	}

	fmt.Println(tf.Info())

	fmt.Printf("twter: %s\n", twter.DomainNick())
	fmt.Printf("nick: %s\n", twter.Nick)
	fmt.Printf("url: %s\n", twter.URI)
	fmt.Printf("avatar: %s\n", twter.Avatar)
	fmt.Printf("tagline: %s\n", twter.Tagline)

	fmt.Println("metadata:")
	for _, c := range tf.Info().GetAll("") {
		fmt.Printf("  %s = %s\n", c.Key(), c.Value())
	}

	fmt.Println("following:")
	for _, c := range tf.Info().Following() {
		fmt.Printf("  % -30s = %s\n", c.Nick, c.URI)
	}

	fmt.Println("twts: ", len(tf.Twts()))

	fmt.Printf("days of week:\n%v\n", daysOfWeek(tf.Twts()))

	fmt.Println("tags: ", len(tf.Twts().Tags()))
	fmt.Println(getTags(tf.Twts().Tags()))

	fmt.Println("mentions: ", len(tf.Twts().Mentions()))
	fmt.Println(getMentions(tf.Twts(), tf.Info().Following()))

	fmt.Println("subjects: ", len(tf.Twts().Subjects()))
	var subjects stats
	for subject, count := range tf.Twts().SubjectCount() {
		subjects = append(subjects, stat{count, subject})
	}
	fmt.Println(subjects)

	fmt.Println("links: ", len(tf.Twts().Links()))
	var links stats
	for link, count := range tf.Twts().LinkCount() {
		links = append(links, stat{count, link})
	}
	fmt.Println(links)
}

func daysOfWeek(twts types.Twts) stats {
	s := make(map[string]int)

	for _, twt := range twts {
		s[fmt.Sprint(twt.Created().Format("tz-Z0700"))]++
		s[fmt.Sprint(twt.Created().Format("dow-Mon"))]++
		s[fmt.Sprint(twt.Created().Format("year-2006"))]++
		s[fmt.Sprint(twt.Created().Format("day-2006-01-02"))]++
	}

	var lis stats
	for k, v := range s {
		lis = append(lis, stat{v, k})
	}
	return lis
}

func getMentions(twts types.Twts, follows []types.Twter) stats {
	counts := make(map[string]int)
	for _, m := range twts.Mentions() {
		t := m.Twter()
		counts[fmt.Sprint(t.Nick, "\t", t.URI)]++
	}

	lis := make(stats, 0, len(counts))
	for name, count := range counts {
		lis = append(lis, stat{count, name})
	}

	return lis
}

func getTags(twts types.TagList) stats {
	counts := make(map[string]int)
	for _, m := range twts {
		counts[fmt.Sprint(m.Text(), "\t", m.Target())]++
	}

	lis := make(stats, 0, len(counts))
	for name, count := range counts {
		lis = append(lis, stat{count, name})
	}

	return lis
}

type stat struct {
	count int
	text  string
}

func (s stat) String() string {
	return fmt.Sprintf("  %v : %v\n", s.count, s.text)
}

func (s stats) Len() int {
	return len(s)
}
func (s stats) Less(i, j int) bool {
	return s[i].count > s[j].count
}
func (s stats) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type stats []stat

func (s stats) String() string {
	var b strings.Builder
	sort.Sort(s)
	for _, line := range s {
		b.WriteString(line.String())
	}
	return b.String()
}
