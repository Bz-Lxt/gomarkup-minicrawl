package mocksite

import (
	"fmt"
	"net/http"
	"strings"
)

const PageCount = 120

func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "User-agent: *\nDisallow: /secret\n")
	})
	mux.HandleFunc("/secret/hidden.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><head><title>Hidden Vault</title></head><body><p>should-not-index-secret-token</p></body></html>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			fmt.Fprint(w, indexHTML())
			return
		}
		var n int
		if _, err := fmt.Sscanf(path, "page/%d.html", &n); err == nil && n >= 0 && n < PageCount {
			fmt.Fprint(w, pageHTML(n))
			return
		}
		http.NotFound(w, r)
	})
	return mux
}

func indexHTML() string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>MiniCrawl Fixture Hub 倒排索引</title></head><body>`)
	b.WriteString(`<h1>MiniCrawl 内置测试站点</h1>`)
	b.WriteString(`<p>这是用于离线验收的网页爬虫语料：inverted index、worker pool、rate limiter 与去重队列。</p>`)
	b.WriteString(`<nav><a href="secret/hidden.html">secret</a></nav>`)
	b.WriteString(`<ul>`)
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, `<li><a href="page/%d.html">页面 %d</a></li>`, i, i)
	}
	b.WriteString(`</ul></body></html>`)
	return b.String()
}

func pageHTML(n int) string {
	next := (n + 1) % PageCount
	jump := (n + 7) % PageCount
	topics := []string{
		"并发爬虫 worker pool 调度",
		"token bucket 限速器 rate limiter",
		"URL 规范化与去重队列",
		"HTML 解析提取关键词",
		"内存倒排索引 inverted index",
		"全文检索与高亮 highlight",
		"robots.txt 合规抓取",
		"图谱可视化节点与边",
	}
	topic := topics[n%len(topics)]
	return fmt.Sprintf(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>语料页 %d · %s</title></head>
<body>
<h1>语料页 %d</h1>
<p>MiniCrawl 高性能并发网页爬虫与搜索引擎。本页主题：%s。关键词：minicrawl inverted crawler 索引 检索。</p>
<p>英文段落: A concurrent web crawler builds an inverted index from HTML keywords for full-text search with highlight.</p>
<p><a href="%d.html">下一页</a> · <a href="%d.html">跳转</a> · <a href="../index.html">首页</a></p>
</body></html>`, n, topic, n, topic, next, jump)
}

func SeedPath() string {
	return "/index.html"
}
