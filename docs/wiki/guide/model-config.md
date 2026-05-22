# 模型配置指南

## 概述

Superagent Base 支持两种方式配置 LLM 模型：

1. **环境变量方式**：通过 `MODEL_PROTOCOL_0` 等 env 配置单个主力模型（最常用）
2. **YAML 文件方式**：在 `resources/conf/model/` 目录下放置 `.yaml` 文件，支持多个模型

此外，`BUILTIN_CM_*` 系列 env 控制知识库检索、NL2SQL 等内置能力所使用的底层模型。

---

## 方式一：环境变量配置（推荐用于开发）

在 `.env` 或 `.env.dev` 文件中设置：

```bash
MODEL_PROTOCOL_0=openai       # 协议类型，见下方支持列表
MODEL_OPENCOZE_ID_0=100001    # 平台内部数字 ID，任意唯一正整数
MODEL_NAME_0=Qwen3-Coder-Next # 界面展示名称
MODEL_ID_0=Qwen3-Coder-Next-4bit  # 实际传给 API 的 model 字符串
MODEL_API_KEY_0=your-api-key  # API 密钥
MODEL_BASE_URL_0=http://host.docker.internal:8000/v1  # API 基础 URL
```

**注意**：目前只支持索引 `_0`，即单个模型。如需多模型，请使用 YAML 文件方式。

---

## 方式二：YAML 文件配置

在 `backend/resources/conf/model/` 目录下创建 `.yaml` 文件，格式如下（参考现有 Ollama 示例）：

```yaml
id: 100002                    # 唯一正整数 ID
name: my-gpt4o                # 展示名称
description:
  zh: GPT-4o 模型
  en: GPT-4o model
meta:
  protocol: openai            # 协议类型
  conn_config:
    base_url: "https://api.openai.com/v1"
    api_key: "sk-xxx"
    timeout: 60s
    model: "gpt-4o"
    temperature: 0.7
    max_tokens: 4096
    top_p: 0.95
```

---

## 支持的模型协议

| `protocol` 值 | 说明 | 典型模型 |
|---------------|------|----------|
| `openai` | OpenAI 兼容 API（最通用） | GPT-4o, LM Studio, Ollama (openai 模式) |
| `deepseek` | DeepSeek 原生 API | deepseek-chat, deepseek-r1 |
| `claude` | Anthropic Claude | claude-opus-4, claude-sonnet-4 |
| `gemini` | Google Gemini | gemini-2.0-flash, gemini-2.5-pro |
| `ark` | 字节跳动火山方舟 | doubao-pro-128k |
| `ollama` | Ollama 原生 API | llama3, qwen2.5 |
| `qwen` | 阿里云通义千问 | qwen-max, qwen-turbo |

---

## 各 Provider 配置示例

### OpenAI

```bash
MODEL_PROTOCOL_0=openai
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=GPT-4o
MODEL_ID_0=gpt-4o
MODEL_API_KEY_0=sk-xxx
MODEL_BASE_URL_0=https://api.openai.com/v1
```

### DeepSeek

```bash
MODEL_PROTOCOL_0=deepseek
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=DeepSeek-R1
MODEL_ID_0=deepseek-reasoner
MODEL_API_KEY_0=sk-xxx
MODEL_BASE_URL_0=https://api.deepseek.com/v1
```

### Claude (Anthropic)

```bash
MODEL_PROTOCOL_0=claude
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=Claude-Sonnet-4
MODEL_ID_0=claude-sonnet-4-5
MODEL_API_KEY_0=sk-ant-xxx
MODEL_BASE_URL_0=https://api.anthropic.com
```

### Gemini

```bash
MODEL_PROTOCOL_0=gemini
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=Gemini-2.0-Flash
MODEL_ID_0=gemini-2.0-flash
MODEL_API_KEY_0=AIza-xxx
MODEL_BASE_URL_0=https://generativelanguage.googleapis.com/v1beta
```

### 阿里云通义千问 (Qwen)

```bash
MODEL_PROTOCOL_0=qwen
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=Qwen-Max
MODEL_ID_0=qwen-max
MODEL_API_KEY_0=sk-xxx
MODEL_BASE_URL_0=https://dashscope.aliyuncs.com/compatible-mode/v1
```

---

## 本地模型配置

### LM Studio（推荐用于 macOS 开发）

