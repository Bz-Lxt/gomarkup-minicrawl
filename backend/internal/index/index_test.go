package index

import "strings"
import "testing"

func TestSearchHighlight(t *testing.T) {
	inv := New()
	inv.Add(Document{ID: 1, URL: "http://x/a", Title: "MiniCrawl Index", Body: "A concurrent web crawler builds an inverted index for search."})
	inv.Add(Document{ID: 2, URL: "http://x/b", Title: "Other", Body: "unrelated text about bananas"})
	hits := inv.Search("inverted", true, 10)
	if len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}
	if !strings.Contains(hits[0].Snippet, "<mark>") {
		t.Fatalf("no mark in %s", hits[0].Snippet)
	}
	if !strings.Contains(strings.ToLower(hits[0].Snippet), "inverted") {
		t.Fatalf("snippet missing term: %s", hits[0].Snippet)
	}
}

func TestChineseBigram(t *testing.T) {
	inv := New()
	inv.Add(Document{ID: 1, URL: "http://x/c", Title: "倒排索引", Body: "网页爬虫构建内存倒排索引"})
	hits := inv.Search("倒排", true, 5)
	if len(hits) == 0 {
		t.Fatal("expected chinese hit")
	}
}

func TestRemoveDoc(t *testing.T) {
	inv := New()
	inv.Add(Document{ID: 1, URL: "http://x/a", Title: "alpha", Body: "unique-token-xyz"})
	inv.Remove(1)
	if hits := inv.Search("unique-token-xyz", false, 5); len(hits) != 0 {
		t.Fatal("removed doc still searchable")
	}
}
