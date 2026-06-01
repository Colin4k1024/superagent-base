# TurnLoop — 会话抢占与中止

TurnLoop 为每个会话的 turn（轮次）生命周期提供精确控制，实现两种核心能力：

| 能力 | 描述 |
|------|------|
| **Preempt（抢占）** | 用户在 Agent 仍在生成时发送新消息，当前 turn 被立即取消，Agent 开始回答新问题 |
| **Abort（中止）** | 用户主动点击停止，Agent 立即停止当前 turn |

这与 Eino 框架的 `adk.TurnLoop` 设计理念一致，针对项目自有 Agent 接口（`Chat(ctx, sessionID, message)`）实现了等价语义。

---

## 工作原理

```
用户发送消息 A
  │
  ▼ SessionLoop.StartTurn("s1", ctx)
  │  └─ 若当前有 turn 在运行 → 调用 cancel() 抢占
  │
  ▼ Agent.Chat(turnCtx, "s1", "消息 A") ── 流式输出 token
  │
  │   ← 用户在此刻发送消息 B（抢占）
  │       │
  │       ▼ StartTurn("s1", ctx2)
  │         └─ 调用旧 cancel() → turnCtx 被取消
  │
  ▼ Agent 检测到 ctx.Done() → 停止生成，关闭 channel
  │
  ▼ 服务端：turnCtx.Err() != nil && ctx.Err() == nil
  │         → 向客户端发送 SSE preempted 事件
  │
  ▼ 新 turn 开始，流式输出消息 B 的回答
```

### 中止（Abort）流程

```
用户点击停止
  │
  ▼ POST /api/v2/chat/abort  { session_id: "s1" }
  │
  ▼ SessionLoop.Abort("s1")
  │  └─ 调用 cancel() → turnCtx 被取消
  │
  ▼ Agent 停止生成，SSE 流关闭
```

---

## 核心组件

### SessionLoop（`pkg/agentdef/session_loop.go`）

```go
// SessionLoop 管理每个 session 的 turn 生命周期。
type SessionLoop struct {
    sessions sync.Map // sessionID → *turnEntry
}

// StartTurn 开启新 turn；若有活跃 turn 则先抢占（cancel）它。
// 返回的 context 同时在以下两种情况被取消：
//   - parent（HTTP 请求 ctx）取消：客户端断连
//   - Abort() 或新 StartTurn() 调用：显式中止或抢占
func (s *SessionLoop) StartTurn(sessionID string, parent context.Context) context.Context

// EndTurn 标记 turn 正常结束，释放 cancel func。
func (s *SessionLoop) EndTurn(sessionID string)

// Abort 立即取消 session 的活跃 turn。
// 返回 true 表示有 turn 被取消，false 表示无活跃 turn。
func (s *SessionLoop) Abort(sessionID string) bool
```

### 抢占检测逻辑

```go
// turn 结束后判断原因：
if turnCtx.Err() != nil && ctx.Err() == nil {
    // turnCtx 被取消（抢占/中止），但 HTTP 连接仍然存活
    // → 向客户端发送 preempted 事件
}
```

| 条件 | 含义 |
|------|------|
| `turnCtx.Err() == nil` | 正常完成 → 发送 `[DONE]` / `done` 事件 |
| `turnCtx.Err() != nil && ctx.Err() == nil` | 被抢占或中止，客户端仍连接 → 发送 `preempted` 事件 |
| `ctx.Err() != nil` | 客户端断连 → 静默结束，无需发送事件 |

---

## API 端点

### 发送消息（含自动抢占）

```bash
POST /api/v2/chat/stream
Content-Type: application/json
X-A2UI: true

{
  "agent_id": "research-agent",
  "session_id": "s1",
  "message": "新问题"
}
```

同 session 若有活跃 turn，自动触发抢占。

### 中止当前 Turn

```bash
POST /api/v2/chat/abort
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "s1"
}
```

**响应：**

```json
{ "aborted": true }
```

`aborted: false` 表示该 session 当前无活跃 turn。

---

## SSE 事件

**A2UI 模式**下新增 `preempted` 事件类型：

```
event: preempted
data: {"type":"preempted"}
```

客户端收到此事件时，表示当前流被服务端取消（抢占或中止）。

完整事件类型一览：

| event | 触发时机 |
|-------|---------|
| `text` | 流式文本 token |
| `thinking` | 思考过程 |
| `tool_call` | 工具调用开始 |
| `tool_result` | 工具调用结果 |
| `done` | turn 正常完成 |
| `preempted` | turn 被抢占或中止（新增） |
| `error` | 发生错误 |

---

## 前端集成

### chatApi 使用示例

```typescript
import { chatApi } from './lib/api'

// 发送消息（自动支持抢占：新消息会覆盖旧 turn）
const controller = chatApi.sendMessage(agentId, sessionId, text, {
  onToken: (token) => { /* 追加文本 */ },
  onDone: () => { setIsLoading(false) },
  onPreempted: () => {
    // 服务端确认 turn 已中止
    setIsLoading(false)
  },
  onError: (err) => { console.error(err) },
})

// 中止当前 turn（用于停止按钮）
await chatApi.abort(agentId, sessionId)
controller.abort() // 同时关闭 SSE 连接
```

### ChatMessage 中的 preempted 标记

```typescript
interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  preempted?: boolean  // true 表示该消息被中断，未完整生成
  // ...
}
```

前端在抢占时会将旧 assistant 消息的 `preempted` 置为 `true`，并在 UI 中显示「已中断」标记。

### 请求 ID 防污染机制

多轮快速抢占时，旧请求的回调可能延迟到达。前端通过 `requestIdRef` 避免旧回调污染新消息：

```typescript
requestIdRef.current++
const myRequestId = requestIdRef.current

const callbacks = {
  onToken: (token) => {
    if (requestIdRef.current !== myRequestId) return  // 忽略旧请求的 token
    // ...
  },
}
```

---

## 与 Interrupt/Resume 的区别

| 维度 | TurnLoop（Preempt/Abort） | Interrupt/Resume |
|------|--------------------------|-----------------|
| 触发方 | 用户（新消息 / 停止按钮） | Agent（检测到确认关键词） |
| 目的 | 取消当前 turn | 暂停并等待用户确认 |
| 恢复 | 不需要（直接开始新 turn） | 需要 POST /chat/resume |
| 状态持久化 | 不持久化 | 存入 CheckpointStore |
| 适用场景 | 用户改变主意、快速打断 | 高风险操作二次确认 |

---

## 并发安全性

`SessionLoop` 使用 `sync.Map` 管理会话，每个 `turnEntry` 有独立 `sync.Mutex`，所有方法均为并发安全：

- 多个 goroutine 可安全地对不同 sessionID 并发调用
- 同一 sessionID 的 `StartTurn` / `Abort` / `EndTurn` 通过 mutex 串行化

---

## 最佳实践

1. **每个 session 使用稳定的 session_id** — TurnLoop 按 sessionID 隔离状态，随机 ID 会导致无法抢占
2. **前端不需要等待 abort 响应再发送新消息** — `POST /chat/stream` 的 `StartTurn` 会自动抢占，abort 请求是辅助信号
3. **Legacy 模式** — 不发送 `preempted` SSE 事件，被抢占时流直接关闭（不输出 `[DONE]`），前端通过 AbortController 感知
4. **超时与中止配合使用** — 在 Agent YAML 中设置 `middleware.timeout` 作为兜底，TurnLoop 的 Abort 作为用户主动中止
