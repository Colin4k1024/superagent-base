# Changelog

All notable changes to this project will be documented in this file.

> **Note**: Versions 0.1.0–0.4.0 were developed as sequential internal phases and
> published together on the same date (2026-05-11). Future releases will carry
> distinct dates reflecting actual public release timelines.

## [0.4.0] - 2026-05-11

### Phase 4：Interrupt/Resume + A2UI + Workflow + 文档完善

#### Added
- **Interrupt/Resume 中断/恢复系统**
  - `InterruptableAgent` 包装器：自动检测模型输出中的确认关键词（12 个英文模式）
  - `CheckpointStore` 接口：内存（默认）和 Redis 两种持久化后端
  - `InterruptState` / `InputField` 数据结构，支持 text / confirm / select 表单类型
  - HTTP API：`POST /api/v1/chat/resume`（恢复对话）、`GET /api/v1/chat/interrupt_state`（查询状态）
  - Agent YAML `spec.interrupt` 配置块（enabled / checkpoint_backend / timeout_seconds）
  - 中断状态过期自动清理（默认 5 分钟）

- **A2UI 协议（Agent-to-UI Structured Events）**
  - `pkg/a2ui/event.go`：10 种结构化事件类型（text / thinking / tool_call / tool_result / code_block / interrupt / error / done / progress / agent_switch）
  - `pkg/a2ui/encoder.go`：`EncodeSSE`（命名事件帧）和 `EncodeCompatible`（向后兼容模式）
  - `ChatSSEHandler` 支持 A2UI 模式，通过 `X-A2UI: true` 头或 `?a2ui=true` 参数激活
  - `EventAgent` / `EventStream` 将原始 token 流转换为结构化事件流

- **Workflow 图执行（Graph Tool）**
  - `WorkflowAgent`：DAG 执行引擎，Kahn 算法拓扑排序
  - 5 种节点类型：`llm_call` / `agent_call` / `tool_call` / `code` / `condition`
  - 边定义支持条件跳转（`condition` 字段）
  - 共享状态机（`state map[string]string`）+ `{{.varName}}` 模板语法
  - `input_mapping`（`$.key` 和模板语法）+ `variables`（命名别名）
  - Agent YAML `spec.workflow` 配置块（nodes / edges / variables）
  - `spec.type: workflow` 类型支持

- **新 HTTP 端点**
  - `GET /api/v1/agents`：列出所有已加载 Agent（name + description）
  - `POST /api/v1/chat/resume`：恢复中断会话（SSE 流）
  - `GET /api/v1/chat/interrupt_state`：查询中断状态

- **完善 Agent YAML Schema**
  - 新增 `spec.interrupt`（InterruptConfig）字段
  - 新增 `spec.sub_agents`（[]SubAgentRef）字段
  - 新增 `spec.orchestration`（OrchestrationSpec）字段
  - 新增 `spec.workflow`（WorkflowSpec）字段
  - `spec.type` 新增 `supervisor` / `sequential` / `parallel` / `workflow` 类型
  - `WorkflowSpec`：nodes / edges / variables
  - `WorkflowNode`：id / type / agent / tool / prompt / code / language / condition / input_mapping
  - `WorkflowEdge`：from / to / condition
  - `WorkflowVariable`：name / from

- **文档（全量更新）**
  - 重写 `README.md`：完整特性列表、架构图、API 快速参考
  - 重写 `docs/agent-yaml-spec.md`：覆盖所有 5 种 Agent 类型和全部配置字段
  - 重写 `docs/architecture.md`：更新模块地图、数据流、Builder 决策树
  - 新增 `docs/a2ui-protocol.md`：A2UI 事件类型、SSE 格式、客户端集成
  - 新增 `docs/interrupt-resume.md`：中断检测、HTTP API、端到端示例
  - 新增 `docs/workflow-guide.md`：节点类型、边定义、变量映射、完整示例
  - 新增 `docs/skill-development.md`：内置技能、本地技能、HTTP 技能托管
  - 更新 `docs/deployment.md`：新增 Agent 运行时环境变量（AGENT_CONFIG_DIR 等）

---

## [0.3.0] - 2026-05-11

### Phase 3：Multi-Agent 编排 + SkillsHub 技能系统

#### Added
- **Multi-Agent 编排**
  - `SupervisorAgent`：主 LLM 协调多个子 Agent，将 sub_agents 元数据注入 system prompt
  - `SequentialAgent`：有序执行 sub_agents，前一个输出作为下一个输入
  - `ParallelAgent`：并发执行所有 sub_agents（`sync.WaitGroup`），合并输出
  - `AgentBuilder` 新增 `WithAgentRegistry` 选项，支持 sub_agents 解析
  - `OrchestrationSpec`（mode / max_rounds）
  - `SubAgentRef`（ref / role / config）

