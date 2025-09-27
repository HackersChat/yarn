// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	text_template "text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	humanize "github.com/dustin/go-humanize"
	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"go.yarn.social/types"
)

const (
	baseTemplate     = "base.html"
	partialsTemplate = "partials.html"
	baseName         = "base"
	contentName      = "content"
	metadataName     = "metadata"
)

var (
	customTimeMagnitudes = []humanize.RelTimeMagnitude{
		{D: time.Second, Format: "now", DivBy: time.Second},
		{D: time.Minute, Format: "%ds %s", DivBy: time.Second},
		{D: time.Hour, Format: "%dm %s", DivBy: time.Minute},
		{D: humanize.Day, Format: "%dh %s", DivBy: time.Hour},
		{D: humanize.Week, Format: "%dd %s", DivBy: humanize.Day},
		{D: humanize.Year, Format: "%dw %s", DivBy: humanize.Week},
		{D: humanize.LongTime, Format: "%dy %s", DivBy: humanize.Year},
		{D: math.MaxInt64, Format: "a long while %s", DivBy: 1},
	}

	lastseenTimeMagnitudes = []humanize.RelTimeMagnitude{
		{D: humanize.Day, Format: "today", DivBy: time.Hour},
		{D: humanize.Week, Format: "%dd %s", DivBy: humanize.Day},
		{D: humanize.Year, Format: "%dw %s", DivBy: humanize.Week},
		{D: humanize.LongTime, Format: "%dy %s", DivBy: humanize.Year},
		{D: math.MaxInt64, Format: "never", DivBy: 1},
	}
)

func CustomRelTime(a, b time.Time, albl, blbl string) string {
	return humanize.CustomRelTime(a, b, albl, blbl, customTimeMagnitudes)
}

func CustomTime(then time.Time) string {
	return CustomRelTime(then, time.Now(), "ago", "from now")
}

func LastSeenRelTime(a, b time.Time, albl, blbl string) string {
	return humanize.CustomRelTime(a, b, albl, blbl, lastseenTimeMagnitudes)
}

func LastSeenTime(then time.Time) string {
	return LastSeenRelTime(then, time.Now(), "ago", "from now")
}

type TemplateManager struct {
	sync.RWMutex

	debug   bool
	tmplFS  fs.FS
	tmplMap map[string]*template.Template
	funcMap template.FuncMap
}

