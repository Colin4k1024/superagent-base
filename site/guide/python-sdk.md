# Python SDK (superagent-base-python)

基于 **AgentScope 2.0** 的 Python Agent 基座，API 与 Go/Java 基座完全对等。

## 快速开始

### 安装

```bash
cd python
pip install -e ".[dev]"
```

### 启动服务

```bash
uvicorn superagent.server:app --reload --port 8889
```

### Docker

```bash
docker build -t superagent-py -f python/Dockerfile python/
docker run -p 8889:8889 --env-file python/.env superagent-py
```

## 核心能力

### Agent 类型

| 类型 | 说明 | 类名 |
|------|------|------|
| `chat_model_agent` | 单模型 ReAct Agent | `ChatModelAgent` |
| `supervisor` | 多 Agent 协调者 | `SupervisorAgent` |
| `sequential` | 顺序流水线 | `SequentialAgent` |
| `parallel` | 并发执行 | `ParallelAgent` |
| `workflow` | DAG 工作流 | `WorkflowAgent` |
| `agentloop` | 自主循环 | `AgentLoopAgent` |

### MCP 集成

```python
from superagent.tools.mcp import MCPRegistry, MCPClient

registry = MCPRegistry()
await registry.connect({
    "name": "filesystem",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
})

client = registry.get_client("filesystem")
tools = await client.list_tools()
result = await client.call_tool("read_file", {"path": "/tmp/test.txt"})
```

### Skills 系统

```python
from superagent.skills import SkillManager, LocalInvoker

manager = SkillManager()

# 注册本地技能
manager.register_local("datetime", lambda input: {"date": "2026-06-21"})

# 从 Hub 安装
await manager.install("web-search", "1.0.0")

# 调用技能
result = await manager.invoke("datetime", {})
```

### Tool 中间件链

```python
from superagent.tools.middleware import (
    retry_middleware,
    timeout_middleware,
    cache_middleware,
    chain,
)

# 组合中间件
pipeline = chain(
    retry_middleware(max_retries=3, backoff=1.0),
    timeout_middleware(timeout=30),
    cache_middleware(ttl=300),
)

# 应用到工具调用
wrapped_invoker = pipeline(my_tool_invoker)
```

### 消息与事件系统

```python
from superagent.message import Msg, UserMsg, AssistantMsg, TextBlock, ToolCallBlock
from superagent.event import (
    ReplyStartEvent,
    TextBlockDeltaEvent,
    ToolCallStartEvent,
    ToolResultEndEvent,
    ReplyEndEvent,
)

# 创建消息
user_msg = UserMsg(name="user", content="Hello")
assistant_msg = AssistantMsg(name="agent", content="Hi there!")

# 流式事件处理
async for event in agent.run_stream("Hello"):
    if isinstance(event, TextBlockDeltaEvent):
        print(event.delta, end="")
    elif isinstance(event, ToolCallStartEvent):
        print(f"\n[Calling {event.tool_call_name}...]")

# 从事件流重建消息
msg = None
async for event in agent.run_stream("Hello"):
    if isinstance(event, ReplyStartEvent):
        msg = AssistantMsg(name=event.name, content=[], id=event.reply_id)
    else:
        msg.append_event(event)
```

### 上下文注入

```python
from superagent.context import ContextInjectionMiddleware

middleware = ContextInjectionMiddleware(
    inject_timestamp=True,
    inject_session_metadata=True,
    static_context="You are a helpful assistant.",
)

# 注入上下文到消息列表
messages = middleware.inject(messages)
```

### AgentLoop

```python
from superagent.agents import AgentLoopAgent

loop_agent = AgentLoopAgent(
    agent_id="loop-1",
    name="autonomous-agent",
    child=chat_agent,
    max_turns=25,
)

# 自主循环执行，直到输出 [DONE] 或达到 max_turns
result = await loop_agent.run("Research quantum computing")
```

## API 端点

所有端点与 Go 基座完全对等：

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v2/chat/stream` | SSE 流式对话 |
| `POST` | `/api/v2/chat/resume` | 恢复中断对话 |
| `GET` | `/api/v2/chat/interrupt_state` | 查询中断状态 |
| `POST` | `/api/v2/chat/abort` | 中止对话 |
| `GET` | `/api/v2/agents` | Agent 列表 |
| `GET` | `/api/v2/conversations` | 会话列表 |
| `POST` | `/api/v2/conversations` | 创建会话 |
| `GET` | `/api/v2/tools` | 工具列表 |
| `GET` | `/api/v2/skills` | 技能列表 |
| `GET` | `/api/v2/mcp/servers` | MCP 服务器列表 |
| `GET` | `/health` | 健康检查 |
| `GET` | `/metrics` | Prometheus 指标 |

## 配置

### 环境变量

```bash
MODEL_API_KEY_0=sk-...          # 模型 API Key
MODEL_BASE_URL_0=http://...     # 模型端点
REDIS_URL=redis://localhost:6379  # Redis 连接
AGENTS_DIR=configs/agents        # Agent YAML 目录
```

### Agent YAML

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
  system_prompt: "You are a helpful assistant."
  tools:
    - ref: builtin/web_search
    - ref: mcp://filesystem/read_file
```

## 依赖

```toml
[project]
dependencies = [
    "agentscope>=2.0.0",
    "fastapi>=0.110.0",
    "uvicorn>=0.29.0",
    "sse-starlette>=2.0.0",
    "pyyaml>=6.0",
    "pydantic>=2.0",
    "httpx>=0.27.0",
    "redis>=5.0.0",
]
```
