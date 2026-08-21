package crawler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/mocksite"
	"minicrawl/internal/store"
)

func fixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/fixture/", http.StripPrefix("/fixture", mocksite.Handler()))
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "User-agent: *\nDisallow: /fixture/secret\n")
	})
	return httptest.NewServer(mux)
}

func TestEngineCrawlSearchGraph(t *testing.T) {
	ts := fixtureServer()
	defer ts.Close()

	corpus := store.NewCorpus()
	inv := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(log, corpus, inv, "mock", ts.URL+"/fixture")

	view, err := eng.Create(Strategy{
		Workers:  8,
		MaxDepth: 6,
		MaxPages: 80,
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		got, ok := eng.Get(view.ID)
		if !ok {
			t.Fatal("task missing")
		}
		if got.Status == StatusCompleted {
			break
		}
		if got.Status == StatusFailed {
			t.Fatalf("failed: %s", got.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout status=%s crawled=%d", got.Status, got.Crawled)
		}
		time.Sleep(50 * time.Millisecond)
	}
	got, _ := eng.Get(view.ID)
	if got.Crawled < 20 {
		t.Fatalf("crawled %d, want >= 20", got.Crawled)
	}

	hits := inv.Search("minicrawl", true, 10)
	if len(hits) == 0 {
		t.Fatal("expected search hits")
	}
	if !strings.Contains(hits[0].Snippet, "<mark>") {
		t.Fatalf("missing highlight: %s", hits[0].Snippet)
	}

	secret := inv.Search("should-not-index-secret-token", false, 10)
	if len(secret) != 0 {
		t.Fatal("robots.txt violated")
	}

	g := corpus.Graph(400)
	if len(g.Nodes) == 0 || len(g.Edges) == 0 {
		t.Fatalf("graph empty nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
	}
}

func TestEngineThroughput(t *testing.T) {
	ts := fixtureServer()
	defer ts.Close()
	corpus := store.NewCorpus()
	inv := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(log, corpus, inv, "mock", ts.URL+"/fixture")
	start := time.Now()
	view, err := eng.Create(Strategy{Workers: 8, MaxDepth: 8, MaxPages: 100})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		got, _ := eng.Get(view.ID)
		if got.Status == StatusCompleted || got.Status == StatusFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}
	elapsed := time.Since(start).Seconds()
	got, _ := eng.Get(view.ID)
	if got.Status != StatusCompleted {
		t.Fatalf("status %s err=%s", got.Status, got.Error)
	}
	rps := float64(got.Crawled) / elapsed
	if rps < 30 {
		t.Fatalf("throughput %.1f pages/s < 30 (crawled=%d elapsed=%.2fs)", rps, got.Crawled, elapsed)
	}
}
