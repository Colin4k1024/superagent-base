# 海尔企业知识问答工具

`haier_knowledge_qa` 是对接海尔内部 RAG 知识库的内置工具，Agent 可通过该工具查询企业内部文档、制度、技术指南等知识内容。

## 快速使用

在 Agent YAML 中引用：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: enterprise-assistant
spec:
  model: qwen
  system_prompt: |
    你是一个企业知识助手，能够查询内部知识库回答问题。
    当用户提问涉及企业制度、技术文档或业务流程时，使用知识库工具查询。
  tools:
    - ref: builtin/haier_knowledge_qa
```

## 环境变量配置

工具通过环境变量进行条件注册，**4 个变量全部设置后工具才会生效**：

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `HAIER_RAG_BASE_URL` | RAG 服务地址 | `https://rag-test.haier.net`（测试）/ `https://rag.haier.net`（生产） |
| `HAIER_RAG_ACCESS_TOKEN` | 用户访问令牌 | 从海尔 RAG 平台获取 |
| `HAIER_RAG_APP_TOKEN` | 应用访问令牌 | 从海尔 RAG 平台获取 |
| `HAIER_RAG_K_CODE` | K-Code 身份标识 | 海尔工号 |

`.env` 配置示例：

```bash
HAIER_RAG_BASE_URL=https://rag-test.haier.net
HAIER_RAG_ACCESS_TOKEN=your-access-token
HAIER_RAG_APP_TOKEN=your-app-access-token
HAIER_RAG_K_CODE=your-kcode
```

## 参数说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `query` | string | 是 | — | 查询问题 |
| `top_k` | integer | 否 | 8 | 检索返回的文档片段数量 |
| `score_threshold` | number | 否 | 0.8 | 相关性分数阈值（0-1） |
| `kb_type` | integer | 否 | 0 | 知识库类型 |
| `model` | string | 否 | qwen | 回答生成模型 |
| `enable_multi_turn` | boolean | 否 | false | 是否启用多轮对话 |
| `session_id` | string | 否 | — | 多轮对话会话 ID |

### 知识库类型 (kb_type)

| 值 | 说明 |
|----|------|
| 0 | 全部知识库 |
| 1 | 非结构化文档（PDF、Word 等） |
| 2 | 结构化数据 |
| 3 | QA 问答对 |
| 4 | 表格数据 |

### 支持的模型 (model)

| 模型标识 | 说明 |
|---------|------|
| `qwen` | 通义千问（默认，推荐通用场景） |
| `DeepSeek-R1` | DeepSeek R1 推理模型 |
| `deepseek-v3` | DeepSeek V3 |
| `Qwen3-32B` | Qwen3 32B |

## 返回结果

工具返回 JSON 格式结果：

```json
{
  "answer": "根据知识库内容，海尔的数字化转型战略包括...",
  "sources": [
    {
      "filename": "数字化转型白皮书.pdf",
      "kb_type": 1,
      "file_url": "https://..."
    }
  ],
  "is_sufficient": true
}
```

| 字段 | 说明 |
|------|------|
| `answer` | 基于知识库生成的回答文本 |
| `sources` | 引用来源列表（文件名、类型、下载链接） |
| `is_sufficient` | 知识库是否有足够信息回答该问题 |

当 `is_sufficient` 为 `false` 时，表示内部知识库没有找到相关内容，Agent 应告知用户该问题超出知识库覆盖范围。

## 多轮对话

启用多轮对话后，RAG 服务会维护对话上下文：

```yaml
spec:
  system_prompt: |
    使用 haier_knowledge_qa 工具时，如果用户在追问，
    请传入 enable_multi_turn=true 和之前的 session_id。
  tools:
    - ref: builtin/haier_knowledge_qa
```

## 完整 Agent 示例

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: haier-knowledge-agent
  description: 海尔企业知识问答助手
spec:
  model: qwen
  system_prompt: |
    你是海尔集团的企业知识助手。

    职责：
    1. 回答关于公司制度、流程、技术规范的问题
    2. 查询内部知识库获取准确信息
    3. 如果知识库没有相关信息，明确告知用户

    使用规则：
    - 收到问题后先调用 haier_knowledge_qa 工具查询
    - 根据返回的 is_sufficient 判断是否有答案
    - 引用 sources 中的文件名作为来源说明
    - 对于知识库无法回答的问题，建议用户咨询相关部门

  tools:
    - ref: builtin/haier_knowledge_qa

  interrupt:
    enabled: false
```

## 注意事项

- 工具默认 60 秒超时，复杂查询可能需要较长时间
- `score_threshold` 设置过高可能导致无结果，过低可能引入不相关内容
- 生产环境和测试环境的 `HAIER_RAG_BASE_URL` 不同，注意区分
- Token 和 K-Code 需要在海尔 RAG 平台申请授权
