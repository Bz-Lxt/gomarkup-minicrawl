package sitemap

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

type URL struct {
	Loc        string
	LastMod    time.Time
	ChangeFreq string
	Priority   float64
}

type Index struct {
	Maps []string
}

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float64 `xml:"priority"`
}

type sitemapIndex struct {
	XMLName xml.Name      `xml:"sitemapindex"`
	Maps    []sitemapRef  `xml:"sitemap"`
}

type sitemapRef struct {
	Loc string `xml:"loc"`
}

func Parse(r io.Reader) ([]URL, *Index, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return nil, nil, err
	}
	text := strings.TrimSpace(string(raw))
	if strings.Contains(text, "<sitemapindex") {
		var idx sitemapIndex
		if err := xml.Unmarshal(raw, &idx); err != nil {
			return nil, nil, err
		}
		out := &Index{}
		for _, m := range idx.Maps {
			if loc := cleanLoc(m.Loc); loc != "" {
				out.Maps = append(out.Maps, loc)
			}
		}
		return nil, out, nil
	}
	var set urlset
	if err := xml.Unmarshal(raw, &set); err != nil {
		return nil, nil, err
	}
	out := make([]URL, 0, len(set.URLs))
	for _, e := range set.URLs {
		item, ok := toURL(e)
		if ok {
			out = append(out, item)
		}
	}
	return out, nil, nil
}

func ParseBytes(b []byte) ([]URL, *Index, error) {
	return Parse(strings.NewReader(string(b)))
}

func toURL(e urlEntry) (URL, bool) {
	loc := cleanLoc(e.Loc)
	if loc == "" {
		return URL{}, false
	}
	item := URL{Loc: loc, ChangeFreq: strings.ToLower(strings.TrimSpace(e.ChangeFreq)), Priority: e.Priority}
	if e.LastMod != "" {
		if t, err := time.Parse(time.RFC3339, e.LastMod); err == nil {
			item.LastMod = t
		} else if t, err := time.Parse("2006-01-02", e.LastMod); err == nil {
			item.LastMod = t
		}
	}
	if item.Priority < 0 {
		item.Priority = 0
	}
	if item.Priority > 1 {
		item.Priority = 1
	}
	return item, true
}

func cleanLoc(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func FilterPrefix(items []URL, prefix string) []URL {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return items
	}
	out := items[:0]
	for _, item := range items {
		if strings.HasPrefix(item.Loc, prefix) {
			out = append(out, item)
		}
	}
	return append([]URL(nil), out...)
}

func Locs(items []URL) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Loc)
	}
	return out
}

func ValidateFreq(freq string) error {
	switch strings.ToLower(strings.TrimSpace(freq)) {
	case "", "always", "hourly", "daily", "weekly", "monthly", "yearly", "never":
		return nil
	default:
		return fmt.Errorf("unknown changefreq %q", freq)
	}
}
