// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"

	"git.mills.io/yarnsocial/yarn/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.yarn.social/client"
)

// syncCmd represents the pub command
var syncCmd = &cobra.Command{
	Use:   "sync [flags] file",
	Short: "Sync synchronizes a local twtxt.txt feed with a Yarn.social feed",
	Long: `The sync command synchronizes a local twtxt.feed with a Yarn.social feed
hosted on a Yarn.social pod. An 3-way merge is performed and the resulting merged twts
returned with the option of rewriting local or remote versionf ot the feed.`,
	Args: cobra.ExactArgs(1),
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

		delete, err := cmd.Flags().GetBool("delete")
		if err != nil {
			log.WithError(err).Error("error getting delete flag")
			os.Exit(1)
		}

		feed, err := cmd.Flags().GetString("feed")
		if err != nil {
			log.WithError(err).Error("error getting feed flag")
			os.Exit(1)
		}

		sync(cli, delete, feed, args[0])
	},
}

func init() {
	RootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolP(
		"delete", "d", false,
		"Delete twts from pod not found in local feed",
	)

	syncCmd.Flags().StringP(
		"feed", "f", "",
		"Feed to synchornize (empty for primary feed)",
	)
}

func sync(cli *client.Client, delete bool, feed string, fn string) {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_RDONLY, os.FileMode(0644))
	if err != nil {
		log.WithError(err).Error("error opening feed")
		os.Exit(1)
	}
	defer f.Close()

	res, err := cli.Sync(delete, feed, f)
	if err != nil {
		log.WithError(err).Error("error performing sync")
		os.Exit(1)
	}

	if !res.Success {
		fmt.Fprintf(os.Stderr, "error syncing feed: %s", res.Error)
		os.Exit(1)
	}

	if err := internal.RewriteFeed(fn, res.Merged, nil); err != nil {
		log.WithError(err).Errorf("error rewritign feed")
		os.Exit(1)
	}
}