LM Studio 暴露 OpenAI 兼容 API，默认监听 `http://127.0.0.1:8000/v1`。

**直接运行 backend（非 Docker）：**

```bash
MODEL_PROTOCOL_0=openai
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=Qwen3-Coder-Next
MODEL_ID_0=Qwen3-Coder-Next-4bit
MODEL_API_KEY_0=123456
MODEL_BASE_URL_0=http://127.0.0.1:8000/v1
```

**backend 运行在 Docker 内部时，需要使用 `host.docker.internal`：**

```bash
MODEL_BASE_URL_0=http://host.docker.internal:8000/v1
```

### Ollama

```bash
MODEL_PROTOCOL_0=ollama
MODEL_OPENCOZE_ID_0=100001
MODEL_NAME_0=Llama3
MODEL_ID_0=llama3:8b
MODEL_API_KEY_0=ollama
MODEL_BASE_URL_0=http://127.0.0.1:11434

# 或者使用 openai 兼容端点
MODEL_PROTOCOL_0=openai
MODEL_BASE_URL_0=http://127.0.0.1:11434/v1
```

---

## 内置能力模型配置（BUILTIN_CM）

知识库检索、NL2SQL、图片标注等内置能力使用独立的模型配置，通过 `BUILTIN_CM_*` env 控制：

```bash
BUILTIN_CM_TYPE=openai        # 协议类型（同 MODEL_PROTOCOL_0 取值）

# OpenAI 协议示例
BUILTIN_CM_OPENAI_BASE_URL=http://127.0.0.1:8000/v1
BUILTIN_CM_OPENAI_API_KEY=123456
BUILTIN_CM_OPENAI_MODEL=Qwen3-Coder-Next-4bit
```

**其他协议对应的 env key 模式：**

| 协议 | BASE_URL | API_KEY | MODEL |
|------|----------|---------|-------|
| openai | `BUILTIN_CM_OPENAI_BASE_URL` | `BUILTIN_CM_OPENAI_API_KEY` | `BUILTIN_CM_OPENAI_MODEL` |
| deepseek | `BUILTIN_CM_DEEPSEEK_BASE_URL` | `BUILTIN_CM_DEEPSEEK_API_KEY` | `BUILTIN_CM_DEEPSEEK_MODEL` |
| ollama | `BUILTIN_CM_OLLAMA_BASE_URL` | — | `BUILTIN_CM_OLLAMA_MODEL` |
| qwen | `BUILTIN_CM_QWEN_BASE_URL` | `BUILTIN_CM_QWEN_API_KEY` | `BUILTIN_CM_QWEN_MODEL` |
| ark | `BUILTIN_CM_ARK_BASE_URL` | `BUILTIN_CM_ARK_API_KEY` | `BUILTIN_CM_ARK_MODEL` |
| gemini | `BUILTIN_CM_GEMINI_BASE_URL` | `BUILTIN_CM_GEMINI_API_KEY` | `BUILTIN_CM_GEMINI_MODEL` |

特定用途可通过前缀覆盖：

```bash
NL2SQL_BUILTIN_CM_TYPE=openai          # NL2SQL 专用模型
M2Q_BUILTIN_CM_TYPE=openai            # messages2query 专用模型
IA_BUILTIN_CM_TYPE=openai             # 图片标注专用模型
WKR_BUILTIN_CM_TYPE=openai            # workflow 知识库召回专用模型
```

---

## Embedding 模型配置（向量化）

知识库功能需要 Embedding 模型，开发环境可通过 `EMBEDDING_TYPE=` 留空关闭：

```bash
EMBEDDING_TYPE=                  # 留空禁用（dev 模式）

# 生产环境示例（OpenAI）
EMBEDDING_TYPE=openai
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
MODEL_API_KEY_0=sk-xxx

# Ollama 示例
EMBEDDING_TYPE=ollama
OLLAMA_EMBEDDING_MODEL=nomic-embed-text
```

---

## 环境变量加载机制

backend 启动时根据 `APP_ENV` 环境变量加载对应 `.env` 文件：

```
APP_ENV=          → 加载 .env
APP_ENV=dev       → 加载 .env.dev
APP_ENV=debug     → 加载 .env.debug
APP_ENV=prod      → 加载 .env.prod
```

文件路径相对于 backend 二进制所在目录（或 `go run` 的工作目录）。
