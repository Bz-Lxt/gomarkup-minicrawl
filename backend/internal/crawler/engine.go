package crawler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/parser"
	"minicrawl/internal/store"
	"minicrawl/internal/timeutil"
	"minicrawl/internal/urlfilter"
)

type task struct {
	id        string
	status    Status
	strategy  Strategy
	crawled   atomic.Int32
	err       string
	createdAt time.Time
	startedAt time.Time
	finished  time.Time
	queue     *DedupQueue
	cancel    context.CancelFunc
	paused    atomic.Bool
}

type Engine struct {
	log     *slog.Logger
	client  *http.Client
	corpus  *store.Corpus
	index   *index.Inverted
	mode    string
	fixture string

	mu     sync.Mutex
	tasks  map[string]*task
	active *task

	pages atomic.Int64
	tickMu sync.Mutex
	samples []Sample
	lastPages int64
	lastRPS   float64
}

func NewEngine(log *slog.Logger, corpus *store.Corpus, inv *index.Inverted, mode, fixtureBase string) *Engine {
	e := &Engine{
		log:     log,
		corpus:  corpus,
		index:   inv,
		mode:    mode,
		fixture: strings.TrimRight(fixtureBase, "/"),
		tasks:   make(map[string]*task),
		client: &http.Client{
			Timeout: 12 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
	go e.sampleLoop()
	return e
}

func (e *Engine) sampleLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		cur := e.pages.Load()
		e.tickMu.Lock()
		delta := cur - e.lastPages
		e.lastPages = cur
		e.lastRPS = float64(delta)
		e.samples = append(e.samples, Sample{
			Time:  timeutil.Format(timeutil.Now()),
			Pages: int(cur),
			RPS:   float64(delta),
		})
		if len(e.samples) > 120 {
			e.samples = e.samples[len(e.samples)-120:]
		}
		e.tickMu.Unlock()
	}
}

func (e *Engine) Timeseries() []Sample {
	e.tickMu.Lock()
	defer e.tickMu.Unlock()
	out := make([]Sample, len(e.samples))
	copy(out, e.samples)
	return out
}

func (e *Engine) CurrentRPS() float64 {
	e.tickMu.Lock()
	defer e.tickMu.Unlock()
	return e.lastRPS
}

func (e *Engine) PagesCrawled() int {
	return int(e.pages.Load())
}

func (e *Engine) Mode() string { return e.mode }

func (e *Engine) FixtureSeed() string {
	return e.fixture + "/index.html"
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *Engine) Create(s Strategy) (*TaskView, error) {
	s = clampStrategy(s)

	seeds := make([]string, 0, len(s.Seeds))
	for _, raw := range s.Seeds {
		if urlfilter.ShouldSkip(raw) && !strings.HasPrefix(raw, "/") {
			continue
		}
		n, err := Normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid seed: %s", raw)
		}
		if e.mode == "mock" && !e.allowedMockURL(n) {
			return nil, fmt.Errorf("mock mode only allows fixture URLs")
		}
		seeds = append(seeds, n)
	}
	if len(seeds) == 0 {
		if e.mode == "mock" {
			seeds = []string{e.FixtureSeed()}
		} else {
			return nil, fmt.Errorf("seeds required")
		}
	}
	s.Seeds = seeds

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active != nil && (e.active.status == StatusRunning || e.active.status == StatusPaused) {
		return nil, fmt.Errorf("a task is already %s", e.active.status)
	}

	t := &task{
		id:        newID(),
		status:    StatusPending,
		strategy:  s,
		createdAt: timeutil.Now(),
		queue:     NewDedupQueue(),
	}
	e.tasks[t.id] = t
	e.active = t
	go e.run(t)
	return e.viewLocked(t), nil
}

func (e *Engine) allowedMockURL(n string) bool {
	return strings.HasPrefix(n, e.fixture)
}

func (e *Engine) Get(id string) (*TaskView, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.tasks[id]
	if !ok {
		return nil, false
	}
	return e.viewLocked(t), true
}

func (e *Engine) List() []*TaskView {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*TaskView, 0, len(e.tasks))
	for _, t := range e.tasks {
		out = append(out, e.viewLocked(t))
	}
	return out
}

func (e *Engine) viewLocked(t *task) *TaskView {
	v := &TaskView{
		ID:          t.id,
		Status:      t.status,
		Strategy:    t.strategy,
		Crawled:     int(t.crawled.Load()),
		QueueLength: t.queue.Len(),
		Error:       t.err,
		CreatedAt:   timeutil.Format(t.createdAt),
		StartedAt:   timeutil.Format(t.startedAt),
		FinishedAt:  timeutil.Format(t.finished),
	}
	return v
}

