# TurnLoop 会话循环

TurnLoop 是基于 Eino ADK 的会话循环机制，为 `chat_model_agent` 提供 **Push / Preempt / Abort** 语义。与传统的单次请求-响应模式不同，TurnLoop 允许用户在 Agent 生成回复的过程中发送新消息、安全打断当前生成、或立即终止会话。

## 核心特性

| 特性 | 说明 |
|------|------|
| Push | 向活跃会话推入新消息，无需等待当前回复结束 |
| Preempt | 新消息到达时，在工具调用完成后安全打断当前 turn（10s 超时） |
| Abort | 通过 HTTP API 立即终止整个会话循环 |
| Idle 自动回收 | 5 分钟无新消息自动停止 TurnLoop，释放资源 |
| 热重载兼容 | Agent YAML 变更后，旧 TurnLoop 会话自动 stop 并重建 |
| 零配置接入 | ADK-backed Agent 自动启用，无需额外 YAML 配置 |

## 适用场景

- **流式对话打断** — 用户发现 Agent 理解错误，立即发送新消息修正方向
- **停止生成** — 前端"停止"按钮一键中止冗长回复
- **连续快速提问** — 无需等待上一轮完成，新消息自动 preempt 旧 turn
- **资源保护** — 空闲会话自动回收，防止内存泄漏

## 工作原理

```
用户请求 → ChatSSEHandler
              │
              ├─ TurnLoopManager.Chat()
              │     ├─ 检测 Agent 是否为 ADK-backed
              │     ├─ 是 → Push 到 TurnLoop，返回 token channel
              │     └─ 否 → fallback 到 Agent.Chat()
              │
              └─ SSE 流式输出 token channel
```

### 会话生命周期

```
创建 TurnLoop Session
     ↓
[idle timeout 5min] ← 每次 Push 重置计时
     ↓
TurnLoop.Stop(UntilIdleFor)
     ↓
Wait() → 清理 unhandled items → 从 Manager 移除
```

### Preempt 流程

当用户在 Agent 正在生成回复时发送新消息：

1. 新消息通过 `Push()` 进入 TurnLoop
2. TurnLoop 在当前 turn 的**工具调用完成后**安全打断（`AfterToolCalls`）
3. 超时 10s 后强制打断
4. 旧消息的 output channel 被关闭
5. 新消息开始生成回复

## HTTP API

### 发送消息

```bash
POST /api/v2/chat/stream
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "s1",
  "message": "你好"
}
```

无需额外配置。ADK-backed Agent 自动走 TurnLoop 路径，非 ADK Agent 自动 fallback 到原有逻辑。

### 中止会话

```bash
POST /api/v2/chat/abort
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "s1"
}
```

**响应**：

```json
// 成功终止
{"status": "aborted"}

// 无活跃会话
{"status": "no_active_loop"}
```

## 前端集成

ChatPage 中的"停止生成"按钮自动调用 abort API：

```typescript
// 停止生成
const handleStop = useCallback(() => {
  if (selectedAgent) {
    void chatApi.abort(selectedAgent, sessionId)
  }
  // 同时中止本地 SSE 连接
  abortRefs.current.forEach((controller) => controller.abort())
  abortRefs.current.clear()
  setActiveStreamCount(0)
}, [selectedAgent, sessionId])
```

## 架构组件

| 组件 | 位置 | 职责 |
|------|------|------|
| `TurnLoopManager` | `pkg/agentdef/turnloop.go` | 管理所有 session 的 TurnLoop 实例 |
| `turnLoopSession` | `pkg/agentdef/turnloop.go` | 单个会话循环：GenInput / PrepareAgent / OnAgentEvents |
| `ChatSSEHandler` | `api/handler/coze/chat_sse.go` | HTTP 层，优先尝试 TurnLoop 再 fallback |

### GenInput 逻辑

- 多个待处理消息时，**只取最后一条**（最新用户意图），关闭其余 channel
- 从 Memory 后端加载历史消息，拼接 system prompt
- 持久化用户消息到 Memory

### OnAgentEvents 逻辑

- 流式消费 Agent 输出事件（支持 streaming + 非 streaming）
- 支持中断检测（interrupt handler）
- 记录模型延迟指标（ModelRouter TTFT）
- 完成后将 assistant 回复持久化到 Memory

## 配置常量

| 常量 | 值 | 说明 |
|------|----|------|
| `turnLoopPreemptTimeout` | 10s | Preempt 等待工具调用完成的超时 |
| `turnLoopIdleTimeout` | 5min | 无新消息后自动回收 session |

## 与其他特性的关系

### 与中断/恢复

TurnLoop 的 interrupt handler 在检测到 Agent 请求确认时，通过 output channel 发送 interrupt 事件。恢复仍走独立的 `POST /api/v2/chat/resume` 端点。

### 与 A2UI 协议

TurnLoop 同时支持 Legacy 和 A2UI 两种流式输出模式。A2UI 模式下，TurnLoop 的 token channel 会被 `TokenStreamToEventStream` 转换为结构化事件流。

### 兼容性

TurnLoop 采用**渐进式接入**设计：

- ADK-backed Agent（使用 `react.NewAgent` 或 Eino ChatModel）→ 自动使用 TurnLoop
- 非 ADK Agent（workflow、supervisor 等编排类型）→ 自动 fallback 到 `Agent.Chat()`
- 无需修改现有 Agent YAML 或代码
