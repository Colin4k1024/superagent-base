# Langfuse 可观测性集成

Superagent Base 通过 OpenTelemetry 原生协议将所有 LLM 调用、Tool 执行和 Agent 编排的 trace 自动上报到 [Langfuse](https://langfuse.com)，实现 LLM 应用级别的可观测性。

## 概述

```
┌─────────────────┐         OTLP/HTTP (protobuf)         ┌───────────────┐
│  Superagent     │ ──────────────────────────────────▶   │   Langfuse    │
│  Backend        │                                       │   Dashboard   │
│                 │         OTLP/gRPC (可选)              ├───────────────┤
│  pkg/observe    │ ──────────────────────────────────▶   │  OTel         │
│                 │                                       │  Collector    │
└─────────────────┘                                       └───────────────┘
```

### 自动追踪能力

| 追踪维度 | 记录内容 |
|---------|---------|
| LLM 调用 | 模型名称、Provider、输入 Prompt、输出 Completion、Token 用量、延迟 |
| Tool 调用 | 工具名称、输入参数、输出结果、执行时长、成功/失败 |
| Agent 编排 | Agent ID、操作类型、会话关联、执行链路 |
| 会话追踪 | Session ID、User ID、Trace Name、自定义 Tags |

## 快速配置

### 环境变量

在 `backend/.env` 中添加以下配置：

```bash
# 启用 Langfuse 追踪
LANGFUSE_ENABLED=true

# Langfuse 项目凭证（从 Langfuse 项目设置页面获取）
LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxx
LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxx

# Langfuse 服务地址
# Cloud 版本：
#   EU: https://cloud.langfuse.com (默认)
#   US: https://us.cloud.langfuse.com
# 自建版本：填写你的 Langfuse 实例地址
LANGFUSE_HOST=https://cloud.langfuse.com

# 可选配置
# LANGFUSE_SAMPLE_RATE=1.0    # 采样率 0.0-1.0，默认 1.0 (100%)
# LANGFUSE_DEBUG=false         # 调试模式
```

### 获取凭证

1. 登录 [Langfuse Dashboard](https://cloud.langfuse.com)
2. 创建或选择一个 Project
3. 进入 **Settings → API Keys**
4. 复制 Public Key 和 Secret Key

### 内部环境参考

| 环境 | LANGFUSE_HOST | 说明 |
|------|---------------|------|
| 测试环境 | `http://10.250.5.144:3000` | 内网自建实例 |
| 生产环境 | `https://langfuse.haier.net` | 生产域名 |

测试环境配置示例：

```bash
LANGFUSE_ENABLED=true
LANGFUSE_PUBLIC_KEY=pk-lf-1a3ecbeb-82ec-4af5-a491-221113148deb
LANGFUSE_SECRET_KEY=sk-lf-59d4be03-5eeb-4361-88a7-03f782598120
LANGFUSE_HOST=http://10.250.5.144:3000
```

## 工作原理

### 双导出架构

Langfuse 集成采用 **双导出** 设计，可以同时向 Langfuse 和现有的 OTel Collector 发送 trace：

```bash
# 仅启用 Langfuse（推荐入门配置）
LANGFUSE_ENABLED=true
OTEL_ENABLED=false

# 同时启用（生产环境推荐）
LANGFUSE_ENABLED=true
OTEL_ENABLED=true
OTEL_ENDPOINT=otel-collector:4317
```

### gen_ai 语义规范

所有 LLM 调用的 span 遵循 [OpenTelemetry gen_ai 语义规范](https://opentelemetry.io/docs/specs/semconv/gen-ai/)，确保 Langfuse 正确解析：

| Span 属性 | 说明 | 示例值 |
|-----------|------|--------|
| `gen_ai.system` | 模型提供商 | `openai`, `anthropic` |
| `gen_ai.request.model` | 请求模型 ID | `gpt-4o`, `claude-3.5-sonnet` |
| `gen_ai.operation.name` | 操作类型 | `chat` |
| `gen_ai.usage.input_tokens` | 输入 Token 数 | `1024` |
| `gen_ai.usage.output_tokens` | 输出 Token 数 | `512` |
| `gen_ai.usage.total_tokens` | 总 Token 数 | `1536` |
| `input.value` | 完整输入内容 | `[{"role":"user","content":"..."}]` |
| `output.value` | 完整输出内容 | `{"role":"assistant","content":"..."}` |

### Langfuse 追踪上下文

每个 Agent 请求自动注入以下上下文属性，支持在 Langfuse 中按会话、用户维度分析：

| 属性 | 来源 | 用途 |
|------|------|------|
| `langfuse.session.id` | 请求中的 `session_id` | 对话会话分组 |
| `langfuse.trace.name` | `agent.{agent_id}` | Trace 命名 |
| `langfuse.trace.tags` | 协议模式 (`a2ui`/`legacy`) | 标签筛选 |

## 架构集成点

### Eino 回调链

Langfuse trace 数据通过 Eino 组件生命周期回调自动采集，无需在业务代码中手动埋点：

```
Agent 请求
  └─ StartAgentSpan (agent.chat)
       └─ EinoObserveCallback.OnStart
            ├─ Model 调用 → gen_ai.chat span (含 input/output/tokens)
            ├─ Tool 调用 → tool.invoke span (含 input/output)
            └─ 子 Agent 调用 → 递归追踪
```

### 代码层接入

如果需要在自定义代码中注入额外的 Langfuse 上下文：

```go
import "github.com/superagent-ai/superagent-base/backend/pkg/observe"

// 方式一：通过 context 注入（推荐，所有子 span 自动继承）
ctx = observe.WithLangfuseContext(ctx, &observe.LangfuseTraceContext{
    SessionID: "session-123",
    UserID:    "user-456",
    TraceName: "custom-workflow",
    Tags:      []string{"production", "v2"},
    Metadata:  map[string]string{"env": "prod"},
})

// 方式二：直接设置当前 span 属性
observe.SetLangfuseSpanAttrs(ctx, "session-123", "user-456", "custom-trace")
```

## 在 Langfuse 中查看数据

配置完成并发送请求后，在 Langfuse Dashboard 中可以看到：

1. **Traces** — 每个 Agent 请求对应一条 trace，包含完整的调用链路
2. **Generations** — LLM 调用详情，包含 prompt、completion 和 token 用量
3. **Sessions** — 按 session_id 聚合的对话视图
4. **Metrics** — Token 消耗趋势、延迟分布、错误率

## 采样与性能

- **异步导出**: Trace 数据通过 `BatchSpanProcessor` 异步批量发送，不阻塞业务请求
- **采样控制**: 通过 `LANGFUSE_SAMPLE_RATE` 控制采样比例，降低高流量场景的成本
- **优雅降级**: Langfuse 不可达时自动丢弃 span，不影响业务正常运行
- **零开销**: `LANGFUSE_ENABLED=false` 时完全不创建导出器，零性能开销

## 与现有监控的关系

| 维度 | Prometheus + Grafana | Langfuse |
|------|---------------------|----------|
| 定位 | 基础设施监控 | LLM 应用可观测性 |
| 数据 | 聚合指标（计数、延迟分位） | 单次调用明细 + 内容 |
| 用途 | 告警、容量规划、SLO | 调试、评估、成本分析、Prompt 优化 |
| 接入 | `/metrics` 端点 | OTLP 自动上报 |

两者互补，建议生产环境同时启用。

## 自建 Langfuse

如需自建 Langfuse 实例，参考 [Langfuse Self-hosting 文档](https://langfuse.com/docs/deployment/self-host)：

```bash
# Docker Compose 快速部署
git clone https://github.com/langfuse/langfuse.git
cd langfuse
docker compose up -d

# 配置环境变量指向自建实例
LANGFUSE_HOST=http://your-langfuse-host:3000
```

## 故障排查

| 问题 | 排查方向 |
|------|---------|
| Langfuse 中无数据 | 检查 `LANGFUSE_ENABLED=true`、Key 是否正确、网络是否可达 |
| 部分 trace 缺失 | 检查 `LANGFUSE_SAMPLE_RATE` 是否 < 1.0 |
| Token 数为 0 | 确认模型 Provider 返回了 usage 信息 |
| 启动时报错 | 检查 `LANGFUSE_PUBLIC_KEY` 和 `LANGFUSE_SECRET_KEY` 是否都已设置 |

开启 debug 模式获取详细日志：

```bash
LANGFUSE_DEBUG=true
```
