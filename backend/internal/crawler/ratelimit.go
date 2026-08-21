package crawler

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type Limiter struct {
	global *rate.Limiter
	hostRPS float64
	mu      sync.Mutex
	hosts   map[string]*rate.Limiter
}

func NewLimiter(globalRPS, hostRPS float64) *Limiter {
	l := &Limiter{hostRPS: hostRPS, hosts: make(map[string]*rate.Limiter)}
	if globalRPS > 0 {
		burst := int(globalRPS)
		if burst < 1 {
			burst = 1
		}
		l.global = rate.NewLimiter(rate.Limit(globalRPS), burst)
	}
	return l
}

func (l *Limiter) Wait(ctx context.Context, host string) error {
	if l.global != nil {
		if err := l.global.Wait(ctx); err != nil {
			return err
		}
	}
	if l.hostRPS <= 0 || host == "" {
		return nil
	}
	l.mu.Lock()
	lim, ok := l.hosts[host]
	if !ok {
		burst := int(l.hostRPS)
		if burst < 1 {
			burst = 1
		}
		lim = rate.NewLimiter(rate.Limit(l.hostRPS), burst)
		l.hosts[host] = lim
	}
	l.mu.Unlock()
	return lim.Wait(ctx)
}
