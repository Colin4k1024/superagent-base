# 架构概览

## 系统分层架构图

```
┌────────────────────────────────────────────────────────────────────────┐
│                            客户端层                                      │
│   Web UI (React+Vite+Tailwind)  │  API 客户端  │  gRPC 客户端 (50051)  │
└────────────┬────────────────────┴──────────────┴──────────┬────────────┘
             │ HTTP/SSE (:8888)                              │ gRPC
┌────────────▼──────────────────────────────────────────────▼────────────┐
│                        网关层（Hertz HTTP）                               │
│  ContextCacheMW → RequestInspectorMW → SetHostMW → SetLogIDMW          │
│  CORS → AccessLogMW → OpenapiAuthMW → SessionAuthMW → I18nMW           │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │
┌────────────────────────────────────▼───────────────────────────────────┐
│                             API 层 (backend/api/)                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ HTTP 端点（chat_sse.go）                                          │  │
│  │  POST /api/v1/chat/stream         流式对话（Legacy + A2UI 模式）  │  │
│  │  GET  /api/v1/agents              列出所有 Agent                   │  │
│  │  POST /api/v1/chat/resume         恢复中断会话                     │  │
│  │  GET  /api/v1/chat/interrupt_state 查询中断状态                   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ gRPC 服务                                                         │  │
│  │  AgentService / ConversationService / ModelService / ToolService  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼───────────────────────────────────┐
│                        Application 层 (backend/application/)             │
│  singleagent · conversation · knowledge · workflow · modelmgr ...      │
└─────────┬──────────────────────────────────────────────┬───────────────┘
          │                                              │
┌─────────▼──────────────┐             ┌────────────────▼────────────────┐
│   Domain / CrossDomain  │             │    Agent Runtime (pkg/agentdef/) │
│   agent · conversation  │             │  schema.go      YAML Schema 定义 │
│   knowledge · memory    │             │  parser.go      解析 + 校验      │
│   plugin · workflow     │             │  loader.go      目录批量加载     │
└────────────────────────┘             │  builder.go     定义 → Agent 实例│
                                       │  runtime.go     生命周期管理     │
                                       │  reload.go      增量更新         │
                                       │  watcher.go     热重载监听       │
                                       │  interrupt.go   中断/恢复        │
                                       │  workflow_builder.go  图执行     │
                                       │  orchestration.go  多 Agent 编排 │
                                       └──────────┬──────────────────────┘
                                                  │
┌─────────────────────────────────────────────────▼─────────────────────┐
│                         工具与技能层                                     │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/tool/                                                        │  │
│  │   Manager             工具生命周期管理                            │  │
│  │   middleware.go        retry / timeout / rate-limit / cache      │  │
│  │   builtin/             web_search / http_request / code_execute  │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/skill/                                                       │  │
│  │   LocalInvoker        注册并调用本地 SkillFunc                   │  │
│  │   HTTPInvoker         POST <endpoint>/invoke                     │  │
│  │   CompositeInvoker    本地优先，回退到 HTTP                      │  │
│  │   Manager             安装/查找/卸载技能                         │  │
│  │   builtin/            datetime / calculator / uuid               │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/mcp/                                                         │  │
│  │   Client              stdio + SSE 传输，消费外部 MCP 工具         │  │
│  │   Server              暴露平台能力作为 MCP 端点                   │  │
│  │   Registry            注册 MCP 服务器                            │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/modelrouter/                                                 │  │
│  │   capability-based / cost-optimized / latency 路由策略 + fallback│  │
│  └─────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/memory/            builtin / Mem0 / Zep / Letta 适配器      │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ pkg/a2ui/              Event 协议 + SSE 编码（EncodeSSE）         │  │
│  └─────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────┬──────────────────────────────────┘
                                     │ Eino SDK
┌────────────────────────────────────▼──────────────────────────────────┐
│                       LLM 推理层（cloudwego/eino）                       │
│  ChatModel（OpenAI-compatible）                                         │
│  ReAct Agent（工具调用循环，最多 10 步）                                  │
│  Stream Reader（token 流）                                               │
└────────────────────────────────────┬──────────────────────────────────┘
                                     │ OpenAI-compatible API
┌────────────────────────────────────▼──────────────────────────────────┐
│                    LLM Provider                                         │
│  LM Studio · Ollama · OpenAI · DeepSeek · Claude · Gemini · Qwen · Ark│
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│                       基础设施层（backend/infra/）                        │
│  orm/      MySQL 8.x（GORM，连接池，软删除）                            │
│  rdb/      Redis 7（会话缓存，分布式锁）                                │
│  storage/  MinIO / TOS / S3（对象存储）                                 │
│  es/       Elasticsearch 8（知识库全文检索，dev 可关闭）                 │
│  eventbus/ NSQ / Kafka / RabbitMQ（异步事件，dev 可关闭）               │
│  embedding/ 向量化（OpenAI/Ark/Ollama/Qwen/Gemini，dev 可关闭）        │
│  checkpoint/ Redis/内存 CheckpointStore（中断状态持久化）               │
│  sse/      Server-Sent Events 流输出                                    │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 核心数据流

### HTTP SSE 对话（Legacy 模式）

```
客户端 POST /api/v1/chat/stream
  │  Body: {"agent_id":"research-agent","session_id":"s1","message":"..."}
  │
  ▼ Hertz HTTP Server (:8888) → 中间件链
  │
  ▼ api/handler/coze/ChatSSEHandler.HandleChatStream()
  │  AgentRuntime.GetAgent("research-agent")
  │
  ▼ agent.Chat(ctx, sessionID, message)
  │  ├── einoChatAgent.Stream()  —— 无工具，直接调用 ChatModel
  │  └── einoReactAgent.Stream() —— 有工具，ReAct 循环（最多 10 步）
  │
  ▼ <-chan string（token 流）
  │
  ▼ SSE Writer → HTTP SSE 响应（text/event-stream）
  │  data: <token>\n\n
  │  data: [DONE]\n\n
  │
