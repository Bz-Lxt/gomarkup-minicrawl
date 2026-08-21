package crawler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"minicrawl/internal/index"
	"minicrawl/internal/store"
)

// TestQueueRaceReproduction reproduces the concurrency bug where the queue
// auto-closes while a worker is still processing an item, causing links
// discovered by that worker to be silently dropped.
//
// Scenario:
//   - Push two seeds A and B.
//   - Two workers pop both: inflight=2, items=[].
//   - Worker A finishes immediately (no new links), calls Done().
//   - With the OLD code, Done() sees len(items)==0 and closes the queue.
//   - Worker B then tries to Push new links discovered while processing B.
//     With the bug, Push returns false (queue closed) -> links dropped.
//   - Worker B's Pop returns false -> worker exits -> premature "completed".
//
// With the fix, Done() must keep the queue open while inflight>0.
func TestQueueRaceReproduction(t *testing.T) {
	q := NewDedupQueue()
	ctx := context.Background()

	// Two seeds.
	if !q.Push("http://a.test/1", 0) {
		t.Fatal("push A")
	}
	if !q.Push("http://b.test/2", 0) {
		t.Fatal("push B")
	}

	itemA, ok := q.Pop(ctx)
	if !ok {
		t.Fatal("pop A")
	}
	itemB, ok := q.Pop(ctx)
	if !ok {
		t.Fatal("pop B")
	}
	_ = itemA
	_ = itemB

	// At this point items is empty, inflight == 2.

	var wg sync.WaitGroup

	// Worker A: finishes fast, no new links found.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Simulate fast processing with no links discovered.
		q.Done()
	}()

	// Worker B: finishes slower, discovers new links that MUST be queued.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Simulate slower processing.
		time.Sleep(20 * time.Millisecond)
		// Worker B discovers new links while processing itemB.
		if !q.Push("http://b.test/child1", 1) {
			t.Errorf("push child1 should succeed but queue was closed prematurely")
		}
		if !q.Push("http://b.test/child2", 1) {
			t.Errorf("push child2 should succeed but queue was closed prematurely")
		}
		q.Done()
	}()

	wg.Wait()

	// After both workers finish, the two children must be poppable.
	got := 0
	for {
		item, ok := q.Pop(ctx)
		if !ok {
			break
		}
		got++
		_ = item
		q.Done()
	}
	if got != 2 {
		t.Fatalf("expected 2 children in queue, got %d (links were dropped)", got)
	}
}

// TestQueueCompletesWhenIdle verifies that the queue still auto-closes
// correctly when all work is genuinely done (inflight==0, items empty).
func TestQueueCompletesWhenIdle(t *testing.T) {
	q := NewDedupQueue()
	ctx := context.Background()

	q.Push("http://x.test/1", 0)
	item, ok := q.Pop(ctx)
	if !ok {
		t.Fatal("pop")
	}
	_ = item
	// No new links discovered.
	q.Done()

	// Queue should now be closed (inflight==0, items==0).
	_, ok = q.Pop(ctx)
	if ok {
		t.Fatal("queue should be closed after idle")
	}
}

// TestQueueStaysOpenWithInflightAndItems verifies that when there are
// pending items AND inflight pages, Done() does not close.
func TestQueueStaysOpenWithInflightAndItems(t *testing.T) {
	q := NewDedupQueue()
	ctx := context.Background()

	q.Push("http://x.test/1", 0)
	q.Push("http://x.test/2", 0)
	q.Push("http://x.test/3", 0)

	item, _ := q.Pop(ctx)
	_ = item
	// inflight=1, items has 2 entries.
	q.Done()
	// inflight=0, items still has 2 -> should NOT close.

	item2, ok := q.Pop(ctx)
	if !ok {
		t.Fatal("queue should still be open")
	}
	_ = item2
	q.Done()
	// items=1, inflight=0 -> should NOT close.

	item3, ok := q.Pop(ctx)
	if !ok {
		t.Fatal("queue should still be open")
	}
	_ = item3
	q.Done()
	// items=0, inflight=0 -> should close.

	_, ok = q.Pop(ctx)
	if ok {
		t.Fatal("queue should be closed now")
	}
}

// TestEngineMultiSeedConcurrentNoDrop is a regression test for the
// premature-completion bug. With multiple seeds and multiple workers, the
// queue must not close while a worker is still processing a page that may
// discover new links. This test uses a mock site where each page links to
// the next, so a premature close would cause many pages to be missed.
//
// We use two independent chains of 10 pages each. With the buggy Done()
// (which only checked len(items)==0, ignoring inflight), the faster worker's
// Done() call would close the queue while the slower worker was still
// processing, silently dropping the rest of the slower chain.
func TestEngineMultiSeedConcurrentNoDrop(t *testing.T) {
	const chainLen = 10
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "User-agent: *\nAllow: /\n")
	})
	for i := 0; i < chainLen; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/m/%d.html", i), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if i+1 < chainLen {
				fmt.Fprintf(w, `<html><head><title>page %d</title></head><body><p>minicrawl page %d</p><a href="/m/%d.html">next</a></body></html>`, i, i, i+1)
			} else {
				fmt.Fprintf(w, `<html><head><title>page %d</title></head><body><p>minicrawl page %d end</p></body></html>`, i, i)
			}
		})
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	corpus := store.NewCorpus()
	inv := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(log, corpus, inv, "live", "")

	// Two seeds: start of two independent chains. With the bug, whichever
	// worker finishes first would close the queue while the other is still
	// processing, dropping the rest of the chain.
	view, err := eng.Create(Strategy{
		Seeds:    []string{ts.URL + "/m/0.html", ts.URL + "/m/5.html"},
		Workers:  8,
		MaxDepth: 10,
		MaxPages: 200,
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for {
		got, ok := eng.Get(view.ID)
		if !ok {
			t.Fatal("task missing")
		}
		if got.Status == StatusCompleted || got.Status == StatusFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout status=%s crawled=%d", got.Status, got.Crawled)
		}
		time.Sleep(20 * time.Millisecond)
	}

	got, _ := eng.Get(view.ID)
	if got.Status != StatusCompleted {
		t.Fatalf("status=%s err=%s", got.Status, got.Error)
	}
	// Chain 0..9 has 10 pages, chain 5..9 has 5 pages (5..9 overlap with
	// the first chain). Union = 10 unique pages. If the queue closed
	// prematurely we would see fewer.
	if got.Crawled < 10 {
		t.Fatalf("crawled %d, want >= 10 (links were dropped by premature close)", got.Crawled)
	}
}
