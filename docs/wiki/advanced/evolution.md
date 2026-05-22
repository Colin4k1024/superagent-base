# Experience Self-Evolution

Superagent Base 内置**经验自进化**能力。系统自动收集 Agent 执行信号，提炼为"基因"（Gene）存储在本地 MySQL，并在后续对话中将最优策略推荐注入 system prompt，形成闭环进化。

**零外部依赖** — 只需 MySQL（已有的开发中间件），无需额外 Docker 容器。

## 核心概念

| 概念 | 说明 |
|------|------|
| **Signal** | 执行信号 — Agent 运行中产生的每一次工具调用、模型推理、节点执行事件 |
| **Gene** | 基因 — 从 Signal 中提炼出的策略模式，带有置信度和成功率评分 |
| **LocalGeneStore** | 本地基因库 — MySQL 表，存储和管理所有 Gene |
| **Advisor** | 进化顾问 — 查询 Gene 推荐并注入 Agent 系统提示 |

## 工作流程

### 信号收集

```
Agent 执行 → Eino Callback → SignalCollector.Collect()
    → [bounded goroutine pool, max=64]
    → context.WithTimeout(Background, 5s)
    → LocalGeneStore.SaveGene() → MySQL (evolution_genes 表)
```

每次 Agent 执行工具调用或模型推理，Eino 全局 callback 自动捕获事件，异步写入本地 MySQL。收集器使用信号量限制并发，并与 HTTP 请求上下文解耦以避免提前取消。

### 基因推荐

```
AgentBuilder.Build()
    → EvolutionAdvisor.Recommend(agent_name)
    → LocalGeneStore.Search() → 高置信度 Gene 列表
    → 注入 system prompt 前缀
```

在 Agent 构建时（启动或热重载），Advisor 查询与该 Agent 相关的最优策略，自动注入 system prompt 前缀。Agent 无需修改即可获得进化能力。

## 环境变量配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `EVOLUTION_ENABLED` | 启用经验自进化 | `false` |
| `EVOLUTION_SENDER_ID` | 本节点唯一 ID | `superagent-node-1` |
| `EVOLUTION_MIN_CONFIDENCE` | 推荐最低置信度 | `0.5` |
| `EVOLUTION_MAX_SUGGESTIONS` | 单次最大推荐数 | `3` |

::: tip
Evolution 使用与应用相同的 MySQL 连接（`MYSQL_DSN`），不需要额外配置数据库。
:::

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
    advise:
      min_confidence: 0.5
      max_suggestions: 3
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
  "mode": "local",
  "sender_id": "superagent-node-1",
  "min_confidence": 0.5,
  "max_suggestions": 3,
  "total_genes": 42,
  "avg_confidence": 0.72,
  "success_rate": 0.85
}
```

### 查询基因库

```bash
GET /api/v1/admin/evolution/genes?q=web_search&min_confidence=0.5&limit=20
```

**参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `q` | string | 搜索关键词（匹配 label/component/agent_name） |
| `min_confidence` | float | 最低置信度过滤 (0.0~1.0) |
| `limit` | int | 最大返回数 (1~100) |

**响应示例：**

```json
{
  "enabled": true,
  "genes": [
    {
      "id": "gene-abc123",
      "strategy": {"component": "web_search", "type": "tool_success"},
      "confidence": 0.85,
      "use_count": 142,
      "success_rate": 0.92
    }
  ],
  "total": 1
}
```

## Prometheus 指标

| 指标 | 说明 |
|------|------|
| `evolution_signals_total{signal_type}` | 信号收集计数（按类型） |
| `evolution_genes_shared_total` | 基因写入成功计数 |
| `evolution_share_failed_total` | 基因写入失败计数 |
| `evolution_share_dropped_total` | 信号丢弃计数（背压保护） |
| `evolution_recommendations_served_total` | 基因推荐服务计数 |

## Web UI

访问 `/evolution` 页面，包含三个 Tab：

| Tab | 功能 |
|-----|------|
| **Overview** | 引擎状态、基因统计、置信度配置 |
| **Gene Library** | 搜索和浏览基因库，查看策略详情、置信度和成功率 |
| **Federation** | 预留（本地模式下不可用） |

## 快速启用

```bash
# 1. 配置环境变量（在 backend/.env 中添加）
echo 'export EVOLUTION_ENABLED="true"' >> backend/.env

# 2. 启动 (或重启)
make dev

# 3. 执行几轮对话后，信号自动收集
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"hello"}'

# 4. 查看引擎状态和基因库
curl http://localhost:8888/api/v1/admin/evolution/stats
curl http://localhost:8888/api/v1/admin/evolution/genes?q=web_search

# 5. 或访问 Web UI: http://localhost:3000/evolution
```

## 架构细节

### 包结构

```
backend/pkg/evolution/
├── store.go           LocalGeneStore — GORM + MySQL 本地存储
├── config.go          配置加载（环境变量 → Config struct）
├── evolution.go       Engine 门面（Init / Shutdown）
├── collector.go       SignalCollector — 异步信号收集（semaphore bounded）
├── advisor.go         EvolutionAdvisor — Gene 推荐查询
├── callback.go        Eino 全局 Callback — Tool/Model 事件自动捕获
└── evolution_test.go  单元测试（11 cases: store / nil safety / payload / config）
```

### 数据表结构

```sql
CREATE TABLE evolution_genes (
  id           VARCHAR(64) PRIMARY KEY,
  label        VARCHAR(512),
  signal_type  VARCHAR(64),
  agent_name   VARCHAR(128),
  component    VARCHAR(128),
  signals      TEXT,
  strategy     TEXT,
  validation   TEXT,
  confidence   DOUBLE DEFAULT 0.5,
  use_count    INT DEFAULT 0,
  success_count INT DEFAULT 0,
  sender_id    VARCHAR(128),
  created_at   DATETIME,
  updated_at   DATETIME,
  INDEX idx_label (label),
  INDEX idx_signal_type (signal_type),
  INDEX idx_agent_name (agent_name)
);
```

### 设计决策

- **本地优先**：所有 Gene 存储在本地 MySQL，零网络依赖
- **nil receiver 安全**：所有 Engine/Collector/Advisor 方法对 nil 接收器安全，disabled 时零代价
- **信号量背压**：64 并发上限，满时丢弃并记录 metric，避免 goroutine 泄漏
- **上下文解耦**：background context + 5s timeout，不依赖 HTTP 请求生命周期
- **Build-time injection**：推荐在 AgentBuilder.Build() 时注入，非运行时
- **AutoMigrate**：启动时自动创建/更新表结构，无需手动运行 DDL
