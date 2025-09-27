// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	timelinec "git.mills.io/yarnsocial/yarn/cmd/yarnc/timeline"
	"go.yarn.social/client"
)

// timelineCmd represents the pub command
var timelineCmd = &cobra.Command{
	Use:     "timeline [flags]",
	Aliases: []string{"view", "show", "events"},
	Short:   "Display your timeline",
	Long: `The timeline command retrieve the timeline from the logged in
Yarn.social account.`,
	//Args:    cobra.NArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		uri := viper.GetString("uri")
		token := viper.GetString("token")
		cli, err := client.NewClient(
			client.WithURI(uri),
			client.WithToken(token),
		)
		if err != nil {
			log.WithError(err).Error("error creating client")
			os.Exit(1)
		}

		outputJSON, err := cmd.Flags().GetBool("json")
		if err != nil {
			log.WithError(err).Error("error getting json flag")
			os.Exit(1)
		}

		outputRAW, err := cmd.Flags().GetBool("raw")
		if err != nil {
			log.WithError(err).Error("error getting raw flag")
			os.Exit(1)
		}

		outputGIT, err := cmd.Flags().GetBool("git")
		if err != nil {
			log.WithError(err).Error("error getting git flag")
			os.Exit(1)
		}

		outputFollow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			log.WithError(err).Error("error getting git flag")
			os.Exit(1)
		}

		reverseOrder, err := cmd.Flags().GetBool("reverse")
		if err != nil {
			log.WithError(err).Error("error getting reverse flag")
			os.Exit(1)
		}

		nTwts, err := cmd.Flags().GetInt("twts")
		if err != nil {
			log.WithError(err).Error("error getting twts flag")
			os.Exit(1)
		}

		page, err := cmd.Flags().GetInt("page")
		if err != nil {
			log.WithError(err).Error("error getting page flag")
			os.Exit(1)
		}

		timeline(cli, outputJSON, outputRAW, outputGIT, outputFollow, reverseOrder, nTwts, page, args)
	},
}

func init() {
	RootCmd.AddCommand(timelineCmd)

	timelineCmd.Flags().IntP(
		"twts", "n", -1,
		"Number of Twts to display (default all)",
	)

	timelineCmd.Flags().IntP(
		"page", "p", 0,
		"Page number of Twts to retrieve",
	)

	timelineCmd.Flags().BoolP(
		"reverse", "r", false,
		"Reverse chronological order (newest first)",
	)

	timelineCmd.Flags().BoolP(
		"json", "J", false,
		"Output in JSON for processing with eg jq",
	)

	timelineCmd.Flags().BoolP(
		"raw", "R", false,
		"Output in raw text for processing with eg grep",
	)

	timelineCmd.Flags().BoolP(
		"git", "G", false,
		"Output in git log format",
	)

	timelineCmd.Flags().BoolP(
		"follow", "f", false,
		"output appended data as new twts arrive",
	)
}

func timeline(cli *client.Client, outputJSON, outputRAW, outputGIT, outputFollow, reverseOrder bool, nTwts, page int, args []string) {
	res, err := cli.Timeline(page)
	if err != nil {
		log.WithError(err).Error("error retrieving timeline")
		os.Exit(1)
	}

	if reverseOrder && !outputFollow {
		sort.Sort(res.Twts)
	} else {
		sort.Sort(sort.Reverse(res.Twts))
	}

	twts := res.Twts[:]

	if nTwts > 0 && nTwts < len(twts) {
		twts = twts[(len(twts) - nTwts):]
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	err = timelinec.GetParser(timelinec.Options{
		OutputJSON:   outputJSON,
		OutputRAW:    outputRAW,
		OutputGIT:    outputGIT,
		OutputFollow: outputFollow,
		Client:       cli,
	}).Parse(ctx, twts, cli.Twter)

	if err != nil {
		os.Exit(1)
	}
}
