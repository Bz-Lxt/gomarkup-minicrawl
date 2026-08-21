package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestConcurrentSeedsDoNotCloseDiscoveryQueue(t *testing.T) {
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = io.WriteString(w, "User-agent: *\n")
		case "/slow":
			close(slowStarted)
			<-releaseSlow
			time.Sleep(100 * time.Millisecond)
			_, _ = io.WriteString(w, `<html><body>slow <a href="/child">child</a></body></html>`)
		case "/empty":
			<-slowStarted
			close(releaseSlow)
			_, _ = io.WriteString(w, `<html><body>empty</body></html>`)
		case "/child":
			_, _ = io.WriteString(w, `<html><body>unique-child-token</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	engine := crawler.NewEngine(log, corpus, inv, "live", "")
	server := httptest.NewServer(New(log, engine, corpus, inv, "live"))
	defer server.Close()

	payload, err := json.Marshal(map[string]any{
		"seeds":     []string{upstream.URL + "/slow", upstream.URL + "/empty"},
		"workers":   2,
		"max_depth": 2,
		"max_pages": 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(server.URL+"/api/crawl/tasks", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create status=%d body=%s", resp.StatusCode, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err = http.Get(server.URL + "/api/crawl/tasks/" + created.ID)
		if err != nil {
			t.Fatal(err)
		}
		var task struct {
			Status  string `json:"status"`
			Crawled int    `json:"crawled"`
		}
		err = json.NewDecoder(resp.Body).Decode(&task)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if task.Status == "completed" {
			if task.Crawled != 3 {
				t.Fatalf("completed after crawling %d pages, want 3", task.Crawled)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not complete: status=%s crawled=%d", task.Status, task.Crawled)
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp, err = http.Get(server.URL + "/api/search?q=unique-child-token")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var results struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if results.Total != 1 {
		t.Fatalf("child page search returned %d hits, want 1", results.Total)
	}
}
