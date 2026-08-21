package politeness

import (
	"sync"
	"time"
)

type HostClock struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func New() *HostClock {
	return &HostClock{last: make(map[string]time.Time)}
}

func (c *HostClock) Wait(host string, minGap time.Duration) {
	if host == "" || minGap <= 0 {
		return
	}
	c.mu.Lock()
	prev, ok := c.last[host]
	now := time.Now()
	if ok {
		if wait := minGap - now.Sub(prev); wait > 0 {
			c.mu.Unlock()
			time.Sleep(wait)
			c.mu.Lock()
			now = time.Now()
		}
	}
	c.last[host] = now
	c.mu.Unlock()
}

func GapFromRPS(rps float64) time.Duration {
	if rps <= 0 {
		return 0
	}
	return time.Duration(float64(time.Second) / rps)
}
