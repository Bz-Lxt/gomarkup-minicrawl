package store

import (
	"sync"

	"minicrawl/internal/index"
	"minicrawl/internal/timeutil"
)

type NodeStatus string

const (
	NodeCrawled NodeStatus = "crawled"
	NodePending NodeStatus = "pending"
)

type Node struct {
	ID     string     `json:"id"`
	URL    string     `json:"url"`
	Title  string     `json:"title"`
	Status NodeStatus `json:"status"`
	Degree int        `json:"degree"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Corpus struct {
	mu      sync.RWMutex
	docs    map[string]*index.Document
	byID    map[int]*index.Document
	nextID  int
	nodes   map[string]*Node
	edgeSet map[string]Edge
}

func NewCorpus() *Corpus {
	return &Corpus{
		docs:    make(map[string]*index.Document),
		byID:    make(map[int]*index.Document),
		nodes:   make(map[string]*Node),
		edgeSet: make(map[string]Edge),
		nextID:  1,
	}
}

func (c *Corpus) Upsert(url, title, body string) *index.Document {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d, ok := c.docs[url]; ok {
		d.Title = title
		d.Body = body
		if n, ok := c.nodes[url]; ok {
			n.Title = title
			n.Status = NodeCrawled
		}
		return d
	}
	d := &index.Document{ID: c.nextID, URL: url, Title: title, Body: body}
	c.nextID++
	c.docs[url] = d
	c.byID[d.ID] = d
	c.nodes[url] = &Node{ID: url, URL: url, Title: title, Status: NodeCrawled}
	return d
}

func (c *Corpus) Has(url string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.docs[url]
	return ok
}

func (c *Corpus) AddEdge(from, to string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if from == "" || to == "" || from == to {
		return
	}
	key := from + " -> " + to
	if _, ok := c.edgeSet[key]; ok {
		return
	}
	c.edgeSet[key] = Edge{From: from, To: to}
	if _, ok := c.nodes[from]; !ok {
		c.nodes[from] = &Node{ID: from, URL: from, Status: NodePending}
	}
	if _, ok := c.nodes[to]; !ok {
		c.nodes[to] = &Node{ID: to, URL: to, Status: NodePending}
	}
}

func (c *Corpus) Graph(limit int) Graph {
	if limit <= 0 || limit > 800 {
		limit = 400
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	deg := map[string]int{}
	edges := make([]Edge, 0, len(c.edgeSet))
	for _, e := range c.edgeSet {
		edges = append(edges, e)
		deg[e.From]++
		deg[e.To]++
	}
	nodes := make([]Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		item := *n
		item.Degree = deg[n.ID]
		nodes = append(nodes, item)
	}
	if len(nodes) > limit {
		nodes = nodes[:limit]
		keep := map[string]struct{}{}
		for _, n := range nodes {
			keep[n.ID] = struct{}{}
		}
		filtered := edges[:0]
		for _, e := range edges {
			if _, a := keep[e.From]; !a {
				continue
			}
			if _, b := keep[e.To]; !b {
				continue
			}
			filtered = append(filtered, e)
		}
		edges = filtered
	}
	_ = timeutil.Now
	return Graph{Nodes: nodes, Edges: edges}
}

func (c *Corpus) DocCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.docs)
}

func (c *Corpus) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.docs = make(map[string]*index.Document)
	c.byID = make(map[int]*index.Document)
	c.nodes = make(map[string]*Node)
	c.edgeSet = make(map[string]Edge)
	c.nextID = 1
}
