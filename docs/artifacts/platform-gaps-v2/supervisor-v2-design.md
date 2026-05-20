# Supervisor V2 — 结构化调度设计

## 背景

当前 Supervisor Agent (v1) 是占位实现：将子 Agent 列表拼入 system prompt，LLM 输出自由文本，`maxRounds` 字段预留但未使用，子 Agent 从未被实际调用（`orchestration.go:54-56` 直接透传 mainAgent 输出）。

## 目标

实现真正的多轮 Supervisor 调度：LLM 通过结构化指令委派子 Agent，基于执行结果决定下一步，支持并行 delegation、错误恢复和中断续做。

## 设计方案

### 核心架构

```
用户消息 → Supervisor LLM (ReAct loop)
                ↓ tool_call: delegate_to_agent
          执行子 Agent（可并行）
                ↓ tool_result: 子 Agent 输出
          Supervisor LLM 评估结果
                ↓ final_answer 或继续 delegation
          流式输出最终答案
```

### 结构化 Delegation 指令

采用 **Tool call 机制**，将 `delegate_to_agent` 注册为 Supervisor 的内置工具：

```go
// 委派工具定义
type DelegateToolInput struct {
    AgentName string `json:"agent_name" description:"子 Agent 名称"`
    Task      string `json:"task" description:"委派任务描述"`
    Context   string `json:"context,omitempty" description:"可选上下文"`
}

type DelegateToolOutput struct {
    AgentName string `json:"agent_name"`
    Result    string `json:"result"`
    Status    string `json:"status"` // success | error | timeout
    Duration  string `json:"duration"`
}
```

LLM 可在单次响应中调用多个 `delegate_to_agent`（parallel tool calls），runtime 并行执行。

### 多轮调度循环

```go
func (s *SupervisorAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
    ch := make(chan string, 100)
    go func() {
        defer close(ch)
        input := message
        for round := 0; round < s.maxRounds; round++ {
            // 1. Supervisor LLM 决策（通过 ReAct agent with delegate tool）
            decision, tokens := s.decide(ctx, sessionID, input, round)

            // 2. 若是 final_answer，流式输出并退出
            if decision.FinalAnswer != "" {
                for _, t := range tokens { ch <- t }
                return
            }

            // 3. 执行委派（支持并行）
            results := s.executeDelegations(ctx, sessionID, decision.Delegations)

            // 4. 聚合结果作为下一轮输入
            input = s.aggregateResults(results)

            // 5. 发送中间事件
            ch <- formatAgentSwitchEvent(round, results)
        }
        ch <- formatError("max rounds exceeded")
    }()
    return ch, nil
}
```

### 回退/超时/失败处理

| 策略 | 行为 |
|------|------|
| `skip` | 跳过失败子 Agent，将错误信息反馈 Supervisor |
| `abort` | 任一失败立即终止 workflow |
| `ask_supervisor` | 将错误作为 observation 反馈 LLM 重新决策 |

实现：
- 超时：`context.WithTimeout` 包裹每次子 Agent 调用
- 重试：复用 `pkg/tool` 的 retry middleware 模式
- 并行度：信号量控制 `parallel_max`

### YAML Schema 扩展

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: project-manager
spec:
  type: supervisor
  model:
    primary: gpt-4o
  system_prompt: |
    你是项目经理，根据用户需求调度子 Agent 完成任务。
  orchestration:
    mode: supervisor
    max_rounds: 10
    delegation:
      timeout: 30s
      retry: 1
      fallback_strategy: ask_supervisor
      parallel_max: 3
    result_aggregation: concat  # concat | summarize | structured
  sub_agents:
    - ref: researcher
      description: "负责信息检索和文献调研"
    - ref: coder
      description: "负责代码编写和调试"
    - ref: reviewer
      description: "负责代码审查和质量把关"
```

新增 Go 类型：

```go
type DelegationConfig struct {
    Timeout          string `yaml:"timeout,omitempty"`
    Retry            int    `yaml:"retry,omitempty"`
    FallbackStrategy string `yaml:"fallback_strategy,omitempty"` // skip|abort|ask_supervisor
    ParallelMax      int    `yaml:"parallel_max,omitempty"`
}

type OrchestrationSpec struct {
    Mode              string            `yaml:"mode"`
    MaxRounds         int               `yaml:"max_rounds,omitempty"`
    Delegation        *DelegationConfig `yaml:"delegation,omitempty"`
    ResultAggregation string            `yaml:"result_aggregation,omitempty"`
}
```

### 集成点

| 集成点 | 方式 |
|--------|------|
| Tool calling | 通过 `pkg/tool` 注册 `builtin/delegate` 工具 |
| A2UI 协议 | 调度事件用 `agent_switch` + `progress` event type |
| Interrupt/Resume | 每轮循环 checkpoint 保存状态 |
| Observability | 每次 delegation 创建子 span |
| 热重载 | 子 Agent 引用通过 registry 解析 |

## 技术决策

| 决策 | 选择 | 备选 | 理由 |
|------|------|------|------|
| 调度指令格式 | Tool call | JSON block 文本解析 | 结构化可靠，复用 ReAct 基础设施 |
| 循环终止 | maxRounds + final_answer | 仅 maxRounds | 双保险，LLM 可提前结束 |
| 并行方式 | 按 delegation 批次 | 全局并行池 | 保持因果关系清晰 |
| 状态管理 | 复用 checkpoint | 新建状态层 | 减少新增复杂度 |

## 风险

- 依赖模型支持 function calling（弱模型可降级为文本解析）
- 多轮循环增加 token 消耗（maxRounds 兜底）
- 并行 delegation 结果合并语义需明确定义

## 关键代码位置

- `backend/pkg/agentdef/orchestration.go:28-84` — 当前实现
- `backend/pkg/agentdef/builder.go:862-899` — 构建逻辑
- `backend/pkg/agentdef/schema.go:174-180` — OrchestrationSpec
