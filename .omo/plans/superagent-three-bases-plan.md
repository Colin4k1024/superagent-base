# Superagent 三基座对等架构实施计划

## 1. 项目概述

### 目标
基于当前 Go 基座（superagent-base），创建功能完全对等的 Python 和 Java 基座：
- `superagent-base-python` - 基于 AgentScope Python 2.0
- `superagent-base-java` - 基于 AgentScope Java 2.0

### 核心原则
- **能力对等**：三基座必须支持相同的功能集
- **配置共享**：Agent YAML 定义三基座共用
- **协议互通**：通过 A2A 协议实现跨基座 Agent 协作
- **独立部署**：每个基座可独立运行，也可组合部署

---

## 2. 框架选型决策

### 选择 AgentScope 的理由

| 评估维度 | AgentScope | Google ADK | 决策 |
|---------|------------|------------|------|
| Python/Java 功能对等 | ✅ 完全对等 | ⚠️ 有差异 | AgentScope |
| Harness 工程化 | ✅ 内置 | ❌ 裸 Agent | AgentScope |
| Workspace 沙箱 | ✅ Docker/E2B | ❌ | AgentScope |
| 权限系统 | ✅ 三态决策 | ❌ | AgentScope |
| A2A 协议 | ✅ | ✅ | 平手 |
| AG-UI 协议 | ✅ | ❌ | AgentScope |
| 企业级部署 | ✅ 无状态扩展 | ✅ Cloud Run | 平手 |

**结论**：AgentScope 更贴近当前 Go 基座的企业级定位

---

## 3. 三基座架构设计

### 3.1 目录结构

```
superagent-base/
├── go/                         ← 当前 Go 基座（Eino）- 保持不变
│   ├── backend/
│   │   ├── pkg/agentdef/       ← Agent 运行时
│   │   ├── pkg/tool/           ← 工具系统
│   │   ├── pkg/mcp/            ← MCP 集成
│   │   ├── pkg/memory/         ← 记忆系统
│   │   ├── pkg/modelrouter/    ← 模型路由
│   │   ├── pkg/a2ui/           ← A2UI 协议
│   │   └── pkg/evolution/      ← 自进化系统
│   └── configs/agents/         ← Agent YAML 定义
│
├── python/                     ← superagent-base-python
│   ├── src/
│   │   ├── superagent/
│   │   │   ├── __init__.py
│   │   │   ├── server.py       ← HTTP/SSE 服务
│   │   │   ├── agents/         ← Agent 实现
│   │   │   │   ├── base.py     ← BaseAgent 封装
│   │   │   │   ├── chat.py     ← chat_model_agent
│   │   │   │   ├── supervisor.py
│   │   │   │   ├── sequential.py
│   │   │   │   ├── parallel.py
│   │   │   │   └── workflow.py
│   │   │   ├── tools/          ← 工具实现
│   │   │   │   ├── builtin/    ← 内置工具
│   │   │   │   └── mcp/        ← MCP 工具
│   │   │   ├── memory/         ← 记忆后端
│   │   │   ├── models/         ← 模型路由
│   │   │   ├── harness/        ← Workspace 配置
│   │   │   └── evolution/      ← 自进化（可选）
│   │   └── tests/
│   ├── configs/                ← 符号链接到 shared/agents
│   ├── pyproject.toml
│   └── Dockerfile
│
├── java/                       ← superagent-base-java
│   ├── src/main/java/
│   │   └── io/superagent/
│   │       ├── SuperagentApplication.java
│   │       ├── server/         ← HTTP/SSE 服务
│   │       ├── agents/         ← Agent 实现
│   │       │   ├── BaseAgentWrapper.java
│   │       │   ├── ChatModelAgent.java
│   │       │   ├── SupervisorAgent.java
│   │       │   ├── SequentialAgent.java
│   │       │   ├── ParallelAgent.java
│   │       │   └── WorkflowAgent.java
│   │       ├── tools/          ← 工具实现
│   │       ├── memory/         ← 记忆后端
│   │       ├── models/         ← 模型路由
│   │       ├── harness/        ← Workspace 配置
│   │       └── evolution/      ← 自进化（可选）
│   ├── src/main/resources/
│   │   └── agents/             ← 符号链接到 shared/agents
│   ├── src/test/java/
│   ├── pom.xml
│   └── Dockerfile
│
└── shared/                     ← 共享配置
    ├── agents/                 ← Agent YAML 定义（三基座共用）
    │   ├── research-agent.yaml
    │   ├── react-tools-agent.yaml
    │   ├── team-supervisor.yaml
    │   └── ...
    ├── schemas/                ← JSON Schema 校验
    │   └── agent-schema.yaml
    ├── docker/                 ← 统一部署配置
    │   ├── docker-compose.yml
    │   └── docker-compose-dev.yml
    └── proto/                  ← gRPC/A2A 协议定义
        └── a2a/
```

