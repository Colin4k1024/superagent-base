# MCP 集成

## 概述

Superagent Base 同时支持 MCP Client 和 MCP Server：

- **Client**：连接外部 MCP Server，获取和调用其工具
- **Server**：将平台 Agent/Tool 暴露为 MCP Server，供外部工具调用

## MCP Client

### 支持的 Transport

| Transport | 描述 | 适用场景 |
|-----------|------|---------|
| stdio | 启动子进程，通过 stdin/stdout 通信 | 本地 MCP Server |
| SSE | HTTP SSE 流 + POST 请求 | 远程 MCP Server |

### 在 Agent 中使用 MCP 工具

```yaml
spec:
  tools:
    - ref: mcp://filesystem/read_file
    - ref: mcp://github/search_repos
```

格式：`mcp://<server-name>/<tool-name>`

### MCP Server 注册

通过配置文件注册 MCP Server：

```yaml
# configs/mcp-servers.yaml
servers:
  - name: filesystem
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
  - name: github
    transport: sse
    url: http://mcp-github:3000/sse
    env:
      GITHUB_TOKEN: your-token
```

## MCP Server

### 暴露 Agent 为 MCP 工具

平台的 Agent 和 Workflow 可以被外部 MCP Client（如 Claude Code、Cursor）调用：

```go
// 代码示例
server := mcp.NewServer("superagent", "1.0.0")

def, handler := mcp.ExposeAgentAsTool(
    "research-agent",
    "Research and analyze topics",
    func(ctx context.Context, msg string) (string, error) {
        agent, _ := runtime.GetAgent("research-agent")
        // ... call agent
    },
)
server.RegisterTool(def, handler)
```

### SSE 端点

MCP Server 通过 HTTP SSE 暴露：

- `GET /mcp/sse` — SSE 事件流（Server → Client）
- `POST /mcp/message` — JSON-RPC 请求（Client → Server）

### 外部工具连接示例

在 Claude Code 中使用：

```json
{
  "mcpServers": {
    "superagent": {
      "url": "http://localhost:8888/mcp/sse"
    }
  }
}
```
