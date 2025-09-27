// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"go.yarn.social/client"
)

// loginCmd represents the pub command
var loginCmd = &cobra.Command{
	Use:     "login [flags]",
	Aliases: []string{"auth"},
	Short:   "Login and authenticate to a Yarn.social pod",
	Long: `The login command allows you to login a user ot login to a
Yarn.social pod running yarnd. Once successfully authenticated with a valid
account a API token is generated on the account and a configuration file is
written to store the endpoint logged in to and the token for future used by
the command-line client.`,
	Args: cobra.MaximumNArgs(0),
	Run: func(_ *cobra.Command, _ []string) {
		uri := viper.GetString("uri")
		cli, err := client.NewClient(client.WithURI(uri))
		if err != nil {
			log.WithError(err).Error("error creating client")
			os.Exit(1)
		}

		login(cli)
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
}

func readCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Error("error reading username")
		return "", "", err
	}

	fmt.Print("Password: ")
	data, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.WithError(err).Error("error reading password")
		return "", "", err
	}
	password := string(data)

	return username, password, nil
}

func login(cli *client.Client) {
	username, password, err := readCredentials()
	if err != nil {
		log.WithError(err).Error("error reading credentials")
		os.Exit(1)
	}

	res, err := cli.Login(username, password)
	if err != nil {
		log.WithError(err).Error("error making login request")
		os.Exit(1)
	}

	log.Info("login successful")

	cli.Config.Token = res.Token
	if err := cli.Config.Save(viper.ConfigFileUsed()); err != nil {
		log.WithError(err).Error("error saving config")
		os.Exit(1)
	}
}
