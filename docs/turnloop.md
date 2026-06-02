# TurnLoop — Eino ADK 会话循环

> Superagent Base 集成 Eino ADK 的 TurnLoop 机制，为 chat_model_agent 提供 **Push / Preempt / Abort** 语义，实现更精细的流式会话控制。

---

## 概述

TurnLoop 是基于 [Eino ADK](https://github.com/cloudwego/eino) 的会话循环抽象。每个 `(agent_id, session_id)` 对应一个独立的 TurnLoop 会话，支持：

| 能力 | 说明 |
|------|------|
| **Push** | 向活跃会话推入新消息，无需等待当前回复结束 |
| **Preempt** | 新消息到达时，在工具调用完成后安全打断当前 turn（10s 超时） |
| **Abort** | 立即终止整个会话循环 |
| **Idle 自动回收** | 5 分钟无新消息则自动停止 TurnLoop，释放资源 |
| **热重载兼容** | Agent YAML 变更后，旧 TurnLoop 会话自动 stop + 重建 |

---

## 工作原理

```
用户请求 → ChatSSEHandler
              │
              ├─ TurnLoopManager.Chat()
              │     ├─ 检测 Agent 是否为 ADK-backed（内部有 adkChatModelAgent）
              │     ├─ 是 → Push 到 TurnLoop，返回 token channel
              │     └─ 否 → fallback 到 Agent.Chat()（原有路径）
              │
              └─ SSE 流式输出 token channel
```

### 会话生命周期

```
创建 TurnLoop Session
     ↓
[idle timeout 5min] ← Push 重置计时
     ↓
TurnLoop.Stop(UntilIdleFor)
     ↓
Wait() → 清理 unhandled items → 从 Manager 移除
```

### Preempt 语义

当用户在 Agent 正在生成回复时发送新消息：

1. 新消息通过 `Push()` 进入 TurnLoop
2. TurnLoop 在当前 turn 的**工具调用完成后**安全打断（`AfterToolCalls`）
3. 超时 10s 后强制打断
4. 旧的未处理消息的 output channel 被关闭
5. 新消息开始生成回复

---

## HTTP API

### 发送消息（自动使用 TurnLoop）

```
POST /api/v2/chat/stream
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "s1",
  "message": "你好"
}
```

无需额外配置，ADK-backed Agent 自动走 TurnLoop 路径。

### 中止会话

```
POST /api/v2/chat/abort
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "s1"
}
```

**响应**：

| status | 说明 |
|--------|------|
| `"aborted"` | 成功终止活跃 TurnLoop |
| `"no_active_loop"` | 该 session 无活跃 TurnLoop（已结束或未使用） |

---

## 前端集成

ChatPage 中的"停止生成"按钮调用 `chatApi.abort(agentId, sessionId)`，触发服务端 TurnLoop 立即终止：

```typescript
// web/src/lib/api.ts
export const chatApi = {
  async abort(agentId: string, sessionId: string): Promise<{ status: string }> {
    const res = await fetch(`${API_BASE}/chat/abort`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ agent_id: agentId, session_id: sessionId }),
    })
    return res.json()
  },
  // ...
}
```

---

## 架构细节

### 核心组件

| 组件 | 位置 | 职责 |
|------|------|------|
| `TurnLoopManager` | `pkg/agentdef/turnloop.go` | 管理所有 session 的 TurnLoop 实例 |
| `turnLoopSession` | `pkg/agentdef/turnloop.go` | 单个 session 的循环：GenInput / PrepareAgent / OnAgentEvents |
| `ChatSSEHandler` | `api/handler/coze/chat_sse.go` | HTTP 层集成，优先尝试 TurnLoop 再 fallback |

### GenInput 逻辑

- 多个待处理消息时，**只取最后一条**（最新意图），关闭其余 channel
- 从 Memory 后端加载历史消息，拼接 system prompt
- 持久化用户消息到 Memory

### OnAgentEvents 逻辑

- 流式消费 Agent 输出事件
- 支持中断检测（interrupt handler）
- 记录模型延迟指标（ModelRouter TTFT）
- 完成后将 assistant 回复持久化到 Memory

### 与中断/恢复的关系

TurnLoop 的 `handleInterrupt` 在检测到 Agent 请求确认时，通过 output channel 发送 interrupt 事件。恢复仍走 `HandleChatResume` 端点（独立于 TurnLoop）。

---

## 配置

TurnLoop 为**自动启用**，无需额外 YAML 配置。只要 Agent 内部构建为 ADK ChatModel（即使用 `react.NewAgent` 或 Eino ChatModel），TurnLoopManager 会自动检测并接管。

### 常量

| 常量 | 值 | 说明 |
|------|----|------|
| `turnLoopPreemptTimeout` | 10s | Preempt 等待工具调用完成的超时 |
| `turnLoopIdleTimeout` | 5min | 无新消息后自动回收 session |

---

## 与原有 Agent.Chat() 的兼容性

TurnLoop 采用**渐进式接入**设计：

1. `TurnLoopManager.Chat()` 先尝试 unwrap Agent 为 ADK ChatModel
2. 若成功 → 走 TurnLoop 路径
3. 若失败（返回 `handled=false`）→ fallback 到 `Agent.Chat()`

这意味着：
- 非 ADK Agent（如纯 workflow、supervisor 编排）仍走原路径
- 无需修改现有 Agent 实现即可受益于 TurnLoop
- 未来所有 chat_model_agent 默认获得 push/preempt/abort 能力
