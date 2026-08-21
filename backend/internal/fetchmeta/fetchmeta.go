package fetchmeta

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Meta struct {
	Status       int
	ContentType  string
	Length       int64
	ETag         string
	LastModified time.Time
	CacheControl string
	Redirect     string
	FinalURL     string
}

func FromHeader(h http.Header, status int) Meta {
	m := Meta{
		Status:       status,
		ContentType:  h.Get("Content-Type"),
		ETag:         strings.Trim(h.Get("ETag"), `"`),
		CacheControl: h.Get("Cache-Control"),
		Redirect:     strings.TrimSpace(h.Get("Location")),
	}
	if cl := h.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil && n >= 0 {
			m.Length = n
		}
	}
	if lm := h.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			m.LastModified = t
		}
	}
	return m
}

func (m Meta) OK() bool { return m.Status >= 200 && m.Status < 300 }

func (m Meta) Redirected() bool { return m.Status >= 300 && m.Status < 400 && m.Redirect != "" }

func (m Meta) NotModified() bool { return m.Status == http.StatusNotModified }

func (m Meta) Retryable() bool {
	return m.Status == 408 || m.Status == 429 || m.Status >= 500
}

func (m Meta) NoStore() bool {
	cc := strings.ToLower(m.CacheControl)
	return strings.Contains(cc, "no-store") || strings.Contains(cc, "private")
}

func (m Meta) MaxAge() time.Duration {
	cc := strings.ToLower(m.CacheControl)
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			n, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil && n > 0 {
				return time.Duration(n) * time.Second
			}
		}
	}
	return 0
}

func ConditionalHeaders(etag string, lastMod time.Time) http.Header {
	h := make(http.Header)
	if etag != "" {
		h.Set("If-None-Match", `"`+etag+`"`)
	}
	if !lastMod.IsZero() {
		h.Set("If-Modified-Since", lastMod.UTC().Format(http.TimeFormat))
	}
	return h
}

func Merge(dst, src Meta) Meta {
	if src.Status != 0 {
		dst.Status = src.Status
	}
	if src.ContentType != "" {
		dst.ContentType = src.ContentType
	}
	if src.Length > 0 {
		dst.Length = src.Length
	}
	if src.ETag != "" {
		dst.ETag = src.ETag
	}
	if !src.LastModified.IsZero() {
		dst.LastModified = src.LastModified
	}
	if src.CacheControl != "" {
		dst.CacheControl = src.CacheControl
	}
	if src.Redirect != "" {
		dst.Redirect = src.Redirect
	}
	if src.FinalURL != "" {
		dst.FinalURL = src.FinalURL
	}
	return dst
}