### 3.2 协议层设计

```
┌─────────────────────────────────────────────────────────────┐
│                      L0 协议层                               │
├─────────────────────────────────────────────────────────────┤
│  MCP (工具调用)  │  A2A (智能体互通)  │  AG-UI (前端交互)   │
└─────────────────────────────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   ┌──────────┐      ┌──────────┐      ┌──────────┐
   │ Go 基座  │      │ Python   │      │ Java     │
   │ (Eino)   │◄────►│ 基座     │◄────►│ 基座     │
   └──────────┘      └──────────┘      └──────────┘
         │                 │                 │
         └─────────────────┼─────────────────┘
                           ▼
                    ┌──────────────┐
                    │ 共享配置层   │
                    │ (Agent YAML) │
                    └──────────────┘
```

---

## 4. 能力对等映射表

### 4.1 核心 Agent 类型

| Go 基座 (Eino) | Python 基座 (AgentScope) | Java 基座 (AgentScope) |
|---------------|-------------------------|----------------------|
| `chat_model_agent` | `ReActAgent` | `ReActAgent` |
| `deep_agent` | `ReActAgent` + 深度提示 | `ReActAgent` + 深度提示 |
| `supervisor` | `Supervisor` | `Supervisor` |
| `sequential` | `Pipeline` | `Pipeline` |
| `parallel` | `MsgHub` 并发模式 | `MsgHub` 并发模式 |
| `plan_execute` | `Supervisor` + `Pipeline` | `Supervisor` + `Pipeline` |
| `workflow` | `Custom Workflow` | `Custom Workflow` |
| `agentloop` | `LoopAgent` | `LoopAgent` |
| `eino_graph` | N/A（Go 专属） | N/A（Go 专属） |

### 4.2 工具系统

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| `builtin/web_search` | `@tool def web_search()` | `@Tool public String webSearch()` |
| `builtin/http_request` | `@tool def http_request()` | `@Tool public String httpRequest()` |
| `builtin/code_execute` | `@tool def code_execute()` | `@Tool public String codeExecute()` |
| MCP 工具 | `McpTool` | `McpTool` |
| Skill 工具 | `Skill` | `Skill` |
| 工具中间件链 | Middleware 注入 | Middleware 注入 |

### 4.3 记忆系统

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| `builtin` 记忆 | `InMemoryStore` | `InMemoryStateStore` |
| `mem0` 集成 | `Mem0Integration` | `Mem0Integration` |
| `zep` 集成 | `ZepIntegration` | `ZepIntegration` |
| `letta` 集成 | `LettaIntegration` | `LettaIntegration` |
| MySQL 持久化 | `MySQLStateStore` | `MySQLStateStore` |
| Redis 持久化 | `RedisStateStore` | `RedisStateStore` |

### 4.4 模型路由

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| 能力路由 | `ModelRegistry` | `ModelRegistry` |
| 成本路由 | 自定义策略 | 自定义策略 |
| 延迟路由 | 自定义策略 | 自定义策略 |
| Fallback | 自动切换备用模型 | 自动切换备用模型 |

### 4.5 中断/恢复

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| Checkpoint 保存 | `Session` 持久化 | `AgentStateStore` |
| Resume API | `POST /resume` | `POST /resume` |
| Human-in-the-loop | `PermissionSystem` | `PermissionSystem` |

### 4.6 流式输出

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| A2UI 事件 | `Event` 类型化事件 | `Event` 类型化事件 |
| SSE 推流 | `streamEvents()` | `streamEvents()` |
| 文本增量 | `TextDeltaEvent` | `TextDeltaEvent` |
| 工具调用 | `ToolCallEvent` | `ToolCallEvent` |
| 工具结果 | `ToolResultEvent` | `ToolResultEvent` |

### 4.7 可观测性

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| OpenTelemetry | `opentelemetry-sdk` | `opentelemetry-java` |
| Prometheus 指标 | `prometheus_client` | `micrometer-registry-prometheus` |
| 链路追踪 | `TracerProvider` | `TracerProvider` |
| Studio 集成 | AgentScope Studio | AgentScope Studio |