客户端接收流式 token
```

### HTTP SSE 对话（A2UI 模式）

```
客户端 POST /api/v1/chat/stream (Header: X-A2UI: true)
  │
  ▼ ChatSSEHandler → 检测到 X-A2UI 头
  │
  ▼ agentdef.NewEventAgent(agent).ChatWithEvents()
  │  包装 Agent，将原始 token 流转换为结构化事件
  │
  ▼ EventStream.Chan() → <-chan *a2ui.Event
  │  事件类型：text / thinking / tool_call / tool_result /
  │            code_block / interrupt / error / done /
  │            progress / agent_switch
  │
  ▼ a2ui.EncodeSSE(evt) → 命名 SSE 帧
  │  event: text\ndata: {"type":"text","timestamp":...,"data":...}\n\n
  │  event: done\ndata: {...}\n\n
  │
客户端使用 addEventListener("text", ...) 监听各类事件
```

### 中断/恢复数据流

```
1. 客户端 POST /api/v1/chat/stream
   │
   ▼ InterruptableAgent.Chat()
   │  → 调用内部 Agent，缓冲完整响应
   │  → 检测确认关键词（"please confirm" / "are you sure" 等）
   │  → 检测到确认请求：
   │    ├── 保存 InterruptState 到 CheckpointStore（redis/memory）
   │    └── 输出 "\x00INTERRUPT:{...json...}" 信号
   │
   ▼ 客户端收到中断信号，弹出确认表单

2. 客户端 POST /api/v1/chat/resume
   │  Body: {"agent_id":"approval-agent","session_id":"s1","input":{"confirm":true}}
   │
   ▼ ChatSSEHandler.HandleChatResume()
   │  → interruptable.Resume(ctx, sessionID, input)
   │  → 格式化用户输入为恢复消息
   │  → 调用内部 Agent 继续对话
   │
   ▼ 流式输出恢复后的响应
```

### gRPC 流式对话

```
客户端 gRPC StreamChat (:50051)
  │
  ▼ api/grpc/AgentHandler
  │  AgentRuntime.GetAgent(name)
  │
  ▼ agent.Chat(ctx, sessionID, message)
  │
  ▼ <-chan string
  │  for chunk := range ch { SendMsg(chunk) }
  │
客户端接收 server-side stream
```

---

## 模块依赖关系

```
main.go
  └── application.Init()
        ├── infra 初始化（MySQL / Redis / Storage / ...）
        ├── pkg/modelrouter.NewRouter()
        ├── pkg/tool.NewManager()
        ├── pkg/skill.NewManager()
        ├── pkg/mcp.NewRegistry()
        ├── pkg/memory.InitFactory()
        ├── agentdef.NewAgentBuilder(
        │     WithModelConfig / WithModelRouter / WithToolManager /
        │     WithMemoryFactory / WithMCPRegistry / WithSkillManager /
        │     WithAgentRegistry
        │   )
        ├── agentdef.NewAgentRuntime(builder)
        ├── agentdef.AgentRuntime.LoadDir()     # 初始加载
        ├── agentdef.NewWatcher() → agentdef.Reloader  # 热重载
        ├── crossdomain 接口绑定
        ├── application service 注册
        └── Hertz HTTP Server + gRPC Server 启动

