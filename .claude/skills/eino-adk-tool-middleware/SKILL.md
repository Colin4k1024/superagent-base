---
name: eino-adk-tool-middleware
description: >-
  Eino ADK WrapInvokableToolCall middleware pattern for intercepting all tool calls.
  TRIGGER when: implementing tool-level middleware (sandbox, logging, auth, metrics) in this project's Eino ADK agents.
  DO NOT TRIGGER when: working with agent-level middleware (timeout, retry) or non-Eino tool systems.
origin: learned
tags: [eino, adk, middleware, tool, go]
---

# Eino ADK Tool Middleware Pattern

How to implement custom middleware that intercepts every tool invocation in an Eino ADK ChatModelAgent.

## When to Activate

- Adding a new cross-cutting concern to all tool calls (security, logging, metrics, caching)
- Need to wrap or replace tool execution for specific tool names
- Building isolation/sandbox/audit layers around tool calls
- The project uses `github.com/cloudwego/eino/adk` with `ChatModelAgent`

## Solution / Pattern

Implement `adk.ChatModelAgentMiddleware` by embedding `*adk.BaseChatModelAgentMiddleware` and overriding `WrapInvokableToolCall`:

```go
package agentdef

import (
    "context"
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool"
)

var _ adk.ChatModelAgentMiddleware = (*myMiddleware)(nil)

type myMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    // custom fields
}

func (m *myMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
        // tCtx.Name = tool name, tCtx.CallID = unique call ID
        
        // Pre-execution logic (validate, log, set up environment)
        
        // Call original endpoint OR replace with custom execution
        result, err := endpoint(ctx, argumentsInJSON, opts...)
        
        // Post-execution logic (audit, transform output)
        
        return result, err
    }, nil
}
```

### Registration in resolveADKHandlers

Register the middleware in `pkg/agentdef/middlewares.go`:

```go
func resolveADKHandlers(ctx context.Context, specs []MiddlewareSpec, ...) ([]adk.ChatModelAgentMiddleware, error) {
    var handlers []adk.ChatModelAgentMiddleware
    // Auto-inject based on agent spec fields
    if someCondition {
        handlers = append(handlers, &myMiddleware{
            BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
        })
    }
    // ... existing switch on spec.Name
    return handlers, nil
}
```

### Injection via ChatModelAgentConfig.Handlers

The middleware list is passed to ADK during agent construction:

```go
adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:      def.Metadata.Name,
    Model:     chatModel,
    Handlers:  adkHandlers,  // []adk.ChatModelAgentMiddleware
    // ...
})
```

## Two-Tier Execution Strategy

For middleware that needs different behavior per tool type:

```go
func (m *myMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        if toolsRequiringFullIsolation[tCtx.Name] {
            // Fully replace: delegate to external backend
            return m.backend.Execute(ctx, tCtx.Name, args)
        }
        // Wrap: call original with constraints
        return m.executeWrapped(ctx, endpoint, tCtx.Name, args, opts...)
    }, nil
}
```

## Key Interface Details (eino v0.9.0-beta.1)

```go
// Endpoint signature
type InvokableToolCallEndpoint func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)

// Tool context
type ToolContext struct {
    Name   string  // tool name
    CallID string  // unique call identifier
}

// Base middleware (embed for no-op defaults)
type BaseChatModelAgentMiddleware = TypedBaseChatModelAgentMiddleware[*schema.Message]
```

## Notes

- `BaseChatModelAgentMiddleware` provides no-op defaults for all interface methods; only override what you need.
- Middleware order matters: first registered = outermost wrapper.
- `WrapInvokableToolCall` is called per-tool-call at request time, not at agent construction time.
- For streaming tools, override `WrapStreamableToolCall` instead.
- The `resolveADKHandlers` function in this project accepts extra parameters (sandbox spec, tool refs) to enable auto-injection logic.
