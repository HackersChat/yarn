// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package renderer

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/mgutz/ansi"
	"github.com/russross/blackfriday"
)

type ConsoleGit struct {
	lists []*list
}

func (options *ConsoleGit) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	s := string(text)
	reg, _ := regexp.Compile(`\n`)

	out.WriteString("\n    ")
	out.WriteString(ansi.ColorCode("015"))
	out.WriteString(reg.ReplaceAllString(s, "\n    "))
	out.WriteString(ansi.ColorCode("reset"))
	out.WriteString("\n")
}

func (options *ConsoleGit) BlockQuote(out *bytes.Buffer, text []byte) {
	s := strings.TrimSpace(string(text))
	reg, _ := regexp.Compile(`\n`)

	out.WriteString("\n  | ")
	out.WriteString(reg.ReplaceAllString(s, "\n  | "))
	out.WriteString("\n\n")
}

func (options *ConsoleGit) BlockHtml(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *ConsoleGit) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	out.WriteString("\n")
	out.WriteString(headerStyles[level-1])

	marker := out.Len()
	if !text() {
		out.Truncate(marker)
		return
	}

	out.WriteString(ansi.ColorCode("reset"))
	out.WriteString("\n\n")
}

func (options *ConsoleGit) HRule(out *bytes.Buffer) {
	out.WriteString("\n\u2015\u2015\u2015\u2015\u2015\n\n")
}

func (options *ConsoleGit) List(out *bytes.Buffer, text func() bool, flags int) {
	out.WriteString("\n")

	kind := UNORDERED
	if flags&blackfriday.LIST_TYPE_ORDERED != 0 {
		kind = ORDERED
	}

	options.lists = append(options.lists, &list{kind, 1})
	text()
	options.lists = options.lists[:len(options.lists)-1]
	out.WriteString("\n")
}

func (options *ConsoleGit) ListItem(out *bytes.Buffer, text []byte, flags int) {
	current := options.lists[len(options.lists)-1]

	for i := 0; i < len(options.lists); i++ {
		out.WriteString("  ")
	}

	if current.kind == ORDERED {
		out.WriteString(fmt.Sprintf("%d. ", current.index))
		current.index += 1
	} else {
		out.WriteString(ansi.ColorCode("red+bh"))
		out.WriteString("* ")
		out.WriteString(ansi.ColorCode("reset"))
	}

	out.Write(text)
	out.WriteString("\n\n")
}

func (options *ConsoleGit) Paragraph(out *bytes.Buffer, text func() bool) {
	marker := out.Len()

	if !text() {
		out.Truncate(marker)
		return
	}

}

func (options *ConsoleGit) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {}
func (options *ConsoleGit) TableRow(out *bytes.Buffer, text []byte)                               {}
func (options *ConsoleGit) TableHeaderCell(out *bytes.Buffer, text []byte, flags int)             {}
func (options *ConsoleGit) TableCell(out *bytes.Buffer, text []byte, flags int)                   {}
func (options *ConsoleGit) Footnotes(out *bytes.Buffer, text func() bool)                         {}
func (options *ConsoleGit) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int)          {}

func (options *ConsoleGit) TitleBlock(out *bytes.Buffer, text []byte) {
	out.WriteString("\n")
	out.WriteString(headerStyles[0])
	out.Write(text)
	out.WriteString(ansi.ColorCode("reset"))
	out.WriteString("\n\n")
}

func (options *ConsoleGit) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	out.WriteString(linkStyle)
	out.Write(link)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) CodeSpan(out *bytes.Buffer, text []byte) {
	out.WriteString(ansi.ColorCode("015+b"))
	out.Write(text)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString(emphasisStyles[1])
	out.Write(text)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) Emphasis(out *bytes.Buffer, text []byte) {
	out.WriteString(emphasisStyles[0])
	out.Write(text)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	out.WriteString(" [ image ] ")
}

func (options *ConsoleGit) LineBreak(out *bytes.Buffer) {
	out.WriteString("\n")
}

func (options *ConsoleGit) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	out.Write(content)
	out.WriteString(" (")
	out.WriteString(linkStyle)
	out.Write(link)
	out.WriteString(ansi.ColorCode("reset"))
	out.WriteString(")")
}

func (options *ConsoleGit) RawHtmlTag(out *bytes.Buffer, tag []byte) {
	out.WriteString(ansi.ColorCode("magenta"))
	out.Write(tag)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) TripleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString(emphasisStyles[2])
	out.Write(text)
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) StrikeThrough(out *bytes.Buffer, text []byte) {
	out.WriteString(ansi.ColorCode("008+s"))
	out.WriteString("\u2015")
	out.Write(text)
	out.WriteString("\u2015")
	out.WriteString(ansi.ColorCode("reset"))
}

func (options *ConsoleGit) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
}

func (options *ConsoleGit) Entity(out *bytes.Buffer, entity []byte) {
	out.WriteString(html.UnescapeString(string(entity)))
}

func (options *ConsoleGit) NormalText(out *bytes.Buffer, text []byte) {
	s := string(text)
	reg, _ := regexp.Compile(`\s+`)

	out.WriteString(reg.ReplaceAllString(s, " "))
}

func (options *ConsoleGit) DocumentHeader(out *bytes.Buffer) {
}

func (options *ConsoleGit) DocumentFooter(out *bytes.Buffer) {
}

func (options *ConsoleGit) GetFlags() int {
	return 0
}
