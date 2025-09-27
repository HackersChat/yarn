// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/james4k/fmatter"
	"github.com/julienschmidt/httprouter"
	sync "github.com/sasha-s/go-deadlock"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const pagesDir = "pages"

//go:embed pages/*.md
var builtinPages embed.FS

type FrontMatter struct {
	Title       string
	Description string
}

type Page struct {
	Content      string
	LastModified time.Time
}

// PageHandler ...
func (s *Server) PageHandler(name string) httprouter.Handle {
	pagesBaseDir := filepath.Join(s.config.Data, pagesDir)

	pageMutex := &sync.RWMutex{}
	pageCache := make(map[string]*Page)

	getPage := func(name string) (*Page, error) {
		fn := filepath.Join(pagesBaseDir, fmt.Sprintf("%s.md", name))

		pageMutex.RLock()
		page, isCached := pageCache[name]
		pageMutex.RUnlock()

		if isCached && FileExists(fn) {
			if fileInfo, err := os.Stat(fn); err == nil {
				if fileInfo.ModTime().After(page.LastModified) {
					data, err := os.ReadFile(fn)
					if err != nil {
						log.WithError(err).Warnf("error reading page %s", name)
						return page, nil
					}
					page.Content = string(data)
					page.LastModified = fileInfo.ModTime()

					pageMutex.Lock()
					pageCache[name] = page
					pageMutex.Unlock()

					return page, nil
				}
			}
		}

		page = &Page{}

		if FileExists(fn) {
			fileInfo, err := os.Stat(fn)
			if err != nil {
				log.WithError(err).Errorf("error getting page stats")
				return nil, err
			}
			page.LastModified = fileInfo.ModTime()

			data, err := os.ReadFile(fn)
			if err != nil {
				log.WithError(err).Errorf("error reading page %s", name)
				return nil, err
			}
			page.Content = string(data)
		} else {
			fn := filepath.Join(pagesDir, fmt.Sprintf("%s.md", name))
			data, err := builtinPages.ReadFile(fn)
			if err != nil {
				log.WithError(err).Errorf("error reading custom page %s", name)
				return nil, err
			}
			page.Content = string(data)
		}

		pageMutex.Lock()
		pageCache[name] = page
		pageMutex.Unlock()

		return page, nil
	}

	caser := cases.Title(language.English)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := NewContext(s, r)

		page, err := getPage(name)
		if err != nil {
			if os.IsNotExist(err) {
				ctx.Error = true
				ctx.Message = s.tr(ctx, "PageNotFoundTitle")
				w.WriteHeader(http.StatusNotFound)
				s.render("404", r, w, ctx)
				return
			}
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorPageError"))
			return
		}

		markdownContent, err := RenderHTML(page.Content, ctx)
		if err != nil {
			log.WithError(err).Errorf("error rendering page %s", name)
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorRenderingPage"))
			return
		}

		var frontmatter FrontMatter
		content, err := fmatter.Read([]byte(markdownContent), &frontmatter)
		if err != nil {
			log.WithError(err).Error("error parsing front matter")
			s.renderError(r, w, http.StatusInternalServerError, s.tr(ctx, "ErrorLoadingPage"))
			return
		}

		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		p := parser.NewWithExtensions(extensions)

		htmlFlags := html.CommonFlags
		opts := html.RendererOptions{
			Flags:     htmlFlags,
			Generator: "",
		}
		renderer := html.NewRenderer(opts)

		html := markdown.ToHTML(content, p, renderer)

		var title string

		if frontmatter.Title != "" {
			title = frontmatter.Title
		} else {
			title = caser.String(name)
		}
		ctx.Title = title
		ctx.Meta.Description = frontmatter.Description

		ctx.Page = name
		ctx.Content = template.HTML(html)

		s.render("page", r, w, ctx)
	}
}
