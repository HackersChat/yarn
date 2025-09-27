// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yarn.social/lextwt"
	"go.yarn.social/types"
)

func TestParseUserAgent(t *testing.T) {
	testCases := []struct {
		name     string
		ua       string
		err      error
		expected TwtxtUserAgent
	}{
		{
			name: "non-Twtxt User Agent",
			ua:   `Linguee Bot (http://www.linguee.com/bot; bot@linguee.com)`,
			err:  ErrInvalidUserAgent,
		},
		{
			name: "Single-User Twtxt User Agent",
			ua:   `twtxt/1.2.3 (+https://foo.com/twtxt.txt; @foo)`,
			expected: &SingleUserAgent{
				twtxtUserAgent: twtxtUserAgent{Client: "twtxt/1.2.3"},
				Nick:           "foo",
				URI:            "https://foo.com/twtxt.txt",
			},
		},
		{
			name: "Multi-User Twtxt User Agent",
			ua:   `yarnd/0.8.0@d4e265e (~https://example.com/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/support)`,
			expected: &MultiUserAgent{
				twtxtUserAgent: twtxtUserAgent{Client: "yarnd/0.8.0@d4e265e"},
				WhoFollowsURL:  "https://example.com/whoFollows?followers=14&token=iABA0yhUz",
				SupportURL:     "https://example.com/support",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ParseUserAgent(testCase.ua)
			if testCase.err != nil {
				assert.Equal(t, testCase.err, err)
			} else {
				assert.NoError(t, err)
				assert.IsType(t, testCase.expected, actual)
				switch act := actual.(type) {
				case *SingleUserAgent:
					assert.Equal(t, testCase.expected.(*SingleUserAgent).Client, act.Client)
					assert.Equal(t, testCase.expected.(*SingleUserAgent).Nick, act.Nick)
					assert.Equal(t, testCase.expected.(*SingleUserAgent).URI, act.URI)
				case *MultiUserAgent:
					assert.Equal(t, testCase.expected.(*MultiUserAgent).Client, act.Client)
					assert.Equal(t, testCase.expected.(*MultiUserAgent).WhoFollowsURL, act.WhoFollowsURL)
					assert.Equal(t, testCase.expected.(*MultiUserAgent).SupportURL, act.SupportURL)
				default:
					assert.Fail(t, "test setup error: unsupported user agent type")
				}
			}
		})
	}
}

func TestTwtxtUserAgent_IsPod(t *testing.T) {
	testCases := []struct {
		name     string
		ua       string
		expected bool
	}{
		{
			name:     "Single-User non-yarnd User Agent",
			ua:       "twtxt/1.2.3 (+https://example.com/twtxt.txt; @foo)",
			expected: false,
		},
		{
			name:     "Single-User yarnd User Agent",
			ua:       "yarnd/0.8.0@d4e265e (+https://example.com/user/foo/twtxt.txt; @foo)",
			expected: true,
		},
		{
			name:     "Multi-User non-yarnd User Agent",
			ua:       "bernd/0.8.0@d4e265e (~https://example.com/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/support)",
			expected: false,
		},
		{
			name:     "Multi-User yarnd User Agent",
			ua:       "yarnd/0.8.0@d4e265e (~https://example.com/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/support)",
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ua, err := ParseUserAgent(testCase.ua)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, ua.IsPod())
		})
	}
}

func TestTwtxtUserAgent_PodBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Single-User non-yarnd User Agent",
			ua:       "twtxt/1.2.3 (+https://example.com/twtxt.txt; @foo)",
			expected: "",
		},
		{
			name:     "Single-User yarnd User Agent",
			ua:       "yarnd/0.8.0@d4e265e (+https://example.com/user/foo/twtxt.txt; @foo)",
			expected: "https://example.com",
		},
		{
			name:     "Single-User yarnd with subdirectory User Agent",
			ua:       "yarnd/0.8.0@d4e265e (+https://example.com/subdir/user/foo/twtxt.txt; @foo)",
			expected: "https://example.com/subdir",
		},
		{
			name:     "Multi-User non-yarnd User Agent",
			ua:       "bernd/0.8.0@d4e265e (~https://example.com/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/support)",
			expected: "",
		},
		{
			name:     "Multi-User yarnd User Agent",
			ua:       "yarnd/0.8.0@d4e265e (~https://example.com/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/support)",
			expected: "https://example.com",
		},
		{
			name:     "Multi-User yarnd with subdirectory User Agent",
			ua:       "yarnd/0.8.0@d4e265e (~https://example.com/subdir/whoFollows?followers=14&token=iABA0yhUz; contact=https://example.com/subdir/support)",
			expected: "https://example.com/subdir",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ua, err := ParseUserAgent(testCase.ua)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, ua.PodBaseURL())
		})
	}
}

