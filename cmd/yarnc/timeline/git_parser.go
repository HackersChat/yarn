package timeline

import (
	"context"
	"strings"

	"git.mills.io/yarnsocial/yarn/cmd/yarnc/timeline/renderer"
	"github.com/dustin/go-humanize"
	"github.com/russross/blackfriday"
	"go.yarn.social/types"
)

type twtWithChild struct {
	types.Twt
	child    []*twtWithChild
	ParentId string
}

func (c *twtWithChild) HasParent() bool {
	return c.ParentId != ""
}

func (c *twtWithChild) GetChildren() []*twtWithChild {
	return c.child
}

func (c *twtWithChild) AddChilddren(twt *twtWithChild) {
	c.child = append(c.child, twt)
}

func (c *twtWithChild) HasChildren() bool {
	return len(c.child) != 0
}

type twtWithChilds []*twtWithChild

func newTwtWithChilds(twts types.Twts) twtWithChilds {
	m := make(map[string]*twtWithChild)
	twtc := []*twtWithChild{}
	for _, v := range twts {
		t := twtWithChild{v, []*twtWithChild{}, ""}
		twtc = append(twtc, &t)
		m[v.Hash()] = &t
	}

	for i, v := range twtc {
		for _, tag := range v.Tags() {
			if m1, ok := m[tag.Text()]; ok {
				twtc[i].ParentId = tag.Text()
				m1.AddChilddren(twtc[i])
			}
		}
	}
	return twtc
}

type gitParser struct {
	printer printer
}

func (d gitParser) Parse(ctx context.Context, twts types.Twts, me types.Twter) error {
	twtc := newTwtWithChilds(twts)

	for _, v := range twtc {
		if !v.HasParent() {
			d.recursivePrint([]string{}, v)
		}
	}
	return nil
}

func (d gitParser) recursivePrint(prefix []string, twt *twtWithChild) {
	d.print(strings.Join(prefix, ""), twt)

	if twt.HasChildren() {
		d.printer.Print(strings.Join(append(prefix, HashColor(twt.Hash())(" | \\ \n")), ""))
	}

	for _, c := range twt.GetChildren() {
		d.recursivePrint(append(prefix, HashColor(twt.Hash())(" |")), c)
	}
}

func (d gitParser) print(prefix string, twt *twtWithChild) {
	r := &renderer.ConsoleGit{}
	extensions := 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_DEFINITION_LISTS |
		blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK |
		blackfriday.EXTENSION_JOIN_LINES

	input := []byte(twt.FormatText(types.MarkdownFmt, nil))
	output := blackfriday.Markdown(input, r, extensions)
	d.printer.Print("%s %s - %s (%s) <%s>\n",
		prefix+white(" * "),
		redBold(twt.Hash()),
		string(output),
		green(humanize.Time(twt.Created())),
		blue(twt.Twter().DomainNick()))
}
