# Go SDK

轻量级 Go SDK，封装 Superagent 基座核心能力，大幅降低使用门槛。

## 快速开始

### 安装

```bash
go get github.com/superagent-ai/superagent-base/sdk
```

### 最简用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/superagent-ai/superagent-base/sdk"
)

func main() {
    agent, err := sdk.LoadAgent("configs/agents/research-agent.yaml")
    if err != nil {
        log.Fatal(err)
    }

    result, err := agent.ChatSync(context.Background(), "", "什么是量子计算？")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
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
if err != nil {
    log.Fatal(err)
}
defer rt.Shutdown()

agent, ok := rt.GetAgent("research-agent")
if !ok {
    log.Fatal("agent not found")
}
```

## 流式对话

```go
ch, err := agent.Chat(ctx, "session-1", "Hello")
if err != nil {
    log.Fatal(err)
}

for chunk := range ch {
    fmt.Print(chunk)
}
fmt.Println()
```

## 工具注册

```go
import "github.com/superagent-ai/superagent-base/sdk/tool"

registry := tool.NewRegistry()

registry.Register(tool.New("web_search", "Search the web", func(ctx context.Context, args map[string]any) (map[string]any, error) {
    query := args["query"].(string)
    return map[string]any{"results": []string{"Result for: " + query}}, nil
}))

registry.Register(tool.New("calculate", "Calculate math", func(ctx context.Context, args map[string]any) (map[string]any, error) {
    return map[string]any{"result": "42"}, nil
}))
```

## 工具中间件

```go
import (
    "time"
    "github.com/superagent-ai/superagent-base/sdk/tool"
)

chain := tool.Chain(
    tool.RetryMiddleware(3, time.Second),
    tool.TimeoutMiddleware(30 * time.Second),
    tool.LogMiddleware(func(name string, args map[string]any, err error) {
        log.Printf("Tool %s called: %v (err: %v)", name, args, err)
    }),
)
```

## MCP 集成

```go
import "github.com/superagent-ai/superagent-base/sdk/mcp"

registry := mcp.NewRegistry()

err := registry.Connect(ctx, mcp.ServerConfig{
    Name:      "filesystem",
    Transport: "stdio",
    Command:   "npx",
    Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
})
if err != nil {
    log.Fatal(err)
}

client, ok := registry.GetClient("filesystem")
if ok {
    tools, _ := client.ListTools(ctx)
    fmt.Printf("MCP tools: %v\n", tools)
}
```

## 技能系统

```go
import (
    "time"
    "github.com/superagent-ai/superagent-base/sdk/skill"
)

invoker := skill.NewLocalInvoker()
manager := skill.NewManager(invoker)

manager.RegisterLocal(skill.SkillMeta{
    Name:        "datetime",
    Version:     "1.0.0",
    Description: "Get current date/time",
}, func(ctx context.Context, input map[string]any) (map[string]any, error) {
    return map[string]any{"date": time.Now().Format(time.RFC3339)}, nil
})

result, err := manager.Invoke(ctx, "datetime", map[string]any{})
```

## 记忆系统

```go
import "github.com/superagent-ai/superagent-base/sdk/memory"

mem := memory.NewBuiltin()

err := mem.AddMessage(ctx, "session-1", memory.Message{
    Role:    "user",
    Content: "Hello",
})

msgs, err := mem.GetMessages(ctx, "session-1", 10)
```

## 模型路由

```go
import "github.com/superagent-ai/superagent-base/sdk/model"

registry := model.NewRegistry()

registry.Register(model.Config{
    Name:     "openai",
    Provider: "openai",
    BaseURL:  "https://api.openai.com/v1",
    APIKey:   "sk-xxx",
    ModelID:  "gpt-4o",
})

registry.Register(model.Config{
    Name:     "deepseek",
    Provider: "deepseek",
    BaseURL:  "https://api.deepseek.com/v1",
    APIKey:   "sk-xxx",
    ModelID:  "deepseek-r1",
})

router := model.NewRouter(registry)
cfg, ok := router.Resolve("gpt-4o")
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/superagent-ai/superagent-base/sdk"
    "github.com/superagent-ai/superagent-base/sdk/tool"
)

func main() {
    // 注册工具
    registry := tool.NewRegistry()
    registry.Register(tool.New("web_search", "Search the web", func(ctx context.Context, args map[string]any) (map[string]any, error) {
        query := args["query"].(string)
        return map[string]any{"results": []string{"Result for: " + query}}, nil
    }))

    // 创建 Runtime
    rt, err := sdk.NewRuntime(
        sdk.WithAgentsDir("configs/agents"),
        sdk.WithModel(sdk.ModelRuntimeConfig{
            BaseURL: "http://localhost:8000/v1",
            APIKey:  "sk-xxx",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer rt.Shutdown()

    // 获取 Agent
    agent, ok := rt.GetAgent("research-agent")
    if !ok {
        log.Fatal("agent not found")
    }

    // 流式对话
    ch, err := agent.Chat(context.Background(), "session-1", "什么是量子计算？")
    if err != nil {
        log.Fatal(err)
    }

    for chunk := range ch {
        fmt.Print(chunk)
    }
    fmt.Println()
}
```

## API 参考

### sdk 包

| 函数/类型 | 说明 |
|----------|------|
| `LoadAgent(path)` | 从 YAML 文件加载 Agent |
| `NewRuntime(opts...)` | 创建 Runtime 实例 |
| `WithAgentsDir(dir)` | 设置 Agent 目录 |
| `WithModel(cfg)` | 设置模型配置 |
| `WithRedis(addr, pwd)` | 设置 Redis 连接 |
| `Agent` | Agent 接口 |
| `AgentDefinition` | Agent 定义结构 |

### Agent 接口

| 方法 | 说明 |
|------|------|
| `Name()` | 获取 Agent 名称 |
| `Description()` | 获取描述 |
| `Chat(ctx, sessionID, message)` | 流式对话 |
| `ChatSync(ctx, sessionID, message)` | 同步对话 |
| `GetDefinition()` | 获取定义 |

### tool 包

| 函数/类型 | 说明 |
|----------|------|
| `New(name, desc, fn)` | 创建工具 |
| `NewRegistry()` | 创建工具注册表 |
| `Chain(mws...)` | 组合中间件 |
| `RetryMiddleware(n, backoff)` | 重试中间件 |
| `TimeoutMiddleware(timeout)` | 超时中间件 |

### mcp 包

| 函数/类型 | 说明 |
|----------|------|
| `NewRegistry()` | 创建 MCP 注册表 |
| `Connect(ctx, cfg)` | 连接 MCP 服务器 |
| `GetClient(name)` | 获取客户端 |
| `ListServers()` | 列出服务器 |

### skill 包

| 函数/类型 | 说明 |
|----------|------|
| `NewLocalInvoker()` | 创建本地调用器 |
| `NewManager(invoker)` | 创建技能管理器 |
| `RegisterLocal(meta, fn)` | 注册本地技能 |
| `Invoke(ctx, name, input)` | 调用技能 |

### memory 包

| 函数/类型 | 说明 |
|----------|------|
| `NewBuiltin()` | 创建内置记忆 |
| `GetMessages(ctx, sessionID, limit)` | 获取消息 |
| `AddMessage(ctx, sessionID, msg)` | 添加消息 |
| `Clear(ctx, sessionID)` | 清空记忆 |

### model 包

| 函数/类型 | 说明 |
|----------|------|
| `NewRegistry()` | 创建模型注册表 |
| `Register(cfg)` | 注册模型 |
| `NewRouter(registry)` | 创建路由器 |
| `Resolve(modelID)` | 解析模型 |
