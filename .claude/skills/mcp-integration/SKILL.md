---
name: mcp-integration
description: >
  Patterns for integrating MCP (Model Context Protocol) servers and tools into
  Superagent agents. TRIGGER when: wiring an MCP server into an agent YAML,
  implementing a new MCP client/server, debugging "mcp tool not found" errors,
  or when the user asks "how do I add an MCP tool", "MCP 怎么接入", "connect filesystem MCP".
  DO NOT TRIGGER when: writing pure builtin tools (use builtin/<name> directly)
  or REST API integrations unrelated to MCP.
origin: learned
tags: [mcp, tools, stdio, sse, agent, protocol]
---

# MCP Integration

Source: `backend/pkg/mcp/`. Supports **stdio** and **SSE** transports.

## Agent YAML — Use MCP Tool

```yaml
spec:
  tools:
    - mcp://filesystem/read_file
    - mcp://filesystem/write_file
    - mcp://brave-search/search
```

URI format: `mcp://<server-name>/<tool-name>`

## Server Registration (config)

```yaml
# configs/agents/my-agent.yaml
spec:
  mcp_servers:
    filesystem:
      transport: stdio
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
    brave-search:
      transport: sse
      url: http://localhost:3001/sse
      api_key: ${BRAVE_API_KEY}
```

## Programmatic Registration (Go)

```go
// stdio server
server := mcp.NewStdioServer("filesystem", mcp.StdioConfig{
    Command: "npx",
    Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/workspace"},
})
registry.Register(server)

// SSE server
server := mcp.NewSSEServer("brave-search", mcp.SSEConfig{
    URL:    "http://localhost:3001/sse",
    APIKey: os.Getenv("BRAVE_API_KEY"),
})
registry.Register(server)
```

## Implementing a Custom MCP Server

```go
// Implement mcp.Server interface
type MyServer struct{}

func (s *MyServer) Name() string { return "my-server" }
func (s *MyServer) Tools() []mcp.ToolDef {
    return []mcp.ToolDef{{
        Name:        "my_tool",
        Description: "Does X",
        InputSchema: mcp.Schema{...},
    }}
}
func (s *MyServer) Call(ctx context.Context, tool string, input map[string]any) (map[string]any, error) {
    // implement
}
```

## Debugging

- Tool not found: ensure server name in YAML matches `Name()` exactly.
- stdio timeout: increase `timeout_ms` in server config or check subprocess startup.
- SSE connection refused: verify the SSE server is running before agent starts.

```bash
# List registered MCP tools at runtime
GET /api/v1/tools?type=mcp
```

## Notes

- MCP tools are lazy-connected: the client connects on first invocation.
- The Registry (`pkg/mcp/registry.go`) is a singleton; servers registered at startup.
- Each MCP tool call goes through the tool middleware chain (retry/timeout/cache).