// NewTemplateManager ...
func NewTemplateManager(conf *Config, translator *Translator, cache Cacher) (*TemplateManager, error) {
	tmplMap := make(map[string]*template.Template)

	funcMap := sprig.FuncMap()

	funcMap["time"] = CustomTime
	funcMap["lastseen"] = LastSeenTime
	funcMap["hostnameFromURL"] = HostnameFromURL
	funcMap["baseFromURL"] = BaseFromURL
	funcMap["prettyURL"] = PrettyURL
	funcMap["isLocalURL"] = IsLocalURLFactory(conf)
	funcMap["formatTwt"] = func(twt types.Twt, u *User) template.HTML {
		return FormatTwtFactory(conf, cache)(twt, types.HTMLFmt, u)
	}
	funcMap["unparseTwt"] = UnparseTwtFactory(conf, cache)
	funcMap["formatTwtContext"] = FormatTwtContextFactory(conf, cache)
	funcMap["getRootTwt"] = GetRootTwtFactory(conf, cache)
	funcMap["formatForDateTime"] = FormatForDateTime
	funcMap["urlForConv"] = URLForConvFactory(conf, cache)
	funcMap["urlForFork"] = URLForForkFactory(conf, cache)
	funcMap["urlForRootConv"] = URLForRootConvFactory(conf, cache, false)
	funcMap["urlForRootConvWithPager"] = URLForRootConvFactory(conf, cache, true)
	funcMap["getConvLength"] = GetConvLength(conf, cache)
	funcMap["getForkLength"] = GetForkLength(conf, cache)
	funcMap["getFeedTypeClass"] = GetFeedTypeClass(conf, cache)
	funcMap["isAdminUser"] = IsAdminUserFactory(conf)
	funcMap["isFeatureEnabled"] = func(name string) bool {
		return IsFeatureEnabled(conf.Features, name)
	}
	funcMap["hasFilter"] = func(r *http.Request, name string) bool {
		return HasString(r.URL.Query()["f"], name)
	}
	funcMap["toggleFilter"] = func(r *http.Request, name string) string {
		u, _ := url.Parse(r.URL.String())
		q := u.Query()
		if HasString(q["f"], name) {
			v := url.Values{}
			for key, val := range q {
				if key != "f" {
					v[key] = val
				}
			}
			for _, x := range q["f"] {
				if !strings.EqualFold(x, name) {
					v.Add("f", x)
				}
			}
			u.RawQuery = v.Encode()
		} else {
			q.Add("f", name)
			u.RawQuery = q.Encode()
		}
		return u.String()
	}
	funcMap["clearFilters"] = func(r *http.Request) string {
		u, _ := url.Parse(r.URL.String())
		q := u.Query()
		q.Del("f")
		u.RawQuery = q.Encode()
		return u.String()
	}

	funcMap["html"] = func(text string) template.HTML { return template.HTML(text) }
	funcMap["tr"] = func(ctx *Context, msgid string, data ...interface{}) string {
		return translator.Translate(ctx, msgid, data...)
	}
	funcMap["attr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}
	funcMap["safe"] = func(s string) template.HTML {
		return template.HTML(s)
	}
	funcMap["url"] = func(s string) template.URL {
		return template.URL(s)
	}
	funcMap["getCachedFeed"] = func(url string) *Feed {
		return cache.GetCachedFeed(url)
	}
	funcMap["mentionForProfile"] = func(profile types.Profile) string {
		if profile.Nick == "" {
			return fmt.Sprintf("@<%s>", profile.URI)
		}
		return fmt.Sprintf("@<%s %s>", profile.Nick, profile.URI)
	}

	m := &TemplateManager{
		debug:   conf.Debug,
		tmplFS:  conf.TemplatesFS(),
		tmplMap: tmplMap,
		funcMap: funcMap,
	}

	if err := m.LoadTemplates(); err != nil {
		log.WithError(err).Error("error loading templates")
		return nil, fmt.Errorf("error loading templates: %w", err)
	}

	return m, nil
}

func (m *TemplateManager) LoadTemplates() error {
	m.Lock()
	defer m.Unlock()

	err := fs.WalkDir(m.tmplFS, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).Error("error walking templates")
			return fmt.Errorf("error walking templates: %w", err)
		}

		fname := info.Name()
		if !info.IsDir() && filepath.Base(path) != baseTemplate {
			// Skip partials.html and also editor swap files, to improve the development
			// cycle. Editors often add suffixes to their swap files, e.g "~" or ".swp"
			// (Vim) and those files are not parsable as templates, causing panics.
			if fname == partialsTemplate || !strings.HasSuffix(fname, ".html") {
				return nil
			}

			name := strings.TrimSuffix(fname, filepath.Ext(fname))
			t := template.New(name).Option("missingkey=zero")
			t.Funcs(m.funcMap)

			if f, err := fs.ReadFile(m.tmplFS, path); err == nil {
				template.Must(t.Parse(string(f)))
			} else {
				return fmt.Errorf("error parsing template %s: %w", path, err)
			}

			if f, err := fs.ReadFile(m.tmplFS, partialsTemplate); err == nil {
				template.Must(t.Parse(string(f)))
			} else {
				return fmt.Errorf("error parsing partials template %s: %w", partialsTemplate, err)
			}

			if f, err := fs.ReadFile(m.tmplFS, baseTemplate); err == nil {
				template.Must(t.Parse(string(f)))
			} else {
				return fmt.Errorf("error parsing base template %s: %w", baseTemplate, err)
			}

			m.tmplMap[name] = t
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("error loading templates")
		return fmt.Errorf("error loading templates: %w", err)
	}
	return nil
}

func (m *TemplateManager) Add(name string, template *template.Template) {
	m.Lock()
	defer m.Unlock()

	m.tmplMap[name] = template
}

func (m *TemplateManager) exec(name string, partial bool, ctx *Context) (io.WriterTo, error) {
	if m.debug {
		log.Debug("reloading templates in debug mode...")
		if err := m.LoadTemplates(); err != nil {
			log.WithError(err).Error("error reloading templates")
			return nil, fmt.Errorf("error reloading templates: %w", err)
		}
	}

	m.RLock()
	template, ok := m.tmplMap[name]
	m.RUnlock()

	if !ok {
		log.WithField("name", name).Errorf("template not found")
		return nil, fmt.Errorf("no such template: %s", name)
	}

	if ctx == nil {
		ctx = &Context{}
	}

	var err error

	buf := bytes.NewBuffer(nil)

	if partial {
		err = func() error {
			if err := template.ExecuteTemplate(buf, contentName, ctx); err != nil {
				return err
			}
			if err := template.ExecuteTemplate(buf, metadataName, ctx); err != nil {
				return err
			}
			return nil
		}()
	} else {
		err = template.ExecuteTemplate(buf, baseName, ctx)
	}

	if err != nil {
		log.WithError(err).WithField("name", name).Errorf("error executing template")
		return nil, fmt.Errorf("error executing template %s: %w", name, err)
	}

	return buf, nil
}

func (m *TemplateManager) Exec(name string, ctx *Context) (io.WriterTo, error) {
	return m.exec(name, false, ctx)
}
func (m *TemplateManager) ExecPartial(name string, ctx *Context) (io.WriterTo, error) {
	return m.exec(name, true, ctx)
}

// RenderHTML ...
func RenderHTML(tpl string, ctx *Context) (string, error) {
	t := template.Must(template.New("tpl").Parse(tpl))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, ctx)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderPlainText ...
func RenderPlainText(tpl string, ctx interface{}) (string, error) {
	t := text_template.Must(text_template.New("tpl").Parse(tpl))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, ctx)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
