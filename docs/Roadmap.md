# MiniCrawl 路线图

> 版本：v1.1  日期：2026-08-20（GMT+8）

## 阶段顺序决策

**Logic-First（交换 Phase 2 / Phase 3）**

可视化页的力导向图谱、词频柱、速率折线均由「文档节点 / 超链接边 / 倒排词项 / 时序采样」数据模型派生，先实现引擎与 API 契约，UI 才能按真实 schema 绘制，而不是对着假数据空画。

## 规模与边界

预估 4,000–7,000 LoC，低于 10k，一次交付 MVP=V1。无 V2 分期。

不实现：微信小程序（`frontend-mp` 不适用）、用户登录、索引落盘。

## 任务清单

### P1 架构骨架
- [x] Git 初始化与 `.gitignore`
- [x] `docker-compose.yml` 随机端口（18421/18422/18423）
- [x] 目录：`backend/`、`frontend-admin/`、`frontend-user/`

### P3 后端引擎（先于 UI）
- [x] Worker Pool + Dedup Queue + Token Bucket 限速（全局/按域）
- [x] robots.txt 基础解析
- [x] HTML 解析、中英关键词、内存倒排索引、高亮检索
- [x] 图谱 / 词频 / 时序 API
- [x] 内置 fixture 站点 + `CRAWL_MODE` 切换
- [x] 统一 slog Logger、GMT+8、Dockerfile（多阶段，ARM64/AMD64）
- [x] 单元测试：去重、限速、索引、搜索高亮、爬取

### P2 前端
- [x] `docs/DesignSpec.md`（雷达指挥室美学）
- [x] 管理页：任务 CRUD、监控、搜索台
- [x] 可视化页：图谱画布、词频、时序、检索

### P4 QA
- [x] `tests/api_smoke.py` + `tests/e2e_flow.spec.ts`（Mock，¥0）
- [x] Docker 内执行，写入 `docs/QA_Record.md`

### P5 审计与知识回收
- [x] `docs/AuditReport.md`
- [x] `/learn` 入库
