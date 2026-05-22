# A2UI 协议

## 概述

**A2UI（Agent-to-UI）** 是 Superagent Base 定义的结构化流式事件协议。相比传统的纯文本 token 流，A2UI 允许前端对 Agent 执行过程的每个阶段（文本输出、工具调用、代码块、中断请求、错误等）做精确的 UI 渲染和交互响应。

A2UI 基于 HTTP Server-Sent Events（SSE）传输，每帧都是一个命名 SSE 事件，浏览器可通过 `EventSource.addEventListener('text', ...)` 等按类型监听。

---

## 事件类型

所有事件共用同一个 Envelope：

```json
{
  "type": "<EventType>",
  "timestamp": 1746000000000,
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 事件类型，见下表 |
| `timestamp` | int64 | Unix 毫秒时间戳 |
| `data` | any | 事件数据，结构因 type 而异 |

### 事件类型一览

| EventType | 说明 | 典型时机 |
|-----------|------|---------|
| `text` | 流式文本 token | 模型输出文本内容时 |
| `thinking` | 推理过程 token | 支持 CoT 的模型（如 DeepSeek-R1）输出思考过程时 |
| `tool_call` | 工具调用 | 模型决定调用工具时 |
| `tool_result` | 工具返回结果 | 工具执行完成后 |
| `code_block` | 代码块 | 模型输出代码片段时 |
| `interrupt` | 中断请求 | Agent 检测到确认请求，需要用户输入时 |
| `error` | 错误 | Agent 执行出现错误时 |
| `done` | 流结束 | 完整响应结束 |
| `progress` | 进度信息 | 长时任务报告执行进度 |
| `agent_switch` | Agent 切换 | 多 Agent 编排中发生 Agent 切换时 |

---

## 各事件 Data 结构

### text（文本 token）

```json
{
  "content": "目前累积的完整文本",
  "delta": "新增的 token 片段"
}
```

| 字段 | 说明 |
|------|------|
| `content` | 截至当前的完整文本 |
| `delta` | 本帧新增内容（增量） |

### thinking（推理过程）

```json
{
  "content": "推理过程的完整内容",
  "delta": "新增推理 token"
}
```

结构与 `text` 相同，但用于区分"思考内容"和"最终回答"，前端可选择折叠/展示。

### tool_call（工具调用）

```json
{
  "id": "call_abc123",
  "name": "web_search",
  "arguments": {
    "query": "量子计算最新进展 2025"
  },
  "status": "calling"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 工具调用唯一 ID |
| `name` | string | 工具名称 |
| `arguments` | map[string]any | 调用参数 |
| `status` | string | `calling` / `success` / `error` |

### tool_result（工具结果）

```json
{
  "id": "call_abc123",
  "name": "web_search",
  "result": "搜索结果内容...",
  "is_error": false
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 对应 tool_call 的 ID |
| `name` | string | 工具名称 |
| `result` | string | 工具返回内容 |
| `is_error` | bool | 是否为错误结果 |

### code_block（代码块）

```json
{
  "language": "python",
  "code": "def hello():\n    print('world')\n",
  "delta": "    print('world')\n"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `language` | string | 编程语言 |
| `code` | string | 截至当前的完整代码 |
| `delta` | string | 本帧新增内容 |

### interrupt（中断请求）

```json
{
  "reason": "即将执行删除操作，请确认是否继续？",
  "fields": [
    {
      "name": "confirm",
      "type": "confirm",
      "label": "确认操作",
      "required": true
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `reason` | string | 中断原因说明（来自模型输出） |
| `fields` | []InterruptField | 需要用户填写的字段 |

**InterruptField 字段类型：**

| `type` | 说明 |
|--------|------|
| `text` | 自由文本输入 |
| `confirm` | 是/否确认 |
| `select` | 从 `options` 中选择 |

### error（错误）

```json
{
  "code": "TOOL_INVOKE_FAILED",
  "message": "web_search 工具调用超时"
}
```

### done（流结束）

`data` 字段为 `null` 或空对象，标志整个流的结束。

### progress（进度）

```json
{
  "agent_name": "research-workflow",
  "step": "summarize",
  "total": 3,
  "current": 2
}
```

### agent_switch（Agent 切换）

```json
{
  "from_agent": "project-manager",
  "to_agent": "code-review-agent",
  "reason": "需要进行代码质量分析"
}
```

---

## SSE 传输格式

每个事件编码为命名 SSE 帧（`EncodeSSE`）：

```
event: text
data: {"type":"text","timestamp":1746000001234,"data":{"content":"你好","delta":"你好"}}

event: tool_call
data: {"type":"tool_call","timestamp":1746000002345,"data":{"id":"call_1","name":"web_search","arguments":{"query":"AI 2025"},"status":"calling"}}

event: tool_result
data: {"type":"tool_result","timestamp":1746000003456,"data":{"id":"call_1","name":"web_search","result":"...","is_error":false}}

event: text
data: {"type":"text","timestamp":1746000004567,"data":{"content":"你好，根据搜索结果...","delta":"，根据搜索结果..."}}

event: done
data: {"type":"done","timestamp":1746000005678,"data":null}

```

### 兼容模式（Legacy）

不携带 `X-A2UI` 头时，使用 `EncodeCompatible` 编码：
- `text` 事件输出原始 delta 文本：`data: <token>\n\n`
- `done` 事件输出：`data: [DONE]\n\n`
- 其他类型回退到 JSON：`data: {"type":"...","data":{...}}\n\n`

---

## 启用 A2UI 模式

### 方式一：请求头

```bash
curl -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -H "X-A2UI: true" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"搜索量子计算"}'
```

### 方式二：查询参数

```
POST /api/v1/chat/stream?a2ui=true
```

---

## 客户端集成示例

### 原生 EventSource（浏览器）

```javascript
// 注意：EventSource 不支持 POST，需要使用 fetch + ReadableStream
const response = await fetch('/api/v1/chat/stream', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-A2UI': 'true'
  },
  body: JSON.stringify({
    agent_id: 'research-agent',
    session_id: 'session-001',
    message: '搜索量子计算最新进展'
  })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

let buffer = '';
while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  
  buffer += decoder.decode(value, { stream: true });
  const lines = buffer.split('\n\n');
  buffer = lines.pop(); // 保留不完整帧
  
  for (const frame of lines) {
    if (!frame.trim()) continue;
    
    // 解析 event: <type> 和 data: <json>
    const eventMatch = frame.match(/^event: (.+)/m);
    const dataMatch = frame.match(/^data: (.+)/m);
    
    if (eventMatch && dataMatch) {
      const eventType = eventMatch[1];
      const event = JSON.parse(dataMatch[1]);
      handleA2UIEvent(eventType, event);
    }
  }
}

