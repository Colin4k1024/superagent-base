# 快速开始

## 前提条件

- Go 1.24+
- Docker + Docker Compose
- Node.js 21+（Web UI 开发）
- 一个 LLM 模型服务（LM Studio / Ollama / OpenAI API）

## 三步启动

### 1. 克隆项目

```bash
git clone https://github.com/Colin4k1024/superagent-base.git
cd superagent-base
```

### 2. 配置模型

复制环境配置并填入你的模型信息：

```bash
cp .env.example backend/.env
```

编辑 `backend/.env`，配置模型端点：

```bash
MODEL_BASE_URL_0=http://127.0.0.1:8000/v1   # 你的模型服务地址
MODEL_API_KEY_0=your-api-key                  # API Key
MODEL_ID_0=your-model-id                      # 模型 ID
```

### 3. 启动服务

```bash
make dev
```

这会自动：
- 启动 MySQL + Redis（Docker）
- 编译并运行后端服务（:8888 HTTP + :50051 gRPC）
- 加载 `configs/agents/` 中的 Agent 定义

## 验证

```bash
# 查看已加载的 Agent
curl http://localhost:8888/api/v1/agents

# 流式对话
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"你好"}'
```

## 创建你的第一个 Agent

在 `configs/agents/` 下创建 YAML 文件：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-first-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: your-model-id
  system_prompt: |
    你是一个友好的助手。
  tools:
    - ref: builtin/web_search
  memory:
    backend: builtin
```

保存后 Agent 会自动热加载（无需重启服务）。

## 下一步

- [YAML 规范](/guide/agent-yaml-spec) — 完整的 Agent 定义语法
- [架构概览](/guide/architecture) — 系统设计和模块关系
- [模型配置](/guide/model-config) — 多模型路由设置
- [A2UI 协议](/advanced/a2ui-protocol) — 结构化 UI 事件流
