# Oris Go SDK 集成计划：经验自进化能力

## Context

Superagent-base 需要引入"经验自进化"能力，使 Agent 在执行过程中自动积累成功策略（Gene）、识别失败模式、复用已验证经验（Capsule），形成闭环进化。

Oris 项目（https://github.com/Colin4k1024/Oris）提供 Go SDK 作为远程客户端，连接 Oris Runtime 服务（Experience Repo / Hub / Execution Runtime）。superagent-base 通过 SDK 与进化系统交互，触发范围覆盖全链路。

## Oris Go SDK

### 安装

```bash
go get github.com/Colin4k1024/Oris/sdks/go
```

- 模块路径：`github.com/Colin4k1024/Oris/sdks/go`
- Go 版本：1.21+
- 无外部依赖（纯 HTTP 客户端 + Ed25519 签名）
- SDK 版本：0.1.0

### SDK 结构

```
sdks/go/
├── oris.go              # 包入口，Version = "0.1.0"
├── hub/                 # Hub 服务客户端（节点注册/发现/订阅）
│   └── client.go
├── execution/           # 执行运行时客户端（Job 生命周期管理）
│   ├── client.go
│   └── types.go
├── experience/          # 经验仓库客户端（Gene 共享/查询）★ 核心集成点
│   ├── client.go
│   ├── client_test.go
│   └── types.go
└── internal/
    └── signing.go       # Ed25519 签名工具
```

### 三个客户端包

#### `experience` — 经验进化（核心集成点）

```go
import "github.com/Colin4k1024/Oris/sdks/go/experience"

client := experience.NewClient(experience.Config{
    BaseURL:  "http://localhost:8090",   // Oris Experience Repo 地址
    APIKey:   "your-api-key",
    Seed:     [32]byte{...},            // Ed25519 seed
    SenderID: "superagent-node-1",
})

// 首次注册公钥
client.RegisterPublicKey(ctx)

// 贡献经验（成功策略 → Gene）
resp, err := client.Share(ctx, payload)
// resp.GeneID, resp.Status, resp.PublishedAt

// 查询已有经验（复用 Gene）
results, err := client.Fetch(ctx, &experience.FetchQuery{
    Q:             "tool_call web_search success",
    MinConfidence: 0.5,
    Limit:         10,
    Cursor:        "",  // 分页游标
})
// results.Assets → []NetworkAsset
// results.NextCursor → 下一页游标
// results.SyncAudit → {TotalAvailable, Returned}
```

#### `hub` — 节点注册与联邦发现

```go
import "github.com/Colin4k1024/Oris/sdks/go/hub"

h := hub.New(hub.Config{
    BaseURL: "http://hub:8080",
    APIKey:  "key",
    Seed:    seed,
    NodeID:  "superagent-1",
})

h.Register("http://my-node:9000", []string{"evolve"}, "0.1.0")
h.Heartbeat(ctx)
nodes, _ := h.Discover(ctx)
results, _ := h.Search(ctx, query)
h.Subscribe(ctx, topic)
```

#### `execution` — Job 执行管理

```go
import "github.com/Colin4k1024/Oris/sdks/go/execution"

exec := execution.NewClient(execution.Config{
    BaseURL: "http://runtime:8091",
    Token:   "bearer-token",
})

resp, _ := exec.RunJob(ctx, execution.RunJobRequest{
    ThreadID: "thread-1",
    Input:    map[string]any{"task": "..."},
    Timeout:  30,
    Priority: 1,
})
exec.GetState(ctx, jobID)
exec.Cancel(ctx, jobID, "reason")
exec.Resume(ctx, jobID, checkpointID, newValues)
```

### 关键数据类型

```go
// experience.NetworkAsset — Gene/Capsule 的统一表示
type NetworkAsset struct {
    Type          string    `json:"type"`           // "gene" | "capsule"
    ID            string    `json:"id"`
    Signals       any       `json:"signals"`        // 触发信号
    Strategy      any       `json:"strategy"`       // 策略内容
    Validation    any       `json:"validation"`     // 验证结果
    Confidence    float64   `json:"confidence"`     // 置信度 (0.0-1.0)
    QualityScore  float64   `json:"quality_score"`  // 质量评分
    UseCount      int       `json:"use_count"`      // 使用次数
    SuccessCount  int       `json:"success_count"`  // 成功次数
    CreatedAt     time.Time `json:"created_at"`
    ContributorID string    `json:"contributor_id"`
}

// experience.OenEnvelope — 签名消息信封
type OenEnvelope struct {
    SenderID    string `json:"sender_id"`
    MessageType string `json:"message_type"`
    Payload     any    `json:"payload"`
    Signature   string `json:"signature"`    // Ed25519 签名
    Timestamp   string `json:"timestamp"`
}

// experience.FetchQuery — 经验查询参数
type FetchQuery struct {
    Q             string  `json:"q,omitempty"`              // 语义搜索
    MinConfidence float64 `json:"min_confidence,omitempty"` // 最低置信度
    Limit         int     `json:"limit,omitempty"`          // 返回数量
    Cursor        string  `json:"cursor,omitempty"`         // 分页游标
}
```

