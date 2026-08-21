package crawler

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

type singlePageTransport func(*http.Request) (*http.Response, error)

func (f singlePageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestEngineCompletesAfterQueueDrains(t *testing.T) {
	corpus := store.NewCorpus()
	engine := NewEngine(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		corpus,
		index.New(),
		"live",
		"",
	)
	engine.client.Transport = singlePageTransport(func(req *http.Request) (*http.Response, error) {
		body := "<html><title>Only page</title><body>done</body></html>"
		contentType := "text/html"
		if req.URL.Path == "/robots.txt" {
			body = "User-agent: *\n"
			contentType = "text/plain"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{contentType}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	created, err := engine.Create(Strategy{
		Seeds:    []string{"http://single-page.test/only"},
		Workers:  2,
		MaxDepth: 1,
		MaxPages: 10,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		view, ok := engine.Get(created.ID)
		if !ok {
			t.Fatal("created task disappeared")
		}
		if view.Status == StatusCompleted {
			if view.Crawled != 1 {
				t.Fatalf("completed after crawling %d pages, want 1", view.Crawled)
			}
			return
		}
		if view.Status == StatusFailed {
			t.Fatalf("task failed: %s", view.Error)
		}
		time.Sleep(5 * time.Millisecond)
	}

	view, _ := engine.Get(created.ID)
	t.Fatalf("task did not complete after its queue drained: status=%s crawled=%d queue_length=%d", view.Status, view.Crawled, view.QueueLength)
}
