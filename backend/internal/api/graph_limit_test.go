package api

import (
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

func TestGraphHonorsRequestedLimit(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	corpus := store.NewCorpus()
	corpus.AddEdge("https://example.test/a", "https://example.test/b")
	corpus.AddEdge("https://example.test/b", "https://example.test/c")
	inv := index.New()
	engine := crawler.NewEngine(log, corpus, inv, "mock", "http://example.test/fixture")
	handler := New(log, engine, corpus, inv, "mock")

	req := httptest.NewRequest(http.MethodGet, "/api/graph?limit=2", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	resp := recorder.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("graph status: want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var graph store.Graph
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("graph nodes: want 2, got %d", len(graph.Nodes))
	}
	nodes := make(map[string]bool, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = true
	}
	for _, edge := range graph.Edges {
		if !nodes[edge.From] || !nodes[edge.To] {
			t.Fatalf("edge %q -> %q references a node outside the response", edge.From, edge.To)
		}
	}
}
