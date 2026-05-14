# Superagent Base — 后续迭代计划

> 更新时间：2026-05-14
> 当前状态：平台基座全面就绪，以下为增量优化项

---

## P1 — 短期优先（1-2 周内）

### 1. RBAC 接入中间件
- **现状**：`pkg/rbac/` 已实现权限模型 + UserStore，但未接入 HTTP/gRPC 中间件
- **目标**：
  - 创建 `middleware/rbac_middleware.go`，从请求头解析 API Key → 查 UserStore → 注入 User 到 ctx
  - Admin 端点要求 `admin` 角色
  - Agent 编辑端点要求 `editor+` 角色
  - Chat 端点要求 `viewer+` 角色
  - 添加 User CRUD admin API (`/api/v1/admin/users`)
- **依赖**：无

### 2. 前端剩余页面 i18n 迁移
- **现状**：LoginPage + AgentsPage + Sidebar 已迁移，其余页面仍硬编码
- **目标**：迁移以下页面到 `t()` 调用：
  - `AgentEditPage.tsx`
  - `WorkflowEditorPage.tsx`
  - `SkillsPage.tsx`
  - `SettingsPage.tsx`
  - `ChatPage.tsx`
  - `MonitorPage.tsx` + 子组件 (StatusPanel, MetricsPanel, LogsPanel, AdminPanel)
- **依赖**：翻译 key 已在 `zh.json`/`en.json` 中定义

### 3. Playwright CI 集成
- **现状**：E2E spec 文件已写好，但未配 CI 流程
- **目标**：
  - 在 GitHub Actions 中添加 `playwright.yml` workflow
  - 步骤：install browsers → start backend (docker compose) → start frontend → run tests
  - 上传失败截图为 artifact
- **依赖**：需要 CI runner 环境有 Docker

### 4. MCP Server 管理 HTTP API
- **现状**：`pkg/mcp/registry.go` 有 Connect/Disconnect/ListServers 方法，但无 HTTP 端点
- **目标**：
  - 创建 `mcp_admin_handler.go`
  - `GET /api/v1/admin/mcp/servers` → 列出已连接服务器
  - `POST /api/v1/admin/mcp/servers` → 动态连接新服务器
  - `DELETE /api/v1/admin/mcp/servers/:name` → 断开
  - `GET /api/v1/admin/mcp/servers/:name/tools` → 列出该服务器暴露的工具
  - 前端 Settings MCP tab 对接真实 API（替换当前"coming soon"占位）
- **依赖**：无

---

## P2 — 中期增强（2-4 周）

### 5. 前端暗色模式
- **现状**：仅亮色主题
- **目标**：
  - Tailwind `darkMode: 'class'` 配置
  - 全局主题切换 (localStorage 持久化)
  - 所有组件添加 `dark:` 变体
  - Monaco editor 切换 `vs-dark` 主题
- **工作量**：中

### 6. Agent 版本管理
- **现状**：YAML 文件覆盖式更新，无历史
- **目标**：
  - 每次保存 agent YAML 时，将旧版本备份到 `configs/agents/.history/<name>/<timestamp>.yaml`
  - 提供 API: `GET /api/v1/admin/agents/:name/versions` 列出历史版本
  - 前端编辑器添加"版本历史"面板，支持 diff 对比和回滚
- **工作量**：中

### 7. 后端测试覆盖提升
- **现状**：`pkg/` 核心包有测试，但 `application/`、`domain/`、`api/handler/` 基本无覆盖
- **目标**：
  - `api/handler/coze/agent_admin_handler_test.go` — 测试 CRUD 端点
  - `application/conversation/` — 测试会话创建/消息列表
  - 整体 pkg 覆盖率 > 60%
- **工作量**：大

### 8. Workflow 执行可视化
- **现状**：图编辑器可设计 workflow，但运行时无可视化反馈
- **目标**：
  - 在 A2UI `progress` 事件中携带当前执行 node_id
  - 前端图编辑器 "运行" 模式：实时高亮正在执行的节点
  - 每个节点展示耗时和输出摘要
- **依赖**：需后端在 workflow 执行器中发射 progress 事件

