---
name: interrupt-resume
description: >
  Implement and debug agent interrupt/resume (human-in-the-loop checkpointing) in
  Superagent. TRIGGER when: enabling interrupt on an agent, implementing resume API,
  debugging "checkpoint not found" errors, adding human approval steps, or when the
  user asks "interrupt/resume 怎么用", "how does checkpoint work", "pause agent for approval".
  DO NOT TRIGGER when: building non-interactive batch agents or simple one-shot queries.
origin: learned
tags: [interrupt, resume, checkpoint, human-in-the-loop, streaming]
---

# Interrupt / Resume

Agents with `spec.interrupt.enabled: true` can pause mid-execution,
save state to a checkpoint, and resume after user input.

## Enable in Agent YAML

```yaml
spec:
  type: chat_model_agent   # works with all agent types
  interrupt:
    enabled: true
  checkpoint:
    backend: redis          # redis (persistent) | memory (dev only)
    ttl_seconds: 3600       # how long to keep checkpoint
```

## Interrupt Flow

```
1. Agent emits SSE:  {"type":"interrupt","reason":"need_approval","prompt":"Confirm delete?"}
2. Stream pauses (no "done" event yet)
3. Client stores session_id from response headers
4. User sees prompt → submits answer
5. Client calls: POST /api/v1/chat/resume
6. Agent continues from saved state
```

## Resume API

```http
POST /api/v1/chat/resume
Content-Type: application/json

{
  "session_id": "sess_abc123",
  "answer": "yes"
}
```

Response: same SSE stream as the original request, continuing from the interrupt point.

## Emitting an Interrupt (Go — custom agent)

```go
// Inside your agent Run() method:
if needsApproval {
    return nil, &a2ui.InterruptError{
        Reason: "need_approval",
        Prompt: "Are you sure you want to delete all records?",
    }
}
```

## Checkpoint Internals

- State stored as JSON in Redis key `checkpoint:{session_id}`.
- Contains: agent name, conversation history, tool call stack, interrupt reason.
- On resume: state is restored, execution continues from the interrupted node/step.

## Debugging

```bash
# Check if checkpoint exists
redis-cli GET "checkpoint:sess_abc123"

# Watch checkpoint creation/retrieval
grep "checkpoint" logs/app.log
```

Common errors:
- `checkpoint not found` — TTL expired or wrong `session_id`; ensure client stores the ID from the first response.
- `already resumed` — each checkpoint can only be resumed once; a new interrupt creates a new checkpoint.

## Frontend Pattern (React)

```ts
let sessionId: string

// First request
const res = await fetch('/api/v1/chat', { method: 'POST', body: ... })
sessionId = res.headers.get('X-Session-Id')!

// On interrupt event
if (event.type === 'interrupt') {
  const answer = await showDialog(event.prompt)
  await fetch('/api/v1/chat/resume', {
    method: 'POST',
    body: JSON.stringify({ session_id: sessionId, answer })
  })
}
```
