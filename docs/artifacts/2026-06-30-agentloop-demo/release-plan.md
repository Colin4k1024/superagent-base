# Release Plan: AgentLoop Demo Page

> 状态：released | 主责：devops-engineer | 日期：2026-06-30

## 发布信息

| 项目 | 内容 |
|------|------|
| 发布内容 | AgentLoop Demo 页面 + A2UI 事件增强 + EventAgent turn header detection |
| 发布环境 | 本地开发环境 |
| 发布方式 | 代码热更新（Vite HMR + Go 后台进程） |
| 放行依据 | launch-acceptance.md — 允许上线 |

## 变更与风险

### 后端变更（1 文件）
- `backend/pkg/agentdef/event_agent.go` — +30 行（parseTurnHeader + progress event）
- 风险：低（仅影响 A2UI 模式下的 turn header 输出格式）

### 前端变更（9 新增 + 4 修改）
- 新增：agentloop-types.ts, agentloop-reducer.ts, AgentLoopChatPage.tsx, 6 个组件
- 修改：api.ts, router.tsx, Sidebar.tsx, globals.css
- 风险：低（独立路由，不影响现有页面）

## 执行步骤

| # | 步骤 | 状态 | 验证 |
|---|------|------|------|
| 1 | 后端编译通过 | ✅ | `go build ./pkg/agentdef/...` |
| 2 | 后端测试通过 | ✅ | 4/4 agentloop tests PASS |
| 3 | 前端 HMR 无错误 | ✅ | Vite 控制台无 error |
| 4 | `/agentloop-demo` 页面可访问 | ✅ | HTTP 200 |
| 5 | 后端 API 正常 | ✅ | `/api/v2/agents` 返回 200 |
| 6 | Docker 容器健康 | ✅ | sa-mysql + sa-redis healthy |

## 验证与监控

### 已完成验证
- 后端编译 + 4 个 agentloop 单测全部通过
- 前端 Vite HMR 无编译错误
- `/agentloop-demo` 页面 HTTP 200
- MySQL + Redis 容器 healthy

### 后续观察项
- 首次实际 AgentLoop 多轮执行时 progress event 渲染效果
- agentAdminApi 在生产环境的行为（需 admin API key）
- 长时间执行（15+ turns）时 UI 性能

## 回滚方案

| 场景 | 回滚命令 | 影响 |
|------|----------|------|
| 前端异常 | `git checkout HEAD~1 -- web/src/pages/AgentLoopChatPage.tsx web/src/components/agentloop/ web/src/lib/agentloop-*.ts web/src/router.tsx web/src/styles/globals.css` + 恢复 Sidebar | AgentLoop 页面不可访问，其余功能不受影响 |
| 后端异常 | `git checkout HEAD~1 -- backend/pkg/agentdef/event_agent.go` + `make dev-server` | AgentLoop turn header 恢复为 text event |

## 放行结论

**已放行。**

所有执行步骤已完成且验证通过。变更范围小（独立页面 + 后端最小改动），回滚路径清晰。无需值守或灰度策略。
