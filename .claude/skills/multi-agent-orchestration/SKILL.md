---
name: multi-agent-orchestration
description: >
  Design multi-agent systems using Superagent's supervisor, sequential, and parallel
  agent types. TRIGGER when: building orchestration agents, routing tasks between
  sub-agents, designing fan-out/fan-in pipelines, or when the user asks
  "multi-agent 怎么设计", "supervisor agent", "parallel agents", "agent orchestration".
  DO NOT TRIGGER when: working on single-model agents (use agent-yaml-authoring)
  or workflow DAG nodes (use workflow-dag skill).
origin: learned
tags: [multi-agent, orchestration, supervisor, sequential, parallel, plan-execute]
---

# Multi-Agent Orchestration

Superagent supports four orchestration patterns beyond the single `chat_model_agent`.

## Two-Pass Build Rule

**Critical:** Leaf agents must be defined (and registered) before orchestration agents
that reference them. In `configs/agents/`, name files so leaves sort first, or use
separate files and ensure the runtime loads them in dependency order.

## 1. Supervisor (Dynamic Routing)

The supervisor LLM decides at runtime which sub-agent handles each message.

```yaml
spec:
  type: supervisor
  model: gpt-4o
  system_prompt: |
    Route tasks to the appropriate specialist agent.
    - For code questions: use code-assistant
    - For research questions: use research-agent
  agents:
    - name: code-assistant
    - name: research-agent
```

Agent switch events are emitted as `{"type":"agent_switch","from":"supervisor","to":"research-agent"}`.

## 2. Sequential (Pipeline)

Each sub-agent runs in order; output is piped to the next.

```yaml
spec:
  type: sequential
  agents:
    - name: extractor      # Step 1
    - name: summarizer     # Step 2: receives extractor output
    - name: formatter      # Step 3: receives summarizer output
```

## 3. Parallel (Fan-Out / Fan-In)

All sub-agents run simultaneously; results are merged.

```yaml
spec:
  type: parallel
  agents:
    - name: web-searcher
    - name: db-researcher
    - name: doc-reader
  merge_strategy: concat   # concat | first | vote
```

`merge_strategy`:
- `concat` — concatenate all outputs (default)
- `first` — return the first successful output
- `vote` — majority-vote across outputs (useful for fact-checking)

## 4. Plan-Execute

A planner agent generates a task list; an executor agent runs each task.

```yaml
spec:
  type: plan_execute
  planner:
    name: task-planner
  executor:
    name: task-executor
  max_iterations: 5
```

## Skill References Between Agents

Any agent can be used as a tool by another agent:

```yaml
# In the orchestrator's tools list:
tools:
  - skill://research-agent
  - skill://code-assistant
```

## Common Patterns

**Research + Write pipeline (sequential):**
```
research-agent → draft-writer → editor-agent
```

**Multi-source research (parallel):**
```
supervisor → [web-searcher, arxiv-searcher, db-searcher] → merge → synthesizer
```

**Approval gate (supervisor + interrupt):**
```
planner → supervisor routes to: executor | approver(interrupt)
```

## Debugging

```bash
# Trace agent switches in streaming output
grep "agent_switch" logs/app.log

# Check sub-agent registration order
grep "agent registered" logs/app.log | head -20
```
