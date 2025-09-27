// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tj/go-editor"

	"go.yarn.social/client"
)

// postCmd represents the pub command
var postCmd = &cobra.Command{
	Use:     "post [flags]  [- | [text]]",
	Aliases: []string{"tweet", "twt", "new", "yarn"},
	Short:   "Post a new twt to a Yarn.social pod",
	Long: `The post command makes a new post to a Yarn.social pod.
if the optional flag -a/--post-as is used the post is made from that specified
feed (persona) if the logged in user owned that feed. The post is read from
command-line arguments if provided, standard input if "-" is the only argument
provided or opens your $EDITOR for writing if no arguments are supplied.`,
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

		post(cli, args)
	},
}

func init() {
	RootCmd.AddCommand(postCmd)

	postCmd.Flags().StringP(
		"post-as", "a", "",
		"Post as a different feed (default: primary account feed)",
	)
}

func post(cli *client.Client, args []string) {
	var text string

	readFromStdin := len(args) == 1 && args[0] == "-"
	readFromEditor := len(args) == 0

	if readFromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.WithError(err).Error("error reading text from stdin")
			os.Exit(1)
		}
		text = string(data)
	} else if readFromEditor {
		data, err := editor.Read()
		if err != nil {
			log.WithError(err).Error("error reading text from editor")
			os.Exit(1)
		}
		text = string(data)
	} else {
		text = strings.Join(args, " ")
	}

	if text == "" {
		log.Error("no text provided")
		os.Exit(1)
	}

	_, err := cli.Post(text, "")
	if err != nil {
		log.WithError(err).Error("error making post")
		os.Exit(1)
	}
}
