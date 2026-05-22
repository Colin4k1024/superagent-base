# Workflow 图执行指南

## 概述

Superagent Base 的 **Workflow**（图工作流）功能允许将多个处理步骤组合成一个 **有向无环图（DAG）**，由系统自动进行拓扑排序并顺序执行。工作流中的每个节点可以是 LLM 调用、Agent 调用、工具调用、代码执行或条件判断，节点之间通过边传递数据。

---

## 核心概念

```
                 START
                   │
                   ▼
            [search 节点]  ← agent_call: research-agent
                   │
          输出存入 state["search.output"]
                   │
                   ▼
          [summarize 节点] ← llm_call: 使用 state["search.output"]
                   │
          输出存入 state["summarize.output"]
                   │
                   ▼
         [quality_check 节点] ← condition: 评估条件
                   │
                  END
```

执行引擎使用 **Kahn 算法**进行拓扑排序，确保所有依赖节点先于下游节点执行。每个节点的输出自动存入共享状态（`state`），供后续节点通过 `input_mapping` 引用。

---

## YAML 语法

### 完整结构

```yaml
spec:
  type: workflow
  workflow:
    nodes:
      - id: <node-id>          # 节点唯一 ID（workflow 内唯一）
        type: <node-type>      # 节点类型（见下文）
        # ... 节点专属字段

    edges:
      - from: <source-id>      # 源节点 ID 或 "START"
        to: <target-id>        # 目标节点 ID 或 "END"
        condition: <expr>      # 可选：条件表达式

    variables:                  # 可选：命名变量别名
      - name: <var-name>
        from: <node-id>.output
```

---

::: v-pre

## 节点类型

### llm_call（LLM 推理节点）

调用 LLM 模型生成文本。

```yaml
- id: summarize
  type: llm_call
  prompt: "请对以下内容进行摘要：{{.search_result}}"
  input_mapping:
    content: "$.search.output"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `prompt` | 否 | 系统提示词，支持 `{{.varName}}` 模板语法 |
| `input_mapping` | 否 | 输入字段映射（见下文） |

执行时会基于工作流的模型配置（`spec.model`）创建一个临时 ChatAgent。

### agent_call（Agent 调用节点）

调用另一个已注册的 Agent（必须是 `AgentRuntime` 中已加载的 Agent）。

```yaml
- id: search
  type: agent_call
  agent: research-agent
  input_mapping:
    query: "$.message"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `agent` | 是 | 被调用 Agent 的 `metadata.name` |
| `input_mapping` | 否 | 输入字段映射 |

### tool_call（工具调用节点）

调用工具（当前为占位符，完整实现接入 `tool.Manager` 后支持真实工具调用）。

```yaml
- id: web_search
  type: tool_call
  tool: builtin/web_search
  input_mapping:
    query: "$.message"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `tool` | 是 | 工具 URI（如 `builtin/web_search`、`mcp://server/tool`） |
| `input_mapping` | 否 | 输入字段映射 |

### code（代码执行节点）

执行代码片段（当前为占位符，输出代码预览）。

```yaml
- id: process
  type: code
  language: python
  code: |
    result = input_data.upper()
    print(result)
  input_mapping:
    input_data: "$.message"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `code` | 是 | 执行代码，支持模板替换 |
| `language` | 否 | 编程语言（`python` / `javascript` 等） |
| `input_mapping` | 否 | 输入字段映射 |

### condition（条件判断节点）

评估条件表达式，输出 `"true"` 或 `"false"`，供条件边使用。

```yaml
- id: quality_check
  type: condition
  condition: "{{.summarize.output}}"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `condition` | 是 | 条件表达式；非空且非 `"false"` / `"0"` 时输出 `"true"` |

---

## 边（Edges）

```yaml
edges:
  - from: START
    to: search

  - from: search
    to: summarize

  - from: summarize
    to: quality_check

  - from: quality_check
    to: publish
    condition: "true"   # 仅在 quality_check 输出为 "true" 时跟随此边

  - from: quality_check
    to: revise
    condition: "false"

  - from: publish
    to: END

  - from: revise
    to: summarize   # 回路（不允许，会形成环，导致校验失败）
```

**保留标识符：**
- `START`：工作流入口（作为 `from` 值），被此边指向的节点从初始状态接收输入
- `END`：工作流出口（作为 `to` 值），仅作标记

**条件边：**
- `condition` 字段非空时，仅在上游节点输出与该值匹配时跟随此边
- 常用于 condition 节点的分支跳转

---

## 变量映射

### input_mapping

`input_mapping` 将工作流状态中的值映射为节点的输入字段：

```yaml
input_mapping:
  query: "$.message"           # 从 state["message"] 读取（即原始用户消息）
  context: "$.search.output"  # 从 state["search.output"] 读取（search 节点输出）
  combined: "{{.summary}} {{.context}}"  # 模板语法，拼接多个变量
```

**两种引用方式：**

