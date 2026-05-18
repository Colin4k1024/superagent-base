# Agent Loop 自主循环

Agent Loop 是一种自主迭代执行模式。与常规的单轮对话 Agent 不同，`agentloop` 类型的 Agent 会持续推进任务，自动将前一轮输出作为下一轮输入，直到任务完成或达到最大迭代次数。

## 核心特性

| 特性 | 说明 |
|------|------|
| 自主迭代 | Agent 自动循环推进，无需人工逐轮触发 |
| 上下文累积 | 每轮输出自动拼接为下一轮上下文 |
| 完成信号 | Agent 输出 `[DONE]` 标记即终止循环 |
| 最大轮次保护 | 默认 25 轮，可通过 `max_turns` 配置 |
| 流式输出 | 每轮的 token 实时流式返回，不阻塞 |
| 上下文取消 | 支持 `context.Context` 取消，随时中止循环 |

## 适用场景

- **自主编程** — Agent 分步分析需求、编写代码、自检修复
- **深度研究** — 多轮搜索、归纳、验证，直到得出结论
- **数据处理** — 逐步清洗、转换、校验数据集
- **任务规划与执行** — 自动拆解复杂任务并逐步完成

## YAML 配置

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: code-agent
  labels:
    team: engineering
    tier: production
spec:
  type: agentloop
  model:
    primary: claude-sonnet-4-6
    temperature: 0.2
    max_tokens: 8192
  system_prompt: |
    You are an autonomous coding agent. You work iteratively on the user's task:
    - Break down complex problems into steps.
    - Execute one step per turn, showing your work.
    - Use available tools (code execution, web search) as needed.
    - After each step, evaluate progress and decide next action.
    - When the task is fully complete, include [DONE] in your response.
    - If you are stuck or need user input, explain clearly and include [DONE].
  tools:
    - ref: builtin/code_execute
    - ref: builtin/web_search
  memory:
    backend: builtin
    session_window: 20
  max_turns: 25
  middleware:
    - name: timeout
      seconds: 120
    - name: retry
      max_attempts: 2
  observability:
    metrics: true
    trace: true
```

## 关键字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `spec.type` | string | 必须为 `agentloop` |
| `spec.max_turns` | int | 最大迭代轮次，默认 25 |
| `spec.system_prompt` | string | 必须引导 Agent 在完成时输出 `[DONE]` |
| `spec.tools` | list | Agent 在循环中可调用的工具 |
| `spec.memory` | object | 记忆配置，用于保持多轮上下文 |

## 执行流程

```
┌──────────────────────────────────────────────────────┐
│                  User Message                        │
└──────────────────────┬───────────────────────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   Turn 1        │
              │  调用底层 Agent  │──────────▶ 流式输出 token
              └────────┬────────┘
                       │
                       ▼
              ┌─────────────────┐
              │ 输出含 [DONE]?  │──── Yes ──▶ 结束循环
              └────────┬────────┘
                       │ No
                       ▼
              ┌─────────────────┐
              │  累积上下文      │
              │  拼接历史输出    │
              └────────┬────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   Turn 2        │
              │  调用底层 Agent  │──────────▶ 流式输出 token
              └────────┬────────┘
                       │
                       ▼
                      ...
                       │
                       ▼
              ┌─────────────────┐
              │ 达到 max_turns? │──── Yes ──▶ 输出达到上限提示
              └─────────────────┘
```

## System Prompt 编写指南

Agent Loop 的 system prompt 必须清晰引导模型：

1. **分步执行** — 每轮只做一步，展示中间结果
2. **自我评估** — 每步完成后判断进展
3. **完成信号** — 任务完成时必须输出 `[DONE]`
4. **卡住处理** — 无法继续时也输出 `[DONE]` 并说明原因

```yaml
system_prompt: |
  You are an autonomous agent. Follow these rules:
  1. Work iteratively — one step per turn.
  2. After each step, assess if the task is complete.
  3. When fully done, include [DONE] in your response.
  4. If blocked, explain why and include [DONE].
```

## 与其他类型的对比

| 特性 | chat_model_agent | agentloop | workflow |
|------|-----------------|-----------|---------|
| 执行模式 | 单轮对话 | 多轮自主循环 | DAG 图执行 |
| 上下文管理 | 手动多轮 | 自动累积 | 节点间传递 |
| 完成判断 | 单次响应即完成 | `[DONE]` 信号 | 图执行完毕 |
| 适合场景 | 简单问答 | 复杂自主任务 | 固定流程 |
| 人工干预 | 每轮需输入 | 无需干预 | 无需干预 |

## API 调用

Agent Loop 类型的 Agent 使用与其他 Agent 完全相同的 HTTP SSE 接口：

```bash
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "code-agent",
    "session_id": "s1",
    "message": "帮我实现一个 Go HTTP 服务器，包含健康检查和优雅关闭"
  }'
```

返回的 SSE 流中，每轮迭代之间会插入分隔标记：

```
data: {"type":"text","content":"## Turn 1\n分析需求..."}
data: {"type":"text","content":"\n--- Turn 2/25 ---\n"}
data: {"type":"text","content":"实现 HTTP server..."}
data: {"type":"text","content":"\n--- Turn 3/25 ---\n"}
data: {"type":"text","content":"添加优雅关闭... [DONE]"}
data: {"type":"done"}
```

## 最佳实践

1. **合理设置 max_turns** — 简单任务 5-10 轮，复杂任务 15-25 轮
2. **配合工具使用** — 给 Agent 提供代码执行、搜索等工具以增强能力
3. **设置超时中间件** — 防止单轮执行时间过长
4. **启用 observability** — 便于追踪每轮执行情况和性能瓶颈
5. **明确 system prompt** — 必须包含 `[DONE]` 信号约定
