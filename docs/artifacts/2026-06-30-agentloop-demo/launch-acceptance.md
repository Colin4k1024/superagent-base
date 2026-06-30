# Launch Acceptance: AgentLoop Demo Page

> 状态：accepted | 主责：qa-engineer | 日期：2026-06-30

## 验收概览

| 项目 | 内容 |
|------|------|
| 验收对象 | AgentLoop Demo 页面 (`/agentloop-demo`) |
| 验收方式 | 代码审查 + 编译验证 + 后端测试 |
| 验收角色 | qa-engineer |
| 关联 PRD | `docs/artifacts/2026-06-30-agentloop-demo/prd.md` |

## 验收范围

### 业务
- AgentLoop agent 选择、消息发送、多轮执行可视化 ✅
- Turn 边界渲染（轮次 + max_turns + 耗时） ✅
- 工具调用详情展开 ✅
- 思考过程折叠/展开 ✅

### 技术
- A2UI 事件解析（8/10 种实际使用） ✅
- 后端 EventAgent turn header → progress event ✅
- Reducer 不可变状态管理 ✅
- Preempt 机制 ✅

### 非功能
- 后端编译通过 ✅
- 后端 4 个 agentloop 测试全部通过 ✅
- 前端 HMR 无编译错误 ✅
- 页面 HTTP 200 ✅

## 验收证据

| 证据 | 位置 |
|------|------|
| 后端测试结果 | `go test ./pkg/agentdef/... -run TestAgentLoop` → PASS (4/4) |
| 前端编译 | Vite HMR 无错误日志 |
| 页面访问 | `curl http://localhost:3500/agentloop-demo` → HTTP 200 |
| Code Review | code-reviewer agent: 1 CRITICAL + 2 HIGH 已修复 |
| Security Review | security-reviewer agent: 1 CRITICAL pre-existing + 本次相关已修复/缓解 |

## 风险判断

| 已满足项 | 说明 |
|----------|------|
| AgentLoop 多轮执行可视化 | Turn 分隔线 + 状态面板 + 消息渲染 |
| A2UI 全事件渲染 | 8 种实际事件 + 2 种预留组件 |
| Preempt 中断 | 复用现有 abort 机制 + requestIdRef 防护 |
| Agent 类型过滤 | agentAdminApi + fallback |

| 可接受风险 | 说明 |
|-----------|------|
| SSE 解析重复 | P2 重构，不影响功能 |
| Admin API fallback | 开发环境可用，生产需配置 |
| rehype-raw XSS | Pre-existing，非本次引入 |

| 阻塞项 | 说明 |
|--------|------|
| 无 | — |

## 上线结论

**允许上线。**

前提条件：
1. CRITICAL 和 HIGH review 问题已修复（✅ 已完成）
2. 后端测试通过（✅ 已完成）
3. 前端无编译错误（✅ 已完成）

观察重点：
1. 首次实际 AgentLoop 多轮执行时的 progress event 渲染效果
2. Agent 过滤在生产环境的行为（admin API key 是否配置）
3. 长时间执行（15+ turns）时的 UI 性能

已接受风险：SSE 解析重复、InterruptField 类型重复、rehype-raw（pre-existing）— 均标记为 P2 后续增量。