| 语法 | 说明 |
|------|------|
| `$.key` | 直接从状态中读取 key 的值 |
| `{{.varName}}` | 模板语法，可在字符串中嵌入变量 |

### variables（命名别名）

```yaml
variables:
  - name: search_result
    from: search.output      # state["search_result"] = state["search.output"]
  - name: summary
    from: summarize.output
```

声明变量后，下游节点可以通过 `$.search_result` 或 `{{.search_result}}` 引用，而无需写完整的 `$.search.output`。

### 状态键规则

| 键 | 说明 |
|----|------|
| `message` | 初始用户消息（自动注入） |
| `<node_id>.output` | 节点的完整输出（自动写入） |
| `<var_name>` | 通过 `variables` 声明的命名别名 |

---

## 完整示例

### 示例 1：研究工作流

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-workflow
  version: "1.0.0"
  tags: [workflow, research]
spec:
  type: workflow
  model:
    primary: gpt-4o
  system_prompt: "研究工作流"
  workflow:
    nodes:
      # 步骤 1：调用研究 Agent 搜索信息
      - id: search
        type: agent_call
        agent: research-agent
        input_mapping:
          query: "$.message"

      # 步骤 2：LLM 整理搜索结果
      - id: summarize
        type: llm_call
        prompt: |
          你是一个技术文档编写专家，请将以下研究结果整理成结构化摘要：
          {{.search_result}}
        input_mapping:
          content: "$.search.output"

      # 步骤 3：质量检查
      - id: quality_check
        type: condition
        condition: "{{.summary}}"

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

### 示例 2：多步骤数据处理

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: data-pipeline
  version: "1.0.0"
spec:
  type: workflow
  model:
    primary: deepseek-r1
  workflow:
    nodes:
      - id: extract
        type: llm_call
        prompt: "从用户输入中提取关键实体和数据点"

      - id: validate
        type: condition
        condition: "{{.extract.output}}"

      - id: transform
        type: llm_call
        prompt: "将提取的数据转换为标准 JSON 格式"
        input_mapping:
          data: "$.extract.output"

      - id: report
        type: llm_call
        prompt: "生成数据处理报告"
        input_mapping:
          transformed: "$.transform.output"
          original: "$.message"

    edges:
      - from: START
        to: extract
      - from: extract
        to: validate
      - from: validate
        to: transform
        condition: "true"
      - from: validate
        to: END
        condition: "false"
      - from: transform
        to: report
      - from: report
        to: END

    variables:
      - name: extracted_data
        from: extract.output
      - name: final_report
        from: report.output
```

---

## 执行机制

### 拓扑排序

使用 Kahn 算法对节点进行拓扑排序：

1. 计算每个节点的入度（被多少条边指向）
2. 将入度为 0 的节点加入执行队列
3. 依次执行队列中的节点，执行后减少其出边目标节点的入度
4. 重复直到所有节点执行完毕
5. 若最终执行节点数 < 总节点数，则存在环形依赖，抛出错误

`START` 和 `END` 作为哨兵值，不参与拓扑排序计算。

### 数据传播

```
state = { "message": "<user input>" }

执行 node A:
  input = resolveInput(node.input_mapping, state)
  output = executeNode(node, input)
  state["A.output"] = output

执行 node B:
  input_mapping:
    foo: "$.A.output"  → input["foo"] = state["A.output"]
  output = executeNode(node, input)
  state["B.output"] = output

最终输出 = state[lastNode + ".output"]
```

### 流式输出

`WorkflowAgent.Chat()` 以 channel 方式返回，仅将最后一个节点的输出流式传输给调用方。中间节点的输出只保存在 state 中。

---

## 错误处理

| 情形 | 行为 |
|------|------|
| 节点执行失败 | 输出 `[workflow error] node "X": <err>`，停止执行 |
| 图中存在环 | 返回 `[workflow error] workflow graph contains a cycle` |
| 节点 ID 未找到 | 返回 `[workflow error] node "X" not found` |
| agent_call 目标不存在 | 返回 `executeAgentNode: agent "X" not found` |
| 没有可执行节点 | 返回 `[workflow error] no executable nodes found` |
| spec.workflow 为空 | YAML 校验失败：`spec.workflow must have at least one node` |
| edges 为空 | YAML 校验失败：`spec.workflow must have at least one edge` |

---

## 与多 Agent 编排的区别

| 特性 | Workflow | Supervisor/Sequential/Parallel |
|------|----------|-------------------------------|
| 执行顺序 | 由 DAG 拓扑排序决定 | Supervisor：LLM 决策；Sequential：声明顺序；Parallel：并发 |
| 条件分支 | 支持（condition 节点 + 条件边） | 不支持（Supervisor 通过 LLM 路由） |
| 状态传递 | 显式 input_mapping + variables | Sequential 自动传递（前一个输出→下一个输入） |
| 适用场景 | 明确定义的多步处理管道 | 动态协作、并发分析 |
| 循环 | 不允许（DAG 限制） | Supervisor 支持 max_rounds 迭代 |

:::
