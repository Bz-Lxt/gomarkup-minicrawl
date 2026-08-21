package api_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"minicrawl/internal/api"
	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestTopKeywordsAvailableBeforeCrawl(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	inv.Add(index.Document{ID: 1, URL: "https://example.test/", Title: "Go crawler", Body: "crawler crawler"})
	engine := crawler.NewEngine(log, corpus, inv, "live", "")
	handler := api.New(log, engine, corpus, inv, "live")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/keywords/top?limit=5", nil)
	serveWithoutDisconnect(handler, recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/keywords/top returned %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body == "" || body == "{\"keywords\":null}\n" {
		t.Fatalf("GET /api/keywords/top returned no indexed keywords: %s", body)
	}
}

func serveWithoutDisconnect(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recover() != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()
	handler.ServeHTTP(w, r)
}
