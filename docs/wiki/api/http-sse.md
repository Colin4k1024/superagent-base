# HTTP SSE API

## 端点列表

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/agents` | 列出所有已加载的 Agent |
| POST | `/api/v1/chat/stream` | 流式对话（SSE） |
| POST | `/api/v1/chat/resume` | 恢复中断的对话 |
| GET | `/api/v1/chat/interrupt_state` | 查询中断状态 |
| GET | `/api/v1/admin/evolution/stats` | Evolution 引擎状态 |
| GET | `/api/v1/admin/evolution/genes` | 基因列表查询 |
| GET | `/api/v1/admin/evolution/federated` | 联邦搜索 |
| GET | `/metrics` | Prometheus 指标 |

## 流式对话

### 请求

```bash
POST /api/v1/chat/stream
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

## 恢复中断

```bash
POST /api/v1/chat/resume
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

## 列出 Agent

```bash
GET /api/v1/agents

{
  "agents": [
    {"name": "research-agent", "description": "研究助手"},
    {"name": "code-review-agent", "description": "代码审查"}
  ]
}
```
