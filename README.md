# MiniCrawl

Go 并发爬虫 + 内存倒排索引 + 管理指挥台 + 声呐图谱可视化。

## 1. 如何启动

```bash
docker compose up --build -d
```

## 2. 使用说明

打开管理页提交种子或勾选内置 fixture，观察态势面板；在检索试验台输入关键词查看高亮结果。可视化页展示链接图谱、词频与速率。

## 3. 服务列表及API说明

| 服务 | URL |
|---|---|
| API | http://localhost:18421 |
| 管理指挥台 | http://localhost:18422 |
| 声呐可视化 | http://localhost:18423 |

接口契约见 `docs/API.md`。

## 4. 测试账号

无登录。内网演示工具。

## 5. 题目内容

使用 Go 实现高性能并发网页爬虫与搜索引擎：Worker Pool、限速器、去重队列、倒排索引、REST 全文检索高亮、后台管理与可视化页面。

## 6. 项目结构

```
backend/            Go 引擎与 API
frontend-admin/    管理指挥台
frontend-user/     可视化页
tests/             API smoke / e2e 脚本
docs/              需求与审核
```

## 7. API 模拟与切换指南

爬虫始终走真实 `net/http`。切换的是**抓取目标**：

- `CRAWL_MODE=mock`（默认）：只允许爬取进程内 fixture（`/fixture/`），QA 离线、花费 ¥0。
- `CRAWL_MODE=real`：接受用户提供的任意 http/https 种子。

在 `docker-compose.yml` 的 `backend.environment` 修改 `CRAWL_MODE` 后重建即可。管理页「使用内置 fixture 站点」对应 mock 种子 `http://127.0.0.1:8080/fixture/index.html`。
