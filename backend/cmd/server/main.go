package main

import (
	"log/slog"
	"net/http"
	"os"

	"minicrawl/internal/api"
	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/logger"
	"minicrawl/internal/store"
)

func main() {
	level := getenv("LOG_LEVEL", "info")
	mode := getenv("CRAWL_MODE", "mock")
	addr := getenv("HTTP_ADDR", ":8080")
	log := logger.New(level)

	corpus := store.NewCorpus()
	inv := index.New()
	fixture := getenv("FIXTURE_BASE", "http://127.0.0.1:8080/fixture")
	engine := crawler.NewEngine(log, corpus, inv, mode, fixture)
	handler := api.New(log, engine, corpus, inv, mode)

	log.Info("minicrawl listening", slog.String("addr", addr), slog.String("mode", mode))
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("server exit", "err", err)
		os.Exit(1)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
