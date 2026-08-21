package retrywait

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Policy struct {
	Base    time.Duration
	Max     time.Duration
	Factor  float64
	Jitter  float64
	mu      sync.Mutex
	rnd     *rand.Rand
}

func New(base, max time.Duration) *Policy {
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	if max < base {
		max = 8 * base
	}
	return &Policy{
		Base:   base,
		Max:    max,
		Factor: 2,
		Jitter: 0.2,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *Policy) ForAttempt(n int) time.Duration {
	if n < 0 {
		n = 0
	}
	mult := math.Pow(p.factor(), float64(n))
	d := time.Duration(float64(p.Base) * mult)
	if d > p.Max {
		d = p.Max
	}
	return p.withJitter(d)
}

func (p *Policy) Sequence(n int) []time.Duration {
	if n <= 0 {
		return nil
	}
	out := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		out[i] = p.ForAttempt(i)
	}
	return out
}

func (p *Policy) ShouldRetry(status, attempt, maxAttempts int) bool {
	if maxAttempts > 0 && attempt >= maxAttempts {
		return false
	}
	return status == 408 || status == 429 || status >= 500 || status == 0
}

func (p *Policy) factor() float64 {
	if p.Factor <= 1 {
		return 2
	}
	return p.Factor
}

func (p *Policy) withJitter(d time.Duration) time.Duration {
	j := p.Jitter
	if j <= 0 {
		return d
	}
	if j > 1 {
		j = 1
	}
	p.mu.Lock()
	f := 1 + (p.rnd.Float64()*2-1)*j
	p.mu.Unlock()
	if f < 0.1 {
		f = 0.1
	}
	return time.Duration(float64(d) * f)
}

func Sleep(d time.Duration, now time.Time, until time.Time) time.Duration {
	if until.IsZero() {
		return d
	}
	left := until.Sub(now)
	if left < 0 {
		return 0
	}
	if d > left {
		return left
	}
	return d
}
