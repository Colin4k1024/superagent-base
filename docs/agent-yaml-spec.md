# Agent YAML 规范

## 概述

Superagent Base 使用 YAML 文件声明式定义 AI Agent。格式参考 Kubernetes 资源清单：每个文件包含一个 `AgentDefinition`，由 `apiVersion`、`kind`、`metadata`、`spec` 四个顶层字段组成。

Agent YAML 文件存放在配置目录（默认 `agents/`）中，运行时启动时会自动加载全部文件，并通过文件系统 Watcher 实现热重载（2 秒防抖）。

---

## Schema

### 顶层字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `apiVersion` | string | 是 | 固定为 `superagent/v1` |
| `kind` | string | 是 | 固定为 `Agent` |
| `metadata` | object | 是 | Agent 身份信息 |
| `spec` | object | 是 | Agent 行为配置 |

---

### metadata

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 唯一标识符，必须匹配 `[a-z0-9-]+`（小写字母、数字、连字符）|
| `version` | string | 否 | 语义版本号，如 `"1.0.0"` |
| `tags` | []string | 否 | 自由标签，用于分组和过滤 |
| `labels` | map[string]string | 否 | 任意键值对 |

**约束：**
- `name` 不能为空
- `name` 只能包含 `[a-z0-9-]`，不允许大写字母、下划线、空格

---

### spec

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 执行模式，见下表 |
| `model` | object | 是 | 模型选择与路由配置 |
| `system_prompt` | string | 否 | 系统级指令，注入到每次对话 |
| `tools` | []ToolRef | 否 | 工具引用列表 |
| `memory` | object | 否 | 记忆后端配置 |
| `middleware` | []MiddlewareSpec | 否 | 中间件管道（有序） |
| `observability` | object | 否 | 可观测性配置 |

**spec.type 可选值：**

| 值 | 说明 |
|----|------|
| `chat_model_agent` | 标准对话模型，直接调用 LLM |
| `deep_agent` | 深度推理模式，支持多步规划 |
| `workflow` | 工作流模式，按预定义步骤执行 |

---

### spec.model

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `primary` | string | 是 | 默认模型 ID，如 `gpt-4o`、`deepseek-r1` |
| `fallback` | string | 否 | 主模型不可用时的备用模型 ID |
| `router` | string | 否 | 路由策略名称（如 `capability-based`）；为空时直接使用 `primary` |

---

### spec.tools[]

工具引用格式（`ref` 字段）支持三种 URI Scheme：

| 格式 | Scheme | 说明 | 示例 |
|------|--------|------|------|
| `builtin/<name>` | builtin | 内置工具 | `builtin/web_search` |
| `mcp://<server>/<tool>` | mcp | MCP 服务器工具 | `mcp://my-server/search` |
| `skill://<name>` | skill | SkillsHub 技能 | `skill://summarize` |

**内置工具列表：**

| 工具名 | ref | 说明 |
|--------|-----|------|
| web_search | `builtin/web_search` | 网页搜索 |
| http_request | `builtin/http_request` | HTTP 请求 |
| code_execute | `builtin/code_execute` | 代码执行 |

每个工具引用支持可选的 `config` 字段，用于传递工具专属配置：

```yaml
tools:
  - ref: builtin/web_search
    config:
      max_results: 5
  - ref: mcp://my-server/search
    config:
      timeout: 10s
```

---

### spec.memory

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `backend` | string | 否 | 记忆后端类型，见下表 |
| `config` | map[string]any | 否 | 后端专属配置 |

**可用后端：**

| 值 | 说明 |
|----|------|
| `builtin` | 内置会话内记忆（默认，无需外部服务） |
| `mem0` | [Mem0](https://mem0.ai) 长期记忆服务 |
| `zep` | [Zep](https://www.getzep.com) 对话历史 + 知识图谱 |
| `letta` | [Letta](https://letta.com) 持久化 Agent 记忆 |

---

### spec.middleware[]

有序的中间件管道，每个条目包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 中间件标识符（可选，缺省从 inline key 推断） |
| inline keys | map[string]any | 中间件配置，通过 YAML inline 展开 |

---

### spec.observability

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `tracing` | bool | false | 启用 OpenTelemetry 分布式追踪 |
| `metrics` | bool | false | 启用 Prometheus 指标采集 |
| `log_level` | string | — | 日志级别：`trace`、`debug`、`info`、`warn`、`error` |

---

## 校验规则

解析时 `Validate()` 函数依次检查：

1. `apiVersion` 必须等于 `superagent/v1`
2. `kind` 必须等于 `Agent`
3. `metadata.name` 不能为空
4. `metadata.name` 必须匹配 `^[a-z0-9-]+$`
5. `spec.type` 必须是 `chat_model_agent`、`deep_agent`、`workflow` 之一
6. `spec.model.primary` 不能为空

任何一项不满足都返回错误，文件不会被加载到运行时。

---

## 示例

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

### 示例 2：带系统提示词 + 工具的研究 Agent

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-agent
  version: "1.0.0"
  tags: [research, tools]
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
    fallback: deepseek-r1
  system_prompt: |
    你是一个专业的研究助手。请使用搜索工具查找最新信息，
    并给出有依据的回答。
  tools:
    - ref: builtin/web_search
      config:
        max_results: 5
    - ref: builtin/http_request
  memory:
    backend: builtin
  observability:
    tracing: true
    log_level: info
```

### 示例 3：本地模型（LM Studio / Ollama）

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: local-agent
  version: "1.0.0"
  tags: [local, dev]
spec:
  type: chat_model_agent
  model:
    primary: Qwen3-Coder-Next-4bit
  system_prompt: "你是一个代码助手，帮助用户编写和调试代码。"
  memory:
    backend: builtin
  observability:
    log_level: debug
```

### 示例 4：深度推理 Agent + MCP 工具

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: deep-analyst
  version: "2.0.0"
  tags: [analysis, mcp]
  labels:
    team: data
    env: production
spec:
  type: deep_agent
  model:
    primary: deepseek-r1
    router: capability-based
  system_prompt: |
    你是一个数据分析专家。请系统性地分析问题，
    给出详细的推理过程和结论。
  tools:
    - ref: mcp://data-server/query
      config:
        database: analytics
    - ref: skill://summarize
  memory:
    backend: mem0
    config:
      api_key: "${MEM0_API_KEY}"
  observability:
    tracing: true
    metrics: true
    log_level: info
```

---

## 运行时行为

- **加载**：启动时从 `ConfigDir` 读取所有 `.yaml` / `.yml` 文件，逐个解析并构建 Agent 实例
- **热重载**：文件系统 Watcher 监听目录变更，2 秒防抖后触发 `ReloadDir`，增量更新已加载的 Agent
- **失败处理**：单个文件解析失败不阻塞其他文件加载，失败信息记录到日志
- **并发安全**：`AgentRuntime` 内部使用 `sync.RWMutex` 保护 Agent 注册表
