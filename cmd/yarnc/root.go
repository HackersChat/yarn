// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"git.mills.io/yarnsocial/yarn"
	"go.yarn.social/client"
	_ "go.yarn.social/lextwt"
)

const (
	// DefaultConfigFilename ...
	DefaultConfigFilename = ".yarnc.yml"

	// DefaultEnvPrefix ...
	DefaultEnvPrefix = "YARNC"
)

var (
	ConfigFile        string
	DefaultConfigFile string
)

func init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.WithError(err).Fatal("error finding user home directory")
	}

	DefaultConfigFile = filepath.Join(homeDir, DefaultConfigFilename)
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "yarnc",
	Version: yarn.FullVersion(),
	Short:   "Command-line client for yarnd",
	Long: `This is the command-line client for Yarn.social pods running
yarnd. This tool allows a user to interact with a pod to view their timeline,
following feeds, make posts and managing their account.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// set logging level
		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
	},
}

// Execute adds all child commands to the root command
// and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.WithError(err).Error("error executing command")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(
		&ConfigFile, "config", "c", DefaultConfigFile,
		"set a custom config file",
	)

	RootCmd.PersistentFlags().BoolP(
		"debug", "D", false,
		"Enable debug logging",
	)

	RootCmd.PersistentFlags().StringP(
		"uri", "U", client.DefaultURI,
		"Pod URL to connect to",
	)

	RootCmd.PersistentFlags().StringP(
		"token", "T", fmt.Sprintf("$%s_TOKEN", DefaultEnvPrefix),
		"yarnd API token to use to authenticate to endpoints",
	)

	viper.BindPFlag("uri", RootCmd.PersistentFlags().Lookup("uri"))
	viper.SetDefault("uri", client.DefaultURI)

	viper.BindPFlag("token", RootCmd.PersistentFlags().Lookup("token"))
	viper.SetDefault("token", os.Getenv(fmt.Sprintf("%s_TOKEN", DefaultEnvPrefix)))

	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))
	viper.SetDefault("debug", false)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigFile(ConfigFile)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Warnf("error loading config file: %s", viper.ConfigFileUsed())
	}

	// from the environment
	viper.SetEnvPrefix(DefaultEnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}
