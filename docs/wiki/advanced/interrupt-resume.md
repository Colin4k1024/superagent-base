# 中断/恢复（Interrupt/Resume）

## 概述

中断/恢复功能允许 Agent 在输出中检测到需要用户确认的内容时，**暂停执行并等待用户输入**，然后在用户提供响应后无缝恢复对话。这对于涉及高风险操作（删除数据、发送邮件、部署代码等）的场景尤为重要。

---

## 工作原理

```
用户发起对话
  │
  ▼ Agent 执行，缓冲完整响应
  │
  ▼ InterruptableAgent 检测响应中是否包含确认关键词
  │  (please confirm / are you sure / shall i proceed / ...)
  │
  ├── 未检测到 ─→ 正常流式输出所有 token
  │
  └── 检测到确认请求
        │
        ▼ 保存 InterruptState 到 CheckpointStore
        │  state = { session_id, agent_name, reason, fields, timestamp, checkpoint_id }
        │
        ▼ 向客户端发送中断信号
        │  A2UI 模式：event: interrupt + InterruptData
        │  Legacy 模式："\x00INTERRUPT:{...json...}" 前缀 token
        │
        ▼ 客户端渲染确认表单，等待用户输入
        │
        ▼ 用户提交确认 → 客户端 POST /api/v1/chat/resume
        │  body: { agent_id, session_id, input: { confirm: true } }
        │
        ▼ InterruptableAgent.Resume()
        │  清除 interrupt state
        │  格式化用户输入为恢复消息
        │
        ▼ 调用内部 Agent 继续对话
        │
        ▼ 流式输出恢复后的响应
```

---

## Agent YAML 配置

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: approval-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
  system_prompt: |
    你是一个审批助手。在执行任何不可逆操作前，
    使用"Please confirm"短语明确请求用户确认。
  interrupt:
    enabled: true
    checkpoint_backend: redis    # redis（持久化）或 memory（内存，默认）
    timeout_seconds: 300         # 中断状态保留时长，默认 300 秒（5 分钟）
  memory:
    backend: builtin
```

### interrupt 配置字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | false | 启用中断/恢复包装 |
| `checkpoint_backend` | string | `memory` | 状态持久化后端：`redis` 或 `memory` |
| `timeout_seconds` | int | 300 | 中断状态保留秒数，超时后需重新对话 |

---

## 中断检测规则

当前版本通过关键词匹配检测模型输出中的确认请求（大小写不敏感）：

| 关键词 | 示例触发语 |
|--------|-----------|
| `please confirm` | "Please confirm you want to delete..." |
| `do you confirm` | "Do you confirm this action?" |
| `are you sure` | "Are you sure you want to proceed?" |
| `shall i proceed` | "Shall I proceed with the deployment?" |
| `would you like me to proceed` | "Would you like me to proceed?" |
| `do you want to proceed` | "Do you want to proceed with deletion?" |
| `do you want me to` | "Do you want me to send this email?" |
| `should i go ahead` | "Should I go ahead and reset the database?" |
| `confirm before` | "Please confirm before I execute this." |
| `confirmation required` | "Confirmation required for this action." |
| `waiting for your confirmation` | "Waiting for your confirmation..." |
| `need your approval` | "This action needs your approval." |

---

## HTTP API

### 1. 发起对话（可能产生中断）

```http
POST /api/v1/chat/stream
Content-Type: application/json

{
  "agent_id": "approval-agent",
  "session_id": "session-123",
  "message": "请帮我删除 prod 数据库中所有过期用户数据"
}
```

**正常响应（无中断）：** SSE token 流 + `[DONE]`

**中断响应（Legacy 模式）：**
```
data: 我将要删除

data: 以下过期用户数据：共 1,234 条记录。

data: \x00INTERRUPT:{"session_id":"session-123","agent_name":"approval-agent","reason":"Please confirm: 即将删除 1,234 条记录，此操作不可逆。","fields":[{"name":"confirm","type":"confirm","label":"确认删除","required":true}],...}
```

**中断响应（A2UI 模式，使用 `X-A2UI: true`）：**
```
event: text
data: {"type":"text",...,"data":{"content":"我将要删除以下过期用户数据：共 1,234 条记录。","delta":"..."}}

