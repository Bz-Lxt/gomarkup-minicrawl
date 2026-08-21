package crawler

import "testing"

func TestNormalizeDedup(t *testing.T) {
	a, err := Normalize("HTTP://Example.COM:80/a/../b/?c=2&c=1&b=1#frag")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Normalize("http://example.com/b?b=1&c=1&c=2")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("normalize mismatch %s vs %s", a, b)
	}
}

func TestDedupQueueNoDuplicate(t *testing.T) {
	q := NewDedupQueue()
	if !q.Push("http://x.test/p#a", 0) {
		t.Fatal("first push")
	}
	if q.Push("http://x.test/p#b", 1) {
		t.Fatal("duplicate should be rejected")
	}
	if q.SeenCount() != 1 {
		t.Fatalf("seen=%d", q.SeenCount())
	}
	if q.Len() != 1 {
		t.Fatalf("len=%d", q.Len())
	}
}
