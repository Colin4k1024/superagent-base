# ADR: Frozen A2UI Event Schema Contract

> **Date:** 2026-09-07
> **Issue:** #16 (Go+Python migration unified plan, P1)
> **Status:** Accepted

## Context

The Go+Python migration unified plan (#16) designates the `/api/v2` HTTP/SSE
contract and the A2UI SSE event schema as the **canonical interop surface**
between the Go and Python runtime lanes. Both runtimes must emit/translate to
this schema — never to each other's internal types.

Before the Python lane (DeerFlow migration) begins, the contract must be
frozen so that both sides have a stable target.

## Decision

The A2UI event schema defined in `backend/pkg/a2ui/event.go` is the frozen
contract. The contract tests in `backend/pkg/a2ui/contract_test.go` are the
executable specification. No field name, type, or SSE encoding format may
change without:

1. Updating the contract tests
2. Mirroring the change on the Python side
3. Recording the change in this ADR

## Contract summary

### Event envelope

Every event is a JSON object with three fields:

```json
{"type": "<event_type>", "timestamp": <unix_millis>, "data": <data_or_null>}
```

### Event types

| Type | Data struct | Key fields (JSON) |
|------|------------|---------------------|
| `text` | `TextData` | `content`, `delta` |
| `thinking` | `ThinkingData` | `content`, `delta` |
| `tool_call` | `ToolCallData` | `id`, `name`, `arguments`, `status` |
| `tool_result` | `ToolResultData` | `id`, `name`, `result`, `is_error` |
| `code_block` | `CodeBlockData` | `language`, `code`, `delta` |
| `interrupt` | `InterruptData` | `reason`, `fields[]` |
| `error` | `ErrorData` | `code`, `message` |
| `done` | `null` | — |
| `progress` | `ProgressData` | `agent_name`, `step`, `total`, `current` |
| `agent_switch` | `AgentSwitchData` | `from_agent`, `to_agent`, `reason` |

### SSE encoding

Named SSE frame format: `event: <type>\ndata: <json>\n\n`

Compatible (data-only) format:
- `text` events: `data: <delta>\n\n`
- `done` events: `data: [DONE]\n\n`
- All other types fall back to full JSON in the data line.

### Ordering invariants

1. `tool_call` must precede its corresponding `tool_result` (matched by `id`).
2. `done` must be the terminal event — no events after it.
3. All `text` deltas must precede `done`.

## Runtime selector

Agent YAML now supports a `runtime:` field:

```yaml
spec:
  type: chat_model_agent
  runtime: go      # default; Go backend handles the hot path
  # runtime: python  # delegated to internal Python lane
```

- `go` (default): the Go backend (ADK Go / eino DAG engine) is the sole
  externally exposed entrypoint via `/api/v2`.
- `python`: the agent is delegated to the internal Python lane (DeerFlow/
  LangGraph), reached via a private Go-side proxy. Python services are never
  exposed directly to clients.

## Enforcement

The contract is enforced by `backend/pkg/a2ui/contract_test.go`, which
validates:

- Every event type's JSON round-trip with expected field names
- SSE encoding format for all 10 event types
- Compatible encoding for text and done
- Event ordering invariants (tool_call before tool_result, done is terminal)
- Event envelope shape (type, timestamp, data fields)
