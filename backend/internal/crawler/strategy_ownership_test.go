package crawler

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

func TestCreateRetainsItsSeedConfiguration(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(log, store.NewCorpus(), index.New(), "live", "")
	seeds := []string{"http://127.0.0.1:1/first"}
	created, err := eng.Create(Strategy{Seeds: seeds, Workers: 1, MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		got, ok := eng.Get(created.ID)
		if !ok {
			t.Fatal("created task disappeared")
		}
		if got.Status == StatusCompleted {
			break
		}
		if got.Status == StatusFailed {
			t.Fatalf("task failed: %s", got.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not complete: status=%s", got.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	seeds[0] = "http://127.0.0.1:1/next-task"
	got, ok := eng.Get(created.ID)
	if !ok {
		t.Fatal("created task disappeared")
	}
	if got.Strategy.Seeds[0] != "http://127.0.0.1:1/first" {
		t.Fatalf("stored seed changed after caller reused its slice: got %q", got.Strategy.Seeds[0])
	}
}