### 认证方式

| 服务 | 写操作 | 读操作 |
|------|--------|--------|
| Hub | Ed25519 签名 (`X-OEN-Signature`) | Bearer Token (`APIKey`) |
| Execution | Bearer Token | Bearer Token |
| Experience | API Key + Ed25519 签名 | 无认证（公开读） |

---

## Oris 核心概念

| 概念 | 说明 |
|------|------|
| **Gene** | 可复用策略 DNA — 带置信度评分、衰减机制、语义标签 |
| **Capsule** | 已验证的经验快照 — 签名保证完整性 |
| **Confidence** | 初始 0.70，成功 +0.05，失败 -0.08，衰减 -0.002/query，<0.30 触发重进化 |
| **Task Class** | 语义等价类，用于匹配相似任务复用已有 Gene |
| **Mutation Evaluator** | 两阶段门控：静态反模式检测 + LLM 评审（5 维度打分） |
| **八阶段进化循环** | Detect → Select → Mutate → Execute → Validate → Evaluate → Solidify → Reuse |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                    Agent YAML (spec.evolution)                   │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│              pkg/evolution/ (集成层)                              │
│  ┌──────────┐  ┌──────────────┐  ┌────────────────────────┐    │
│  │ Collector│  │ Advisor      │  │ Eino Callback          │    │
│  │ (信号采集)│  │ (经验查询复用) │  │ (全链路自动Hook)       │    │
│  └────┬─────┘  └──────┬───────┘  └───────────┬────────────┘    │
│       │               │                      │                 │
│  ┌────▼───────────────▼──────────────────────▼────────────┐    │
│  │         Oris Go SDK (远程客户端)                         │    │
│  │  experience.Client │ hub.Client │ execution.Client      │    │
│  └────────────────────────────┬───────────────────────────┘    │
└───────────────────────────────┼────────────────────────────────┘
                                │ HTTP
┌───────────────────────────────▼────────────────────────────────┐
│              Oris Runtime Services (Docker Sidecar)              │
│  ┌──────────────┐  ┌──────────┐  ┌────────────────────────┐   │
│  │Experience Repo│  │   Hub    │  │  Execution Runtime     │   │
│  │   :8090      │  │  :8080   │  │       :8091            │   │
│  └──────────────┘  └──────────┘  └────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## 集成点映射

| 触发源 | 信号类型 | 采集内容 | 进化动作 |
|--------|---------|---------|---------|
| Tool 调用成功 | tool_success | tool_name, args, result, latency | Share → 固化为 Gene |
| Tool 调用失败 | tool_error | tool_name, args, error, context | Share → 标记失败模式 |
| Model 调用 | model_invoke | model_id, prompt_hash, tokens, latency | Share → 路由策略 Gene |
| Agent 会话完成 | agent_done | agent_name, session, total_steps, outcome | Share → 对话策略 Gene |
| Workflow 节点完成 | node_done | node_id, type, input_hash, output, duration | Share → 节点策略 Gene |
| ReAct 工具选择 | react_step | step_n, tool_chosen, reasoning | Share → 工具选择策略 |

## 新增模块结构

```
backend/pkg/evolution/
├── evolution.go          # 顶层 Facade：Init(), Engine, config
├── collector.go          # SignalCollector: 采集信号 → experience.Client.Share()
├── advisor.go            # EvolutionAdvisor: experience.Client.Fetch() → 推荐策略
├── callback.go           # Eino Callback Handler（全局自动 Hook）
├── config.go             # EvolutionConfig (env / YAML)
└── evolution_test.go     # 集成测试
```

## 核心接口设计（superagent-base 侧）

```go
// pkg/evolution/evolution.go

import (
    "github.com/Colin4k1024/Oris/sdks/go/experience"
    "github.com/Colin4k1024/Oris/sdks/go/hub"
)

// Engine 是进化能力的顶层入口，封装 Oris SDK 客户端。
type Engine struct {
    expClient *experience.Client  // 经验仓库客户端
    hubClient *hub.Client         // Hub 客户端（可选，用于联邦）
    collector *SignalCollector
    advisor   *EvolutionAdvisor
    config    Config
}

// Config 进化引擎配置。
type Config struct {
    Enabled       bool     // 总开关
    ExperienceURL string   // Experience Repo 地址
    HubURL        string   // Hub 地址（可选）
    APIKey        string   // API Key
    Seed          [32]byte // Ed25519 seed
    SenderID      string   // 本节点 ID
    MinConfidence float64  // Advisor 最低置信度阈值
    MaxSuggestions int     // Advisor 最大推荐数
}

// Signal 描述一次执行事件的观测数据。
type Signal struct {
    Type       string         // tool_success, tool_error, model_invoke, agent_done, etc.
    AgentName  string
    SessionID  string
    Component  string         // tool name / model id / node id
    Input      string         // input hash or summary
    Output     string         // result summary
    Error      string         // error message if failed
    Duration   time.Duration
    Metadata   map[string]any // extra context
    Timestamp  time.Time
}

// Recommendation 是 Advisor 返回的经验建议。
type Recommendation struct {
    GeneID     string
    Strategy   any     // 策略内容（来自 NetworkAsset.Strategy）
    Confidence float64
    UseCount   int
    SuccessRate float64 // SuccessCount / UseCount
}
```

