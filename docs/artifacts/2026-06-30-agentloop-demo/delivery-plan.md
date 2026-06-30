# Delivery Plan: AgentLoop Demo Page

> 状态：plan | 主责：tech-lead | 日期：2026-06-30
> 关联 PRD：`docs/artifacts/2026-06-30-agentloop-demo/prd.md`

## 版本目标

交付一个独立的 `/agentloop-demo` 页面，完整展示 AgentLoop 多轮自主执行全流程，支持 A2UI 协议 10 种事件渲染（实际使用 8 种，2 种预留），作为平台 AgentLoop 能力的标准参考实现。

**放行标准**：
- 用户可选择 agentloop agent、发送消息、观察多轮迭代执行
- Turn 边界以结构化分隔线渲染（轮次编号 + max_turns + 耗时）
- thinking、tool_call/tool_result、text 事件正确渲染
- preempt（中断当前执行）正常工作
- 页面无控制台错误，streaming 不卡顿

## 需求挑战会结论

| 质疑 | 来源 | 决策 |
|------|------|------|
| 目标用户是 demo 观众还是调试开发者？ | PM | 两者兼顾，demo 优先。调试能力（按轮折叠、失败高亮）P2 增量 |
| Turn 边界正则匹配是脆弱的 workaround | Architect | 采纳替代方案：EventAgent 层 pattern detection → emit progress event |
| progress/agent_switch/code_block 无发送路径 | Architect | 第一期 progress 有发送路径（turn header 改造），其余 2 种预留组件 |
| Agent 过滤缺 type 字段 | PM | 使用 `agentAdminApi.list()` 返回的 `AgentDetail.type` 字段过滤 |
| ChatInput preempt 支持？ | PM | 已验证：ChatPage 已有 preempt 逻辑，ChatInput 是 UI 壳，无阻塞 |
| 独立页面 vs 可复用组件 | Architect | 独立页面优先，组件内聚，后续可提取 |

## Brownfield 快照

### 现有模块边界

| 模块 | 文件 | 状态 |
|------|------|------|
| SSE 解析 | `web/src/lib/api.ts` chatApi.sendMessage | 需扩展（+progress/interrupt） |
| 消息渲染 | `web/src/components/chat/MessageBubble.tsx` | 不直接复用，接口不兼容 |
| ThinkingBlock | `web/src/components/chat/ThinkingBlock.tsx` | 直接复用 |
| ToolCallBlock | `web/src/components/chat/ToolCallBlock.tsx` | 直接复用 |
| MarkdownRenderer | `web/src/components/chat/MarkdownRenderer.tsx` | 直接复用 |
| ChatInput | `web/src/components/chat/ChatInput.tsx` | 直接复用 |
| 路由 | `web/src/router.tsx` | 需增加路由 |
| 侧边栏 | `web/src/components/Sidebar.tsx` | 需增加导航入口 |
| EventAgent | `backend/pkg/agentdef/event_agent.go` | 需增加 turn header pattern detection（~15 行） |

### 外部依赖

- 后端 `POST /api/v1/chat/stream` + `X-A2UI: true`（已可用）
- 后端 `POST /api/v1/chat/resume`（已可用，前端需新增调用）
- 后端 `GET /api/v1/admin/agents`（已可用，返回 AgentDetail 含 type 字段）
- `code-assistant` agent（已配置，agentloop 类型）

## Story Slices

### Slice 1：后端 EventAgent Turn Header Detection（后端最小改动）

**目标**：在 EventAgent 层检测 turn header 文本，emit `progress` event 替代 `text` event

**改动范围**：
- `backend/pkg/agentdef/event_agent.go` — `eventAgentWrapper.ChatWithEvents()` 增加 ~15 行 pattern detection

**验收标准**：
- AgentLoop 进入 Turn 2+ 时，SSE 流中出现 `event: progress` 而非 `event: text`（含 turn header 内容）
- progress event data: `{agent_name, step: "turn", total: N, current: M}`
- 非 agentloop agent 的 text 流不受影响
- 现有 agentloop 单测继续通过

**Owner**：frontend-engineer（后端改动极小，前端工程师可完成）
**依赖**：无

### Slice 2：A2UI 事件解析扩展 + 类型系统（前端数据层）

**目标**：扩展 api.ts 解析 progress/interrupt 事件，定义 AgentLoop 数据类型

**改动范围**：
- `web/src/lib/api.ts` — ChatStreamCallbacks 增加 `onProgress`/`onInterrupt`，sendMessage 解析新事件
- `web/src/lib/agentloop-types.ts` — 新增，AgentLoopMessage/Turn 类型定义
- `web/src/lib/agentloop-reducer.ts` — 新增，状态 reducer

**验收标准**：
- `onProgress` 回调正确接收 `{agent_name, step, total, current}` 数据
- `onInterrupt` 回调正确接收 `{reason, fields}` 数据
- reducer 正确将 SSE 事件归组到 AgentLoopMessage → AgentLoopTurn 结构
- tool_call/tool_result 按 id 匹配

**Owner**：frontend-engineer
**依赖**：Slice 1（progress event 格式需确认）

### Slice 3：AgentLoop 页面骨架 + 路由（前端页面层）

**目标**：创建页面容器、路由、sidebar 入口、agent 选择器

**改动范围**：
- `web/src/pages/AgentLoopChatPage.tsx` — 新增
- `web/src/router.tsx` — 增加 `/agentloop-demo` 路由
- `web/src/components/Sidebar.tsx` — 增加导航项

