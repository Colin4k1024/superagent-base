# 架构概览

## 系统架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                        客户端层                                    │
│   Web UI (React)    │   API 客户端   │   gRPC 客户端              │
└──────────┬──────────┴───────┬────────┴──────────┬────────────────┘
           │ HTTP/SSE          │ REST              │ gRPC (port 50051)
┌──────────▼──────────────────▼───────────────────▼────────────────┐
│                        网关层 (Hertz)                              │
│   ContextCacheMW → RequestInspectorMW → SetHostMW → SetLogIDMW   │
│   CORS → AccessLogMW → OpenapiAuthMW → SessionAuthMW → I18nMW    │
└──────────────────────────────┬───────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                        API 层                                      │
│   backend/api/                                                     │
│   ├── router/          HTTP 路由注册 (Hertz 生成)                  │
│   ├── grpc/            gRPC 服务实现 (Agent / Conversation / ...)  │
│   └── middleware/      请求中间件                                  │
└──────────────────────────────┬───────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                        Application 层                              │
│   backend/application/                                             │
│   ├── singleagent/     单 Agent 对话编排                           │
│   ├── conversation/    会话管理                                    │
│   ├── knowledge/       知识库处理                                  │
│   ├── workflow/        工作流执行                                  │
│   ├── modelmgr/        模型配置管理                                │
│   └── ...                                                         │
└──────────┬───────────────────────────────────┬────────────────────┘
           │                                   │
┌──────────▼─────────────┐       ┌─────────────▼──────────────────┐
│    Domain 层            │       │    CrossDomain 层               │
│    backend/domain/      │       │    backend/crossdomain/         │
│    ├── agent/           │       │    ├── agentrun/    Agent 运行   │
│    ├── conversation/    │       │    ├── conversation/ 跨域会话    │
│    ├── knowledge/       │◄─────►│    ├── knowledge/  跨域知识库   │
│    ├── memory/          │       │    └── ...                      │
│    ├── plugin/          │       └─────────────────────────────────┘
│    └── workflow/        │
└──────────┬──────────────┘
           │
┌──────────▼───────────────────────────────────────────────────────┐
│                        Agent Runtime 层                            │
│   backend/pkg/agentdef/                                           │
│   ├── schema.go        YAML Schema 定义                           │
│   ├── parser.go        解析 + 校验                                 │
│   ├── loader.go        目录批量加载                                │
│   ├── builder.go       AgentBuilder → Eino Agent                  │
│   ├── runtime.go       AgentRuntime（生命周期管理）                │
│   ├── reload.go        Reloader（增量更新）                        │
│   └── watcher.go       Watcher（fsnotify 热重载）                  │
└──────────┬───────────────────────────────────────────────────────┘
           │ Eino SDK
┌──────────▼───────────────────────────────────────────────────────┐
│                        LLM 推理层                                  │
│   github.com/cloudwego/eino                                       │
│   ├── ChatModel (OpenAI-compatible)                               │
│   ├── ReAct Agent (工具调用循环，最多 10 步)                       │
│   └── Stream Reader (SSE 流式输出)                                 │
└──────────┬───────────────────────────────────────────────────────┘
           │ OpenAI-compatible API
┌──────────▼───────────────────────────────────────────────────────┐
│                        LLM 服务                                    │
│   LM Studio (local)  │  Ollama (local)  │  OpenAI / DeepSeek / ...│
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                        基础设施层                                  │
│   backend/infra/                                                  │
│   ├── orm/       MySQL (GORM)                                     │
│   ├── rdb/       Redis                                            │
│   ├── storage/   MinIO / TOS / S3                                 │
│   ├── es/        Elasticsearch（知识库检索，dev 可关闭）            │
│   ├── eventbus/  NSQ / Kafka / RabbitMQ（dev 可关闭）             │
│   ├── embedding/ 向量化（dev 可关闭）                              │
│   └── sse/       Server-Sent Events 流输出                        │
└──────────────────────────────────────────────────────────────────┘
```

---

## 模块依赖关系

```
main.go
  └── application.Init()
        ├── infra 初始化（MySQL / Redis / Storage / ...）
        ├── crossdomain 接口绑定
        ├── application service 注册
        └── agentdef.AgentRuntime.Start()