func TestFormatMentionsAndTags(t *testing.T) {
	conf := &Config{BaseURL: "http://0.0.0.0:8000"}

	testCases := []struct {
		text     string
		format   TwtTextFormat
		expected string
	}{
		{
			text:     "@<test http://0.0.0.0:8000/user/test/twtxt.txt>",
			format:   HTMLFmt,
			expected: `<a href="http://0.0.0.0:8000/user/test/">@test</a>`,
		},
		{
			text:     "@<test http://0.0.0.0:8000/user/test/twtxt.txt>",
			format:   MarkdownFmt,
			expected: "[@test](http://0.0.0.0:8000/user/test/twtxt.txt#test)",
		},
		{
			text:     "@<iamexternal http://iamexternal.com/twtxt.txt>",
			format:   HTMLFmt,
			expected: fmt.Sprintf(`<a href="%s">@iamexternal</a>`, URLForExternalProfile(conf, "http://iamexternal.com/twtxt.txt")),
		},
		{
			text:     "@<iamexternal http://iamexternal.com/twtxt.txt>",
			format:   MarkdownFmt,
			expected: "[@iamexternal](http://iamexternal.com/twtxt.txt#iamexternal)",
		},
		{
			text:     "#<test http://0.0.0.0:8000/search?q=%23test>",
			format:   HTMLFmt,
			expected: `<a href="http://0.0.0.0:8000/search?q=%23test">#test</a>`,
		},
		{
			text:     "#<test http://0.0.0.0:8000/search?q=%23test>",
			format:   MarkdownFmt,
			expected: `[#test](http://0.0.0.0:8000/search?q=%23test)`,
		},
	}

	for _, testCase := range testCases {
		actual := FormatMentionsAndTags(conf, testCase.text, testCase.format)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestIsLocalURL(t *testing.T) {
	testCases := []struct {
		url      string
		baseURL  string
		expected bool
	}{
		{
			url:      "https://feeds.twtxt.cc",
			baseURL:  "https://www.twtxt.cc",
			expected: false,
		},
		{
			url:      "http://localhost:8001",
			baseURL:  "http://localhost:8000",
			expected: false,
		},
		{
			url:      "http://localhost:8000/ext",
			baseURL:  "http://localhost:8000",
			expected: true,
		},
	}

	for _, testCase := range testCases {
		actual := strings.HasPrefix(NormalizeURL(testCase.url), NormalizeURL(testCase.baseURL))
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestFormatTwtFactory(t *testing.T) {
	assert := assert.New(t)

	cfg := NewConfig()
	cfg.baseURL = &url.URL{Host: "example.com"}
	twter := types.Twter{
		Nick: "test",
		URI:  "https://example.com/twtxt.txt",
	}
	cache, _ := NewSqliteCache(":memory")
	cache.SetTwter(twter.URI, twter)
	factory := FormatTwtFactory(cfg, cache)

	t.Run("image", func(_ *testing.T) {
		actual := string(factory(lextwt.NewTwt(twter,
			lextwt.NewDateTime(parseTime("2021-01-24T02:19:54Z"), "2021-01-24T02:19:54Z"),
			lextwt.NewMedia("This is alt", "https://example.com/hot.png", "'This is title'"),
		), types.HTMLFmt, NewUser()))
		expected := "<p><p><img loading=\"lazy\" src=\"//example.com/hot.png\" title=\"This is title\" alt=\"This is alt\"/></p>\n</p>"
		assert.Equal(expected, actual)
	})

	t.Run("when nick for URL-only mention is resolvable, then use nick", func(_ *testing.T) {
		actual := string(factory(lextwt.NewTwt(twter,
			lextwt.NewDateTime(parseTime("2025-01-22T22:39:50+01:00"), "2025-01-22T22:39:50+01:00"),
			lextwt.NewMention("", "https://example.com/twtxt.txt"),
		), types.HTMLFmt, NewUser()))
		expected := `<p><p><a href="http://0.0.0.0:8000/external?uri=https://example.com/twtxt.txt&amp;nick=test" ` +
			`target="_blank" rel="nofollow noopener">@test<em>@example.com</em></a></p>` + "\n</p>"
		assert.Equal(expected, actual)
	})

	t.Run("when nick for URL-only mention is unresolvable, then fall back to domain name as nick", func(_ *testing.T) {
		// this manifests the lextwt fallback behavior
		actual := string(factory(lextwt.NewTwt(twter,
			lextwt.NewDateTime(parseTime("2025-01-22T22:39:50+01:00"), "2025-01-22T22:39:50+01:00"),
			lextwt.NewMention("", "https://example.org/twtxt.txt"),
		), types.HTMLFmt, NewUser()))
		expected := `<p><p><a href="http://0.0.0.0:8000/external?uri=https://example.org/twtxt.txt&amp;nick=example.org" ` +
			`target="_blank" rel="nofollow noopener">@example.org<em>@example.org</em></a></p>` + "\n</p>"
		assert.Equal(expected, actual)
	})
}

func TestSlugify(t *testing.T) {
	for _, tt := range []struct {
		name, url, expected string
	}{
		{
			name:     "only hostname and path are slugified",
			url:      "https://user:pass@example.com:1234/user/kate/twtxt.txt.png?query=ignored&also#anchor",
			expected: "example-com-user-kate-twtxt-txt-png",
		},
		{
			name:     "all chars are converted to lowercase",
			url:      "https://examPLE.COM/Foo/BaR",
			expected: "example-com-foo-bar",
		},
		{
			name:     "no consecutive dashes",
			url:      "https://sub.example.com/bla äöü _test/",
			expected: "sub-example-com-bla-_test",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Slugify(tt.url))
		})
	}
}
