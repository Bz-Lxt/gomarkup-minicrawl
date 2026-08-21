package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

type contextRoundTripFunc func(*http.Request) (*http.Response, error)

func (f contextRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestStopTaskCancelsHostRateLimitWait(t *testing.T) {
	previousTransport := http.DefaultTransport
	http.DefaultTransport = contextRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			return nil, err
		}

		body := ""
		switch r.URL.Path {
		case "/robots.txt":
			body = "User-agent: *\n"
		case "/fixture/index.html":
			body = `<html><body><a href="/fixture/next.html">next</a></body></html>`
		case "/fixture/next.html":
			body = `<html><body>done</body></html>`
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
				Request:    r,
			}, nil
		}

		header := make(http.Header)
		header.Set("Content-Type", "text/html; charset=utf-8")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})
	defer func() { http.DefaultTransport = previousTransport }()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	engine := crawler.NewEngine(log, corpus, inv, "mock", "http://fixture.test/fixture")
	handler := New(log, engine, corpus, inv, "mock")

	createRequest := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks", strings.NewReader(`{
		"seeds":["http://fixture.test/fixture/index.html"],
		"workers":1,
		"global_rps":100,
		"host_rps":0.5,
		"max_depth":1,
		"max_pages":2
	}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, want %d; body=%s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}
	var created crawler.TaskView
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	getTask := func() crawler.TaskView {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/crawl/tasks/"+created.ID, nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("get task status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
		var view crawler.TaskView
		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			t.Fatal(err)
		}
		return view
	}

	readyDeadline := time.Now().Add(time.Second)
	for {
		view := getTask()
		if view.Status == crawler.StatusRunning && view.Crawled == 1 && view.QueueLength == 0 {
			break
		}
		if time.Now().After(readyDeadline) {
			t.Fatalf("task did not reach host-rate wait: status=%s crawled=%d queue=%d", view.Status, view.Crawled, view.QueueLength)
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if view := getTask(); view.Status != crawler.StatusRunning || view.Crawled != 1 {
		t.Fatalf("task left rate-limit wait before stop: status=%s crawled=%d", view.Status, view.Crawled)
	}

	stopRequest := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks/"+created.ID+"/stop", nil)
	stopResponse := httptest.NewRecorder()
	handler.ServeHTTP(stopResponse, stopRequest)
	if stopResponse.Code != http.StatusOK {
		t.Fatalf("stop task status = %d, want %d; body=%s", stopResponse.Code, http.StatusOK, stopResponse.Body.String())
	}

	finishDeadline := time.Now().Add(500 * time.Millisecond)
	for {
		view := getTask()
		if view.Status != crawler.StatusStopped {
			t.Fatalf("task status after stop = %s, want %s", view.Status, crawler.StatusStopped)
		}
		if view.FinishedAt != "" {
			break
		}
		if time.Now().After(finishDeadline) {
			t.Fatalf("stopped task did not finish while waiting for a host rate-limit token: %+v", view)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
