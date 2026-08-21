package hostperm

import (
	"net"
	"net/url"
	"strings"
	"sync"
)

type Rule struct {
	Host    string
	Allow   bool
	Reason  string
}

type List struct {
	mu     sync.RWMutex
	exact  map[string]Rule
	suffix []Rule
}

func New() *List {
	return &List{exact: make(map[string]Rule)}
}

func (l *List) Allow(host, reason string) {
	l.set(host, true, reason)
}

func (l *List) Deny(host, reason string) {
	l.set(host, false, reason)
}

func (l *List) set(host string, allow bool, reason string) {
	host = normHost(host)
	if host == "" {
		return
	}
	rule := Rule{Host: host, Allow: allow, Reason: reason}
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.HasPrefix(host, "*.") {
		l.suffix = append(l.suffix, rule)
		return
	}
	l.exact[host] = rule
}

func (l *List) Check(host string) (Rule, bool) {
	host = normHost(host)
	l.mu.RLock()
	defer l.mu.RUnlock()
	if rule, ok := l.exact[host]; ok {
		return rule, true
	}
	for i := len(l.suffix) - 1; i >= 0; i-- {
		suf := strings.TrimPrefix(l.suffix[i].Host, "*.")
		if host == suf || strings.HasSuffix(host, "."+suf) {
			return l.suffix[i], true
		}
	}
	return Rule{}, false
}

func (l *List) Allowed(host string) bool {
	rule, ok := l.Check(host)
	if !ok {
		return true
	}
	return rule.Allow
}

func (l *List) AllowedURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return l.Allowed(u.Hostname())
}

func (l *List) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.exact) + len(l.suffix)
}

func normHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, ".")
}

func PrivateHost(host string) bool {
	host = normHost(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}
