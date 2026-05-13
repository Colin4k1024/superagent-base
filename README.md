# Superagent Base

**Superagent Base** 是一个开源 AI Agent 开发平台，基于 [Coze Studio](https://github.com/coze-dev/coze-studio) 构建。它为构建、部署和管理 AI Agent 提供完整的后端服务，支持声明式 YAML 定义、多模型路由、工具调用、多 Agent 编排、Workflow 图执行、中断/恢复、A2UI 协议流式输出，以及完整的可观测性栈。

---

## 核心特性

| 能力 | 说明 |
|------|------|
| **声明式 Agent 定义** | 使用 YAML 文件定义 Agent，支持 5 种类型（chat、supervisor、sequential、parallel、workflow），热重载无需重启 |
| **Model Router** | 基于能力、成本、延迟的路由策略 + 自动 fallback |
| **MCP Client + Server** | stdio + SSE 传输，消费外部 MCP 工具并对外暴露平台能力 |
| **Memory 适配器** | 4 种后端：builtin / Mem0 / Zep / Letta |
| **SkillsHub 技能系统** | Local / HTTP / Composite Invoker，内置 datetime / calculator / uuid |
| **Tool 中间件链** | retry / timeout / rate-limit / cache + 内置工具 web_search / http_request / code_execute |
| **多 Agent 编排** | Supervisor / Sequential / Parallel 三种模式 |
| **Workflow / Graph Tool** | DAG 执行，拓扑排序，节点类型：llm_call / agent_call / tool_call / code / condition |
| **中断/恢复** | 检测确认请求 → 保存 checkpoint → HTTP Resume API 恢复对话 |
| **A2UI 协议** | 结构化流式事件（text / thinking / tool_call / tool_result / code_block / interrupt / error / done / progress / agent_switch） |
| **OpenTelemetry + Prometheus** | 分布式追踪 + 指标（Agent 请求数/延迟/错误率、Model Token、Tool 调用、活跃会话数），Eino callback 自动上报 |
| **Monitor Dashboard** | 4 Tab 实时面板（Status / Metrics / Logs / Admin），纯 SVG 图表，SSE 日志流，热重载管理 |
| **监控栈一键部署** | Prometheus + Grafana + OTel Collector，`docker compose -f docker/docker-compose-monitoring.yml up -d` |
| **Admin API** | `GET /api/v1/admin/status`（运行状态）、`POST /api/v1/admin/reload`（热重载）、`GET /api/v1/admin/logs`（实时日志 SSE） |
| **gRPC API** | AgentService / ConversationService / ModelService / ToolService |
| **HTTP SSE 流式 API** | POST /api/v1/chat/stream，GET /api/v1/agents，POST /api/v1/chat/resume |
| **Web UI** | React + Vite + Tailwind，流式对话 + 监控面板 + 技能市场 |
| **Docker Compose + Helm** | 开发环境轻量栈 + 生产级 Kubernetes 部署 |
| **CLI 工具 sactl** | 技能管理和 Agent 管理命令行工具 |

---

## 架构总览

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          客户端层                                          │
│   Web UI (React+Vite)   │   HTTP/SSE 客户端   │   gRPC 客户端            │
└────────────┬────────────┴──────────┬───────────┴──────────┬──────────────┘
             │ HTTP SSE              │ REST                  │ gRPC :50051
┌────────────▼──────────────────────▼───────────────────────▼──────────────┐
│                       网关层 (Hertz HTTP :8888)                            │
│   CORS · 认证 · 日志 · RequestInspector · AccessLog                       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│                           API 层 (backend/api/)                            │
│   POST /api/v1/chat/stream   GET /api/v1/agents                           │
│   POST /api/v1/chat/resume   GET /api/v1/chat/interrupt_state             │
│   gRPC: AgentService / ConversationService / ModelService / ToolService   │
└─────────────────────┬──────────────────────────────────────────────────-─┘
                      │
┌─────────────────────▼────────────────────────────────────────────────────┐
│                       Application 层 (backend/application/)               │
│   singleagent · conversation · knowledge · workflow · modelmgr ...       │
└──────────────────┬─────────────────────────┬────────────────────────────-┘
                   │                         │
   ┌───────────────▼──────────┐  ┌───────────▼──────────────────────────┐
   │   Domain / CrossDomain   │  │     Agent Runtime (pkg/agentdef/)    │
   │   agent · conversation   │  │  schema · parser · builder · runtime  │
   │   knowledge · memory     │  │  interrupt · workflow · orchestration │
   └──────────────────────────┘  └──────────┬───────────────────────────┘
                                             │
   ┌─────────────────────────────────────────▼─────────────────────────┐
   │                       工具与技能层                                   │
   │  pkg/tool (Manager + middleware: retry/timeout/ratelimit/cache)    │
   │  pkg/skill (LocalInvoker + HTTPInvoker + CompositeInvoker)        │
   │  pkg/mcp (Client stdio/SSE + Server + Registry)                   │
   │  pkg/modelrouter (capability/cost/latency + fallback)             │
   │  pkg/memory (builtin / Mem0 / Zep / Letta)                       │
   │  pkg/a2ui (Event 协议 + SSE 编码)                                 │
   └─────────────────────────────────┬─────────────────────────────────┘
                                     │ Eino SDK (CloudWeGo)
   ┌─────────────────────────────────▼─────────────────────────────────┐
   │              LLM 推理层 (github.com/cloudwego/eino)                │
   │   ChatModel · ReAct Agent (最多 10 步) · Stream Reader            │
   └─────────────────────────────────┬─────────────────────────────────┘
                                     │ OpenAI-compatible API
   ┌─────────────────────────────────▼─────────────────────────────────┐
   │              LLM Provider                                          │
   │  LM Studio · Ollama · OpenAI · DeepSeek · Claude · Gemini · ...  │
   └────────────────────────────────────────────────────────────────────┘

   ┌────────────────────────────────────────────────────────────────────┐
   │  基础设施层 (backend/infra/)                                        │
   │  MySQL · Redis · MinIO/S3 · Elasticsearch · NSQ/Kafka · Milvus   │
   └────────────────────────────────────────────────────────────────────┘
```

---

## 快速开始

### 前置条件

- Go 1.24+
- Docker + Docker Compose
- 本地 LLM：[LM Studio](https://lmstudio.ai/) 监听 `http://127.0.0.1:8000/v1`（或 Ollama / 任意 OpenAI 兼容端点）

### 3 步启动

```bash
# 1. 克隆
git clone <repo-url>
cd superagent-base

# 2. （可选）配置模型：编辑 docker/.env.dev
#    填写 MODEL_API_KEY_0 和 MODEL_BASE_URL_0

# 3. 启动（MySQL + Redis + backend）
make dev
```

访问 `http://localhost:8888`，gRPC 在 `localhost:50051`。

```bash
# 停止
make dev-down
```

---

## API 快速参考

### HTTP SSE 流式 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/chat/stream` | 流式对话，支持 Legacy 和 A2UI 两种模式 |
| `GET` | `/api/v1/agents` | 列出所有已加载 Agent |
| `POST` | `/api/v1/chat/resume` | 恢复中断的对话 |
| `GET` | `/api/v1/chat/interrupt_state` | 查询会话中断状态 |

### 监控与管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/admin/status` | 系统运行状态（uptime、agents、health、ready） |
| `POST` | `/api/v1/admin/reload` | 触发 Agent 热重载 |
| `GET` | `/api/v1/admin/logs` | SSE 实时日志流（结构化 JSON） |
| `GET` | `/metrics` | Prometheus 指标端点 |
| `GET` | `/health` | 健康检查 |
| `GET` | `/ready` | 就绪检查（含 Agent Runtime 状态） |

**流式对话示例（Legacy 模式）：**

```bash
curl -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"研究一下量子计算"}' \
  --no-buffer
```

**A2UI 模式（结构化事件）：**

```bash
curl -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -H "X-A2UI: true" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"研究一下量子计算"}' \
  --no-buffer
```

### gRPC API

| 服务 | 说明 |
|------|------|
| `AgentService` | Agent 查询、对话、流式聊天 |
| `ConversationService` | 会话创建、历史管理 |
| `ModelService` | 模型列表、路由查询 |
| `ToolService` | 工具列表、调用 |

Proto 文件：`api/proto/`

---

## Agent YAML 快速参考

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent          # 唯一标识，匹配 [a-z0-9-]+
  version: "1.0.0"
spec:
  type: chat_model_agent  # chat_model_agent | supervisor | sequential | parallel | workflow
  model:
    primary: gpt-4o       # 主模型 ID
    fallback: deepseek-r1 # 备用模型
    router: capability-based  # 路由策略（可选）
  system_prompt: "你是一个助手..."
  tools:
    - ref: builtin/web_search
    - ref: mcp://my-server/search
    - ref: skill://calculator
  memory:
    backend: builtin      # builtin | mem0 | zep | letta
  interrupt:
    enabled: true
    checkpoint_backend: redis
    timeout_seconds: 300
  observability:
    tracing: true
    metrics: true
    log_level: info
```

**有效的 `spec.type` 值：**

| 类型 | 说明 |
|------|------|
| `chat_model_agent` | 标准对话 Agent，可挂载工具 |
| `deep_agent` | 深度推理模式，支持多步规划 |
| `supervisor` | 多 Agent 协调者，通过 LLM 决策分发给 sub_agents |
| `sequential` | 顺序执行 sub_agents，前一个输出作为下一个输入 |
| `parallel` | 并发执行所有 sub_agents，合并输出 |
| `plan_execute` | 先规划后执行的多 Agent 模式 |
| `workflow` | DAG 图执行，通过 spec.workflow 定义节点和边 |

---

## Make 命令

| 命令 | 说明 |
|------|------|
| `make dev` | 启动 MySQL + Redis + backend（开发推荐） |
| `make dev-middleware` | 仅启动 MySQL + Redis |
| `make dev-server` | 构建并运行 backend（需先启动 middleware） |
| `make dev-down` | 停止 dev 容器 |
| `make dev-clean` | 停止容器并删除 MySQL 数据 |
| `make build` | 构建 `bin/superagent` |
| `make test` | 运行 `pkg/` 测试 |
| `make test-all` | 运行全部测试 |
| `make debug` | 启动完整 debug 环境（含 ES / MinIO / NSQ 等） |

### 监控栈

```bash
# 启动 Prometheus + Grafana + OTel Collector
docker compose -f docker/docker-compose-monitoring.yml up -d

# 访问
# Prometheus: http://localhost:9090
# Grafana:    http://localhost:3001 (admin/admin)
# OTel gRPC:  localhost:4317

# 启动后端时开启 OTel 追踪
export OTEL_ENABLED=true
export OTEL_ENDPOINT=localhost:4317
export SERVICE_NAME=superagent
make dev-server
```

**可观测性指标**:

| 指标 | 说明 |
|------|------|
| `superagent_agent_requests_total{agent_id, status}` | Agent 请求计数（区分 legacy/a2ui 模式） |
| `superagent_agent_request_duration_seconds{agent_id}` | 请求延迟直方图 |
| `superagent_active_sessions` | 当前并发会话数 |
| `superagent_model_tokens_total{model_id, provider, type}` | Model Token 消耗 |
| `superagent_model_errors_total{model_id, provider, error_type}` | Model 调用错误 |
| `superagent_tool_invocations_total{tool_name, status}` | Tool 调用计数 |
| `superagent_agent_reload_failures_total{agent_id}` | 热重载失败计数 |

---

## 文档

| 文档 | 说明 |
|------|------|
| [docs/agent-yaml-spec.md](docs/agent-yaml-spec.md) | Agent YAML 完整规范：所有类型、所有字段、完整示例 |
| [docs/architecture.md](docs/architecture.md) | 系统架构、数据流、模块依赖 |
| [docs/model-config.md](docs/model-config.md) | 模型配置：LM Studio / Ollama / OpenAI / DeepSeek / Claude 等 |
| [docs/deployment.md](docs/deployment.md) | 部署指南：本地开发 / Docker Compose / Kubernetes Helm |
| [docs/a2ui-protocol.md](docs/a2ui-protocol.md) | A2UI 协议：事件类型、SSE 格式、客户端集成 |
| [docs/interrupt-resume.md](docs/interrupt-resume.md) | 中断/恢复：工作原理、YAML 配置、HTTP API |
| [docs/workflow-guide.md](docs/workflow-guide.md) | Workflow 图执行：节点类型、边定义、变量映射 |
| [docs/skill-development.md](docs/skill-development.md) | 技能开发：内置技能、自定义技能、HTTP 技能托管 |

---

## 目录结构

```
superagent-base/
├── backend/                  Go 后端
│   ├── api/                  HTTP 路由 + gRPC 处理器
│   │   └── handler/coze/     chat_sse.go（SSE 端点）
│   ├── application/          应用服务编排层
│   ├── crossdomain/          跨域服务接口
│   ├── domain/               核心领域逻辑
│   ├── infra/                基础设施适配器
│   └── pkg/
│       ├── agentdef/         Agent YAML 运行时（schema/parser/builder/runtime/interrupt/workflow/orchestration）
│       ├── a2ui/             A2UI 协议（event.go + encoder.go）
│       ├── modelrouter/      Model Router（路由策略 + fallback）
│       ├── mcp/              MCP Client（stdio/SSE）+ Server
│       ├── memory/           记忆后端适配器
│       ├── skill/            SkillsHub（invoker + manager + builtin）
│       └── tool/             工具中间件链 + 内置工具
├── configs/
│   ├── agents/               Agent YAML 示例（research-agent.yaml 等）
│   └── models/               模型配置 YAML
├── docker/
│   ├── docker-compose-dev.yml     轻量级开发栈（MySQL + Redis）
│   ├── docker-compose-debug.yml   完整 debug 栈
│   ├── docker-compose-monitoring.yml  监控栈（Prometheus + Grafana + OTel）
│   ├── monitoring/                监控配置（prometheus.yml / otel-collector.yaml / grafana）
│   └── .env.dev                   开发环境配置模板
├── helm/                     Kubernetes Helm Chart
├── api/proto/                gRPC Proto 定义
├── scripts/                  开发脚本（dev-start.sh / e2e-test.sh）
└── Makefile
```

---

## 技术栈

| 组件 | 选型 |
|------|------|
| HTTP 框架 | [Hertz](https://github.com/cloudwego/hertz) (CloudWeGo) |
| LLM SDK | [Eino](https://github.com/cloudwego/eino) (CloudWeGo) |
| LLM Provider | OpenAI / Claude / Gemini / DeepSeek / Ark / Ollama / Qwen |
| gRPC | google.golang.org/grpc |
| ORM | GORM + MySQL 8.x |
| 缓存 | Redis 7 |
| 对象存储 | MinIO（开发）/ TOS / S3（生产） |
| 消息队列 | NSQ（默认）/ Kafka / RabbitMQ |
| 文件监听 | fsnotify（Agent YAML 热重载） |
| 可观测性 | OpenTelemetry + Prometheus |
| Web UI | React + TypeScript + Vite + Tailwind |

---

## License

Apache 2.0 — 详见 [LICENSE-APACHE](LICENSE-APACHE)。

## 致谢

本项目基于 [Coze Studio](https://github.com/coze-dev/coze-studio)（coze-dev Authors，Apache 2.0 许可）构建。
