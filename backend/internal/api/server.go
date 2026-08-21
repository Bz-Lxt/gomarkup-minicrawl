package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/mocksite"
	"minicrawl/internal/store"
	"minicrawl/internal/timeutil"
)

type Server struct {
	log    *slog.Logger
	engine *crawler.Engine
	corpus *store.Corpus
	index  *index.Inverted
	mode   string
}

func New(log *slog.Logger, engine *crawler.Engine, corpus *store.Corpus, inv *index.Inverted, mode string) http.Handler {
	s := &Server{log: log, engine: engine, corpus: corpus, index: inv, mode: mode}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/stats/timeseries", s.timeseries)
	mux.HandleFunc("GET /api/graph", s.graph)
	mux.HandleFunc("GET /api/keywords/top", s.keywords)
	mux.HandleFunc("GET /api/search", s.search)
	mux.HandleFunc("POST /api/crawl/tasks", s.createTask)
	mux.HandleFunc("GET /api/crawl/tasks", s.listTasks)
	mux.HandleFunc("GET /api/crawl/tasks/{id}", s.getTask)
	mux.HandleFunc("POST /api/crawl/tasks/{id}/stop", s.stopTask)
	mux.HandleFunc("POST /api/crawl/tasks/{id}/pause", s.pauseTask)
	mux.HandleFunc("POST /api/crawl/tasks/{id}/resume", s.resumeTask)
	mux.HandleFunc("POST /api/index/clear", s.clearIndex)
	mux.Handle("/fixture/", http.StripPrefix("/fixture", mocksite.Handler()))
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /fixture/secret\nDisallow: /api/\n"))
	})
	return withCORS(withLog(log, mux))
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status, code, msg string) {
	writeJSON(w, statusCode(status), map[string]apiError{"error": {Code: code, Message: msg}})
}

func statusCode(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return http.StatusInternalServerError
	}
	return n
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   timeutil.Format(timeutil.Now()),
		"mode":   s.mode,
	})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	docs, terms := s.index.Stats()
	active, configured := s.engine.WorkerSnapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"pages_crawled": s.engine.PagesCrawled(),
		"queue_length":  s.engine.QueueLen(),
		"index_docs":    docs,
		"index_terms":   terms,
		"current_rps":   s.engine.CurrentRPS(),
		"workers": map[string]int{
			"active":      active,
			"configured":  configured,
		},
		"crawl_mode": s.mode,
		"updated_at": timeutil.Format(timeutil.Now()),
	})
}

func (s *Server) timeseries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"points": s.engine.Timeseries()})
}

func (s *Server) graph(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, s.corpus.Graph(limit))
}

func (s *Server) keywords(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, map[string]any{"keywords": s.index.TopKeywords(limit)})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeErr(w, "400", "VALIDATION_ERROR", "q is required")
		return
	}
	hl := r.URL.Query().Get("highlight") != "false"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits := s.index.Search(q, hl, limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"query": q,
		"total": len(hits),
		"hits":  hits,
	})
}

type createBody struct {
	Seeds       []string `json:"seeds"`
	UseFixture  bool     `json:"use_fixture"`
	Workers     int      `json:"workers"`
	GlobalRPS   float64  `json:"global_rps"`
	HostRPS     float64  `json:"host_rps"`
	MaxDepth    int      `json:"max_depth"`
	MaxPages    int      `json:"max_pages"`
	UserAgent   string   `json:"user_agent"`
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeErr(w, "400", "VALIDATION_ERROR", "invalid json")
		return
	}
	seeds := body.Seeds
	if body.UseFixture || (len(seeds) == 0 && s.mode == "mock") {
		seeds = []string{s.engine.FixtureSeed()}
	}
	if body.Workers < 0 || body.Workers > 64 {
		writeErr(w, "400", "VALIDATION_ERROR", "workers must be 0-64")
		return
	}
	if body.MaxDepth < 0 || body.MaxDepth > 20 {
		writeErr(w, "400", "VALIDATION_ERROR", "max_depth must be 0-20")
		return
	}
	if body.MaxPages < 0 || body.MaxPages > 5000 {
		writeErr(w, "400", "VALIDATION_ERROR", "max_pages must be 0-5000")
		return
	}
	view, err := s.engine.Create(crawler.Strategy{
		Seeds:     seeds,
		Workers:   body.Workers,
		GlobalRPS: body.GlobalRPS,
		HostRPS:   body.HostRPS,
		MaxDepth:  body.MaxDepth,
		MaxPages:  body.MaxPages,
		UserAgent: body.UserAgent,
	})
	if err != nil {
		msg := err.Error()
		code := "VALIDATION_ERROR"
		st := "400"
		if strings.Contains(msg, "already") {
			code = "CONFLICT"
			st = "409"
		}
		writeErr(w, st, code, msg)
		return
	}
	w.Header().Set("Location", "/api/crawl/tasks/"+view.ID)
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tasks": s.engine.List()})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	view, ok := s.engine.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, "404", "NOT_FOUND", "task not found")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) stopTask(w http.ResponseWriter, r *http.Request) {
	view, err := s.engine.Stop(r.PathValue("id"))
	if err != nil {
		s.taskActionErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) pauseTask(w http.ResponseWriter, r *http.Request) {
	view, err := s.engine.Pause(r.PathValue("id"))
	if err != nil {
		s.taskActionErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) resumeTask(w http.ResponseWriter, r *http.Request) {
	view, err := s.engine.Resume(r.PathValue("id"))
	if err != nil {
		s.taskActionErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) taskActionErr(w http.ResponseWriter, err error) {
	msg := err.Error()
	if msg == "not found" {
		writeErr(w, "404", "NOT_FOUND", msg)
		return
	}
	writeErr(w, "409", "TASK_STATE", msg)
}

func (s *Server) clearIndex(w http.ResponseWriter, r *http.Request) {
	s.corpus.Clear()
	s.index.Clear()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": timeutil.Format(timeutil.Now())})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Info("http", "method", r.Method, "path", r.URL.Path)
		}
	})
}
