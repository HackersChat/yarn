// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.yarn.social/client"
)

// postCmd represents the pub command
var followCmd = &cobra.Command{
	Use:   "follow <NICK> <URL>",
	Short: "Follow another user of an existing twtxt.txt feed",
	Long:  `Follow another user of an existing twtxt.txt feed`,
	//Args:    cobra.NArgs(0),
	Run: func(_ *cobra.Command, args []string) {
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

		if len(args) != 2 {
			log.Error("wrong arguments")
			os.Exit(1)
		}

		nick := args[0]
		url := args[1]

		follow(cli, nick, url)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("could not follow %s %s", nick, url))
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(followCmd)
}

func follow(cli *client.Client, nick, url string) error {
	err := cli.Follow(nick, url)
	if err != nil {
		return err
	}
	return nil
}
