---
name: a2ui-streaming
description: >
  Reference for the A2UI SSE streaming protocol used by Superagent agents to push
  typed events to the frontend. TRIGGER when: implementing a new event type,
  debugging streaming output, writing frontend SSE consumers, or when the user asks
  "how does streaming work", "what events does the agent emit", "A2UI 协议".
  DO NOT TRIGGER when: working on HTTP REST endpoints unrelated to streaming.
origin: learned
tags: [a2ui, streaming, sse, protocol, frontend, events]
---

# A2UI Streaming Protocol

A2UI is the typed SSE event protocol that connects agent execution to the UI.
Source of truth: `backend/pkg/a2ui/types.go`.

## Event Types

| EventType | Payload fields | Description |
|-----------|---------------|-------------|
| `text` | `content` (string) | Incremental text output |
| `thinking` | `content` (string) | Internal reasoning (collapsible in UI) |
| `tool_call` | `name`, `input` | Tool invocation start |
| `tool_result` | `name`, `output`, `error` | Tool result back |
| `code_block` | `language`, `content` | Fenced code with syntax highlight |
| `interrupt` | `reason`, `prompt` | Agent paused, waiting for user input |
| `error` | `code`, `message` | Non-fatal error message |
| `done` | `usage` | Stream complete, token usage summary |
| `progress` | `step`, `total`, `label` | Multi-step progress indicator |
| `agent_switch` | `from`, `to` | Orchestrator switched active sub-agent |

## SSE Wire Format

```
data: {"type":"text","content":"Hello"}\n\n
data: {"type":"tool_call","name":"datetime","input":{}}\n\n
data: {"type":"tool_result","name":"datetime","output":{"datetime":"2025-01-01"}}\n\n
data: {"type":"done","usage":{"prompt_tokens":100,"completion_tokens":50}}\n\n
```

## Sending Events (Go)

```go
// backend/pkg/a2ui/encoder.go
enc := a2ui.NewEncoder(w)          // w is http.ResponseWriter
enc.Send(a2ui.TextEvent("Hello"))
enc.Send(a2ui.ToolCallEvent("datetime", input))
enc.Send(a2ui.DoneEvent(usage))
```

## Interrupt / Resume Flow

```
Agent → interrupt event → Frontend shows prompt → User replies
→ POST /api/v1/chat/resume {session_id, answer}
→ Agent continues from checkpoint
```

```json
{"type":"interrupt","reason":"need_user_input","prompt":"Which timezone?"}
```

## Frontend Consumer (React)

```ts
const es = new EventSource(`/api/v1/chat/stream?session=${id}`)
es.onmessage = (e) => {
  const ev = JSON.parse(e.data)
  switch (ev.type) {
    case 'text':      appendText(ev.content); break
    case 'interrupt': showPrompt(ev.prompt);  break
    case 'done':      es.close();             break
  }
}
```

## Notes

- Always end a stream with a `done` event — the frontend closes the connection on receipt.
- `thinking` events are optional; the UI should render them collapsed by default.
- `agent_switch` is emitted by supervisor/sequential agents when control transfers.
