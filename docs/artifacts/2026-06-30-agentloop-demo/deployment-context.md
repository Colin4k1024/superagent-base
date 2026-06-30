# Deployment Context: AgentLoop Demo Page

> 状态：released | 主责：devops-engineer | 日期：2026-06-30

## 环境清单

| 环境 | 用途 | 访问入口 | 部署目标 |
|------|------|----------|----------|
| 本地开发 | 功能验证 | http://localhost:3500 (前端) / http://localhost:8888 (API) | Docker Compose + Go run + Vite dev |

## 部署入口

| 入口 | 命令 | 前置条件 |
|------|------|----------|
| 全量启动 | `make dev` | Docker Desktop 运行中 |
| 仅中间件 | `make dev-middleware` | Docker Desktop 运行中 |
| 仅后端 | `make dev-server` | MySQL + Redis 容器运行中 |
| 前端 | `cd web && npm run dev` | 后端运行中 |

## 配置与密钥

| 项目 | 来源 | 说明 |
|------|------|------|
| 数据库连接 | `backend/.env` | MySQL localhost:3306 |
| Redis 连接 | `backend/.env` | Redis localhost:6379 |
| 模型 API Key | `backend/.env` | 需配置 LLM provider key |
| 前端代理 | `web/vite.config.ts` | /api → :8888, /grpc → :50051 |

## 运行保障

- **回滚路径**：`git checkout main` 回退到 `c718f02`
- **监控**：后端日志输出到 stdout（`make dev-server` 终端）
- **观察窗口**：开发环境持续观察

## 恢复能力

| 触发条件 | 回滚路径 | 验证方法 |
|----------|----------|----------|
| 前端编译失败 | `git stash` 或 `git checkout -- web/` | `npm run dev` 无错误 |
| 后端编译失败 | `git stash` 或 `git checkout -- backend/` | `go build ./...` 通过 |
| AgentLoop 页面异常 | 回退 3 个新增文件 + 路由 | `/agentloop-demo` 404 或重定向 |
