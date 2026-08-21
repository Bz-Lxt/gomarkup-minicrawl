package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestRejectedCrawlTaskDoesNotRemainListed(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	engine := crawler.NewEngine(log, corpus, inv, "mock", "http://fixture.invalid/fixture")
	handler := New(log, engine, corpus, inv, "mock")

	create := httptest.NewRequest(http.MethodPost, "/api/crawl/tasks", strings.NewReader(`{"seeds":["http://outside.invalid/"]}`))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", created.Code, http.StatusBadRequest)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/crawl/tasks", nil)
	listedResponse := httptest.NewRecorder()
	handler.ServeHTTP(listedResponse, list)
	if listedResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listedResponse.Code, http.StatusOK)
	}
	var listed struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	if err := json.NewDecoder(listedResponse.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Tasks) != 0 {
		t.Fatalf("rejected create left %d task(s) in the public task list, want 0", len(listed.Tasks))
	}
}
