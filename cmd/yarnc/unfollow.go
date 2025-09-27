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
var unfollowCmd = &cobra.Command{
	Use:   "unfollow <NICK>",
	Short: "Unfollow another user of an existing twtxt.txt feed",
	Long:  `Unfollow another user of an existing twtxt.txt feed`,
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

		if len(args) != 1 {
			log.Error("wrong arguments")
			os.Exit(1)
		}

		nick := args[0]

		unfollow(cli, nick)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("could not unfollow %s", nick))
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(unfollowCmd)
}

func unfollow(cli *client.Client, nick string) error {
	err := cli.Unfollow(nick)
	if err != nil {
		return err
	}
	return nil
}
