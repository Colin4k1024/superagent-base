# Arch Design: Agent 上下文管理

| 字段 | 值 |
|------|------|
| 状态 | draft |
| 主责 | architect |
| 日期 | 2026-06-16 |

---

## 系统边界

本次改动局限于 `pkg/agentdef` 包内部，不触及：
- `pkg/memory` 的 Backend interface（不改协议）
- `api/` HTTP handler 层
- `application/` 服务层
- 外部存储（Redis / MySQL）

唯一的外部交互点：当 `summarize` 模式需要调用 LLM 时，复用现有 `modelrouter` 或直接构造一个轻量 ChatModel 实例。

## 组件拆分

### 新增组件

```
pkg/agentdef/
├── context_policy.go      ← ContextSpec 定义 + BuildContext 函数
├── context_policy_test.go ← 策略单测
└── token_estimator.go     ← token 估算工具（rune-based，预留 tokenizer 接口）
```

### 改动组件

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| schema.go | 扩展 | AgentSpec 新增 `Context ContextSpec` 字段 |
| adk_stream.go | 重构 | buildMessageHistory → 调用 BuildContext |
| chat_agents.go | 重构 | SimpleChatAgent 的 inline history 读取 → 调用 BuildContext |
| orchestration.go | bugfix | Supervisor/Sequential/Parallel 传递子会话 ID |
| orchestration_delegate.go | bugfix + 扩展 | delegateTool 使用子会话 ID；summarize 调用注入的 SummarizeFunc |
| plan_execute.go | bugfix | executor 使用子会话 ID |
| agentloop.go | 重构 | 外层管理上下文，底层 agent 无 memory |
| builder.go | 扩展 | buildAgentLoop 时注入 memBackend=nil；buildSupervisor 注入 SummarizeFunc |

## 关键数据流

### BuildContext 调用流

```
Agent.Chat(ctx, sessionID, message)
  │
  ├─ policy := def.Spec.Context  (未配置时 → defaultPolicy)
  │
  ├─ switch policy.Strategy:
  │    case "stateless":
  │        return [systemPrompt]  // 无历史
  │    case "sliding_window":
  │        msgs = memBackend.GetMessages(ctx, sid, {Limit: policy.MaxMessages})
  │        return [systemPrompt] + msgs
  │    case "token_budget":
  │        msgs = memBackend.GetMessages(ctx, sid, {Limit: 100})  // 取足够多
  │        budget = policy.MaxTokens - estimateTokens(systemPrompt) - policy.ReserveOutputTokens
  │        return [systemPrompt] + truncateByTokenBudget(msgs, budget)
  │    case "summary_buffer":
  │        (P2: 超过 N 轮时调用摘要模型压缩早期消息)
  │
  └─ append(userMessage)
```

### 子会话隔离流

```
SupervisorAgent.Chat(ctx, "session-123", input)
  │
  ├─ mainAgent.Chat(ctx, "session-123", input)  // supervisor 自己的对话
  │
  └─ delegateTool.execute(ctx, DelegateToolInput{AgentName: "researcher"})
       │
       └─ sub.Chat(ctx, "session-123::sub::researcher::r1", task)
            // 子 Agent 写入独立命名空间，不污染主会话
```

## 接口约定

### ContextSpec (schema.go)

```go
type ContextSpec struct {
    // Strategy: stateless | sliding_window | token_budget | summary_buffer
    // Default: sliding_window (向后兼容)
    Strategy           string `yaml:"strategy,omitempty" json:"strategy,omitempty"`
    MaxMessages        int    `yaml:"max_messages,omitempty" json:"max_messages,omitempty"`       // sliding_window 模式
    MaxTokens          int    `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`           // token_budget / summary_buffer
    ReserveOutputTokens int   `yaml:"reserve_output_tokens,omitempty" json:"reserve_output_tokens,omitempty"`
    SummarizeAfterTurns int   `yaml:"summarize_after_turns,omitempty" json:"summarize_after_turns,omitempty"` // summary_buffer
    SummaryModel       string `yaml:"summary_model,omitempty" json:"summary_model,omitempty"`
    Tokenizer          string `yaml:"tokenizer,omitempty" json:"tokenizer,omitempty"`             // 预留: tiktoken | rune_estimate
}

type IsolationSpec struct {
    // SubSessionStrategy: none | per_call | per_round | per_branch
    // Default: per_call（推荐）
    SubSessionStrategy string `yaml:"sub_session_strategy,omitempty" json:"sub_session_strategy,omitempty"`
}
```

### BuildContext 函数签名

```go
// BuildContext constructs the message history for a chat model call based on policy.
// Returns the messages slice ready for LLM input (system + history, WITHOUT current user message).
func BuildContext(ctx context.Context, policy ContextSpec, systemPrompt, sessionID string, memBackend memory.Backend) ([]*schema.Message, error)
```

### SummarizeFunc 类型

```go
// SummarizeFunc compresses multiple text results into a concise summary.
// Used by aggregateResults when mode="summarize".
type SummarizeFunc func(ctx context.Context, inputs []string) (string, error)
```

### 子会话 ID 生成

```go
// SubSessionID generates a namespaced session ID for child agents.
func SubSessionID(parentID, agentName string, qualifier string) string {
    return parentID + "::" + qualifier + "::" + agentName
}
```

## 技术选型

| 决策点 | 选择 | 原因 |
|--------|------|------|
| token 估算 | `len([]rune(s)) * 1.5` (int) | 中文友好、零依赖、P2 可替换 |
| summarize 注入方式 | 函数类型注入（非接口） | 更轻量、测试 mock 简单、单一用途 |
| 子会话分隔符 | `::` | 避免与用户自定义 sessionID 冲突（用户 ID 通常不含 `::`) |
| 默认策略 | sliding_window + max_messages=20 | 向后兼容，存量行为不变 |

## 风险与约束

1. **`GetMessages` 性能**：token_budget 策略需要先取 100 条再裁剪，Redis LRANGE 对长 list 无问题，但需确认 memory backend 实现支持 Limit>20。—— 已确认 `GetMessagesOpts.Limit` 为 int，无硬上限。
2. **子会话数据增长**：每次 supervisor 调用产生 N 个子会话 key。缓解：子会话 TTL 跟随主会话，或在 SessionLoop.EndTurn 时清理。
3. **summary_buffer 的摘要时机**：摘要调用本身消耗时间和 token，不能在每次 Chat 调用时都触发。设计为"仅当消息数超过 SummarizeAfterTurns 且距上次摘要已过 N 轮时"触发。
