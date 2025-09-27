// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.yarn.social/types"
)

// hashCmd represents the hash command
var hashCmd = &cobra.Command{
	Use:   "hash [flags]  [- | [text]]",
	Short: "Calculates a hash of a Twt",
	Long: `The hash command calculates the Twt Hash[1] for a given Twter,
Timestamp and Content. Twt Hashes use a blake2b hashing algorithm with a
base32 encoding and are shortened to 7 characters for human convenience without
sacrificing collisions.

This tool is useful for older Twtxt clients that don't yet implement the
Twt Hash extension, giving users the ability to construct replies to Twts
with their existing tools/clients.

The Twter (-u/--twter) flag accepts a valid URL to the Feed, e.g:
  -u https://twtxt.net/user/prologic/twtxt.txt

The Timestamp (-t/--time) flag accepts a valid RFC3339 timestamp, e.g:
  -t 2020-07-18T12:39:52Z

A full example usage is:

  $ yarnc hash -u https://twtxt.net/user/prologic/twtxt.txt -t 2020-07-18T12:39:52Z "Hello World! 😊"
  o6dsrga

Tip: You can also verify the hash by visiting any Yarn.social[2] Pod and using
the Permalink resource, e.g:

  https://twtxt.net/twt/o6dsrga

[1]: https://dev.twtxt.net/doc/twthashextension.html
[2]: https://yarn.social/
`,
	Run: func(cmd *cobra.Command, args []string) {
		ts, err := cmd.Flags().GetString("time")
		if err != nil {
			log.WithError(err).Error("error getting time flag")
			os.Exit(1)
		}

		uri, err := cmd.Flags().GetString("twter")
		if err != nil {
			log.WithError(err).Error("error getting twter flag")
			os.Exit(1)
		}

		hash(uri, ts, args)
	},
}

func init() {
	RootCmd.AddCommand(hashCmd)

	hashCmd.Flags().StringP(
		"twter", "u", "",
		"Set the Twter's URI (URL of the feed)",
	)

	hashCmd.Flags().StringP(
		"time", "t", "",
		"Post as a different feed (default: primary account feed)",
	)
}

func hash(uri, ts string, args []string) {
	twter := types.Twter{Nick: "nobody", URI: uri}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		log.WithError(err).Errorf("error parsing timestamp %q: %s", ts, err)
		os.Exit(1)
	}

	var text string

	readFromStdin := len(args) == 0 || (len(args) == 1 && args[0] == "-")

	if readFromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.WithError(err).Error("error reading text from stdin")
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

	twt := types.MakeTwt(twter, t, text)
	fmt.Println(twt.Hash())
}
