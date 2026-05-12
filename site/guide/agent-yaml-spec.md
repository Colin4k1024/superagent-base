# Agent YAML 规范

## 概述

Superagent Base 使用 YAML 文件声明式定义 AI Agent。格式参考 Kubernetes 资源清单：每个文件包含一个 `AgentDefinition`，由 `apiVersion`、`kind`、`metadata`、`spec` 四个顶层字段组成。

Agent YAML 文件存放在配置目录（默认 `configs/agents/`）中，运行时启动时会自动加载全部文件，并通过文件系统 Watcher 实现热重载（2 秒防抖）。

---

## 顶层结构

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `apiVersion` | string | 是 | 固定为 `superagent/v1` |
| `kind` | string | 是 | 固定为 `Agent` |
| `metadata` | object | 是 | Agent 身份信息 |
| `spec` | object | 是 | Agent 行为配置 |

---

## metadata

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 唯一标识符，必须匹配 `^[a-z0-9-]+$` |
| `version` | string | 否 | 语义版本号，如 `"1.0.0"` |
| `tags` | []string | 否 | 自由标签，用于分组和过滤 |
| `labels` | map[string]string | 否 | 任意键值对 |

---

## spec

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 执行模式，见下表 |
| `model` | ModelSpec | 是 | 模型选择与路由配置 |
| `system_prompt` | string | 否 | 系统级指令，注入到每次对话 |
| `tools` | []ToolRef | 否 | 工具引用列表 |
| `memory` | MemorySpec | 否 | 记忆后端配置 |
| `middleware` | []MiddlewareSpec | 否 | 中间件管道（有序） |
| `observability` | ObsSpec | 否 | 可观测性配置 |
| `sub_agents` | []SubAgentRef | 否 | 子 Agent 引用（orchestration 类型专用） |
| `orchestration` | OrchestrationSpec | 否 | 多 Agent 协调策略 |
| `workflow` | WorkflowSpec | 否 | 图工作流定义（type=workflow 时必填） |
| `interrupt` | InterruptConfig | 否 | 中断/恢复配置 |

### spec.type 可选值

| 值 | 说明 |
|----|------|
| `chat_model_agent` | 标准对话 Agent，可挂载工具（builtin / mcp / skill） |
| `deep_agent` | 深度推理模式，支持多步规划（扩展自 chat_model_agent） |
| `supervisor` | 多 Agent 协调者，使用 LLM 决策并将任务分发给 sub_agents |
| `sequential` | 顺序执行 sub_agents，前一个的完整输出作为下一个的输入 |
| `parallel` | 并发执行所有 sub_agents，合并输出 |
| `plan_execute` | 先规划后执行的多 Agent 模式 |
| `workflow` | DAG 图执行，通过 `spec.workflow` 定义节点和边 |

---

## spec.model

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `primary` | string | 是 | 默认模型 ID，如 `gpt-4o`、`deepseek-r1` |
| `fallback` | string | 否 | 主模型不可用时的备用模型 ID |
| `router` | string | 否 | 路由策略名称；为空时直接使用 `primary` |

**路由策略（`router` 字段）：**

| 策略名 | 说明 |
|--------|------|
| `capability-based` | 根据任务能力需求选择最合适的模型 |
| `cost-optimized` | 优先选择成本最低的模型 |
| `latency` | 优先选择响应延迟最低的模型 |

---

## spec.tools[]

每个工具引用包含 `ref` 和可选的 `config`：

```yaml
tools:
  - ref: builtin/web_search
    config:
      max_results: 5
  - ref: mcp://my-server/search
    config:
      timeout: 10s
  - ref: skill://calculator
```

**URI 格式：**

| Scheme | 格式 | 说明 |
|--------|------|------|
| `builtin/` | `builtin/<name>` | 内置工具（来自 pkg/tool/builtin） |
| `mcp://` | `mcp://<server>/<tool>` | MCP 服务器上的工具 |
| `skill://` | `skill://<name>` | SkillsHub 技能 |

**内置工具列表：**

| 工具名 | ref | 说明 |
|--------|-----|------|
| web_search | `builtin/web_search` | 网页搜索 |
| http_request | `builtin/http_request` | HTTP 请求 |
| code_execute | `builtin/code_execute` | 代码执行 |

---

## spec.memory

| 字段 | 类型 | 说明 |
|------|------|------|
| `backend` | string | 记忆后端类型：`builtin` / `mem0` / `zep` / `letta` |
| `config` | map[string]any | 后端专属配置（如 api_key、session_window 等） |

