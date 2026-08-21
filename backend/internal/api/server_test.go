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

func TestHealthAndSearchValidation(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	inv := index.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()
	eng := crawler.NewEngine(log, corpus, inv, "mock", ts.URL+"/fixture")
	h := New(log, eng, corpus, inv, "mock")
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/api/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("search want 400 got %d", resp.StatusCode)
	}
	var body map[string]map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"]["code"] != "VALIDATION_ERROR" {
		t.Fatalf("err body %+v", body)
	}
	_ = time.Second
	_ = strings.TrimSpace
}
