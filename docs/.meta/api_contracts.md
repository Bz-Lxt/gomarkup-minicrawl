# 外部 API 契约

本项目不集成任何第三方计量 API（无 LLM / 支付 / 短信）。

爬虫使用通用 `net/http` 抓取用户指定 URL，或在 `CRAWL_MODE=mock` 下抓取进程内 fixture 站点。

| Provider | 用途 | 契约状态 |
|---|---|---|
| 无 | — | N/A |

Contract Gate：跳过（无外部 provider 可探测）。