event: interrupt
data: {"type":"interrupt","timestamp":...,"data":{"reason":"Please confirm: 即将删除 1,234 条记录，此操作不可逆。","fields":[{"name":"confirm","type":"confirm","label":"确认删除","required":true}]}}
```

### 2. 查询中断状态

```http
GET /api/v1/chat/interrupt_state?agent_id=approval-agent&session_id=session-123
```

**响应（有中断）：**
```json
{
  "interrupted": true,
  "state": {
    "session_id": "session-123",
    "agent_name": "approval-agent",
    "reason": "Please confirm: 即将删除 1,234 条记录，此操作不可逆。",
    "fields": [
      {
        "name": "confirm",
        "type": "confirm",
        "label": "确认删除",
        "required": true
      }
    ],
    "timestamp": 1746000000,
    "checkpoint_id": "approval-agent_session-123_1746000000000000000"
  }
}
```

**响应（无中断）：**
```json
{"interrupted": false}
```

### 3. 恢复对话

```http
POST /api/v1/chat/resume
Content-Type: application/json

{
  "agent_id": "approval-agent",
  "session_id": "session-123",
  "input": {
    "confirm": true
  }
}
```

**响应：** 恢复后的 SSE token 流 + `[DONE]`

```
data: 已收到您的确认。正在执行删除操作...

data: 删除完成，共删除 1,234 条过期用户记录。操作日志已记录。

data: [DONE]
```

**错误响应：**

| 状态码 | 说明 |
|--------|------|
| `400` | agent_id 或 session_id 缺失 |
| `404` | Agent 不存在，或 Agent 不支持中断/恢复 |
| `409` | 该会话没有待处理的中断状态 |
| `500` | 内部错误 |

---

## 完整端到端示例

### 场景：部署审批

```bash
# 步骤 1：发起对话
curl -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"approval-agent","session_id":"deploy-123","message":"部署 v2.0 到生产环境"}' \
  --no-buffer

# 收到 interrupt 信号，用户界面显示确认对话框

# 步骤 2：查询中断状态（可选）
curl "http://localhost:8888/api/v1/chat/interrupt_state?agent_id=approval-agent&session_id=deploy-123"

# 步骤 3：用户确认，恢复对话
curl -X POST http://localhost:8888/api/v1/chat/resume \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"approval-agent","session_id":"deploy-123","input":{"confirm":true}}' \
  --no-buffer

# 步骤 4：如果用户拒绝
curl -X POST http://localhost:8888/api/v1/chat/resume \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"approval-agent","session_id":"deploy-123","input":{"confirm":false}}' \
  --no-buffer
```

---

## 客户端处理指南

### JavaScript 客户端

```javascript
async function chatWithInterrupt(agentId, sessionId, message) {
  const response = await fetch('/api/v1/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-A2UI': 'true'
    },
    body: JSON.stringify({ agent_id: agentId, session_id: sessionId, message })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const frames = buffer.split('\n\n');
    buffer = frames.pop();

    for (const frame of frames) {
      const eventMatch = frame.match(/^event: (.+)/m);
      const dataMatch = frame.match(/^data: (.+)/m);
      if (!eventMatch || !dataMatch) continue;

      const event = JSON.parse(dataMatch[1]);

      if (event.type === 'interrupt') {
        // 显示确认对话框
        const userInput = await showConfirmDialog(event.data.reason, event.data.fields);
        
        // 恢复对话
        await resumeChat(agentId, sessionId, userInput);
        return;
      }

      if (event.type === 'text') {
        appendToUI(event.data.delta);
      }
    }
  }
}

async function resumeChat(agentId, sessionId, input) {
  const response = await fetch('/api/v1/chat/resume', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agent_id: agentId, session_id: sessionId, input })
  });

  const reader = response.body.getReader();
  // 读取恢复后的流...
}
```

---

## InterruptState 数据结构

```go
type InterruptState struct {
    SessionID    string       `json:"session_id"`
    AgentName    string       `json:"agent_name"`
    Reason       string       `json:"reason"`        // 模型输出的确认请求原文
    Fields       []InputField `json:"fields"`        // 需要用户填写的表单字段
    Timestamp    int64        `json:"timestamp"`     // Unix 时间戳（秒）
    CheckpointID string       `json:"checkpoint_id"` // 用于 CheckpointStore 的键
}

type InputField struct {
    Name     string   `json:"name"`
    Type     string   `json:"type"`    // text / confirm / select
    Label    string   `json:"label"`
    Required bool     `json:"required"`
    Options  []string `json:"options,omitempty"` // select 类型使用
}
```

---

## 注意事项

- **超时**：中断状态有 `timeout_seconds` 期限（默认 5 分钟），超时后状态自动清除，Resume 请求将返回 `409`
- **单次消费**：`Resume` 成功后中断状态立即清除，不可重复 resume
- **并发安全**：`InterruptableAgent` 内部使用 `sync.RWMutex` 保护中断状态映射
- **Redis 后端**：生产环境推荐使用 `checkpoint_backend: redis` 以支持多实例部署，当前 redis 后端会回退到内存（完整实现在 `infra/checkpoint/redis.go`）
- **不支持 interrupt 的 Agent**：`Resume` 请求对不支持中断的 Agent 返回 `404`