function handleA2UIEvent(type, event) {
  switch (type) {
    case 'text':
      appendText(event.data.delta);
      break;
    case 'thinking':
      appendThinking(event.data.delta);
      break;
    case 'tool_call':
      showToolCall(event.data.name, event.data.status);
      break;
    case 'tool_result':
      updateToolResult(event.data.id, event.data.result);
      break;
    case 'code_block':
      renderCodeBlock(event.data.language, event.data.delta);
      break;
    case 'interrupt':
      showConfirmDialog(event.data.reason, event.data.fields);
      break;
    case 'error':
      showError(event.data.message);
      break;
    case 'done':
      finalize();
      break;
    case 'progress':
      updateProgress(event.data.step, event.data.current, event.data.total);
      break;
    case 'agent_switch':
      showAgentSwitch(event.data.from_agent, event.data.to_agent);
      break;
  }
}
```

### React Hook 示例

```tsx
import { useState, useCallback } from 'react';

interface A2UIEvent {
  type: string;
  timestamp: number;
  data: any;
}

export function useAgentChat(agentId: string) {
  const [messages, setMessages] = useState<string>('');
  const [thinking, setThinking] = useState<string>('');
  const [toolCalls, setToolCalls] = useState<any[]>([]);
  const [interrupted, setInterrupted] = useState<any>(null);

  const sendMessage = useCallback(async (sessionId: string, message: string) => {
    const response = await fetch('/api/v1/chat/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-A2UI': 'true' },
      body: JSON.stringify({ agent_id: agentId, session_id: sessionId, message })
    });

    const reader = response.body!.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const frames = buffer.split('\n\n');
      buffer = frames.pop() ?? '';

      for (const frame of frames) {
        const eventMatch = frame.match(/^event: (.+)/m);
        const dataMatch = frame.match(/^data: (.+)/m);
        if (!eventMatch || !dataMatch) continue;

        const evt: A2UIEvent = JSON.parse(dataMatch[1]);

        switch (evt.type) {
          case 'text':
            setMessages(prev => prev + evt.data.delta);
            break;
          case 'thinking':
            setThinking(prev => prev + evt.data.delta);
            break;
          case 'tool_call':
            setToolCalls(prev => [...prev, evt.data]);
            break;
          case 'interrupt':
            setInterrupted(evt.data);
            return; // 暂停等待用户确认
        }
      }
    }
  }, [agentId]);

  const resume = useCallback(async (sessionId: string, input: Record<string, any>) => {
    setInterrupted(null);
    const response = await fetch('/api/v1/chat/resume', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agent_id: agentId, session_id: sessionId, input })
    });
    // 处理恢复流...
  }, [agentId]);

  return { messages, thinking, toolCalls, interrupted, sendMessage, resume };
}
```

---

## 完整事件流示例（工具调用对话）

以下是一次使用 `research-agent`（带 web_search 工具）对话的完整 A2UI 事件序列：

```
event: thinking
data: {"type":"thinking","timestamp":1746000001000,"data":{"content":"用户想了解量子计算...","delta":"用户想了解量子计算..."}}

event: tool_call
data: {"type":"tool_call","timestamp":1746000002000,"data":{"id":"call_1","name":"web_search","arguments":{"query":"量子计算 2025 进展"},"status":"calling"}}

event: tool_result
data: {"type":"tool_result","timestamp":1746000003000,"data":{"id":"call_1","name":"web_search","result":"IBM 发布 1000 量子比特处理器...","is_error":false}}

event: text
data: {"type":"text","timestamp":1746000004000,"data":{"content":"根据","delta":"根据"}}

event: text
data: {"type":"text","timestamp":1746000004100,"data":{"content":"根据最新研究","delta":"最新研究"}}

event: text
data: {"type":"text","timestamp":1746000004200,"data":{"content":"根据最新研究，IBM 已经","delta":"，IBM 已经"}}

...（更多 text 事件）...

event: done
data: {"type":"done","timestamp":1746000010000,"data":null}
```

---

## 向后兼容性

- 不携带 `X-A2UI: true` 或 `?a2ui=true` 时，使用 Legacy 模式（纯文本 token 流 + `[DONE]`）
- 现有客户端（如基于原始 Coze Studio 协议的集成）无需修改即可继续工作
- A2UI 模式为选择性加入（opt-in），不影响默认行为
