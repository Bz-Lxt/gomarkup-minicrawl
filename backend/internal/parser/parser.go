package parser

import (
	"bytes"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Page struct {
	Title string
	Text  string
	Links []string
}

var skipText = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true,
	"footer": true, "svg": true,
}

func Parse(body []byte, base *url.URL) (*Page, error) {
	doc, err := html.Parse(io.LimitReader(bytes.NewReader(body), 2<<20))
	if err != nil {
		return nil, err
	}
	p := &Page{}
	var text strings.Builder
	var skip int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			name := strings.ToLower(n.Data)
			if name == "title" && p.Title == "" {
				p.Title = strings.TrimSpace(collectText(n))
			}
			if name == "a" {
				if href := attr(n, "href"); href != "" {
					if abs := resolve(base, href); abs != "" {
						p.Links = append(p.Links, abs)
					}
				}
			}
			if skipText[name] {
				skip++
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				skip--
				return
			}
		}
		if n.Type == html.TextNode && skip == 0 {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if text.Len() > 0 {
					text.WriteByte(' ')
				}
				text.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	p.Text = strings.Join(strings.Fields(text.String()), " ")
	return p, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func collectText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func resolve(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if base == nil {
		return u.String()
	}
	return base.ResolveReference(u).String()
}
