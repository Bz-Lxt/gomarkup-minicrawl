package store

import "testing"

func TestGraphLimit(t *testing.T) {
	c := NewCorpus()
	// Add 5 nodes and 4 edges forming a chain: a -> b -> c -> d -> e
	c.AddEdge("a", "b")
	c.AddEdge("b", "c")
	c.AddEdge("c", "d")
	c.AddEdge("d", "e")

	g := c.Graph(2)
	if len(g.Nodes) != 2 {
		t.Fatalf("limit=2 want 2 nodes, got %d", len(g.Nodes))
	}

	// Edges must only reference nodes present in the response.
	keep := map[string]struct{}{}
	for _, n := range g.Nodes {
		keep[n.ID] = struct{}{}
	}
	for _, e := range g.Edges {
		if _, ok := keep[e.From]; !ok {
			t.Fatalf("edge references missing node %q", e.From)
		}
		if _, ok := keep[e.To]; !ok {
			t.Fatalf("edge references missing node %q", e.To)
		}
	}
}

func TestGraphLimitBelowTotal(t *testing.T) {
	c := NewCorpus()
	for i := 0; i < 10; i++ {
		c.AddEdge("n"+itoa(i), "n"+itoa(i+1))
	}
	// Requesting more nodes than exist returns all of them.
	g := c.Graph(100)
	if len(g.Nodes) != 11 {
		t.Fatalf("limit=100 want 11 nodes, got %d", len(g.Nodes))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
