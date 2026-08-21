package htmltext

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	scriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	tagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	wsRe     = regexp.MustCompile(`[ \t\r\n\f]+`)
)

func Strip(html string) string {
	s := scriptRe.ReplaceAllString(html, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = unescape(s)
	s = wsRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func Title(html string) string {
	low := strings.ToLower(html)
	start := strings.Index(low, "<title")
	if start < 0 {
		return ""
	}
	gt := strings.Index(html[start:], ">")
	if gt < 0 {
		return ""
	}
	rest := html[start+gt+1:]
	end := strings.Index(strings.ToLower(rest), "</title>")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(unescape(rest[:end]))
}

func Snippet(html string, n int) string {
	text := Strip(html)
	if n <= 0 || utf8.RuneCountInString(text) <= n {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:n])) + "…"
}

func WordCount(html string) int {
	text := Strip(html)
	if text == "" {
		return 0
	}
	n := 0
	in := false
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			in = false
			continue
		}
		if !in {
			n++
			in = true
		}
	}
	return n
}

func unescape(s string) string {
	repl := []string{
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
	}
	return strings.NewReplacer(repl...).Replace(s)
}

func HasLang(html, lang string) bool {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return false
	}
	low := strings.ToLower(html)
	return strings.Contains(low, `lang="`+lang) || strings.Contains(low, `lang='`+lang)
}
