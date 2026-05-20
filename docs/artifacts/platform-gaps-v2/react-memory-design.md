# ReAct Agent Memory 集成设计

## 背景

`einoReactAgent` 已在构建时持有 `memBackend` 字段（`builder.go:292`），但其 `Chat()` 方法（`builder.go:1238`）用 `_` 丢弃了 sessionID，既不加载历史消息也不保存对话。相比之下，`einoChatAgent.Chat`（`builder.go:1140-1219`）完整实现了 memory 集成。

## 影响

1. **多轮对话无上下文** — 每次请求都是"单轮"
2. **工具调用中间结果丢失** — 中断后无法恢复 tool chain
3. **Agent 状态不持久** — 无法跨会话记忆

## 设计方案

### 分期实施

| 阶段 | 内容 | 改动量 |
|------|------|--------|
| **V1** | 启动加载历史 + 结束保存响应 | ~30 行 |
| **V2** | Eino Callback 逐步持久化 tool_call/result | ~80 行 |

### V1：首尾保存

```go
func (a *einoReactAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
    msgs := make([]*schema.Message, 0, 24)

    // System prompt
    if a.systemPrompt != "" {
        msgs = append(msgs, schema.SystemMessage(a.systemPrompt))
    }

    // 加载历史（与 einoChatAgent 相同模式）
    if a.memBackend != nil && sessionID != "" {
        history, _ := a.memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
        for _, m := range history {
            msgs = append(msgs, mapToSchemaMessage(m))
        }
    }

    // 当前消息
    msgs = append(msgs, schema.UserMessage(message))

    // 保存用户消息
    if a.memBackend != nil && sessionID != "" {
        a.memBackend.AddMessage(ctx, sessionID, memory.Message{
            Role:    "user",
            Content: message,
        })
    }

    // 启动 ReAct stream
    stream, err := a.agent.Stream(ctx, msgs)
    if err != nil {
        return nil, err
    }

    // 收集完整响应并保存
    ch := make(chan string, 100)
    go func() {
        defer close(ch)
        var fullResp strings.Builder
        for chunk := range stream {
            ch <- chunk
            fullResp.WriteString(chunk)
        }
        // 保存 assistant 响应
        if a.memBackend != nil && sessionID != "" {
            a.memBackend.AddMessage(ctx, sessionID, memory.Message{
                Role:    "assistant",
                Content: fullResp.String(),
            })
        }
    }()
    return ch, nil
}
```

### V2：Callback 逐步持久化

利用 Eino `compose.WithCallbacks` 在每步 tool 调用时写入 memory：

```go
// 注册到 react.Agent 的 callback handler
type MemoryCallback struct {
    mem       memory.Backend
    sessionID string
}

func (mc *MemoryCallback) OnToolEnd(ctx context.Context, info *callbacks.ToolEndInfo) {
    mc.mem.AddMessage(ctx, mc.sessionID, memory.Message{
        Role:    "tool",
        Content: info.Output,
        Metadata: map[string]any{
            "tool_call_id": info.ToolCallID,
            "tool_name":    info.ToolName,
            "step":         info.Step,
        },
    })
}
```

### Context Window 管理

- **截断**：`GetMessagesOpts{Limit: 20}` 硬截断最近 N 轮
- **摘要**（可选 V3）：超过阈值时 LLM 生成摘要替换早期消息
- **建议扩展**：`GetMessagesOpts` 增加 `MaxTokens int` 字段

### 接口变更

**Memory interface 无需扩展**：

- `ShortTermMemory.AddMessage` — 通过 `Metadata` 区分消息类型
- `AgentStateMemory.SetState/GetState` — 存 ReAct chain 状态
- `Message.Metadata` 已是 `map[string]any`

唯一建议的小扩展：`GetMessagesOpts` 增加 `Filter map[string]any`，允许按 metadata 过滤。

### 集成方式

**消息注入 + 流后保存（非 callback）**

Eino `react.Agent.Stream(ctx, []*schema.Message)` 接受初始 messages 作为起始上下文。历史消息作为 initial msgs 传入是最自然的集成点，不修改 Eino 框架代码。

## 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| V1 集成方式 | 消息注入 | 不修改 Eino 框架，~30 行改动 |
| 中间结果持久化 | V2 Callback | 可选增强，不阻塞 V1 |
| Context 管理 | 硬截断 Limit:20 | 简单有效，覆盖大多数场景 |
| 接口扩展 | 不扩展 | 现有 Metadata 字段足够 |

## 风险

- ReAct 历史消息可能包含 thought/action/observation 内部格式，注入后需确认 LLM 能正确解读
- V1 阶段中断丢失中间结果（V2 解决）
- Limit:20 可能不够长对话场景（可配置化）

## 关键代码位置

- `backend/pkg/agentdef/builder.go:1238-1277` — einoReactAgent.Chat（需修改）
- `backend/pkg/agentdef/builder.go:1140-1219` — einoChatAgent.Chat（参考实现）
- `backend/pkg/agentdef/builder.go:288-295` — memBackend 赋值
- `backend/pkg/memory/interface.go:22-55` — Backend 接口
