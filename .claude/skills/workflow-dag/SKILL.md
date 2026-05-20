---
name: workflow-dag
description: >
  Design and debug Superagent workflow DAG agents (type: workflow). TRIGGER when:
  designing a multi-step workflow, connecting DAG nodes, handling branch/condition
  logic, or when the user asks "how does workflow execution work", "DAG 怎么定义",
  "add a conditional branch to my workflow".
  DO NOT TRIGGER when: building single-model agents (use agent-yaml-authoring),
  orchestration with supervisor/sequential types, or simple tool-calling agents.
origin: learned
tags: [workflow, dag, nodes, execution, branch, condition]
---

# Workflow DAG

Superagent's `workflow` agent type executes a directed acyclic graph of nodes.
Each node is either a model call, a tool invocation, or a control-flow gate.

## Minimal Workflow YAML

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: doc-pipeline
spec:
  type: workflow
  workflow:
    start: extract
    nodes:
      extract:
        type: model
        model: gpt-4o
        prompt: "Extract key points from: {{input}}"
        next: summarize

      summarize:
        type: model
        model: gpt-4o-mini
        prompt: "Summarize: {{extract.output}}"
        next: END
```

## Node Types

| Type | Description |
|------|-------------|
| `model` | LLM call; output stored as `<node_name>.output` |
| `tool` | Invokes `builtin/<name>`, `mcp://...`, or `skill://...` |
| `branch` | Conditional routing based on expression |
| `parallel` | Fork to multiple nodes; join waits for all |
| `transform` | Pure data mapping / template rendering (no LLM) |

## Branch Node

```yaml
classify:
  type: model
  model: gpt-4o-mini
  prompt: "Classify intent as SEARCH or ANSWER: {{input}}"
  next: route

route:
  type: branch
  conditions:
    - if: "contains(classify.output, 'SEARCH')"
      next: web_search
    - if: "contains(classify.output, 'ANSWER')"
      next: direct_answer
  default: direct_answer
```

## Parallel Fork / Join

```yaml
research:
  type: parallel
  branches:
    - web_search
    - db_lookup
  join: merge_results   # wait for all branches

merge_results:
  type: transform
  template: "Web: {{web_search.output}}\nDB: {{db_lookup.output}}"
  next: END
```

## Variable References

Use `{{node_name.output}}` to pipe output between nodes.
`{{input}}` refers to the original workflow input.

## Interrupt Inside a Workflow

```yaml
confirm:
  type: interrupt
  prompt: "Proceed with deletion? (yes/no)"
  next: delete_node
```

The workflow pauses and emits an `interrupt` SSE event.
Resume via `POST /api/v1/chat/resume`.

## Debugging

- Use `APP_LOG_LEVEL=debug` to trace node execution order and variable values.
- Each node logs its input, output, and duration.
- Failed nodes surface as `error` SSE events with `node` field indicating which node failed.
