package crawler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestEngineDoesNotIndexTruncatedResponse(t *testing.T) {
	const partial = `<html><head><title>partial</title></head><body>truncated-marker`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = io.WriteString(w, "User-agent: *\n")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "4096")
		_, _ = io.WriteString(w, partial)
	}))
	defer ts.Close()

	corpus := store.NewCorpus()
	inv := index.New()
	eng := NewEngine(slog.New(slog.NewTextHandler(io.Discard, nil)), corpus, inv, "live", "")
	view, err := eng.Create(Strategy{Seeds: []string{ts.URL}, Workers: 1, MaxDepth: 1, MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		view, _ = eng.Get(view.ID)
		if view.Status == StatusCompleted || view.Status == StatusFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not finish: status=%s", view.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if view.Crawled != 0 {
		t.Fatalf("truncated response counted as crawled: got %d", view.Crawled)
	}
	if hits := inv.Search("truncated-marker", false, 10); len(hits) != 0 {
		t.Fatalf("truncated response was searchable: got %d hits", len(hits))
	}
}
