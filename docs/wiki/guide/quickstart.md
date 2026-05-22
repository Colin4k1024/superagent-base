# 5 分钟快速上手

从零到运行你的第一个 AI Agent，只需 5 分钟。

---

## 你将完成什么

| 步骤 | 内容 | 耗时 |
|------|------|------|
| 1 | 启动基础设施 | 1 min |
| 2 | 配置模型 | 1 min |
| 3 | 启动服务 | 1 min |
| 4 | 创建并对话 Agent | 1 min |
| 5 | 尝试多 Agent 协同 | 1 min |

---

## Step 1: 克隆并启动基础设施

```bash
git clone https://github.com/Colin4k1024/superagent-base.git
cd superagent-base

# 启动 MySQL + Redis
make dev-middleware
```

::: tip 已有 MySQL/Redis？
设置环境变量指向已有服务即可，跳过 `make dev-middleware`。
:::

## Step 2: 配置你的 LLM 模型

```bash
cp .env.example backend/.env
```

编辑 `backend/.env`，只需改 3 行：

```bash
MODEL_BASE_URL_0=http://127.0.0.1:8000/v1   # LLM 服务地址
MODEL_API_KEY_0=123456                        # API Key
MODEL_ID_0=Qwen3-Coder-Next-4bit             # 模型 ID
```

::: info 支持的模型服务
- **vLLM** / **Ollama** / **LM Studio** — 本地部署
- **OpenAI** / **DeepSeek** / **Qwen** / **Claude** — 云端 API
- 任何 OpenAI 兼容接口
:::

## Step 3: 启动后端

```bash
make dev-server
```

看到日志输出 `agent runtime: reload complete` 表示启动成功。

验证：

```bash
curl http://localhost:8888/api/v1/agents
```

输出已加载的 Agent 列表。

## Step 4: 创建你的 Agent 并对话

在 `backend/configs/agents/` 下创建 `hello.yaml`：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: hello-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: Qwen3-Coder-Next-4bit
  system_prompt: |
    你是一个友好的中文助手，回答简洁明了。
  memory:
    backend: builtin
```

**无需重启** — Agent 自动热加载！立即对话：

```bash
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"hello-agent","session_id":"s1","message":"你好！介绍一下你自己"}'
```

你会看到流式 SSE 响应：

```
event: message
data: 你好！我是一个AI助手...

event: message
data: [DONE]
```

## Step 5: 尝试多 Agent 协同

创建 `backend/configs/agents/team.yaml`：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: team-agent
  version: "1.0.0"
spec:
  type: supervisor
  model:
    primary: Qwen3-Coder-Next-4bit
  system_prompt: |
    你是一个项目经理。将用户的任务委派给合适的子 Agent 完成。
  sub_agents:
    - ref: hello-agent
      role: "通用问答助手"
  orchestration:
    mode: supervisor
    max_rounds: 3
```

测试多 Agent 协同：

```bash
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"team-agent","session_id":"s2","message":"帮我查一下今天的天气"}'
```

Supervisor Agent 会自动分析任务并委派给子 Agent。

---

## 能力一览

完成上手后，你可以继续探索以下核心能力：

### Agent 类型

| 类型 | 用途 | 示例场景 |
|------|------|----------|
| `chat_model_agent` | 单 Agent 对话 | 客服、问答 |
| `workflow` | DAG 流程图 | 数据处理管道 |
| `supervisor` | 主管委派 | 项目协作 |
| `sequential` | 串行流水线 | 翻译→润色→校对 |
| `parallel` | 并行执行 | 多角度分析 |

### 工具生态

```yaml
tools:
  - ref: builtin/web_search       # 内置工具
  - ref: skill://calculator        # SkillHub 技能
  - ref: mcp://browser/navigate    # MCP 外部服务
```

### 记忆系统

```yaml
memory:
  backend: builtin    # 或 mem0 / zep / letta
```

Agent 自动记住多轮对话上下文。

### 管理能力

| 操作 | 方式 |
|------|------|
| 添加 Agent | 放 YAML 到 `configs/agents/` |
| 修改 Agent | 编辑 YAML，自动热重载 |
| 删除 Agent | 删除 YAML 文件 |
| 查看列表 | `GET /api/v1/agents` |
| 安装 Skill | `POST /api/v1/skills/install` |

---

## 下一步

- [YAML 完整规范](/guide/agent-yaml-spec) — 所有字段详解
- [多 Agent 编排](/advanced/multi-agent) — Supervisor/Sequential/Parallel 深度指南
- [工作流编排](/advanced/workflow) — DAG 图执行
- [MCP 集成](/advanced/mcp) — 连接外部工具服务
- [Skill 开发](/advanced/skill-development) — 开发和发布自定义技能
- [API 参考](/api/http-sse) — HTTP SSE / gRPC 完整接口