### 9. 代码分割 + 懒加载路由
- **现状**：manualChunks 已拆分 vendor/monaco/xyflow，但所有路由仍同步加载
- **目标**：
  - 使用 `React.lazy()` + `Suspense` 对大页面延迟加载
  - Monaco editor 页面 / Workflow 页面 按需加载
  - 首屏 JS < 200KB
- **工作量**：小

---

## P3 — 长期演进（1-3 月）

### 10. 多租户 Workspace
- **现状**：单租户，所有 agent 共享一个目录
- **目标**：
  - 引入 Workspace 概念（一组 agents + models + skills 的隔离空间）
  - 数据库存储 workspace 元数据
  - API 添加 workspace context
  - 前端 workspace 切换器
- **工作量**：大

### 11. SSO 集成
- **现状**：API Key 单因素认证
- **目标**：
  - 支持 OAuth2 / OIDC (Google, GitHub, 企业 AD)
  - JWT token + refresh token 流程
  - 前端 OAuth 回调页面
- **工作量**：大

### 12. Agent 市场 / 模板库
- **现状**：4 个示例 YAML 文件
- **目标**：
  - 社区 Agent 模板仓库（类似 Docker Hub）
  - 一键导入模板 → 本地 agent
  - 用户发布自己的 agent 模板
- **工作量**：大

### 13. 计费 / 用量追踪
- **现状**：Prometheus 记录 token 消耗但无计费逻辑
- **目标**：
  - 按 workspace/user 聚合 token 消耗
  - 配额系统（月度 token 上限）
  - 用量仪表盘
- **工作量**：大

### 14. 水平扩展
- **现状**：单实例 Hertz server + gRPC
- **目标**：
  - Session affinity (sticky sessions for streaming)
  - 分布式 checkpoint store (Redis cluster)
  - Agent runtime 多副本同步（leader election + YAML 分发）
  - Helm HPA 配置验证
- **工作量**：大

---

## 技术债

| 项目 | 说明 | 优先级 |
|------|------|--------|
| Chunk size warning | Monaco + xyflow 应按需 import，减少初始加载 | 低 |
| `go.sum` 定期清理 | 每月 `go mod tidy` | 低 |
| E2E 测试 screenshot 文件 | 已在 .gitignore，但 reports/ 积累需定期清理 | 低 |
| 前端 package-lock.json 体积 | 6MB+，考虑迁移到 pnpm | 低 |
| gRPC CreateAgent/UpdateAgent/DeleteAgent | Proto 使用 int64 ID，不适合 YAML 模式，考虑废弃或改 proto | 中 |
| MonitorPage 组件 SSE 重连 | 存在偶发的重连风暴（快速 open/close），需 debounce | 中 |
| web_search DuckDuckGo 限流 | 高频调用会被 block，需添加缓存层或切换到付费 API | 中 |

---

## 完成标记

以下为本次开发周期（2026-05-14）已完成的全部事项：

- [x] Eino DevOps IDE 集成 + eino_graph agent 类型
- [x] 开发环境稳定性（MySQL 8.4 兼容、MinIO noop fallback、Docker auto-start）
- [x] gRPC 全服务实现（Tool/Model/Conversation/Agent）
- [x] 工具系统实现（web_search DuckDuckGo/Serper、code_execute CodeRunner）
- [x] Admin API 鉴权 + IP 速率限制
- [x] 前端基础设施（React Query + Zustand + shadcn-style UI）
- [x] Agent CRUD 编辑器（Monaco YAML + 表单双向同步）
- [x] Workflow 图编辑器（React Flow + 5 节点 + 序列化器）
- [x] Skills 市场对接真实 API
- [x] Settings 模型管理持久化
- [x] 前端认证（API Key + 路由守卫）
- [x] 前端测试（Vitest 31 用例）
- [x] 后端测试补充（+20 用例，graphs/observe/modelrouter/tool）
- [x] Model Router 指标采集（Prometheus + TTFT）
- [x] ErrorBoundary + 代码分割 + 响应式侧边栏
- [x] README 全面更新
- [x] i18n 国际化（中/英双语）
- [x] Playwright E2E 测试框架 (13 specs)
- [x] RBAC 权限模型基础包