## Agent YAML Schema 扩展

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-agent
spec:
  type: chat_model_agent
  # ... existing fields ...
  evolution:
    enabled: true
    collect:
      - tool_success
      - tool_error
      - model_invoke
      - agent_done
    advise:
      min_confidence: 0.5
      max_suggestions: 3
```

## 集成钩子（Eino Callback）

利用现有的 `callbacks.AppendGlobalHandlers()` 机制（参考 `pkg/observe/eino_callback.go`），新增 `EvolutionCallback`：

```go
// pkg/evolution/callback.go

func NewEvolutionCallback(engine *Engine) callbacks.Handler {
    cb := &evolutionCallback{engine: engine}
    return callbacks.NewHandlerBuilder().
        OnEndFn(cb.OnEnd).
        OnErrorFn(cb.OnError).
        Build()
}

// OnEnd 在 Tool/Model 调用成功时采集信号并异步 Share 到 Experience Repo。
func (c *evolutionCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    signal := buildSignal(info, output, nil)
    go c.engine.collector.Collect(ctx, signal) // 异步，不阻塞主流程
    return ctx
}

// OnError 在 Tool/Model 调用失败时采集失败信号。
func (c *evolutionCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    signal := buildSignal(info, nil, err)
    go c.engine.collector.Collect(ctx, signal)
    return ctx
}
```

## main.go 集成

```go
import "github.com/superagent-ai/superagent-base/backend/pkg/evolution"

// 在 application.Init(ctx) 之后、agentBuilder 之前：
evoEngine, err := evolution.Init(ctx, evolution.LoadConfigFromEnv())
if err != nil {
    logs.Warnf("evolution engine init failed (disabled): %v", err)
} else {
    callbacks.AppendGlobalHandlers(evolution.NewEvolutionCallback(evoEngine))
    logs.Infof("evolution engine enabled: %s", evoEngine.Config().ExperienceURL)
}

// builder 可选注入 advisor:
if evoEngine != nil {
    builderOpts = append(builderOpts, agentdef.WithEvolutionAdvisor(evoEngine.Advisor()))
}
```

## 部署变更

### docker-compose-dev.yml 新增 Oris 服务

```yaml
services:
  oris-experience:
    image: ghcr.io/colin4k1024/oris-experience-repo:latest
    ports:
      - "8090:8090"
    environment:
      - DATABASE_URL=sqlite:///data/genes.db
    volumes:
      - oris-data:/data

volumes:
  oris-data:
```

### 环境变量

```env
# Evolution (Oris)
EVOLUTION_ENABLED=true
ORIS_EXPERIENCE_URL=http://localhost:8090
ORIS_HUB_URL=                           # 可选，联邦模式
ORIS_API_KEY=your-api-key
ORIS_SEED=base64-encoded-32-byte-seed
ORIS_SENDER_ID=superagent-node-1
EVOLUTION_MIN_CONFIDENCE=0.5
EVOLUTION_MAX_SUGGESTIONS=3
```

---

## 实施阶段

### Phase 1: 基础框架 + 信号采集
- `go get github.com/Colin4k1024/Oris/sdks/go`
- 创建 `pkg/evolution/` 包骨架
- 实现 Eino Callback 采集 Tool/Model 执行信号
- 通过 `experience.Client.Share()` 异步上报信号
- Docker Compose 增加 Oris Experience Repo 容器
- 环境变量配置

### Phase 2: 经验查询 + Advisor
- 实现 EvolutionAdvisor（通过 `experience.Client.Fetch()` 查询相似 Gene）
- 在 AgentBuilder 中注入 Advisor
- ReAct agent system prompt 增强（附加已有经验建议）
- Agent YAML schema 扩展 `spec.evolution`

### Phase 3: Hub 联邦 + 多实例共享
- 接入 `hub.Client`（注册 + 心跳 + 发现）
- 多 superagent-base 实例共享进化经验
- Prometheus metrics 暴露进化指标

### Phase 4: Workflow + Model Router + Admin
- Workflow 节点级信号采集
- Model Router 策略进化（基于历史路由成功率调整）
- Admin API 暴露 Gene 列表/搜索
- 前端 Evolution 管理页面

## 前置条件

- [x] Oris Go SDK 已发布：`go get github.com/Colin4k1024/Oris/sdks/go`
- [ ] Oris Experience Repo Docker 镜像可用
- [ ] Experience Repo 支持 Gene 语义搜索 + 置信度过滤

## 验证方式

1. **Phase 1 验证**：Agent 对话后，检查 Experience Repo 中是否有新 Gene（`curl http://localhost:8090/experience?limit=5`）
2. **Phase 2 验证**：相同类型任务第二次执行时，Advisor 返回推荐策略
3. **Phase 3 验证**：多实例部署，实例 A 的经验在实例 B 可查询
4. **全链路验证**：Prometheus 面板展示 `evolution_signals_total`, `evolution_genes_shared`, `evolution_recommendations_served`
