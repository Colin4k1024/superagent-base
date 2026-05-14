# Experience Self-Evolution

Superagent Base 集成 [Oris Go SDK](https://github.com/Colin4k1024/Oris)，实现 Agent 的**经验自进化**。系统自动收集 Agent 执行信号，提炼为"基因"（Gene），并在后续对话中将最优策略推荐注入 system prompt，形成闭环进化。

## 核心概念

| 概念 | 说明 |
|------|------|
| **Signal** | 执行信号 — Agent 运行中产生的每一次工具调用、模型推理、节点执行事件 |
| **Gene** | 基因 — 从大量 Signal 中提炼出的策略模式，带有置信度和成功率评分 |
| **Experience Repo** | 经验仓库 — Oris 后端存储和管理所有 Gene 的服务 |
| **Hub** | 联邦中心 — 连接多个 Superagent 节点，跨节点共享 Gene |
| **Advisor** | 进化顾问 — 查询 Gene 推荐并注入 Agent 系统提示 |

## 工作流程

### 信号收集

```
Agent 执行 → Eino Callback → SignalCollector.Collect()
    → [bounded goroutine pool, max=64]
    → context.WithTimeout(Background, 5s)
    → experience.Client.Share() → Oris Experience Repo
```

每次 Agent 执行工具调用或模型推理，Eino 全局 callback 自动捕获事件，异步发送到 Oris Experience Repo。收集器使用信号量限制并发，并与 HTTP 请求上下文解耦以避免提前取消。

### 基因推荐

```
AgentBuilder.Build()
    → EvolutionAdvisor.Recommend(agent_name)
    → experience.Client.Fetch() → 高置信度 Gene 列表
    → 注入 system prompt 前缀
```

在 Agent 构建时（启动或热重载），Advisor 查询与该 Agent 相关的最优策略，自动注入 system prompt 前缀。Agent 无需修改即可获得进化能力。

### 联邦共享

```
Node A ← Hub → Node B
         ↕
       Node C
```

多个 Superagent 节点通过 Hub 注册和心跳保活，跨节点搜索和共享 Gene，实现集群级经验复用。

## 环境变量配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `EVOLUTION_ENABLED` | 启用经验自进化 | `false` |
| `ORIS_EXPERIENCE_URL` | Oris Experience Repo 地址 | — (enabled 时必填) |
| `ORIS_HUB_URL` | Oris Hub 地址（启用联邦） | — (可选) |
| `ORIS_API_KEY` | Oris API Key | — |
| `ORIS_SEED` | 加密种子 | — |
| `ORIS_SENDER_ID` | 本节点唯一 ID | `superagent-node-1` |
| `ORIS_NODE_ENDPOINT` | 本节点回调地址 | — |
| `ORIS_MIN_CONFIDENCE` | 推荐最低置信度 | `0.6` |
| `ORIS_MAX_SUGGESTIONS` | 单次最大推荐数 | `3` |

## Agent YAML 配置

在 Agent YAML 的 `spec` 中添加 `evolution` 字段：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-agent
spec:
  type: chat_model_agent
  model:
    primary: deepseek-r1
  system_prompt: "你是研究助手..."
  tools:
    - ref: builtin/web_search
  evolution:
    enabled: true
    collect:           # 可选：过滤信号类型，为空则收集全部
      - tool_success
      - tool_error
      - model_invoke
      - node_done
```

## Admin API

### 获取引擎状态

```bash
GET /api/v1/admin/evolution/stats
```

**响应示例：**

```json
{
  "enabled": true,
  "experience_url": "http://localhost:9200",
  "hub_url": "http://hub.example.com",
  "sender_id": "superagent-node-1",
  "peer_nodes": 3,
  "min_confidence": 0.6,
  "max_suggestions": 3
}
```

### 查询基因库

```bash
GET /api/v1/admin/evolution/genes?q=web_search&min_confidence=0.5&limit=20
```

**参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `q` | string | 搜索关键词 |
| `min_confidence` | float | 最低置信度过滤 (0.0~1.0) |
| `limit` | int | 最大返回数 (1~100) |

**响应示例：**

```json
{
  "enabled": true,
  "genes": [
    {
      "id": "gene-abc123",
      "strategy": {"pattern": "retry_with_backoff", "max_retries": 3},
      "confidence": 0.85,
      "use_count": 142,
      "success_rate": 0.92
    }
  ],
  "total": 1
}
```

### 联邦搜索

```bash
GET /api/v1/admin/evolution/federated?q=error_handling&min_confidence=0.5&limit=10
```

跨所有 Hub 连接节点搜索 Gene，返回格式同基因库查询。

## Prometheus 指标

| 指标 | 说明 |
|------|------|
| `evolution_signals_total{signal_type}` | 信号收集计数（按类型） |
| `evolution_genes_shared_total` | 基因共享成功计数 |
| `evolution_share_failed_total` | 基因共享失败计数 |
| `evolution_share_dropped_total` | 信号丢弃计数（背压保护） |
| `evolution_recommendations_served_total` | 基因推荐服务计数 |

## Web UI

访问 `/evolution` 页面，包含三个 Tab：

| Tab | 功能 |
|-----|------|
| **Overview** | 引擎状态、连接信息、节点数、置信度配置 |
| **Gene Library** | 搜索和浏览基因库，查看策略详情、置信度和成功率 |
| **Federation** | 跨节点联邦搜索，发现其他节点的高质量 Gene |

## 快速启用

```bash
# 1. 配置环境变量
cat >> backend/.env << 'EOF'
EVOLUTION_ENABLED=true
ORIS_EXPERIENCE_URL=http://localhost:9200
ORIS_SENDER_ID=my-node
EOF

# 2. 启动 (或重启)
make dev

# 3. 执行几轮对话后，信号自动收集
curl -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"hello"}'

# 4. 查看基因库
curl http://localhost:8888/api/v1/admin/evolution/genes?q=web_search

# 5. 或访问 Web UI: http://localhost:3000/evolution
```

## 架构细节

### 包结构

```
backend/pkg/evolution/
├── config.go          配置加载（环境变量 → Config struct）
├── evolution.go       Engine 门面（Init / Shutdown / DiscoverNodes / FederatedSearch）
├── collector.go       SignalCollector — 异步信号收集（semaphore bounded）
├── advisor.go         EvolutionAdvisor — Gene 推荐查询
├── callback.go        Eino 全局 Callback — Tool/Model 事件自动捕获
└── evolution_test.go  单元测试（nil safety / payload / config）
```

### 设计决策

- **nil receiver 安全**：所有 Engine/Collector/Advisor 方法对 nil 接收器安全，disabled 时零代价
- **信号量背压**：64 并发上限，满时丢弃并记录 metric，避免 goroutine 泄漏
- **上下文解耦**：background context + 5s timeout，不依赖 HTTP 请求生命周期
- **缓存 DiscoverNodes**：30s TTL + RWMutex，避免每次请求都调 Hub
- **Build-time injection**：推荐在 AgentBuilder.Build() 时注入，非运行时
