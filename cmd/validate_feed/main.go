// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"git.mills.io/prologic/go-gopher"
	"github.com/makeworld-the-better-one/go-gemini"
	log "github.com/sirupsen/logrus"

	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

func main() {
	flag.Parse()

	uri := flag.Arg(0)
	logger := log.WithField("url", uri)

	url, err := url.Parse(uri)
	if err != nil {
		logger.WithError(err).Error("error parsing uri")
		os.Exit(2)
	}

	var r io.Reader

	switch url.Scheme {
	case "", "file":
		f, err := os.Open(url.Path)
		if err != nil {
			logger.WithError(err).Error("error reading file")
			os.Exit(2)
		}
		r = f
		defer f.Close()
	case "http", "https":
		res, err := http.Get(url.String())
		if err != nil {
			logger.WithError(err).Error("error reading resource")
			os.Exit(2)
		}
		r = res.Body
		defer res.Body.Close()
	case "gopher":
		res, err := gopher.Get(url.String())
		if err != nil {
			logger.WithError(err).Error("error reading resource")
			os.Exit(2)
		}
		r = res.Body
		defer res.Body.Close()
	case "gemini":
		res, err := gemini.Fetch(url.String())
		if err != nil {
			logger.WithError(err).Error("error reading resource")
			os.Exit(2)
		}
		r = res.Body
		defer res.Body.Close()
	default:
		logger.WithError(err).Errorf("unsupported uri scheme")
		os.Exit(2)
	}

	if err := validateFeed(r); err != nil {
		logger.WithError(err).Error("error validating feed")
		os.Exit(2)
	}
}

func validateFeed(r io.Reader) error {
	twter := types.NilTwt.Twter()
	_, err := lextwt.ParseFile(r, &twter)
	if err != nil {
		return fmt.Errorf("error parsing feed: %w", err)
	}
	return nil
}
