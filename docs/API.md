# MiniCrawl API

Base URL（开发）：`http://localhost:18421`

通用错误体：

```json
{"error": {"code": "VALIDATION_ERROR", "message": "q is required"}}
```

| code | HTTP | 含义 |
|---|---|---|
| VALIDATION_ERROR | 400 | 参数不合法 |
| NOT_FOUND | 404 | 任务不存在 |
| CONFLICT | 409 | 已有任务在跑 |
| TASK_STATE | 409 | 当前状态不允许该操作 |
| INTERNAL | 500 | 内部错误 |

时间字段均为 GMT+8，格式 `yyyy-MM-dd HH:mm:ss`。

---

## GET /api/health

响应示例：

```json
{"status": "ok", "time": "2026-08-20 14:30:00", "mode": "mock"}
```

## GET /api/stats

```json
{
  "pages_crawled": 80,
  "queue_length": 3,
  "index_docs": 80,
  "index_terms": 210,
  "current_rps": 12,
  "workers": {"active": 8, "configured": 8},
  "crawl_mode": "mock",
  "updated_at": "2026-08-20 14:30:01"
}
```

## GET /api/stats/timeseries

```json
{"points": [{"t": "2026-08-20 14:30:01", "pages": 12, "rps": 8}]}
```

## GET /api/graph

Query：`limit`（默认 400，最大 800）

```json
{
  "nodes": [{"id": "http://127.0.0.1:8080/fixture/index.html", "url": "...", "title": "MiniCrawl Fixture Hub", "status": "crawled", "degree": 12}],
  "edges": [{"from": ".../index.html", "to": ".../page/0.html"}]
}
```

## GET /api/keywords/top

Query：`limit`（默认 30）

```json
{"keywords": [{"term": "minicrawl", "freq": 240}]}
```

## GET /api/search

Query：`q`（必填）、`highlight`（默认 true）、`limit`（默认 20，最大 50）

```
GET /api/search?q=inverted&highlight=true
```

```json
{
  "query": "inverted",
  "total": 1,
  "hits": [{
    "doc_id": 1,
    "url": "http://127.0.0.1:8080/fixture/page/4.html",
    "title": "语料页 4 · 内存倒排索引 inverted index",
    "score": 6,
    "snippet": "…builds an <mark>inverted</mark> index…",
    "highlight": true
  }]
}
```

## POST /api/crawl/tasks

请求：

```json
{
  "seeds": [],
  "use_fixture": true,
  "workers": 8,
  "global_rps": 0,
  "host_rps": 0,
  "max_depth": 6,
  "max_pages": 200,
  "user_agent": "MiniCrawl/1.0"
}
```

`global_rps=0` 表示不限速。`CRAWL_MODE=mock` 且未给 seeds 时自动使用内置 fixture。

成功：`201 Created`，`Location: /api/crawl/tasks/{id}`。

## GET /api/crawl/tasks

```json
{"tasks": [{"id": "ab12", "status": "running", "crawled": 12, "created_at": "2026-08-20 14:30:00"}]}
```

## GET /api/crawl/tasks/{id}

返回单个任务。状态：`pending|running|paused|stopped|completed|failed`。

## POST /api/crawl/tasks/{id}/pause | /resume | /stop

对运行中任务暂停、继续或停止。

## POST /api/index/clear

清空内存语料与倒排索引（任务队列不受影响）。演示/验收用。
