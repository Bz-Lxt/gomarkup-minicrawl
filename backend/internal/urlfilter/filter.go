package urlfilter

import (
	"net/url"
	"strings"
)

var skipExt = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".svg": {},
	".css": {}, ".js": {}, ".ico": {}, ".woff": {}, ".woff2": {}, ".ttf": {},
	".mp4": {}, ".mp3": {}, ".zip": {}, ".pdf": {},
}

func ShouldSkip(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return true
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return true
	}
	path := strings.ToLower(u.Path)
	for ext := range skipExt {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func SameHost(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname())
}

func StripFragment(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	return u.String()
}
