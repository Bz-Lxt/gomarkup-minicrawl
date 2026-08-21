package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestCreateTaskPreservesRepeatedQueryParameters(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	eng := crawler.NewEngine(log, corpus, inv, "live", "")
	h := New(log, eng, corpus, inv, "live")
	body := []byte(`{"seeds":["http://example.test/items?tag=web&tag=go&sort=new"],"workers":1,"max_pages":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var task struct {
		Strategy struct {
			Seeds []string `json:"seeds"`
		} `json:"strategy"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	want := "http://example.test/items?sort=new&tag=go&tag=web"
	if len(task.Strategy.Seeds) != 1 || task.Strategy.Seeds[0] != want {
		t.Fatalf("returned seeds = %q, want [%q]", task.Strategy.Seeds, want)
	}
}