### 4.8 自进化系统（Evolution）

| Go 基座能力 | Python 实现 | Java 实现 |
|------------|------------|----------|
| 信号收集 | 需自建 | 需自建 |
| 基因存储 | 需自建 | 需自建 |
| 策略推荐 | 需自建 | 需自建 |

**注意**：AgentScope 不包含自进化系统，需要独立实现或使用 L3 层（Oris）

---

## 5. 实施阶段

### Phase 1: Python 基座骨架（Week 1-2）

#### 任务清单

**1.1 项目初始化**
- [ ] 创建 `python/` 目录结构
- [ ] 配置 `pyproject.toml`（依赖 agentscope、fastapi、uvicorn）
- [ ] 配置 `Dockerfile`
- [ ] 创建 `.env.example`

**1.2 核心 Agent 封装**
- [ ] `src/superagent/agents/base.py` - 封装 AgentScope BaseAgent
- [ ] `src/superagent/agents/chat.py` - ReActAgent 封装
- [ ] `src/superagent/agents/supervisor.py` - Supervisor 封装
- [ ] `src/superagent/agents/sequential.py` - Pipeline 封装
- [ ] `src/superagent/agents/parallel.py` - MsgHub 并发封装
- [ ] `src/superagent/agents/workflow.py` - Custom Workflow 封装

**1.3 工具系统**
- [ ] `src/superagent/tools/builtin/web_search.py`
- [ ] `src/superagent/tools/builtin/http_request.py`
- [ ] `src/superagent/tools/builtin/code_execute.py`
- [ ] `src/superagent/tools/mcp/` - MCP 工具集成

**1.4 记忆系统**
- [ ] `src/superagent/memory/backends/` - 记忆后端适配

**1.5 模型路由**
- [ ] `src/superagent/models/registry.py` - ModelRegistry 封装
- [ ] `src/superagent/models/router.py` - 路由策略

**1.6 HTTP 服务层**
- [ ] `src/superagent/server.py` - FastAPI + SSE
- [ ] `POST /api/v2/chat/stream` - 流式对话
- [ ] `POST /api/v2/chat/resume` - 恢复对话
- [ ] `GET /api/v2/agents` - Agent 列表
- [ ] `POST /api/v2/admin/reload` - 热重载

**1.7 Agent YAML 加载**
- [ ] `src/superagent/config/loader.py` - YAML 解析
- [ ] `src/superagent/config/builder.py` - Agent 构建器

**1.8 测试**
- [ ] 单元测试覆盖核心模块
- [ ] 集成测试：与 Go 基座 API 兼容性

#### 验收标准
```bash
# 启动 Python 基座
cd python && python -m superagent.server

# 测试 API 兼容性
curl -X POST http://localhost:8889/api/v2/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","message":"hello"}'

# 预期：返回与 Go 基座相同格式的 SSE 流
```

---

### Phase 2: Java 基座骨架（Week 3-4）

#### 任务清单

**2.1 项目初始化**
- [ ] 创建 `java/` 目录结构
- [ ] 配置 `pom.xml`（依赖 agentscope-harness、spring-boot）
- [ ] 配置 `Dockerfile`
- [ ] 创建 `application.yml`

**2.2 核心 Agent 封装**
- [ ] `BaseAgentWrapper.java` - 封装 HarnessAgent
- [ ] `ChatModelAgent.java` - ReActAgent 封装
- [ ] `SupervisorAgent.java` - Supervisor 封装
- [ ] `SequentialAgent.java` - Pipeline 封装
- [ ] `ParallelAgent.java` - MsgHub 并发封装
- [ ] `WorkflowAgent.java` - Custom Workflow 封装

**2.3 工具系统**
- [ ] `tools/builtin/WebSearchTool.java`
- [ ] `tools/builtin/HttpRequestTool.java`
- [ ] `tools/builtin/CodeExecuteTool.java`
- [ ] `tools/mcp/` - MCP 工具集成

**2.4 记忆系统**
- [ ] `memory/backends/` - 记忆后端适配

**2.5 模型路由**
- [ ] `models/ModelRegistryWrapper.java`
- [ ] `models/RouterStrategy.java`

**2.6 HTTP 服务层**
- [ ] `server/ChatController.java` - Spring MVC + SSE
- [ ] `server/AdminController.java` - 管理 API
- [ ] 与 Go/Python 基座 API 完全兼容

