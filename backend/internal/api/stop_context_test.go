package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestStopTaskCancelsInFlightFetch(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCanceled := make(chan struct{})
	releaseRequest := make(chan struct{})
	var startedOnce sync.Once
	var canceledOnce sync.Once

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/robots.txt" {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    r,
			}, nil
		}
		startedOnce.Do(func() { close(requestStarted) })
		select {
		case <-r.Context().Done():
			canceledOnce.Do(func() { close(requestCanceled) })
		case <-releaseRequest:
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(strings.NewReader("<html><title>released</title></html>")),
			Request:    r,
		}, nil
	})
	defer func() { http.DefaultTransport = originalTransport }()
	defer close(releaseRequest)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	eng := crawler.NewEngine(log, corpus, inv, "live", "")
	handler := New(log, eng, corpus, inv, "live")

	body, err := json.Marshal(map[string]any{
		"seeds":     []string{"http://target.example/slow"},
		"workers":   1,
		"max_depth": 1,
		"max_pages": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	createReq := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create task status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	var created crawler.TaskView
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("crawl request did not reach target")
	}

	stopReq := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks/"+created.ID+"/stop", nil)
	stopReq.SetPathValue("id", created.ID)
	stopRec := httptest.NewRecorder()
	handler.ServeHTTP(stopRec, stopReq)
	if stopRec.Code != http.StatusOK {
		t.Fatalf("stop task status=%d body=%s", stopRec.Code, stopRec.Body.String())
	}
	var stopped crawler.TaskView
	if err := json.NewDecoder(stopRec.Body).Decode(&stopped); err != nil {
		t.Fatal(err)
	}
	if stopped.Status != crawler.StatusStopped {
		t.Fatalf("stop task status=%q, want %q", stopped.Status, crawler.StatusStopped)
	}

	select {
	case <-requestCanceled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("stop returned but the target still has the crawl request open")
	}
}