| 后端 | 说明 |
|------|------|
| `builtin` | 内置会话内记忆（默认，无需外部服务） |
| `mem0` | [Mem0](https://mem0.ai) 长期记忆服务 |
| `zep` | [Zep](https://www.getzep.com) 对话历史 + 知识图谱 |
| `letta` | [Letta](https://letta.com) 持久化 Agent 记忆 |

---

## spec.interrupt

控制中断/恢复行为，当模型输出中检测到确认请求时自动触发中断。

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | false | 启用中断/恢复包装 |
| `checkpoint_backend` | string | `memory` | 持久化后端：`redis` 或 `memory` |
| `timeout_seconds` | int | 300 | 中断状态保留时长（秒） |

详见 [中断与恢复](/advanced/interrupt-resume)。

---

## spec.sub_agents[]

用于 `supervisor` / `sequential` / `parallel` 类型，引用其他已注册的 Agent：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `ref` | string | 是 | 被引用 Agent 的 `metadata.name` |
| `role` | string | 否 | 该子 Agent 在编排中的角色描述 |
| `config` | map[string]any | 否 | 每个子 Agent 的独立配置覆盖 |

---

## spec.orchestration

| 字段 | 类型 | 说明 |
|------|------|------|
| `mode` | string | 编排模式：`supervisor` / `sequential` / `parallel` |
| `max_rounds` | int | 最大迭代轮次（supervisor 默认 5） |

---

## spec.workflow

用于 `type: workflow`，定义 DAG 图执行工作流。

### WorkflowSpec

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `nodes` | []WorkflowNode | 是 | 工作流节点列表 |
| `edges` | []WorkflowEdge | 是 | 有向边列表 |
| `variables` | []WorkflowVariable | 否 | 命名变量，将节点输出映射为状态键 |

### WorkflowNode

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 节点唯一标识（workflow 内唯一） |
| `type` | string | 节点类型：`llm_call` / `agent_call` / `tool_call` / `code` / `condition` |
| `agent` | string | agent_call 节点使用：引用 Agent 的 name |
| `tool` | string | tool_call 节点使用：工具 URI（如 `builtin/web_search`） |
| `prompt` | string | llm_call 节点使用：系统/指令提示词 |
| `code` | string | code 节点使用：执行代码 |
| `language` | string | code 节点使用：`python` / `javascript` |
| `condition` | string | condition 节点使用：布尔表达式 |
| `input_mapping` | map[string]string | 输入字段映射，值为状态表达式（`$.varName`）或模板 |

**节点类型说明：**

| 类型 | 行为 |
|------|------|
| `llm_call` | 调用 LLM 生成文本，使用 `prompt` 作为系统提示词 |
| `agent_call` | 调用另一个已注册的 Agent |
| `tool_call` | 调用工具（当前为占位符，完整实现后接入 tool.Manager） |
| `code` | 执行代码片段（当前为占位符） |
| `condition` | 评估条件表达式，输出 `"true"` 或 `"false"` |

### WorkflowEdge

| 字段 | 类型 | 说明 |
|------|------|------|
| `from` | string | 源节点 ID，或 `START`（工作流入口） |
| `to` | string | 目标节点 ID，或 `END`（工作流出口） |
| `condition` | string | 条件表达式，非空时仅在条件为真时跟随此边 |

::: v-pre

### WorkflowVariable

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 变量名，下游节点通过 `{{.name}}` 引用 |
| `from` | string | 来源格式：`node_id.output` 或 `node_id.field` |

:::

---

## spec.middleware[]

有序的工具调用中间件管道：

```yaml
middleware:
  - name: retry
    max_attempts: 3
    backoff: 1s
  - name: timeout
    duration: 30s
  - name: rate_limit
    rps: 10
  - name: cache
    ttl: 300s
```

---

## spec.observability

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `tracing` | bool | false | 启用 OpenTelemetry 分布式追踪 |
| `metrics` | bool | false | 启用 Prometheus 指标采集 |
| `log_level` | string | — | 日志级别：`trace` / `debug` / `info` / `warn` / `error` |

---

## 校验规则

解析时 `Validate()` 依次检查：

1. `apiVersion` 必须等于 `superagent/v1`
2. `kind` 必须等于 `Agent`
3. `metadata.name` 不能为空，且必须匹配 `^[a-z0-9-]+$`
4. `spec.type` 必须是 `chat_model_agent` / `deep_agent` / `workflow` / `supervisor` / `sequential` / `parallel` / `plan_execute` 之一
5. orchestration 类型（`supervisor` / `sequential` / `parallel` / `plan_execute`）必须有至少一个 `sub_agents` 条目，且每个条目的 `ref` 不能为空
6. `type: workflow` 时，`spec.workflow` 不能为空，且必须包含至少一个 node 和一条 edge
7. 非 orchestration 且非 workflow 类型时，`spec.model.primary` 不能为空

任何一项不满足都返回错误，文件不会加载到运行时。

---

## 完整示例

### 示例 1：最简 Chat Agent

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: hello-agent
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
```

### 示例 2：Research Agent（使用 configs/agents/research-agent.yaml）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-agent
  version: "1.0.0"
  tags: [research, analysis]
  labels:
    team: core
    tier: production
spec:
  type: chat_model_agent
  model:
    router: capability-based
    primary: deepseek-r1
    fallback: gpt-4o
  system_prompt: |
    You are a professional research assistant. Help users analyze topics,
    find information, and provide well-structured summaries.
  tools:
    - ref: builtin/web_search
    - ref: builtin/http_request
  memory:
    backend: mem0
    config:
      session_window: 20
  observability:
    tracing: true
    metrics: true
    log_level: info
```

### 示例 3：Code Review Agent（使用 configs/agents/code-review-agent.yaml）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: code-review-agent
  version: "1.0.0"
  tags: [coding, review]
  labels:
    team: engineering
    tier: production
spec:
  type: chat_model_agent
  model:
    router: capability-based
    primary: claude-sonnet
    fallback: deepseek-r1
  system_prompt: |
    You are a senior code reviewer. Analyze code for bugs, security issues,
    performance problems, and suggest improvements.
  tools:
    - ref: builtin/code_execute
  memory:
    backend: builtin
  observability:
    tracing: true
    metrics: true
    log_level: info
```

### 示例 4：Supervisor 多 Agent（项目经理）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: project-manager
  version: "1.0.0"
  tags: [orchestration, supervisor]
spec:
  type: supervisor
  model:
    primary: gpt-4o
  system_prompt: |
    你是一个项目经理。根据用户需求，协调研究 Agent 和代码审查 Agent 分工合作。
  sub_agents:
    - ref: research-agent
      role: 负责信息调研和分析
    - ref: code-review-agent
      role: 负责代码质量审查
  orchestration:
    mode: supervisor
    max_rounds: 5
  observability:
    tracing: true
    log_level: info
```

### 示例 5：Sequential Pipeline（顺序流水线）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-then-review
  version: "1.0.0"
spec:
  type: sequential
  model:
    primary: gpt-4o
  system_prompt: "先调研，再审查"
  sub_agents:
    - ref: research-agent
      role: step-1-research
    - ref: code-review-agent
      role: step-2-review
```

### 示例 6：Parallel（并行汇总）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: parallel-analysis
  version: "1.0.0"
spec:
  type: parallel
  model:
    primary: gpt-4o
  system_prompt: "并发分析"
  sub_agents:
    - ref: research-agent
    - ref: code-review-agent
```

::: v-pre

### 示例 7：Workflow（DAG 图执行）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-workflow
  version: "1.0.0"
  tags: [workflow]
spec:
  type: workflow
  model:
    primary: gpt-4o
  system_prompt: "研究工作流"
  workflow:
    nodes:
      - id: search
        type: agent_call
        agent: research-agent
        input_mapping:
          query: "$.message"

      - id: summarize
        type: llm_call
        prompt: "请对以下研究结果进行摘要和结构化整理"
        input_mapping:
          content: "$.search.output"

      - id: quality_check
        type: condition
        condition: "{{.summarize.output}}"

    edges:
      - from: START
        to: search
      - from: search
        to: summarize
      - from: summarize
        to: quality_check
      - from: quality_check
        to: END

    variables:
      - name: search_result
        from: search.output
      - name: summary
        from: summarize.output
```

:::

### 示例 8：中断/恢复 Agent

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: approval-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
  system_prompt: |
    你是一个审批助手。在执行任何重要操作前，
    请明确询问用户确认（使用"Please confirm"等短语）。
  interrupt:
    enabled: true
    checkpoint_backend: redis
    timeout_seconds: 300
  memory:
    backend: builtin
  observability:
    tracing: true
    log_level: info
```

---

## 运行时行为

| 行为 | 说明 |
|------|------|
| **加载** | 启动时从 `AGENT_CONFIG_DIR`（默认 `configs/agents/`）读取所有 `.yaml` / `.yml` 文件 |
| **热重载** | fsnotify 监听目录变更，2 秒防抖后触发 `ReloadDir`，增量更新 Agent 注册表 |
| **失败处理** | 单个文件解析失败不阻塞其他文件，失败信息记录到日志 |
| **并发安全** | `AgentRuntime` 内部使用 `sync.RWMutex` 保护注册表 |
| **中断包装** | 当 `spec.interrupt.enabled: true` 时，构建的 Agent 自动包装为 `InterruptableAgent` |
