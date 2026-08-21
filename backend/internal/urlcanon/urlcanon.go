package urlcanon

import (
	"net"
	"net/url"
	"strings"
)

func Canonical(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errEmpty(raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = canonicalHost(u.Host)
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	u.Path = collapsePath(u.Path)
	u.RawQuery = sortQuery(u.RawQuery)
	return u.String(), nil
}

type emptyError string

func (e emptyError) Error() string { return "empty url: " + string(e) }

func errEmpty(raw string) error { return emptyError(raw) }

func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, p, err := net.SplitHostPort(host); err == nil {
		if p == "80" || p == "443" {
			return h
		}
		return net.JoinHostPort(h, p)
	}
	return host
}

func DefaultPort(scheme string) string {
	switch strings.ToLower(scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func StripDefaultPort(host, scheme string) string {
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return host
	}
	if p == DefaultPort(scheme) {
		return h
	}
	return host
}

func collapsePath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	var out []string
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, part)
		}
	}
	joined := "/" + strings.Join(out, "/")
	if strings.HasSuffix(p, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	return joined
}

func sortQuery(raw string) string {
	if raw == "" {
		return ""
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}
	return q.Encode()
}

func Origin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme) + "://" + canonicalHost(u.Host)
}

func SameOrigin(a, b string) bool {
	return Origin(a) != "" && Origin(a) == Origin(b)
}
