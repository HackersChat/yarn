// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package langs

import "embed"

//go:generate goi18n merge active.*.toml translate.*.toml

//go:embed active.*.toml
var LocaleFS embed.FS
