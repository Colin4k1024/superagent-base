# HiAgent RAG 知识库集成

## 概述

`builtin/hiagent_rag` 工具将海尔 HiAgent v2.0.0 平台的 RAG 知识库检索能力接入 superagent-base。声明式 YAML Agent 可通过该工具直接查询 HiAgent 知识库，实现企业知识问答。

## 工作原理

HiAgent API 采用**会话制**：

1. **CreateConversation** — 为用户创建会话，获取 `AppConversationID`
2. **ChatQueryV2** — 使用会话 ID 发送查询，获取知识库检索结果

工具内部封装了完整的会话生命周期管理：
- 自动创建和缓存会话（线程安全）
- 会话过期时自动重建并重试
- 对 LLM 完全透明，只需传入 `query`

## 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `HIAGENT_API_URL` | 是 | HiAgent 服务地址，如 `http://33.234.129.82:32300` |
| `HIAGENT_API_KEY` | 是 | API 密钥（同时作为 HTTP Header `Apikey` 和请求体中的认证凭据） |
| `HIAGENT_APP_KEY` | 否 | 应用密钥，不设置时默认使用 `HIAGENT_API_KEY` 的值 |

> **条件注册**：未设置 `HIAGENT_API_URL` 或 `HIAGENT_API_KEY` 时，工具不会注册到系统中，不影响其他功能正常运行。

## 快速开始

### 1. 配置环境变量

在 `backend/.env` 中添加：

```bash
HIAGENT_API_URL=http://33.234.129.82:32300
HIAGENT_API_KEY=your-api-key-here
```

### 2. 创建 Agent YAML

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: rag-assistant
  version: "1.0.0"
  tags: [rag, hiagent]
  description: "基于 HiAgent 知识库的 RAG 问答助手"

spec:
  type: chat_model_agent
  model:
    primary: qwen-plus

  system_prompt: |
    你是一个知识助手。用户提问时，使用 hiagent_rag 工具检索知识库获取相关信息，
    然后综合检索结果给出准确的回答。如果知识库中没有相关信息，如实告知用户。

  tools:
    - ref: builtin/hiagent_rag

  memory:
    backend: builtin
```

将文件保存到 `configs/agents/rag-assistant.yaml`，服务会通过 fsnotify 自动加载。

### 3. 启动服务并测试

```bash
make dev
```

通过 API 发起对话：

```bash
curl -X POST http://localhost:8888/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "rag-assistant",
    "message": "公司的年假制度是怎样的？"
  }'
```

## 工具参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 要在知识库中检索的问题 |
| `user_id` | string | 否 | 用户标识，用于会话隔离。默认值 `"default"` |

## 工具返回

```json
{
  "answer": "检索到的回答内容...",
  "think_messages": ["思考过程..."],
  "tool_messages": ["工具调用记录..."]
}
```

## 错误处理

| 场景 | 行为 |
|------|------|
| 环境变量缺失 | 工具不注册，Agent 引用该工具时跳过 |
| 网络超时 | 返回错误（60 秒超时） |
| HiAgent 返回 4xx | 解析错误体，返回描述性错误信息 |
| 会话过期 (404/410) | 自动清除缓存 + 重建会话 + 重试一次 |
| 空 query | 立即返回参数错误 |

## 架构说明

```
Agent YAML (ref: builtin/hiagent_rag)
        │
        ▼
┌─────────────────────┐
│  HiAgentRAGTool     │  ← Eino InvokableTool 接口
│  (pkg/tool/builtin) │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  SessionManager     │  ← 线程安全会话缓存
│  (pkg/hiagent)      │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Client             │  ← HTTP 客户端
│  (pkg/hiagent)      │
└────────┬────────────┘
         │
         ▼
   HiAgent API Server
```

## 源码位置

| 文件 | 说明 |
|------|------|
| `backend/pkg/hiagent/types.go` | 请求/响应结构体 |
| `backend/pkg/hiagent/client.go` | HTTP 客户端（CreateConversation + ChatQueryBlocking） |
| `backend/pkg/hiagent/session.go` | 线程安全会话管理器 |
| `backend/pkg/hiagent/client_test.go` | 单元测试 |
| `backend/pkg/tool/builtin/hiagent_rag.go` | Eino InvokableTool 实现 |
| `backend/pkg/tool/builtin/registry.go` | 条件注册逻辑 |
| `configs/agents/rag-assistant.yaml` | 示例 Agent |

## 扩展

### 多知识库切换

如需对接多个 HiAgent 应用，可通过 `HIAGENT_APP_KEY` 区分，或在 Agent YAML 中通过 `config` 字段传入不同配置（后续版本支持）。

### SSE 流式模式

当前实现使用 blocking 模式获取完整结果。如需流式输出（逐字返回），可扩展 `ChatQueryStreaming` 方法并接入 A2UI 事件流。
