package crawler_test

import (
	"io"
	"log/slog"
	"runtime"
	"sync"
	"testing"
	"time"

	"minicrawl/internal/crawler"
	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestCompletedTaskPublishesFinishedAtAtomically(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := crawler.NewEngine(log, store.NewCorpus(), index.New(), "live", "")
	view, err := eng.Create(crawler.Strategy{
		Seeds:    []string{"http://127.0.0.1:1/"},
		Workers:  4,
		MaxDepth: 1,
		MaxPages: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	bad := make(chan crawler.TaskView, 1)
	deadline := time.Now().Add(5 * time.Second)
	for i := 0; i < runtime.GOMAXPROCS(0)*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				got, ok := eng.Get(view.ID)
				if !ok {
					return
				}
				if got.Status == crawler.StatusCompleted {
					if got.FinishedAt == "" {
						select {
						case bad <- *got:
						default:
						}
					}
					return
				}
				runtime.Gosched()
			}
		}()
	}
	wg.Wait()

	select {
	case got := <-bad:
		t.Fatalf("completed task exposed without completion time: %+v", got)
	default:
	}
	got, _ := eng.Get(view.ID)
	if got.Status != crawler.StatusCompleted || got.FinishedAt == "" {
		t.Fatalf("task did not publish a complete terminal view: %+v", got)
	}
}
