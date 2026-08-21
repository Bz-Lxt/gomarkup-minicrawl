package index

import (
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var latinWord = regexp.MustCompile(`[a-z0-9]+`)

var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "of": {}, "to": {},
	"in": {}, "for": {}, "on": {}, "is": {}, "are": {}, "was": {}, "be": {},
	"as": {}, "at": {}, "by": {}, "this": {}, "that": {}, "with": {}, "from": {},
	"it": {}, "its": {}, "we": {}, "you": {}, "your": {}, "not": {}, "but": {},
	"if": {}, "can": {}, "will": {}, "has": {}, "have": {}, "had": {},
}

func Tokenize(s string) []string {
	s = strings.ToLower(s)
	var out []string
	var cjk []rune
	flush := func() {
		if len(cjk) == 1 {
			out = append(out, string(cjk))
		} else {
			for i := 0; i < len(cjk)-1; i++ {
				out = append(out, string(cjk[i:i+2]))
			}
		}
		cjk = cjk[:0]
	}
	for _, r := range s {
		if isCJK(r) {
			cjk = append(cjk, r)
			continue
		}
		flush()
	}
	flush()
	for _, w := range latinWord.FindAllString(s, -1) {
		w = stem(w)
		if len(w) < 2 {
			continue
		}
		if _, stop := stopwords[w]; stop {
			continue
		}
		out = append(out, w)
	}
	return out
}

func QueryTerms(q string) []string {
	seen := map[string]struct{}{}
	var terms []string
	for _, t := range Tokenize(q) {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		terms = append(terms, t)
	}
	return terms
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

func stem(w string) string {
	if len(w) > 4 && strings.HasSuffix(w, "ies") {
		return w[:len(w)-3] + "y"
	}
	if len(w) > 5 && strings.HasSuffix(w, "ing") {
		return w[:len(w)-3]
	}
	if len(w) > 4 && strings.HasSuffix(w, "ed") {
		return w[:len(w)-2]
	}
	if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
		return w[:len(w)-1]
	}
	return w
}

func Highlight(text, query string) string {
	esc := html.EscapeString(text)
	terms := highlightNeedles(query)
	if len(terms) == 0 {
		return esc
	}
	reParts := make([]string, 0, len(terms))
	for _, t := range terms {
		if t == "" {
			continue
		}
		reParts = append(reParts, regexp.QuoteMeta(html.EscapeString(t)))
	}
	if len(reParts) == 0 {
		return esc
	}
	re := regexp.MustCompile(`(?i)(` + strings.Join(reParts, "|") + `)`)
	return re.ReplaceAllString(esc, `<mark>$1</mark>`)
}

func highlightNeedles(query string) []string {
	q := strings.TrimSpace(query)
	var out []string
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		k := strings.ToLower(s)
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	add(q)
	for _, w := range strings.Fields(q) {
		add(w)
	}
	for _, t := range Tokenize(q) {
		if utf8.RuneCountInString(t) >= 2 {
			add(t)
		}
	}
	return out
}

func Snippet(body, query string, highlight bool) string {
	runes := []rune(body)
	if len(runes) == 0 {
		return ""
	}
	q := strings.ToLower(strings.TrimSpace(query))
	idx := strings.Index(strings.ToLower(body), q)
	if idx < 0 {
		fields := strings.Fields(query)
		for _, f := range fields {
			idx = strings.Index(strings.ToLower(body), strings.ToLower(f))
			if idx >= 0 {
				break
			}
		}
	}
	start := 0
	if idx > 0 {
		start = idx - 60
		if start < 0 {
			start = 0
		}
		start = len([]rune(body[:start]))
	}
	end := start + 180
	if end > len(runes) {
		end = len(runes)
	}
	snip := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snip = "…" + snip
	}
	if end < len(runes) {
		snip = snip + "…"
	}
	if highlight {
		return Highlight(snip, query)
	}
	return html.EscapeString(snip)
}
