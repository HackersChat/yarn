// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"
)

func fastHash(data []byte) string {
	sum := blake2b.Sum256(data)

	// Base32 is URL-safe, unlike Base64, and shorter than hex.
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	hash := strings.ToLower(encoding.EncodeToString(sum[:]))

	return hash
}

func main() {
	data, _ := io.ReadAll(os.Stdin)
	fmt.Println(fastHash(data))
}
