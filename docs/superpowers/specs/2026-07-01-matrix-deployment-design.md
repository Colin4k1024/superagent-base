# Matrix Deployment Design — Go / Java / Python 三后端本地测试

**日期**: 2026-07-01  
**状态**: 已批准  
**作者**: Colin4k1024

---

## 1. 目标

在本地通过 Docker 拉起 Go / Java / Python 三套独立后端，配合三个前端实例，对同一 API 契约做功能矩阵测试，验证三套实现行为一致性。

---

## 2. 现状

| 组件 | 路径 | 框架 | 端口 | 完成度 |
|---|---|---|---|---|
| Go 后端 | `backend/` | Hertz + Eino ReAct | 8888 | 完整 |
| Python 后端 | `python/` | FastAPI + AgentScope 2.0 | 8889 | 基本完整（43 endpoints） |
| Java 后端 | `java/` | Spring Boot 3 + AgentScope | 8890 | 完整，部分 TODO |
| Python SDK | `sdks/python/` | httpx async | — | 完整，用于测试 |
| TypeScript SDK | `sdks/typescript/` | fetch | — | 完整，用于测试 |

三套后端均已有 Dockerfile，但没有统一的 docker-compose 将其联合拉起。

---

## 3. 整体拓扑

```
┌──────────────────────────────────────────────────────────────┐
│                 Docker Network: sa-matrix                     │
│                                                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ go-backend  │  │python-backend│  │java-backend │          │
│  │  :8888      │  │  :8889      │  │  :8890      │          │
│  │ Hertz/Eino  │  │ FastAPI/AS  │  │ SpringBoot  │          │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
│         └────────────────┴────────────────┘                  │
│                           │                                   │
│      ┌────────────────────┼────────────────────┐             │
│      ▼                    ▼                    ▼              │
│  mysql:3306          redis:6379          minio:9000           │
│  (3 database)        (key prefix 隔离)   (3 buckets)          │
└──────────────────────────────────────────────────────────────┘

前端三实例（本地 Vite，不进 Docker）：
  :3501  VITE_API_BASE=http://localhost:8888  → Go
  :3502  VITE_API_BASE=http://localhost:8889  → Python
  :3503  VITE_API_BASE=http://localhost:8890  → Java
```

---

## 4. 数据隔离策略

| 中间件 | 隔离方式 |
|---|---|
| MySQL | 三个独立 database：`sa_go` / `sa_python` / `sa_java` |
| Redis | key prefix 隔离：`go:` / `python:` / `java:` |
| MinIO | 三个独立 bucket：`sa-go` / `sa-python` / `sa-java` |
| 配置目录 | 共享 `configs/agents/` YAML（只读挂载） |

---

## 5. API 契约（三套后端必须实现）

基于 `/api/v2/` canonical 接口，与现有 Python SDK / TypeScript SDK 对齐：

```
# 核心 Chat
POST /api/v2/chat/stream          SSE 流式对话（X-A2UI: true）
POST /api/v2/chat/resume          Interrupt/Resume 恢复
POST /api/v2/chat/abort           中断当前对话
GET  /api/v2/chat/interrupt_state 查询中断状态

# Agent
GET  /api/v2/agents               列出所有已加载 Agent
GET  /api/v2/agents/:id/state     Agent 状态（Memory）
POST /api/v2/agents/:id/state
DELETE /api/v2/agents/:id/state/:key

# Admin（需 API Key）
GET  /api/v2/admin/status
POST /api/v2/admin/reload
GET  /api/v2/admin/logs           SSE 日志流
GET  /api/v2/admin/agents         Agent CRUD
POST /api/v2/admin/agents
GET  /api/v2/admin/agents/:name
PUT  /api/v2/admin/agents/:name
DELETE /api/v2/admin/agents/:name
POST /api/v2/admin/agents/validate

# 健康
GET  /health
GET  /ready
GET  /metrics
```

---

## 6. 交付任务

### Task 1：docker/docker-compose-matrix.yml

新增文件，覆盖：
- 共享中间件 profile：mysql / redis / minio
- `go-backend` service（build `./backend`，挂载 `configs/agents`）
- `python-backend` service（build `./python`，挂载 `configs/agents`）
- `java-backend` service（build `./java`，挂载 `configs/agents`）
- 配套 `docker/.env.matrix.example`
- Makefile target：`make matrix-up / matrix-down / matrix-clean`

### Task 2：Java TODO 补全

需修复的文件：
- `java/src/main/.../config/YamlAgentLoader.java` — 补全 YAML 解析逻辑
- `java/src/main/.../memory/RedisMemory.java` — 补全 Redis 操作
- `java/src/main/.../harness/CompactionManager.java` — 补全 context 压缩
- `java/src/main/.../harness/SkillRepository.java` — 补全技能注册
- `java/src/main/.../harness/ToolResultEviction.java` — 补全 tool result 淘汰
- `java/src/main/.../tools/McpToolWrapper.java` — 补全 MCP 工具包装
- `java/src/main/.../tools/builtin/*.java` — WebSearch / HttpRequest / CodeExecute stub 补全

### Task 3：前端多实例

- `web/.env.go`、`web/.env.python`、`web/.env.java` 三份环境变量文件
- Makefile targets：
  - `make matrix-fe-go`（port 3501）
  - `make matrix-fe-python`（port 3502）
  - `make matrix-fe-java`（port 3503）
  - `make matrix-fe`（并行启动三个）

### Task 4：功能矩阵测试套件

路径：`tests/matrix/`

测试矩阵（每个测例对三个后端各执行一次）：

| 测试项 | SDK | 验证点 |
|---|---|---|
| 列出 Agent | Python SDK | 返回非空列表，格式正确 |
| 流式对话（text 事件） | Python SDK | 收到 text 类型事件，collect() 返回字符串 |
| 流式对话（thinking 事件） | TypeScript SDK | thinking 事件正确解析 |
| 对话历史保持 | Python SDK | 同 session_id 多轮对话 |
| Admin status | Python SDK | 返回 uptime / version 字段 |
| Agent CRUD | Python SDK | create → get → update → delete |
| Interrupt/Resume | Python SDK | 触发 interrupt，resume 恢复 |
| 健康检查 | HTTP | /health 返回 200 |

输出：`tests/matrix/report.md`，对比三后端每个测例的 pass/fail/diff。

---

## 7. 启动流程

```bash
# 1. 拉起所有后端 + 中间件
make matrix-up

# 2. 等待三后端健康
make matrix-wait

# 3. 启动三个前端实例（三个终端或 tmux）
make matrix-fe

# 4. 运行功能矩阵测试
make matrix-test

# 5. 查看报告
cat tests/matrix/report.md
```

---

## 8. 端口一览

| 服务 | 端口 |
|---|---|
| Go 后端 | 8888 |
| Python 后端 | 8889 |
| Java 后端 | 8890 |
| 前端 → Go | 3501 |
| 前端 → Python | 3502 |
| 前端 → Java | 3503 |
| MySQL | 3306 |
| Redis | 6379 |
| MinIO API | 9000 |
| MinIO Console | 9001 |

---

## 9. 风险与约束

| 风险 | 缓解 |
|---|---|
| Java TODO 影响功能完整性 | Task 2 优先完成，阻塞矩阵测试 |
| agentscope 依赖版本不一致 | Java/Python 固定同一版本 |
| 端口冲突 | matrix-up 前检查端口占用 |
| MySQL schema 差异 | 三个 database 各自独立初始化 |
