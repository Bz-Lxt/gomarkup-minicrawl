package mimeutil

import (
	"path"
	"strings"
)

var byExt = map[string]string{
	".html": "text/html",
	".htm":  "text/html",
	".xml":  "application/xml",
	".txt":  "text/plain",
	".json": "application/json",
	".css":  "text/css",
	".js":   "application/javascript",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".pdf":  "application/pdf",
	".zip":  "application/zip",
}

func Parse(raw string) (media, charset string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	parts := strings.Split(raw, ";")
	media = strings.ToLower(strings.TrimSpace(parts[0]))
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "charset=") {
			charset = strings.Trim(part[8:], `"' `)
		}
	}
	return media, charset
}

func TypeOnly(raw string) string {
	media, _ := Parse(raw)
	return media
}

func IsHTML(raw string) bool {
	media, _ := Parse(raw)
	return media == "text/html" || media == "application/xhtml+xml"
}

func IsXML(raw string) bool {
	media, _ := Parse(raw)
	return media == "application/xml" || media == "text/xml" || strings.HasSuffix(media, "+xml")
}

func IsText(raw string) bool {
	media, _ := Parse(raw)
	return strings.HasPrefix(media, "text/") || IsHTML(raw) || IsXML(raw) || media == "application/json"
}

func IsBinary(raw string) bool {
	return raw != "" && !IsText(raw)
}

func FromPath(p string) string {
	ext := strings.ToLower(path.Ext(p))
	if t, ok := byExt[ext]; ok {
		return t
	}
	return ""
}

func FromURLPath(raw string) string {
	if i := strings.Index(raw, "?"); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.Index(raw, "#"); i >= 0 {
		raw = raw[:i]
	}
	return FromPath(raw)
}

func AllowCrawl(raw string) bool {
	media, _ := Parse(raw)
	if media == "" {
		return true
	}
	if IsHTML(raw) || IsXML(raw) || media == "text/plain" {
		return true
	}
	return false
}

func Charset(raw string) string {
	_, cs := Parse(raw)
	return strings.ToLower(strings.TrimSpace(cs))
}
