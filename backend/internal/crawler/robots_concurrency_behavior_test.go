package crawler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestConcurrentCrawlWaitsForRobotsPolicy(t *testing.T) {
	firstRobots := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondWorker := make(chan struct{}, 1)
	var robotsRequests atomic.Int32
	var privateRequests atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		if robotsRequests.Add(1) == 1 {
			close(firstRobots)
			<-releaseFirst
		} else {
			select {
			case secondWorker <- struct{}{}:
			default:
			}
		}
		_, _ = io.WriteString(w, "User-agent: *\nDisallow: /private/\n")
	})
	mux.HandleFunc("/private/", func(w http.ResponseWriter, r *http.Request) {
		privateRequests.Add(1)
		select {
		case secondWorker <- struct{}{}:
		default:
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html><title>private</title></html>")
	})
	site := httptest.NewServer(mux)
	defer site.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(log, store.NewCorpus(), index.New(), "live", "")
	view, err := eng.Create(Strategy{
		Seeds:    []string{site.URL + "/private/a", site.URL + "/private/b"},
		Workers:  2,
		MaxDepth: 1,
		MaxPages: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-firstRobots:
	case <-time.After(2 * time.Second):
		t.Fatal("crawler did not request robots.txt")
	}
	select {
	case <-secondWorker:
	case <-time.After(2 * time.Second):
		close(releaseFirst)
		t.Fatal("second worker did not check the host policy")
	}
	close(releaseFirst)

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, ok := eng.Get(view.ID)
		if !ok {
			t.Fatal("task missing")
		}
		if got.Status == StatusCompleted {
			break
		}
		if got.Status == StatusFailed {
			t.Fatalf("task failed: %s", got.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not complete: status=%s", got.Status)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got := privateRequests.Load(); got != 0 {
		t.Fatalf("crawler requested %d URL(s) disallowed by robots.txt", got)
	}
}
