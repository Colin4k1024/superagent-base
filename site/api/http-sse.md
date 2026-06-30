# HTTP API

Superagent Base 提供完整的 RESTful + SSE 流式 API，所有端点统一在 `/api/v2/` 命名空间下（v1 路径保留但标记为 deprecated）。

## 端点总览

### 对话与 Agent

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v2/chat/stream` | 流式对话（SSE），支持 Legacy / A2UI 两种模式 |
| POST | `/api/v2/chat/resume` | 恢复中断的对话 |
| GET | `/api/v2/chat/interrupt_state` | 查询会话中断状态 |
| GET | `/api/v2/agents` | 列出所有已加载的 Agent |

### 会话管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/conversations` | 列出会话 |
| POST | `/api/v2/conversations` | 创建会话 |
| GET | `/api/v2/conversations/:id` | 获取会话详情 |
| PUT | `/api/v2/conversations/:id` | 更新会话 |
| DELETE | `/api/v2/conversations/:id` | 删除会话（需 Admin 认证） |
| GET | `/api/v2/conversations/:conversation_id/messages` | 获取会话消息列表 |
| DELETE | `/api/v2/conversations/:conversation_id/messages/:message_id` | 删除消息（需 Admin 认证） |

### Session（短期记忆）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/sessions/:session_id/messages` | 获取 session 消息历史 |
| DELETE | `/api/v2/sessions/:session_id` | 清空 session（需 Admin 认证） |

### 文件管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v2/files` | 上传文件（multipart/form-data） |
| GET | `/api/v2/files` | 列出所有文件 |
| GET | `/api/v2/files/:id` | 获取文件元信息 |
| GET | `/api/v2/files/:id/content` | 下载文件内容 |
| DELETE | `/api/v2/files/:id` | 删除文件（需 Admin 认证） |

### 长期记忆（LTM）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/memory/long-term?user_id=&limit=&offset=` | 列出用户的长期记忆 |
| POST | `/api/v2/memory/long-term` | 添加记忆条目 |
| GET | `/api/v2/memory/long-term/search?user_id=&q=&limit=&threshold=` | 语义搜索记忆 |
| PUT | `/api/v2/memory/long-term/:id` | 更新记忆内容 |
| DELETE | `/api/v2/memory/long-term/:id` | 删除记忆（需 Admin 认证） |

### Agent 状态（KV Store）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/agents/:agent_id/state` | 获取 Agent 所有状态键值 |
| GET | `/api/v2/agents/:agent_id/state/:key` | 获取单个状态值 |
| POST | `/api/v2/agents/:agent_id/state` | 设置状态键值 |
| DELETE | `/api/v2/agents/:agent_id/state/:key` | 删除状态键（需 Admin 认证） |

### Workflow 执行

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v2/workflows/run` | 同步执行 Workflow |
| POST | `/api/v2/workflows/stream_run` | 流式执行 Workflow（SSE） |
| POST | `/api/v2/workflows/stream_resume` | 恢复流式 Workflow |
| POST | `/api/v2/workflows/chat` | Chat 模式执行 Workflow |
| GET | `/api/v2/workflows/:workflow_id` | 获取 Workflow 信息 |

### Skills 与 Tools

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/skills` | 列出已安装技能 |
| GET | `/api/v2/skills/search?q=` | 搜索可用技能 |
| GET | `/api/v2/tools` | 列出所有注册工具及其 schema |

### 用户身份

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/me` | 获取当前用户信息（需 Admin 认证） |

### 管理 API（需 API Key 认证）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/status` | 系统运行状态 |
| POST | `/api/v2/admin/reload` | 触发 Agent 热重载 |
| GET | `/api/v2/admin/logs` | SSE 实时日志流 |

#### Agent YAML 管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/agents` | 列出所有 Agent 定义 |
| POST | `/api/v2/admin/agents` | 创建 Agent |
| POST | `/api/v2/admin/agents/validate` | 校验 Agent YAML |
| GET | `/api/v2/admin/agents/:name` | 获取 Agent 详情 |
| PUT | `/api/v2/admin/agents/:name` | 更新 Agent |
| DELETE | `/api/v2/admin/agents/:name` | 删除 Agent |

#### 用户管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/users` | 列出用户 |
| POST | `/api/v2/admin/users` | 创建用户 |
| PUT | `/api/v2/admin/users/:id` | 更新用户 |
| DELETE | `/api/v2/admin/users/:id` | 删除用户 |

#### MCP Server 管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/mcp/servers` | 列出已连接 MCP Server |
| POST | `/api/v2/admin/mcp/servers` | 连接新 MCP Server |
| DELETE | `/api/v2/admin/mcp/servers/:name` | 断开 MCP Server |
| GET | `/api/v2/admin/mcp/servers/:name/tools` | 列出某 Server 的工具 |

#### Evolution 管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/evolution/stats` | Evolution 引擎统计 |
| GET | `/api/v2/admin/evolution/genes` | 基因列表（支持 `?q=&min_confidence=&limit=`） |
| POST | `/api/v2/admin/evolution/recommend` | 获取策略推荐 |
| GET | `/api/v2/admin/evolution/federated` | 联邦搜索（本地模式返回空） |

