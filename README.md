# Superagent Base

Superagent Base 是一个开源 AI Agent 开发平台，基于 [Coze Studio](https://github.com/coze-dev/coze-studio) 构建。它为构建、部署和管理 AI Agent 提供完整的后端服务，支持多模型路由、工具调用、知识库检索和多轮对话。

## 核心特性

- **声明式 Agent 定义**：使用 YAML 文件定义 Agent，支持文件热重载（无需重启）
- **多模型支持**：兼容 OpenAI、DeepSeek、Claude、Gemini、通义千问、火山方舟、Ollama、LM Studio
- **Eino ReAct 引擎**：基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 的工具调用循环（最多 10 步）
- **流式输出**：HTTP SSE + gRPC server-side streaming
- **多种工具类型**：内置工具（web_search / http_request / code_execute）、MCP 工具、SkillsHub 技能
- **可扩展记忆后端**：builtin / Mem0 / Zep / Letta
- **完整基础设施**：MySQL、Redis、MinIO/S3、Elasticsearch、NSQ/Kafka、向量存储
- **本地开发友好**：轻量级 dev 栈仅需 MySQL + Redis，重服务均可关闭

## 架构

```
客户端 (Web UI / API / gRPC)
        │
        ▼
Hertz HTTP Server (:8888)  +  gRPC Server (:50051)
        │
        ▼
API 层 → Application 层 → Domain / CrossDomain 层
        │
        ▼
pkg/agentdef (AgentRuntime)
        │  YAML 热重载
        ▼
Eino ChatModel / ReAct Agent
        │  OpenAI-compatible API
        ▼
LLM (LM Studio / Ollama / OpenAI / DeepSeek / ...)
        │
基础设施：MySQL · Redis · MinIO · ES · NSQ
```

详细架构图和数据流见 [docs/architecture.md](docs/architecture.md)。

## 快速开始

### 前置条件

- Go 1.24+
- Docker + Docker Compose
- 本地 LLM（如 [LM Studio](https://lmstudio.ai/) 监听 `http://127.0.0.1:8000/v1`）

### 3 步启动

```bash
# 1. 克隆
git clone <repo-url>
cd superagent-base

# 2. 配置模型（可选，默认已指向 LM Studio）
#    编辑 docker/.env.dev，填写 MODEL_API_KEY_0 和 MODEL_BASE_URL_0

# 3. 启动
make dev
```

访问 `http://localhost:8888`，gRPC 在 `localhost:50051`。

### 停止

```bash
make dev-down
```

## Make 命令

| 命令 | 说明 |
|------|------|
| `make dev` | 启动 MySQL + Redis + backend |
| `make dev-middleware` | 仅启动 MySQL + Redis |
| `make dev-server` | 构建并运行 backend（需先启动 middleware） |
| `make dev-down` | 停止 dev 容器 |
| `make dev-clean` | 停止容器并删除 MySQL 数据 |
| `make build` | 构建 `bin/superagent` |
| `make test` | 运行 `pkg/` 测试 |
| `make test-all` | 运行全部测试 |
| `make debug` | 启动完整 debug 环境（含 ES / MinIO / NSQ 等） |

## 文档

| 文档 | 说明 |
|------|------|
| [docs/agent-yaml-spec.md](docs/agent-yaml-spec.md) | Agent YAML 完整规范与示例 |
| [docs/architecture.md](docs/architecture.md) | 系统架构与模块依赖关系 |
| [docs/model-config.md](docs/model-config.md) | 模型配置：LM Studio / Ollama / OpenAI / DeepSeek / Claude 等 |
| [docs/deployment.md](docs/deployment.md) | 部署指南：本地开发 / Docker Compose / Kubernetes |

## 模块路径

Go module 路径为 `github.com/superagent-ai/superagent-base/backend`。

## 目录结构

```
superagent-base/
├── backend/                Go 后端
│   ├── api/                HTTP 路由 + gRPC 处理器
│   ├── application/        应用服务编排层
│   ├── crossdomain/        跨域服务接口
│   ├── domain/             核心领域逻辑
│   ├── infra/              基础设施适配器
│   ├── pkg/
│   │   └── agentdef/       Agent YAML 运行时（schema / parser / builder / runtime）
│   └── bizpkg/             业务工具包
├── docker/
│   ├── docker-compose-dev.yml     轻量级开发栈
│   ├── docker-compose-debug.yml   完整 debug 栈
│   ├── .env.dev                   开发环境配置
│   └── .env.debug.example         完整配置示例
├── scripts/
│   ├── dev-start.sh        开发环境一键启动脚本
│   ├── dev-stop.sh         停止脚本
│   └── e2e-test.sh         E2E 冒烟测试
├── docs/                   文档目录
├── helm/                   Kubernetes Helm Chart
├── api/proto/              gRPC Proto 文件
└── Makefile
```

## License

Apache 2.0 — 详见 [LICENSE-APACHE](LICENSE-APACHE)。

## 致谢

本项目基于 [Coze Studio](https://github.com/coze-dev/coze-studio)（coze-dev Authors，Apache 2.0 许可）构建。