- **SkillsHub 技能系统**
  - `pkg/skill/invoker.go`：LocalInvoker / HTTPInvoker（POST /invoke）/ CompositeInvoker
  - `pkg/skill/manager.go`：Install / GetTool / ListInstalled / Uninstall
  - `pkg/skill/adapter.go`：将技能包装为 Eino `InvokableTool`
  - `pkg/skill/client.go`：HubClient 接口（从 SkillsHub 拉取 SkillMeta）
  - `pkg/skill/builtin/builtin.go`：内置技能 datetime / calculator / uuid
  - `AgentBuilder` 新增 `WithSkillManager` 选项，支持 `skill://` 引用
  - `spec.tools` 支持 `skill://<name>` URI

- **工具中间件链**
  - `pkg/tool/middleware.go`：retry / timeout / rate-limit / cache 中间件
  - 中间件可配置并组合为管道

---

## [0.2.0] - 2026-05-11

### Phase 2：Model Router + MCP + Memory 适配器

#### Added
- **Model Router（模型路由）**
  - `pkg/modelrouter/router.go`：Router 接口 + 路由请求/响应结构体
  - `pkg/modelrouter/strategy.go`：capability-based / cost-optimized / latency 三种策略
  - `pkg/modelrouter/loader.go`：从 `configs/models/` YAML 文件加载模型配置
  - `AgentBuilder` 新增 `WithModelRouter` 选项
  - `spec.model.router` 字段支持命名路由策略
  - `spec.model.fallback` 字段支持自动降级

- **MCP Client + Server**
  - `pkg/mcp/client.go`：MCP 客户端，消费外部 MCP 工具
  - `pkg/mcp/transport_stdio.go`：stdio 传输（子进程通信）
  - `pkg/mcp/transport_sse.go`：SSE 传输（HTTP 流）
  - `pkg/mcp/server.go` / `server_http.go`：暴露平台能力作为 MCP 端点
  - `pkg/mcp/registry.go`：注册和管理 MCP 服务器
  - `pkg/mcp/eino_adapter.go`：将 MCP 工具适配为 Eino InvokableTool
  - `AgentBuilder` 新增 `WithMCPRegistry` 选项
  - `spec.tools` 支持 `mcp://<server>/<tool>` URI

- **Memory 适配器**
  - `pkg/memory/` 统一接口（Backend / BackendConfig）
  - builtin（会话内记忆）/ Mem0 / Zep / Letta 四种适配器
  - `AgentBuilder` 新增 `WithMemoryFactory` 选项
  - `spec.memory.backend` 和 `spec.memory.config` 字段

---

## [0.1.0] - 2026-05-11

### Phase 1：基础框架 + Agent Runtime

#### Added
- **项目初始化**
  - 基于 Coze Studio 分叉（Apache 2.0）
  - Go module 路径：`github.com/superagent-ai/superagent-base/backend`
  - Go 1.24 + Eino 框架

- **声明式 Agent 定义系统**
  - `pkg/agentdef/schema.go`：AgentDefinition / Metadata / AgentSpec / ModelSpec / ToolRef / MemorySpec / MiddlewareSpec / ObsSpec
  - `pkg/agentdef/parser.go`：Parse + Validate（6 项校验规则）
  - `pkg/agentdef/loader.go`：目录批量加载，返回 `map[name]*AgentDefinition`
  - `pkg/agentdef/builder.go`：AgentBuilder → 可运行 Agent（stub / einoChatAgent / einoReactAgent）
  - `pkg/agentdef/runtime.go`：AgentRuntime（GetAgent / ListAgents，`sync.RWMutex` 保护）
  - `pkg/agentdef/watcher.go`：fsnotify 热重载，2 秒防抖
  - `pkg/agentdef/reload.go`：增量 ReloadDir

- **LLM 集成**
  - Eino ChatModel（OpenAI-compatible）
  - Eino ReAct Agent（工具调用循环，最多 10 步）
  - 支持 7 种 Provider：OpenAI / Claude / Gemini / DeepSeek / Ark / Ollama / Qwen

- **内置工具**
  - `pkg/tool/builtin/`：web_search / http_request / code_execute

- **流式 API**
  - HTTP SSE：`POST /api/v1/chat/stream`（Legacy 模式）
  - gRPC：AgentService / ConversationService / ModelService / ToolService

- **可观测性**
  - OpenTelemetry 分布式追踪（可选）
  - Prometheus 指标（`/metrics` 端点）

- **Web UI**
  - React + TypeScript + Vite + Tailwind 流式对话页面

- **部署**
  - Docker Compose dev 栈（MySQL + Redis + backend）
  - Docker Compose debug 栈（全服务）
  - Kubernetes Helm Chart（`helm/charts/opencoze/`）

- **CLI**
  - `sactl` 命令行工具（技能管理、Agent 管理）

- **配置示例**
  - `configs/agents/research-agent.yaml`
  - `configs/agents/code-review-agent.yaml`

- **文档**
  - docs/architecture.md
  - docs/agent-yaml-spec.md
  - docs/model-config.md
  - docs/deployment.md
