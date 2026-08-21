package crawler

import (
	"context"
	"testing"
	"time"
)

func TestLimiterPace(t *testing.T) {
	l := NewLimiter(20, 0)
	start := time.Now()
	for i := 0; i < 40; i++ {
		if err := l.Wait(context.Background(), "h"); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 800*time.Millisecond {
		t.Fatalf("limiter too fast: %s", elapsed)
	}
	if elapsed > 2500*time.Millisecond {
		t.Fatalf("limiter too slow: %s", elapsed)
	}
}
