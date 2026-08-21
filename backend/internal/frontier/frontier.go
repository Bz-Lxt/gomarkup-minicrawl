package frontier

import (
	"container/heap"
	"net/url"
	"strings"
	"sync"
)

type Item struct {
	URL      string
	Host     string
	Depth    int
	Priority int
	index    int
}

type queue []*Item

func (q queue) Len() int { return len(q) }

func (q queue) Less(i, j int) bool {
	if q[i].Priority != q[j].Priority {
		return q[i].Priority > q[j].Priority
	}
	return q[i].Depth < q[j].Depth
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *queue) Push(x any) {
	item := x.(*Item)
	item.index = len(*q)
	*q = append(*q, item)
}

func (q *queue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*q = old[:n-1]
	return item
}

type Frontier struct {
	mu    sync.Mutex
	q     queue
	seen  map[string]struct{}
	hosts map[string]int
}

func New() *Frontier {
	f := &Frontier{seen: make(map[string]struct{}), hosts: make(map[string]int)}
	heap.Init(&f.q)
	return f
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func (f *Frontier) Push(raw string, depth, priority int) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.seen[raw]; ok {
		return false
	}
	f.seen[raw] = struct{}{}
	h := hostOf(raw)
	item := &Item{URL: raw, Host: h, Depth: depth, Priority: priority}
	heap.Push(&f.q, item)
	f.hosts[h]++
	return true
}

func (f *Frontier) Pop() (Item, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.q.Len() == 0 {
		return Item{}, false
	}
	item := heap.Pop(&f.q).(*Item)
	if item.Host != "" {
		f.hosts[item.Host]--
		if f.hosts[item.Host] <= 0 {
			delete(f.hosts, item.Host)
		}
	}
	out := *item
	out.index = 0
	return out, true
}

func (f *Frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.q.Len()
}

func (f *Frontier) Seen(raw string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.seen[raw]
	return ok
}

func (f *Frontier) HostPending(host string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hosts[strings.ToLower(host)]
}

func (f *Frontier) SeenCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}