**2.7 Agent YAML 加载**
- [ ] `config/YamlAgentLoader.java`
- [ ] `config/AgentBuilder.java`

**2.8 测试**
- [ ] JUnit 5 单元测试
- [ ] 集成测试：三基座 API 一致性

#### 验收标准
```bash
# 启动 Java 基座
cd java && mvn spring-boot:run

# 测试 API 兼容性
curl -X POST http://localhost:8890/api/v2/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","message":"hello"}'

# 预期：返回与 Go/Python 基座相同格式的 SSE 流
```

---

### Phase 3: 共享配置层（Week 5）

#### 任务清单

**3.1 Agent YAML 规范**
- [ ] 定义统一的 Agent YAML Schema
- [ ] 创建 JSON Schema 校验文件
- [ ] 迁移现有 Agent 定义到 `shared/agents/`

**3.2 Docker 统一部署**
- [ ] `shared/docker/docker-compose.yml` - 三基座联合部署
- [ ] `shared/docker/docker-compose-dev.yml` - 开发环境
- [ ] 环境变量统一管理

**3.3 A2A 互通测试**
- [ ] Go Agent 调用 Python Agent
- [ ] Python Agent 调用 Java Agent
- [ ] Java Agent 调用 Go Agent

#### 验收标准
```bash
# 启动三基座
docker compose -f shared/docker/docker-compose.yml up

# 测试跨基座调用
curl -X POST http://localhost:8888/api/v2/chat/stream \
  -d '{"agent_id":"cross-base-supervisor","message":"协作完成任务"}'

# 预期：Supervisor 自动分派给不同基座的子 Agent
```

---

### Phase 4: 企业级特性补齐（Week 6-8）

#### 任务清单

**4.1 自进化系统（Evolution）**
- [ ] Python 基座实现 Evolution 模块
- [ ] Java 基座实现 Evolution 模块
- [ ] 与 Go 基座 Evolution 数据互通

**4.2 可观测性增强**
- [ ] 三基座统一 OTel 配置
- [ ] Prometheus 指标对齐
- [ ] Grafana Dashboard 模板

**4.3 生产部署文档**
- [ ] Kubernetes Helm Chart
- [ ] 扩缩容策略
- [ ] 灾备方案

---

## 6. API 兼容性规范

### 6.1 统一 API 端点

所有基座必须实现以下端点：

```
POST   /api/v2/chat/stream          # 流式对话
POST   /api/v2/chat/resume          # 恢复中断对话
GET    /api/v2/chat/interrupt_state  # 查询中断状态
GET    /api/v2/agents               # Agent 列表
GET    /api/v2/conversations        # 会话列表
POST   /api/v2/conversations        # 创建会话
GET    /api/v2/conversations/:id    # 会话详情
DELETE /api/v2/conversations/:id    # 删除会话
GET    /api/v2/tools                # 工具列表
GET    /api/v2/skills               # 技能列表
POST   /api/v2/admin/reload         # 热重载
GET    /api/v2/admin/status         # 系统状态
GET    /metrics                     # Prometheus 指标
GET    /health                      # 健康检查
GET    /ready                       # 就绪检查
```

### 6.2 SSE 事件格式

所有基座必须使用相同的 A2UI 事件格式：

```json
{"type": "text", "data": {"content": "Hello"}, "timestamp": 1234567890}
{"type": "tool_call", "data": {"name": "web_search", "args": {...}}}
{"type": "tool_result", "data": {"name": "web_search", "result": {...}}}
{"type": "thinking", "data": {"content": "Let me think..."}}
{"type": "error", "data": {"message": "Something went wrong"}}
{"type": "done", "data": {}}
```

---

## 7. 依赖清单

### Python 基座依赖

```toml
[project]
dependencies = [
    "agentscope>=2.0.0",
    "fastapi>=0.110.0",
    "uvicorn>=0.29.0",
    "sse-starlette>=2.0.0",
    "pyyaml>=6.0",
    "pydantic>=2.0",
    "opentelemetry-api>=1.24.0",
    "opentelemetry-sdk>=1.24.0",
    "prometheus-client>=0.20.0",
]
```

### Java 基座依赖

