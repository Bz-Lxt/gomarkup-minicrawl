package crawler

import (
	"context"
	"sync"
)

type Item struct {
	URL   string
	Depth int
}

type DedupQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	items    []Item
	seen     map[string]struct{}
	inflight int
	closed   bool
}

func NewDedupQueue() *DedupQueue {
	q := &DedupQueue{seen: make(map[string]struct{})}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *DedupQueue) Push(raw string, depth int) bool {
	norm, err := Normalize(raw)
	if err != nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	if _, ok := q.seen[norm]; ok {
		return false
	}
	q.seen[norm] = struct{}{}
	q.items = append(q.items, Item{URL: norm, Depth: depth})
	q.cond.Signal()
	return true
}

func (q *DedupQueue) Pop(ctx context.Context) (Item, bool) {
	stop := context.AfterFunc(ctx, func() {
		q.mu.Lock()
		q.cond.Broadcast()
		q.mu.Unlock()
	})
	defer stop()

	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed && ctx.Err() == nil {
		q.cond.Wait()
	}
	if ctx.Err() != nil {
		return Item{}, false
	}
	if len(q.items) == 0 {
		return Item{}, false
	}
	it := q.items[0]
	q.items = q.items[1:]
	q.inflight++
	return it, true
}

func (q *DedupQueue) Done() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.inflight > 0 {
		q.inflight--
	}
	if q.inflight == 0 && len(q.items) == 0 {
		q.closed = true
		q.cond.Broadcast()
	}
}

func (q *DedupQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}

func (q *DedupQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *DedupQueue) SeenCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.seen)
}

func (q *DedupQueue) Has(raw string) bool {
	norm, err := Normalize(raw)
	if err != nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.seen[norm]
	return ok
}
