package crawler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type robotGroup struct {
	disallow []string
}

type Robots struct {
	client *http.Client
	ua     string
	mu     sync.Mutex
	cache  map[string]robotGroup
}

func NewRobots(client *http.Client, ua string) *Robots {
	return &Robots{client: client, ua: ua, cache: make(map[string]robotGroup)}
}

func (r *Robots) Allowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	group := r.group(u.Scheme, u.Host, host)
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	for _, d := range group.disallow {
		if d == "" {
			continue
		}
		if strings.HasPrefix(path, d) {
			return false
		}
	}
	return true
}

func (r *Robots) group(scheme, hostPort, host string) robotGroup {
	r.mu.Lock()
	if g, ok := r.cache[host]; ok {
		r.mu.Unlock()
		return g
	}
	r.mu.Unlock()

	g := r.fetch(scheme, hostPort)
	r.mu.Lock()
	r.cache[host] = g
	r.mu.Unlock()
	return g
}

func (r *Robots) fetch(scheme, hostPort string) robotGroup {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", scheme, hostPort)
	req, err := http.NewRequest(http.MethodGet, robotsURL, nil)
	if err != nil {
		return robotGroup{}
	}
	req.Header.Set("User-Agent", r.ua)
	client := r.client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return robotGroup{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return robotGroup{}
	}
	return parseRobots(io.LimitReader(resp.Body, 64*1024), r.ua)
}

func parseRobots(rd io.Reader, ua string) robotGroup {
	want := strings.ToLower(strings.TrimSpace(ua))
	sc := bufio.NewScanner(rd)
	var currentUA []string
	applies := false
	var dis []string
	starDis := []string{}
	inStar := false

	flush := func() {}
	_ = flush

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "user-agent":
			currentUA = []string{strings.ToLower(val)}
			applies = currentUA[0] == "*" || (want != "" && strings.HasPrefix(want, currentUA[0]))
			inStar = currentUA[0] == "*"
		case "disallow":
			if applies && val != "" {
				dis = append(dis, val)
			}
			if inStar && val != "" {
				starDis = append(starDis, val)
			}
		}
	}
	if len(dis) == 0 {
		dis = starDis
	}
	return robotGroup{disallow: dis}
}
