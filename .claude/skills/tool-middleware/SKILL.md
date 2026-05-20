---
name: tool-middleware
description: >
  Configure and extend the tool middleware chain (retry, timeout, rate-limit, cache)
  in Superagent. TRIGGER when: adding retry logic to a tool, configuring per-tool
  timeouts, implementing tool response caching, debugging "tool timeout" errors,
  or when the user asks "how do I add retry to a tool", "工具超时怎么设置", "tool middleware".
  DO NOT TRIGGER when: implementing the tool's core logic or writing MCP servers.
origin: learned
tags: [tool, middleware, retry, timeout, cache, rate-limit]
---

# Tool Middleware

Source: `backend/pkg/tool/`. Every tool call passes through a configurable middleware chain.

## Default Chain Order

```
ContextCache → RateLimit → Timeout → Retry → [Tool Function]
```

## Agent YAML Configuration

```yaml
spec:
  tool_config:
    defaults:
      timeout_ms: 10000
      retry:
        max_attempts: 3
        backoff: exponential
        initial_delay_ms: 200
    per_tool:
      mcp://brave-search/search:
        timeout_ms: 5000
        retry:
          max_attempts: 2
      builtin/calculator:
        cache:
          ttl_seconds: 0   # disable cache for non-deterministic tools
```

## Middleware Reference

### Retry

```yaml
retry:
  max_attempts: 3          # total attempts (first call + retries)
  backoff: exponential     # exponential | linear | fixed
  initial_delay_ms: 200    # delay before first retry
  max_delay_ms: 5000       # cap on exponential backoff
  retryable_errors:        # defaults: timeout, 5xx, network errors
    - timeout
    - rate_limit
```

### Timeout

```yaml
timeout_ms: 10000   # wall-clock timeout per tool call (includes retries)
```

### Rate Limit

```yaml
rate_limit:
  requests_per_minute: 60
  burst: 10
```

### Context Cache

Caches tool results keyed by `(tool_name, input_hash)` within a single agent turn.
Prevents duplicate calls when the LLM re-calls the same tool with identical input.

```yaml
cache:
  ttl_seconds: 300    # 0 = disabled, -1 = session-scoped (cleared at turn end)
  max_size_kb: 512    # evict LRU entries when exceeded
```

## Programmatic Middleware (Go)

```go
// Add custom middleware to the tool manager
mgr := tool.NewManager()
mgr.Use(tool.TimeoutMiddleware(10 * time.Second))
mgr.Use(tool.RetryMiddleware(tool.RetryConfig{MaxAttempts: 3}))
mgr.Use(myCustomMiddleware)   // implements tool.Middleware interface
```

```go
// Middleware interface
type Middleware func(next InvokeFunc) InvokeFunc
type InvokeFunc func(ctx context.Context, name string, input map[string]any) (map[string]any, error)
```

## Debugging

```bash
# See which middleware is applied per call
APP_LOG_LEVEL=debug make dev-server | grep "tool_middleware"

# Count retries
grep "tool_retry" logs/app.log | wc -l
```

Common issues:
- `tool timeout` with low `timeout_ms` → increase or set per-tool override.
- Duplicate LLM tool calls → enable `cache` with short TTL.
- Rate limit errors from external APIs → configure `rate_limit` below provider limits.