func (e *Engine) Stop(id string) (*TaskView, error) {
	return e.signal(id, StatusStopped)
}

func (e *Engine) Pause(id string) (*TaskView, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.tasks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if t.status != StatusRunning {
		return nil, fmt.Errorf("cannot pause from %s", t.status)
	}
	t.paused.Store(true)
	t.status = StatusPaused
	return e.viewLocked(t), nil
}

func (e *Engine) Resume(id string) (*TaskView, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.tasks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if t.status != StatusPaused {
		return nil, fmt.Errorf("cannot resume from %s", t.status)
	}
	t.paused.Store(false)
	t.status = StatusRunning
	return e.viewLocked(t), nil
}

func (e *Engine) signal(id string, st Status) (*TaskView, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.tasks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if t.status != StatusRunning && t.status != StatusPaused && t.status != StatusPending {
		return nil, fmt.Errorf("cannot stop from %s", t.status)
	}
	t.status = st
	if t.cancel != nil {
		t.cancel()
	}
	t.queue.Close()
	return e.viewLocked(t), nil
}

func (e *Engine) QueueLen() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active == nil {
		return 0
	}
	return e.active.queue.Len()
}

func (e *Engine) WorkerSnapshot() (active, configured int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active == nil {
		return 0, 0
	}
	configured = e.active.strategy.Workers
	if e.active.status == StatusRunning {
		active = configured
	}
	return active, configured
}

func (e *Engine) run(t *task) {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	defer cancel()

	e.mu.Lock()
	t.status = StatusRunning
	t.startedAt = timeutil.Now()
	e.mu.Unlock()

	for _, s := range t.strategy.Seeds {
		t.queue.Push(s, 0)
	}
	if t.queue.Len() == 0 {
		e.finish(t, StatusFailed, "no valid seeds")
		return
	}

	lim := NewLimiter(t.strategy.GlobalRPS, t.strategy.HostRPS)
	robots := NewRobots(e.client, t.strategy.UserAgent)
	var wg sync.WaitGroup
	for i := 0; i < t.strategy.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				for t.paused.Load() {
					if ctx.Err() != nil {
						return
					}
					time.Sleep(40 * time.Millisecond)
				}
				item, ok := t.queue.Pop(ctx)
				if !ok {
					return
				}
				e.process(ctx, t, item, lim, robots)
				t.queue.Done()
			}
		}()
	}
	wg.Wait()

	e.mu.Lock()
	st := t.status
	e.mu.Unlock()
	if st == StatusStopped {
		e.finish(t, StatusStopped, "")
		return
	}
	e.finish(t, StatusCompleted, "")
}

func (e *Engine) finish(t *task, st Status, err string) {
	e.mu.Lock()
	if t.status == StatusRunning || t.status == StatusPaused || t.status == StatusPending {
		t.status = st
	} else if st == StatusCompleted && t.status == StatusStopped {
		// keep stopped
	} else if t.status != StatusStopped {
		t.status = st
	}
	t.err = err
	t.queue.Close()
	e.mu.Unlock()
	t.finished = timeutil.Now()
}

func (e *Engine) process(ctx context.Context, t *task, item Item, lim *Limiter, robots *Robots) {
	if int(t.crawled.Load()) >= t.strategy.MaxPages {
		t.queue.Close()
		return
	}
	if item.Depth > t.strategy.MaxDepth {
		return
	}
	if e.corpus.Has(item.URL) {
		return
	}
	if !robots.Allowed(item.URL) {
		e.log.Info("robots disallow", "url", item.URL)
		return
	}
	if err := lim.Wait(ctx, HostOf(item.URL)); err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", t.strategy.UserAgent)
	resp, err := e.client.Do(req)
	if err != nil {
		e.log.Debug("fetch failed", "url", item.URL, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(strings.ToLower(ct), "html") && !strings.Contains(strings.ToLower(ct), "text/plain") {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return
	}
	base, _ := url.Parse(item.URL)
	page, err := parser.Parse(body, base)
	if err != nil {
		return
	}
	title := page.Title
	if title == "" {
		title = item.URL
	}
	doc := e.corpus.Upsert(item.URL, title, page.Text)
	e.index.Add(*doc)
	t.crawled.Add(1)
	e.pages.Add(1)

	if item.Depth >= t.strategy.MaxDepth {
		return
	}
	if int(t.crawled.Load()) >= t.strategy.MaxPages {
		t.queue.Close()
		return
	}
	for _, link := range page.Links {
		norm, err := Normalize(link)
		if err != nil {
			continue
		}
		if e.mode == "mock" && !e.allowedMockURL(norm) {
			continue
		}
		e.corpus.AddEdge(item.URL, norm)
		t.queue.Push(norm, item.Depth+1)
	}
}
