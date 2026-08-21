package parser

import (
	"net/url"
	"strings"
	"testing"
)

func TestParseExtract(t *testing.T) {
	htmlDoc := `<html><head><title>Hello</title></head><body>
<nav>skip me</nav>
<script>var x=1</script>
<p>Visible text</p>
<a href="/next">n</a>
</body></html>`
	base, _ := url.Parse("http://ex.test/page")
	p, err := Parse([]byte(htmlDoc), base)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Hello" {
		t.Fatalf("title=%s", p.Title)
	}
	if strings.Contains(p.Text, "skip me") || strings.Contains(p.Text, "var x") {
		t.Fatalf("noise leaked: %s", p.Text)
	}
	if !strings.Contains(p.Text, "Visible") {
		t.Fatalf("text=%s", p.Text)
	}
	if len(p.Links) != 1 || p.Links[0] != "http://ex.test/next" {
		t.Fatalf("links=%v", p.Links)
	}
}