**验收标准**：
- 访问 `/agentloop-demo` 显示页面
- Agent 下拉只显示 agentloop 类型（通过 agentAdminApi.list + type 过滤）
- 选择 agent 后 Session ID 自动生成
- Sidebar 显示 "AgentLoop Demo" 导航入口

**Owner**：frontend-engineer
**依赖**：无（可与 Slice 2 并行）

### Slice 4：Turn 分隔线 + 执行状态面板（前端组件层）

**目标**：渲染 turn 边界和执行状态

**改动范围**：
- `web/src/components/agentloop/TurnSeparator.tsx` — 新增
- `web/src/components/agentloop/ExecutionStatusPanel.tsx` — 新增
- `web/src/components/agentloop/index.ts` — barrel export

**验收标准**：
- Turn 分隔线显示 "Turn 2/15" + 本轮耗时
- 状态面板显示当前轮次、max_turns、总耗时、执行状态 badge
- 状态颜色：running=blue, done=green, error=red, preempted=amber
- 进入动画 fade-in + slide-down (200ms)

**Owner**：frontend-engineer
**依赖**：Slice 2（依赖 progress 事件数据结构）

### Slice 5：消息渲染 + 工具调用 + 思考过程（前端组件层）

**目标**：渲染 AgentLoop 消息的完整内容

**改动范围**：
- `web/src/components/agentloop/AgentLoopMessageList.tsx` — 新增
- `web/src/components/agentloop/AgentLoopMessageItem.tsx` — 新增

**验收标准**：
- 用户消息以气泡形式渲染
- 每个 Turn 内的 thinking 以 ThinkingBlock 渲染（复用）
- 每个 Turn 内的 tool_call 以 ToolCallBlock 渲染（复用）
- Turn 内文本以 MarkdownRenderer 渲染（复用）
- 流式 cursor 正确显示

**Owner**：frontend-engineer
**依赖**：Slice 2（依赖 AgentLoopMessage 类型）、Slice 4（依赖 TurnSeparator）

### Slice 6：中断表单 + Preempt + Resume（前端交互层）

**目标**：支持 interrupt 表单和 preempt 机制

**改动范围**：
- `web/src/components/agentloop/InterruptForm.tsx` — 新增
- `web/src/lib/api.ts` — 增加 `chatApi.resume()` 方法
- `web/src/pages/AgentLoopChatPage.tsx` — preempt 逻辑

**验收标准**：
- interrupt 事件渲染确认/输入表单
- 表单提交后调用 `/api/v1/chat/resume` 恢复执行
- 发送新消息时，正在执行的流被 preempt（abort + 标记 preempted）
- 被 preempt 的消息显示中断标记

**Owner**：frontend-engineer
**依赖**：Slice 2（onInterrupt 回调）、Slice 3（页面骨架）

### Slice 7：集成验证 + 边界态 + E2E（QA）

**目标**：端到端验证完整流程

**验收标准**：
- 完整 AgentLoop 执行流程：发送 → 多轮执行 → 工具调用 → 完成
- 中断流程：执行中发送新消息 → preempt → 新响应
- 边界态：agent 返回错误、空响应、超长响应
- 控制台无错误，streaming 不卡顿
- 响应式：桌面 + 平板布局正常

**Owner**：qa-engineer
**依赖**：Slice 1-6 全部完成

## 执行波次

```
Wave 1 (并行):
  ├── Slice 1: EventAgent turn header detection (后端)
  └── Slice 3: 页面骨架 + 路由 (前端)

Wave 2 (依赖 Wave 1):
  └── Slice 2: A2UI 解析扩展 + 类型系统 (前端)

Wave 3 (并行, 依赖 Wave 2):
  ├── Slice 4: Turn 分隔线 + 状态面板 (前端)
  └── Slice 5: 消息渲染组件 (前端)

Wave 4 (依赖 Wave 3):
  └── Slice 6: Interrupt + Preempt + Resume (前端)

Wave 5 (依赖全部):
  └── Slice 7: 集成验证 (QA)
```

## 角色分工

| 角色 | 职责 | Slices |
|------|------|--------|
| tech-lead | 方案仲裁、进度跟踪 | 全局 |
| frontend-engineer | 页面实现、组件开发 | 1-6 |
| architect | 组件拆分评审、代码 review | Review |
| qa-engineer | E2E 验证、边界态测试 | 7 |

## 风险与缓解

| 风险 | 影响 | 缓解 | Owner |
|------|------|------|-------|
| EventAgent turn header pattern 误匹配 | LLM 输出包含相同格式文本 | 使用更唯一的 pattern（含 agentloop 内部标记） | frontend-eng |
| interrupt/resume 联调耗时 | Slice 6 可能延期 | interrupt 作为 P2，第一期可只做 preempt | tech-lead |
| AgentLoop 长时间执行时 UX | 用户感觉卡死 | StatusPanel 实时显示轮次和耗时 | frontend-eng |
| 多 turn 状态归组复杂度 | reducer 逻辑出错 | 充分单测，tool_call/tool_result 按 id 匹配 | frontend-eng |

## 门禁状态

- Pre-flight: ✅ intake 完成、PRD 存在、挑战会已收口
- Revision: 0 项
- Escalation: 0 项
- Abort: ✅ 无阻塞

## 技能装配清单

| 技能 | 触发原因 | 主责角色 |
|------|----------|----------|
| `frontend-engineering` | React 组件设计规范 | frontend-engineer |
| `a2ui-streaming` | A2UI SSE 协议参考 | frontend-engineer |

## 是否需要 ADR

否。本任务不涉及架构偏离、组件选型变更或协议变更。EventAgent 层的 turn header pattern detection 是实现细节，不改变 A2UI 协议定义。
