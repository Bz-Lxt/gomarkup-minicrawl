package api

import (
	"encoding/json"
	"fmt"
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

func TestCrawlContinuesAfterSeedFetchError(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = io.WriteString(w, "User-agent: *\n")
		case "/fixture/broken":
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijacking unsupported", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				return
			}
			_ = conn.Close()
		case "/fixture/healthy":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html><title>Healthy</title><body>resilient-seed-token</body></html>")
		default:
			http.NotFound(w, r)
		}
	}))
	defer fixture.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	eng := crawler.NewEngine(log, corpus, inv, "mock", fixture.URL+"/fixture")
	server := httptest.NewServer(New(log, eng, corpus, inv, "mock"))
	defer server.Close()

	body := fmt.Sprintf(`{"seeds":[%q,%q],"workers":1,"max_depth":1,"max_pages":10}`,
		fixture.URL+"/fixture/broken", fixture.URL+"/fixture/healthy")
	resp, err := http.Post(server.URL+"/api/crawl/tasks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("create task status=%d body=%s", resp.StatusCode, payload)
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
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&task)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if task.Status == "completed" {
			break
		}
		if task.Status == "failed" || task.Status == "stopped" {
			t.Fatalf("task ended as %s: %s", task.Status, task.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not finish: %s", task.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp, err = http.Get(server.URL + "/api/search?q=resilient-seed-token")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("healthy seed search total=%d, want 1", result.Total)
	}
}
