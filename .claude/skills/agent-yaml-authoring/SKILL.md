---
name: agent-yaml-authoring
description: >
  Author and validate Superagent declarative YAML agent definitions
  (apiVersion: superagent/v1, kind: Agent). TRIGGER when: creating a new agent,
  editing configs/agents/*.yaml, reviewing agent spec fields, or when the user
  asks "how do I define an agent", "write me an agent yaml", "agent 配置怎么写".
  DO NOT TRIGGER when: working on workflow DAG nodes (use workflow-dag skill),
  pure Go backend logic, or infrastructure config.
origin: learned
tags: [agent, yaml, superagent, declarative, config]
---

# Agent YAML Authoring

Superagent agents are declared as Kubernetes-style YAML manifests.
Every file in `configs/agents/` is hot-reloaded via fsnotify — no server restart needed.

## Minimal Template

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent          # unique slug, used as skill://my-agent reference
  description: "One-line description visible in the UI"
spec:
  type: chat_model_agent  # see Agent Types below
  model: gpt-4o
  system_prompt: |
    You are a helpful assistant.
  tools: []               # builtin/<name> | mcp://<server>/<tool> | skill://<name>
```

## Required Fields

| Field | Description |
|-------|-------------|
| `apiVersion` | Always `superagent/v1` |
| `kind` | Always `Agent` |
| `metadata.name` | Unique kebab-case slug |
| `spec.type` | One of the agent types below |
| `spec.model` | Model ID (see `configs/models/routing-rules.yaml`) |

## Agent Types

| Type | Description |
|------|-------------|
| `chat_model_agent` | Single model with optional tools (uses Eino ReAct when tools present) |
| `deep_agent` | Extended reasoning, multi-step web search |
| `supervisor` | Orchestrates sub-agents, routes tasks dynamically |
| `sequential` | Runs sub-agents in fixed order, pipes output |
| `parallel` | Fans out to sub-agents simultaneously |
| `plan_execute` | Plan phase then execute phase |
| `workflow` | Full DAG workflow engine |

## Tool URI Schemes

```yaml
tools:
  - builtin/datetime        # built-in Go skill
  - builtin/calculator
  - mcp://filesystem/read   # MCP server tool
  - skill://code-fix        # another agent as a skill
```

## Interrupt / Resume

```yaml
spec:
  interrupt:
    enabled: true
  checkpoint:
    backend: redis          # or memory
```

## Sub-Agent Reference (orchestration)

```yaml
spec:
  type: supervisor
  agents:
    - name: researcher      # must be registered before this agent
    - name: writer
```

Two-pass build order: leaf agents are built first; orchestration agents resolve
`spec.agents` references from the registry. Declare leaves before supervisors.

## Canonical Example

See `configs/agents/research-agent.yaml` for a complete reference with
deep_agent, tools, memory, and interrupt/resume wired up.
