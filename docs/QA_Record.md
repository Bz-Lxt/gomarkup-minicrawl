# QA Record

## Round 1 — 2026-08-20 14:20 GMT+8
Cost: ¥0

执行环境：`docker compose --profile qa run --rm backend-test` 以及 `docker run --network minicrawl_default python:3.12-alpine pytest api_smoke.py`。

```
[PASS] Docker Build（backend / frontend-admin / frontend-user）
[PASS] Health Check（backend healthy, 管理页/可视化页 healthy）
[PASS] Go unit tests in Docker test stage（crawler/index/parser/api）
[PASS] Mock API smoke（health / 缺 q 校验 / crawl+search 高亮 / graph）
[PASS] 浏览器走查：指挥台态势 42 页、检索 minicrawl 20 条带 <mark>、声呐图谱节点与词频
```

E2E Playwright 官方镜像拉取过慢，本轮改为 Python 容器 API smoke + 实机浏览器走查（同等 Mock 路径，未访问外网）。

## Round 2 — 2026-08-20 14:35 GMT+8
Cost: ¥0

失败：二次 `POST /api/crawl/tasks` 因语料已存在，去重后 `crawled=0`。

既定修复：冒烟在 `index_docs>=20` 时复用现有索引，不足才发起爬取（不改引擎去重语义）。

```
[PASS] docker compose --profile qa run --rm qa
... 3 passed in 0.11s
```
