# MiniCrawl 需求文档（Requirements）

> 版本：v1.1（冻结） 日期：2026-08-20（GMT+8）
> 变更：v1.1 增补「可视化页面」（frontend-user）——爬取图谱、词频排行、速率时间序列。
> 原始 Prompt 存档：`docs/.meta/original_prompt.md`

## 1. 项目概述

使用 **Go 语言**实现一个高性能并发网页爬虫与搜索引擎，代号 **MiniCrawl**。系统包含五大子系统：并发爬取引擎、HTML 解析与关键词提取、内存倒排索引、RESTful API + 后台管理页面、**可视化页面**。

## 2. 功能需求

### 2.1 并发爬取引擎
- **Worker Pool**：可配置 worker 数量（默认 8），任务从队列分发。
- **Rate Limiter**：限速器，支持全局速率与按域名速率限制（token bucket），可配置。
- **去重队列**：URL 规范化（去锚点、参数排序、协议/大小写归一）后去重，同一 URL 不重复入队。
- **自定义爬取策略**：最大深度、最大页面数、并发数、速率均可通过 API/配置调整；支持启动、暂停、停止任务。
- 遵守 `robots.txt`（基础解析：Disallow 规则），带自定义 User-Agent。

### 2.2 HTML 解析与关键词提取
- 解析 HTML，提取 `<title>`、正文文本（剔除 script/style/nav 等噪声标签）。
- 提取页面内链接（`<a href>`）供继续爬取。
- 关键词提取：英文按词干/停用词过滤，中文按 bigram 切分，基于词频统计产出关键词。

### 2.3 内存倒排索引（Inverted Index）
- 结构：`关键词 -> [{docID, 词频, 位置列表}]`，并发安全（读写锁）。
- 支持文档增删；索引常驻内存，**重启后丢失（需求固有特性，需在 README 注明）**。

### 2.4 RESTful API
- `POST /api/crawl/tasks`：提交爬取任务（种子 URL 列表 + 策略参数）。
- `GET /api/crawl/tasks/:id`：任务状态查询；`POST /api/crawl/tasks/:id/stop` 停止。
- `GET /api/search?q=关键词&highlight=true`：全文检索，结果带 `<mark>` 高亮、标题、URL、摘要。
- `GET /api/stats`：监控指标（已爬页面数、队列长度、索引文档数、关键词数、当前速率、worker 状态）。
- `GET /api/graph`：爬取图谱（节点=页面，边=超链接），供可视化页渲染。
- `GET /api/keywords/top`：词频 Top N，供词频图。
- `GET /api/stats/timeseries`：爬取速率时间序列。
- `GET /api/health`：健康检查。
- API 文档：独立 `docs/API.md`，含请求/响应示例、参数类型、错误码表。

### 2.5 后台管理页面（frontend-admin）
- **任务操作**：提交种子 URL、配置策略（worker 数/速率/深度/页数上限）、启动/停止任务。
- **监控面板**：实时展示爬取速率、队列深度、索引规模、任务列表与状态（轮询 `/api/stats`）。
- **搜索测试台**：输入关键词即时检索，展示高亮结果。
- 设计标准：现代化 UI（Vue3 + Tailwind），响应式，符合 Aesthetic Excellence 红线。

### 2.6 可视化页面（frontend-user）
- **爬取图谱**：力导向图，节点=页面（标题/URL/状态），边=页面间超链接；支持悬停详情与缩放拖拽。
- **词频排行**：索引内 Top N 关键词横向柱状图，点击关键词触发检索。
- **速率时间序列**：近窗爬取速率折线，随任务推进刷新。
- **检索入口**：与管理页共用搜索 API，结果高亮展示。
- 设计标准：与管理页同一设计体系，但布局以全幅画布可视化为主；响应式（768px / 480px）。

## 3. 非功能需求

- **Docker 交付**：`docker compose up` 一键启动；镜像支持 ARM64 + AMD64；服务经 localhost 可访问；容器时区 `TZ=Asia/Shanghai`。
- **日志**：统一 Logger（含 level 控制），禁止散落 `fmt.Println`；生产模式屏蔽 debug。
- **测试**：Go 单元测试覆盖核心引擎（去重、限速、索引、搜索）；API smoke 测试在 **Mock/离线模式**运行（内置本地测试站点 fixture，不访问外网），QA 预期花费 **¥0**。
- **时区**：所有时间处理使用 GMT+8。

## 4. 验收基线（可测量）

| 指标 | 基线 |
|---|---|
| 爬取吞吐 | 本地 mock 站点下 ≥ 30 页/秒（worker=8，限速关闭时） |
| 搜索延迟 | 索引 10k 文档内，P95 < 100ms |
| URL 去重 | 规范化后同一 URL 重复入队率 = 0% |
| 限速精度 | 实际速率与设定值误差 ≤ ±10% |
| 高亮正确性 | 命中词在摘要中被 `<mark>` 完整包裹 |
| 图谱完整性 | 已爬页面均出现在 `/api/graph` 节点中；页面内链形成对应边 |
| QA 成本 | 全程 Mock，¥0 |

## 5. 风险与边界说明

- 爬取外部网站须遵守目标站点 robots.txt 与使用条款；系统内置 robots 解析与限速即为合规保障，QA 一律使用本地 mock 站点。
- 内存索引不持久化，重启后需重新爬取（需求固有特性）。
- 无用户体系：管理页为单体内网工具，不实现登录鉴权（README 中注明适用边界）。

## 6. 技术栈决议

- **后端**：Go 1.22+，`net/http`（或轻量路由），`golang.org/x/net/html` 解析。
- **前端**：Vue 3 + Vite + Tailwind CSS。`frontend-admin` 管理操作；`frontend-user` 可视化检索。独立 Nginx 容器，`/api` 反代后端。
- **存储**：纯内存，无外部 DB。
- **目录结构**（SOP 强制）：`backend/`、`frontend-admin/`、`frontend-user/`。`frontend-mp` 不适用（非小程序题目，不伪造空壳）。
- **Mock 切换**：`CRAWL_MODE=mock|real`。Mock 使用后端内置 fixture 站点（真实 HTTP 爬取，目标为本地页面）；Real 接受用户种子 URL。切换方式写入 README §7。