API 层 → Application 层 → Domain 层 → Infra 层
API 层 → agentdef.AgentRuntime（直接调用 Agent.Chat）
agentdef.AgentBuilder → Eino SDK → LLM
```

---

## 核心数据流：对话请求

### HTTP SSE 流

```
客户端 POST /api/chat
  │
  ▼ Hertz HTTP Server (:8888)
  │  中间件链（认证 / 日志 / CORS）
  │
  ▼ api/router → conversation handler
  │
  ▼ application/singleagent.Chat()
  │  从 AgentRuntime 获取 Agent 实例
  │
  ▼ agentdef.Agent.Chat(ctx, sessionID, message)
  │  ├── 构建 messages（system prompt + user）
  │  ├── einoChatModel.Stream(ctx, messages) 或
  │  └── einoReactAgent.Stream(ctx, messages)
  │       └── Eino ReAct 循环（最多 10 步工具调用）
  │
  ▼ <-chan string（token 流）
  │
  ▼ infra/sse.WriteStream() → HTTP SSE 响应
  │
客户端收到 text/event-stream
```

### gRPC 流

```
客户端 gRPC StreamChat (port 50051)
  │
  ▼ api/grpc/agent_handler.go
  │  AgentRuntime.GetAgent(name)
  │
  ▼ Agent.Chat(ctx, sessionID, message)
  │
  ▼ <-chan string
  │  for chunk := range ch { SendMsg(chunk) }
  │
客户端收到 server-side stream
```

---

## 组件说明

### backend/pkg/agentdef

Agent 声明式运行时核心，与业务逻辑解耦：

- `AgentDefinition`：YAML Schema Go 结构体（`schema.go`）
- `Parse` / `Validate`：解析校验（`parser.go`）
- `Loader`：目录批量加载，返回 `map[name]*AgentDefinition`（`loader.go`）
- `AgentBuilder`：将 Definition 构建为可运行 `Agent`（`builder.go`）
  - 无 ModelConfig → 返回 stub agent（测试用）
  - 有 ModelConfig，无工具 → `einoChatAgent`（直接 Stream）
  - 有 ModelConfig，有工具 → `einoReactAgent`（ReAct 循环）
- `AgentRuntime`：管理全部 Agent 生命周期，提供 `GetAgent` / `ListAgents`（`runtime.go`）
- `Watcher` + `Reloader`：fsnotify 热重载，2 秒防抖（`watcher.go` / `reload.go`）

### backend/bizpkg/config/modelmgr

模型配置管理：

- 从环境变量 `MODEL_PROTOCOL_0` 等读取单个模型配置（`initModelByEnv`）
- 从 `resources/conf/model/*.yaml` 读取多个预置模型（`initModelByTemplate`）
- `BUILTIN_CM_TYPE` + `BUILTIN_CM_<TYPE>_*` 控制知识库等内置能力的底层模型

### backend/infra

基础设施适配器，通过接口隔离：

| 目录 | 功能 |
|------|------|
| `orm/` | MySQL (GORM)，支持连接池、软删除 |
| `rdb/` | Redis，支持连接池、序列化 |
| `storage/` | 对象存储，支持 MinIO / TOS / S3 |
| `es/` | Elasticsearch v8，知识库全文检索 |
| `eventbus/` | 消息队列，支持 NSQ / Kafka / RMQ |
| `embedding/` | 向量化，支持 OpenAI / Ark / Ollama / Qwen / Gemini |
| `sse/` | Server-Sent Events 流输出 |

---

## 技术选型说明

| 组件 | 选型 | 原因 |
|------|------|------|
| HTTP 框架 | [Hertz](https://github.com/cloudwego/hertz) | CloudWeGo 高性能框架，原 Coze Studio 使用 |
| LLM SDK | [Eino](https://github.com/cloudwego/eino) | CloudWeGo AI 框架，原生支持 ReAct、Stream、工具调用 |
| gRPC | google.golang.org/grpc | 标准 gRPC Go 实现 |
| ORM | [GORM](https://gorm.io) + MySQL 8.x | 成熟稳定，支持迁移 |
| 缓存 | Redis 7 | 会话缓存、分布式锁 |
| 对象存储 | MinIO（开发）/ TOS / S3（生产） | 存储用户文件、模型 icon |
| 消息队列 | NSQ（默认）/ Kafka / RabbitMQ | Agent 异步任务、事件通知 |
| 文件监听 | [fsnotify](https://github.com/fsnotify/fsnotify) | Agent YAML 热重载 |
| 可观测性 | OpenTelemetry + Prometheus | 分布式追踪 + 指标，开发环境可关闭 |
