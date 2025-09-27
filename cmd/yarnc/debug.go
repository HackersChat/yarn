// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"git.mills.io/prologic/go-gopher"
	"github.com/makeworld-the-better-one/go-gemini"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:     "debug [flags] <url|file>",
	Aliases: []string{},
	Short:   "Parses and debugs the Twtxt feed given a URL or local file",
	Long:    `...`,
	Args:    cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		runDebug(args)
	},
}

func init() {
	RootCmd.AddCommand(debugCmd)
}

func runDebug(args []string) {
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

		doDebug(url.String(), f)
	case "http", "https":
		f, err := http.Get(url.String())
		if err != nil {
			log.WithError(err).Error("error reading HTTP feed")
			os.Exit(2)
		}
		defer f.Body.Close()

		doDebug(url.String(), f.Body)
	case "gopher":
		res, err := gopher.Get(url.String())
		if err != nil {
			log.WithError(err).Error("error reading Gopher feed")
			os.Exit(2)
		}
		defer res.Body.Close()

		doDebug(url.String(), res.Body)
	case "gemini":
		res, err := gemini.Fetch(url.String())
		if err != nil {
			log.WithError(err).Error("error reading Gemini feed")
			os.Exit(2)
		}
		defer res.Body.Close()

		doDebug(url.String(), res.Body)
	default:
		log.WithError(err).Errorf("unsupported url scheme: %s", url.Scheme)
		os.Exit(2)
	}
}

func doDebug(url string, r io.Reader) {
	twter := types.NewTwter("", url)
	tf, err := lextwt.ParseFile(r, &twter)
	if err != nil {
		log.WithError(err).Error("error parsing feed")
		os.Exit(2)
	}

	for _, twt := range tf.Twts() {
		fmt.Printf("%s %+l\n", twt.Hash(), twt)
	}
}