#### Webhook 管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v2/admin/webhooks` | 列出 Webhooks |
| POST | `/api/v2/admin/webhooks` | 创建 Webhook |
| GET | `/api/v2/admin/webhooks/:id` | 获取 Webhook 详情 |
| PUT | `/api/v2/admin/webhooks/:id` | 更新 Webhook |
| DELETE | `/api/v2/admin/webhooks/:id` | 删除 Webhook |
| POST | `/api/v2/admin/webhooks/:id/test` | 测试 Webhook |
| GET | `/api/v2/admin/webhooks/:id/logs` | 查看 Webhook 日志 |

### 基础设施

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/metrics` | Prometheus 指标端点 |
| GET | `/health` | 健康检查 |
| GET | `/ready` | 就绪检查（含 Agent Runtime 状态） |

---

## 认证

管理类 API（标记"需 Admin 认证"）通过 `Authorization: Bearer <API_KEY>` 或 `X-API-Key: <API_KEY>` 头认证。API Key 在环境变量 `ADMIN_API_KEY` 中配置。

未认证请求返回 `401 Unauthorized`。

---

## 流式对话

### 请求

```bash
POST /api/v2/chat/stream
Content-Type: application/json

{
  "agent_id": "research-agent",
  "session_id": "session-123",
  "message": "介绍一下 Eino 框架"
}
```

### 响应（Legacy 模式）

```
data: Eino
data:  是一个
data:  基于
data:  Go
data:  的 LLM 框架
data: [DONE]
```

### 响应（A2UI 模式）

添加 `?a2ui=true` 或 `X-A2UI: true` 头：

```
event: text
data: {"type":"text","timestamp":1234567890,"data":{"delta":"Eino "}}

event: tool_call
data: {"type":"tool_call","timestamp":1234567891,"data":{"id":"tc1","name":"web_search","status":"calling"}}

event: tool_result
data: {"type":"tool_result","timestamp":1234567892,"data":{"id":"tc1","name":"web_search","result":"...","is_error":false}}

event: text
data: {"type":"text","timestamp":1234567893,"data":{"delta":"根据搜索结果..."}}

event: done
data: {"type":"done","timestamp":1234567900,"data":null}
```

---

## 恢复中断

```bash
POST /api/v2/chat/resume
Content-Type: application/json

{
  "agent_id": "approval-agent",
  "session_id": "session-123",
  "input": {
    "confirm": true,
    "reason": "用户已确认执行"
  }
}
```

---

## 会话管理

### 创建会话

```bash
POST /api/v2/conversations
Content-Type: application/json

{
  "bot_id": "research-agent"
}
```

### 列出会话

```bash
GET /api/v2/conversations?bot_id=research-agent
```

### 获取消息列表

```bash
GET /api/v2/conversations/12345/messages
```

---

## 长期记忆

### 添加记忆

```bash
POST /api/v2/memory/long-term
Content-Type: application/json

{
  "user_id": "user-001",
  "content": "用户偏好中文回答",
  "metadata": {"source": "preference"}
}
```

### 搜索记忆

```bash
GET /api/v2/memory/long-term/search?user_id=user-001&q=偏好&limit=5
```

---

## Agent 状态

### 设置状态

```bash
POST /api/v2/agents/research-agent/state
Content-Type: application/json

{
  "key": "last_topic",
  "value": "quantum computing"
}
```

### 获取所有状态

```bash
GET /api/v2/agents/research-agent/state

{
  "agent_id": "research-agent",
  "state": {
    "last_topic": "quantum computing",
    "interaction_count": 42
  }
}
```

---

## Workflow 执行

### 同步执行

```bash
POST /api/v2/workflows/run
Content-Type: application/json

{
  "workflow_id": "research-workflow",
  "parameters": {"topic": "AI agents"}
}
```

### 流式执行

```bash
POST /api/v2/workflows/stream_run
Content-Type: application/json

{
  "workflow_id": "research-workflow",
  "parameters": {"topic": "AI agents"}
}
```

---

## 文件管理

### 上传文件

```bash
curl -X POST http://localhost:8888/api/v2/files \
  -F "file=@document.pdf"

# 响应
{"file":{"id":"uuid","filename":"document.pdf","mime_type":"application/pdf","size":1024,"created_at":"..."}}
```

### 下载文件

```bash
curl http://localhost:8888/api/v2/files/{id}/content -o output.pdf
```

---

## 列出 Agent

```bash
GET /api/v2/agents

{
  "agents": [
    {"name": "research-agent", "description": "研究助手", "type": "chat_model_agent"},
    {"name": "team-supervisor", "description": "团队协调", "type": "supervisor"}
  ]
}
```

---

## 列出工具

```bash
GET /api/v2/tools

{
  "tools": [
    {"name": "web_search", "description": "搜索互联网信息"},
    {"name": "http_request", "description": "发起 HTTP 请求"},
    {"name": "code_execute", "description": "执行代码"}
  ],
  "count": 3
}
```

---

## v3 Chat API（Coze 兼容）

保留 Coze Studio 原生 v3 Chat API，适用于已有 Coze SDK 集成的场景：

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/v3/chat` | 创建对话（ChatV3） |
| POST | `/v3/chat/cancel` | 取消对话 |
| GET | `/v3/chat/retrieve` | 查询对话状态 |

---