API 层 → Application 层 → Domain 层 → Infra 层
API 层 → agentdef.AgentRuntime（直接调用 Agent.Chat）
agentdef.AgentBuilder → pkg/tool + pkg/skill + pkg/mcp + pkg/memory + pkg/modelrouter
agentdef.AgentBuilder → Eino SDK → LLM Provider
```

---

## pkg/ 模块地图

| 包路径 | 主要职责 |
|--------|---------|
| `pkg/agentdef` | Agent YAML 运行时：Schema / Parser / Builder / Runtime / Interrupt / Workflow / Orchestration |
| `pkg/a2ui` | A2UI 协议：事件类型定义（event.go）+ SSE 编码（encoder.go） |
| `pkg/modelrouter` | 模型路由：capability-based / cost-optimized / latency 策略 + fallback |
| `pkg/mcp` | MCP 协议：Client（stdio/SSE）/ Server / Registry / EinoAdapter |
| `pkg/memory` | 记忆后端：builtin / Mem0 / Zep / Letta 统一接口 |
| `pkg/skill` | 技能运行时：LocalInvoker / HTTPInvoker / CompositeInvoker / Manager / Cache |
| `pkg/skill/builtin` | 内置技能：datetime / calculator / uuid |
| `pkg/tool` | 工具管理：Manager + middleware（retry/timeout/rate-limit/cache）|
| `pkg/tool/builtin` | 内置工具：web_search / http_request / code_execute |
| `pkg/observe` | 可观测性：OpenTelemetry + Prometheus |
| `pkg/ctxcache` | 请求级 Context 缓存 |
| `pkg/logs` | 统一日志门面 |
| `pkg/errorx` | 统一错误类型 |
| `pkg/safego` | panic-safe goroutine 包装 |
| `pkg/taskgroup` | 并发任务组 |

---

## 技术选型

| 组件 | 选型 | 原因 |
|------|------|------|
| HTTP 框架 | [Hertz](https://github.com/cloudwego/hertz) | CloudWeGo 高性能框架，原 Coze Studio 使用 |
| LLM SDK | [Eino](https://github.com/cloudwego/eino) | CloudWeGo AI 框架，原生支持 ReAct/Stream/工具调用 |
| gRPC | google.golang.org/grpc | 标准 gRPC Go 实现 |
| ORM | GORM + MySQL 8.x | 成熟稳定，支持迁移 |
| 缓存 | Redis 7 | 会话缓存、分布式锁、中断状态持久化 |
| 对象存储 | MinIO（开发）/ TOS / S3（生产） | 存储用户文件、模型 icon |
| 消息队列 | NSQ（默认）/ Kafka / RabbitMQ | Agent 异步任务、事件通知 |
| 向量存储 | Milvus 2.5 | 知识库语义检索 |
| 文件监听 | fsnotify | Agent YAML 热重载 |
| 可观测性 | OpenTelemetry + Prometheus | 分布式追踪 + 指标 |
| Web UI | React + TypeScript + Vite + Tailwind | 现代前端，流式对话支持 |

---

## Agent Builder 决策树

```
AgentBuilder.Build(def)
  │
  ├── def.Spec.Type == "supervisor"  → buildSupervisor()
  │     ├── 解析 sub_agents
  │     ├── 为 supervisor 构建内置 chat_model_agent
  │     └── 返回 SupervisorAgent
  │
  ├── def.Spec.Type == "sequential"  → buildSequential()
  │     ├── 按声明顺序解析 sub_agents
  │     └── 返回 SequentialAgent（链式执行）
  │
  ├── def.Spec.Type == "parallel"    → buildParallel()
  │     ├── 解析 sub_agents
  │     └── 返回 ParallelAgent（并发执行）
  │
  ├── def.Spec.Type == "workflow"    → buildWorkflow()
  │     └── 返回 WorkflowAgent（DAG 拓扑排序执行）
  │
  └── 其他类型（chat_model_agent 等）
        ├── 解析模型 ID（可选 Model Router）
        ├── 解析工具引用（builtin / mcp / skill）
        ├── 初始化记忆后端
        ├── 无 ModelConfig → 返回 stub chatAgent（测试用）
        ├── 有 ModelConfig，无工具 → einoChatAgent（直接 Stream）
        └── 有 ModelConfig，有工具 → einoReactAgent（ReAct 循环）
              └── maybeWrapInterruptable（当 interrupt.enabled=true）
```
