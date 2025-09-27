package indieweb

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func searchLinks(node *html.Node, link *url.URL) bool {
	if node.Type == html.ElementNode {
		if node.DataAtom == atom.A {
			if href := getAttr(node, "href"); href != "" {
				target, err := url.Parse(href)
				if err == nil {
					// pods have the form
					// http://pod.domain.tld/external?uri=uri
					if strings.HasPrefix(target.Path, "/external") && target.Query().Get("uri") == link.String() {
						return true
					}
					if target.String() == link.String() {
						return true
					}
				}
			}
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		found := searchLinks(c, link)
		if found {
			return found
		}
	}
	return false
}

func getAttr(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}
