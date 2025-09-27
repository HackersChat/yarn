// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"net"
	"net/http"
	"time"
)

func init() {
	// Base your Transport off the default, then override only what you need.
	t := http.DefaultTransport.(*http.Transport).Clone()

	// Use DialContext instead of the older Dial
	t.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second, // TCP connect timeout
		KeepAlive: 30 * time.Second, // probe idle connections
	}).DialContext

	// Time to wait for full TLS handshake
	t.TLSHandshakeTimeout = 10 * time.Second

	// Time to first response header
	t.ResponseHeaderTimeout = 10 * time.Second

	// How long to wait for a 100-continue response
	t.ExpectContinueTimeout = 1 * time.Second

	// Reuse idle connections aggressively
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 10
	t.IdleConnTimeout = 90 * time.Second

	// Respect standard proxy env vars
	t.Proxy = http.ProxyFromEnvironment

	http.DefaultClient = &http.Client{
		Transport: t,
		// Overall per-request deadline: connect → read full body
		Timeout: 30 * time.Second,
	}
}