```xml
<dependencies>
    <dependency>
        <groupId>io.agentscope</groupId>
        <artifactId>agentscope-harness</artifactId>
        <version>${agentscope.version}</version>
    </dependency>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-webflux</artifactId>
    </dependency>
    <dependency>
        <groupId>io.micrometer</groupId>
        <artifactId>micrometer-registry-prometheus</artifactId>
    </dependency>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-sdk</artifactId>
    </dependency>
</dependencies>
```

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| AgentScope 版本不一致 | 功能差异 | 锁定版本，定期同步升级 |
| YAML Schema 不兼容 | 配置失败 | JSON Schema 校验，CI 自动检测 |
| A2A 协议实现差异 | 互通失败 | 统一测试套件，端到端验证 |
| 性能差异 | 用户体验不一致 | 基准测试，性能回归检测 |
| 自进化系统缺失 | 能力不对等 | Phase 4 补齐，或集成 Oris |

---

## 9. 成功标准

### 功能完整性
- [ ] 三基座支持相同的 Agent 类型
- [ ] 三基座支持相同的工具集
- [ ] 三基座支持相同的 API 端点
- [ ] 三基座支持相同的 SSE 事件格式

### 互通性
- [ ] Go Agent 可调用 Python/Java Agent
- [ ] Python Agent 可调用 Go/Java Agent
- [ ] Java Agent 可调用 Go/Python Agent

### 可部署性
- [ ] 每个基座可独立部署
- [ ] 三基座可联合部署
- [ ] Docker Compose 一键启动

### 文档完整性
- [ ] 每个基座有 README
- [ ] API 文档（OpenAPI）
- [ ] 部署文档
- [ ] 开发指南

---

## 10. 时间线

| 阶段 | 时间 | 交付物 |
|------|------|--------|
| Phase 1 | Week 1-2 | Python 基座骨架 + 测试 |
| Phase 2 | Week 3-4 | Java 基座骨架 + 测试 |
| Phase 3 | Week 5 | 共享配置 + A2A 互通 |
| Phase 4 | Week 6-8 | 企业级特性 + 文档 |

---

**计划创建时间**：2026-06-20
**最后更新**：2026-06-21
**计划状态**：Phase 1-3 已完成，Phase 4 待实施

## 11. 实施进度

| 阶段 | 状态 | 文件数 | 测试 | 验证结果 |
|------|------|--------|------|---------|
| Phase 1: Python 基座 | ✅ 完成 | 30 | 33/33 | 全部通过 |
| Phase 2: Java 基座 | ✅ 完成 | 30 | 22/22 | 全部通过 |
| Phase 3: 共享配置层 | ✅ 完成 | 3 | - | JSON Schema + 14 YAML + Docker Compose |
| Phase 4: 企业级特性 | ⏳ 待实施 | - | - | - |

### 已完成交付物

**Python 基座** (`python/`)
- `src/superagent/agents/` — 6 种 Agent 类型（base, chat, supervisor, sequential, parallel, workflow）
- `src/superagent/tools/builtin/` — 3 个内置工具（web_search, http_request, code_execute）
- `src/superagent/tools/mcp/client.py` — MCP 客户端
- `src/superagent/memory/backends/` — BuiltinMemory + RedisMemory
- `src/superagent/models/` — ModelRegistry + ModelRouter
- `src/superagent/config/` — YAML 加载器 + Pydantic Schema（兼容 Go 的 ref 对象格式）
- `src/superagent/server.py` — FastAPI + SSE，7 个 API 端点
- `pyproject.toml`, `Dockerfile`, `.env.example`, `README.md`

**Java 基座** (`java/`)
- `src/main/java/io/superagent/agents/` — 6 种 Agent 类型
- `src/main/java/io/superagent/tools/builtin/` — 3 个内置工具
- `src/main/java/io/superagent/tools/` — Tool 接口 + McpToolWrapper
- `src/main/java/io/superagent/memory/` — MemoryStore 接口 + BuiltinMemory + RedisMemory
- `src/main/java/io/superagent/models/` — ModelRegistry + ModelRouter
- `src/main/java/io/superagent/config/` — AgentDefinition record + YamlAgentLoader + AgentBuilderFactory
- `src/main/java/io/superagent/server/` — ChatController + AdminController + HealthController（15 个端点）
- `pom.xml`, `Dockerfile`, `application.yml`, `README.md`

**共享配置** (`shared/`)
- `schemas/agent-definition-v1.json` — 统一 JSON Schema（覆盖 Go 全部字段）
- `agents/*.yaml` — 14 个 Agent 定义（三基座共用）
- `docker/docker-compose.yml` — 三基座联合部署（Go:8888 + Python:8889 + Java:8890）
