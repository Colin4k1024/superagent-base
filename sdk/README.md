# Superagent Go SDK

轻量级 Go SDK，封装 Superagent 基座核心能力。

## 快速开始

```go
import "github.com/superagent-ai/superagent-base/sdk"

// 加载 Agent
agent, err := sdk.LoadAgent("configs/agents/my-agent.yaml")

// 对话
result, err := agent.ChatSync(ctx, "session-1", "Hello")

// 流式对话
ch, err := agent.Chat(ctx, "session-1", "Hello")
for chunk := range ch {
    fmt.Print(chunk)
}
```

## Runtime 管理

```go
rt, err := sdk.NewRuntime(
    sdk.WithAgentsDir("configs/agents"),
    sdk.WithModel(sdk.ModelRuntimeConfig{
        BaseURL: "http://localhost:8000/v1",
        APIKey:  "sk-xxx",
    }),
    sdk.WithRedis("localhost:6379", ""),
)
defer rt.Shutdown()

agent, ok := rt.GetAgent("my-agent")
```

## 工具注册

```go
import "github.com/superagent-ai/superagent-base/sdk/tool"

registry := tool.NewRegistry()
registry.Register(tool.New("web_search", "Search the web", func(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 实现搜索逻辑
    return map[string]any{"results": []string{}}, nil
}))
```

## MCP 集成

```go
import "github.com/superagent-ai/superagent-base/sdk/mcp"

registry := mcp.NewRegistry()
registry.Connect(ctx, mcp.ServerConfig{
    Name:      "filesystem",
    Transport: "stdio",
    Command:   "npx",
    Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
})
```

## 技能系统

```go
import "github.com/superagent-ai/superagent-base/sdk/skill"

invoker := skill.NewLocalInvoker()
manager := skill.NewManager(invoker)
manager.RegisterLocal(skill.SkillMeta{
    Name:        "datetime",
    Description: "Get current date/time",
}, func(ctx context.Context, input map[string]any) (map[string]any, error) {
    return map[string]any{"date": time.Now().Format(time.RFC3339)}, nil
})
```

## 目录结构

```
sdk/
├── agent.go          # Agent 接口 + 类型定义
├── builder.go        # AgentBuilder (从 YAML 构建)
├── runtime.go        # AgentRuntime (生命周期管理)
├── option.go         # 功能选项模式
├── tool/             # 工具接口 + 中间件
├── mcp/              # MCP 客户端
├── skill/            # 技能系统
├── memory/           # 记忆系统
├── model/            # 模型路由
├── event/            # 事件类型
└── examples/         # 示例代码
```
