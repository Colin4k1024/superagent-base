# Arch Design: AgentLoop Demo Page

> 状态：plan | 主责：architect | 日期：2026-06-30
> 关联 PRD：`docs/artifacts/2026-06-30-agentloop-demo/prd.md`
> 关联 Plan：`docs/artifacts/2026-06-30-agentloop-demo/delivery-plan.md`

## 系统边界

### 前端（本次变更范围）

```
Router (/agentloop-demo)
  └── AgentLoopChatPage (页面主组件)
        ├── Header (AgentSelector + SessionID + StatusBadge)
        ├── ExecutionStatusPanel (轮次/耗时/状态)
        ├── AgentLoopMessageList
        │     └── AgentLoopMessageItem (per message)
        │           ├── UserBubble
        │           └── per Turn:
        │                 ├── TurnSeparator
        │                 ├── ThinkingBlock (复用)
        │                 ├── ToolCallBlock (复用)
        │                 ├── MarkdownRenderer (复用)
        │                 └── InterruptForm (新)
        └── ChatInput (复用)
```

### 后端（最小改动）

```
EventAgent.ChatWithEvents()
  └── 新增: turn header pattern detection (~15 行)
        检测 "--- Turn N/M ---" → emit progress event
        替代原来的 text event
```

### 不变部分

- `AgentLoopAgent.Chat()` — 不改
- A2UI 协议定义 — 不改
- `POST /api/v1/chat/stream` — 不改
- `POST /api/v1/chat/resume` — 不改
- SSE 编码器 `EncodeSSE()` — 不改

## 组件拆分

### 新增组件

| 组件 | 文件 | 职责 | 状态 |
|------|------|------|------|
| AgentLoopChatPage | `pages/AgentLoopChatPage.tsx` | 页面容器、SSE 状态机、消息编排 | 新增 |
| AgentLoopMessageList | `components/agentloop/AgentLoopMessageList.tsx` | 消息列表滚动容器 | 新增 |
| AgentLoopMessageItem | `components/agentloop/AgentLoopMessageItem.tsx` | 单条消息渲染（多 turn） | 新增 |
| TurnSeparator | `components/agentloop/TurnSeparator.tsx` | Turn 分隔线（轮次 + 耗时） | 新增 |
| ExecutionStatusPanel | `components/agentloop/ExecutionStatusPanel.tsx` | 执行状态面板 | 新增 |
| InterruptForm | `components/agentloop/InterruptForm.tsx` | 中断确认/输入表单 | 新增 |

### 复用组件（不修改）

| 组件 | 来源 |
|------|------|
| ThinkingBlock | `components/chat/ThinkingBlock.tsx` |
| ToolCallBlock | `components/chat/ToolCallBlock.tsx` |
| MarkdownRenderer | `components/chat/MarkdownRenderer.tsx` |
| ChatInput | `components/chat/ChatInput.tsx` |

### 类型系统

| 文件 | 内容 |
|------|------|
| `lib/agentloop-types.ts` | AgentLoopMessage, AgentLoopTurn, TurnEventBlock, InterruptField |
| `lib/agentloop-reducer.ts` | useReducer: SSE event → AgentLoopMessage[] 归组逻辑 |

## 数据流

```
POST /api/v1/chat/stream (X-A2UI: true)
  │
  ▼ SSE events
chatApi.sendMessage() — 扩展 onProgress / onInterrupt
  │
  ▼ callbacks
AgentLoopChatPage (useReducer)
  │  dispatch({ type: 'TURN_START', turn, total })
  │  dispatch({ type: 'TEXT_DELTA', delta })
  │  dispatch({ type: 'THINKING_DELTA', delta })
  │  dispatch({ type: 'TOOL_CALL', id, name, args })
  │  dispatch({ type: 'TOOL_RESULT', id, name, result })
  │  dispatch({ type: 'DONE' })
  │
  ▼ AgentLoopMessage[]
AgentLoopMessageList → AgentLoopMessageItem → 子组件
```

## A2UI 事件映射

| 事件 | 发送方 | 前端处理 | 渲染组件 |
|------|--------|----------|----------|
| `text` | EventAgent (每 token) | append 到 currentTurn.text | MarkdownRenderer |
| `thinking` | A2UICallback | append 到 currentTurn.thinking | ThinkingBlock |
| `tool_call` | A2UICallback.OnStart | push 到 currentTurn.toolCalls | ToolCallBlock |
| `tool_result` | A2UICallback.OnEnd | 匹配 id 更新 toolCall | ToolCallBlock |
| `progress` | EventAgent (turn header) | 创建新 Turn | TurnSeparator + StatusPanel |
| `interrupt` | EventAgentWrapper | 渲染动态表单 | InterruptForm |
| `error` | EventAgent | message.events push | 行内 ErrorBanner |
| `done` | EventAgent (channel close) | status='done' | StatusPanel badge |
| `preempted` | ChatSSEHandler | status='preempted' | StatusPanel badge |
| `code_block` | 无发送方 | 预留 | — |
| `agent_switch` | 无发送方 | 预留 | — |

## 接口约定

### AgentLoopMessage

```ts
interface AgentLoopMessage {
  id: string
  role: 'user' | 'assistant'
  content?: string              // user message text
  turns: AgentLoopTurn[]        // assistant: per-turn data
  status: 'streaming' | 'done' | 'preempted' | 'error'
}

interface AgentLoopTurn {
  turn: number
  total: number
  text: string
  thinking: string
  toolCalls: ToolCallInfo[]
  startTime: number             // turn 开始时间戳
  endTime?: number              // turn 结束时间戳
  isStreaming: boolean
}
```

### ChatStreamCallbacks 扩展

```ts
// 新增:
onProgress?: (data: { agent_name?: string; step?: string; total: number; current: number }) => void
onInterrupt?: (data: { reason: string; fields: InterruptField[] }) => void
```

### chatApi 新增

```ts
resume: (agentId: string, sessionId: string, input: string) => Promise<Response>
```

## 技术选型

| 决策 | 选择 | 原因 |
|------|------|------|
| 状态管理 | useReducer | 与 ChatPage 一致，不引入 Zustand/Redux |
| 路由 | 独立 `/agentloop-demo` | 便于示例定位，不污染 ChatPage |
| Turn 边界 | EventAgent progress event | 比前端正则更可控 |
| Agent 过滤 | agentAdminApi.list() + type | 已有 API 返回 type 字段 |
| 消息类型 | AgentLoopMessage（独立） | 不污染 ChatMessage 接口 |

## 风险与约束

| 风险 | 等级 | 缓解 |
|------|------|------|
| EventAgent turn header 误匹配 | 中 | 使用唯一 pattern + 降级为普通 text |
| interrupt/resume 联调 | 中 | interrupt 作为 P2，preempt 优先 |
| reducer 跨 turn 归组复杂度 | 中 | tool_call/tool_result 严格按 id 匹配 |
| 长时间执行 UX | 低 | StatusPanel 实时反馈 |
