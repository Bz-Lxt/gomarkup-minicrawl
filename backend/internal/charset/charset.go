package charset

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

func Normalize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	switch name {
	case "utf8", "utf-8", "unicode-1-1-utf-8":
		return "utf-8"
	case "gbk", "gb2312", "gb-2312", "windows-936":
		return "gbk"
	case "gb18030":
		return "gb18030"
	case "big5", "big-5", "cn-big5":
		return "big5"
	case "iso-8859-1", "latin1", "latin-1":
		return "iso-8859-1"
	case "us-ascii", "ascii":
		return "ascii"
	default:
		return name
	}
}

func FromContentType(ct string) string {
	ct = strings.ToLower(ct)
	for _, part := range strings.Split(ct, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "charset=") {
			return Normalize(strings.Trim(part[8:], `"'`))
		}
	}
	return ""
}

func FromMeta(html []byte) string {
	low := bytes.ToLower(html)
	if len(low) > 4096 {
		low = low[:4096]
	}
	key := []byte("charset=")
	i := bytes.Index(low, key)
	if i < 0 {
		return ""
	}
	rest := low[i+len(key):]
	rest = bytes.TrimLeft(rest, " \t\"'")
	end := 0
	for end < len(rest) {
		c := rest[end]
		if c == '"' || c == '\'' || c == ' ' || c == ';' || c == '>' {
			break
		}
		end++
	}
	return Normalize(string(rest[:end]))
}

func LooksUTF8(b []byte) bool {
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return true
	}
	return utf8.Valid(b)
}

func Detect(b []byte, contentType string) string {
	if name := FromContentType(contentType); name != "" {
		return name
	}
	if name := FromMeta(b); name != "" {
		return name
	}
	if LooksUTF8(b) {
		return "utf-8"
	}
	return ""
}

func DecodeAsUTF8(b []byte, name string) []byte {
	name = Normalize(name)
	if name == "" || name == "utf-8" || name == "ascii" {
		return b
	}
	return b
}

func ReadLimit(r io.Reader, n int) ([]byte, error) {
	if n <= 0 {
		n = 1 << 20
	}
	return io.ReadAll(io.LimitReader(r, int64(n)))
}

func SniffHTML(b []byte) bool {
	s := bytes.TrimSpace(bytes.ToLower(b))
	if len(s) > 256 {
		s = s[:256]
	}
	return bytes.Contains(s, []byte("<html")) || bytes.Contains(s, []byte("<!doctype html"))
}
