# Model Router 实时反馈设计

## 背景

当前 Model Router 三种策略（Capability/Cost/Latency）本质都是 `ruleBasedStrategy` 的命名包装，执行相同的 `Match` 字段精确匹配逻辑。`LatencyStrategy` 注释明确为 "extension point"，当前只是占位。路由决策完全基于静态规则，无法感知 provider 实时健康状况。

## 目标

引入实时反馈闭环：收集每个 provider 的延迟、成功率、成本 → 聚合为可评分的指标 → 动态调整路由权重 → 降级时回退到静态规则。

## 设计方案

### 核心组件

#### ProviderStats — 单 provider 指标

```go
type ProviderStats struct {
    mu           sync.RWMutex
    latencyEMA   float64   // TTFT 指数移动平均（秒）
    tpsEMA       float64   // tokens/second EMA
    successRate  float64   // 成功率 EMA
    costPerToken float64   // USD per token EMA
    lastUpdated  time.Time
    sampleCount  int64
}
```

#### FeedbackCollector — 聚合器

```go
type FeedbackCollector struct {
    stats      map[string]*ProviderStats // key = modelID
    alpha      float64                   // EMA 衰减系数，默认 0.1
    minSamples int                       // 冷启动阈值，默认 5
    staleDur   time.Duration             // 过期时间，默认 5m
    pricing    PricingTable              // 从配置加载
}
```

#### AdaptiveStrategy — 动态路由策略

```go
func (as *AdaptiveStrategy) Select(ctx context.Context, req *RouteRequest) (string, error) {
    if !as.collector.HasSufficientData() {
        return as.fallback, nil // 降级
    }
    bestModel, bestScore := "", 0.0
    for _, candidate := range as.candidates {
        if as.collector.IsCircuitOpen(candidate) {
            continue
        }
        score := as.collector.Score(candidate, as.weights)
        if score > bestScore {
            bestModel, bestScore = candidate, score
        }
    }
    if bestModel == "" {
        return as.fallback, nil
    }
    return bestModel, nil
}
```

### 指标采集

#### 延迟

```go
func (fc *FeedbackCollector) RecordLatency(modelID string, ttft, totalDur time.Duration, outputTokens int) {
    stats := fc.getOrCreate(modelID)
    stats.mu.Lock()
    defer stats.mu.Unlock()
    stats.latencyEMA = fc.alpha*ttft.Seconds() + (1-fc.alpha)*stats.latencyEMA
    if outputTokens > 0 {
        tps := float64(outputTokens) / totalDur.Seconds()
        stats.tpsEMA = fc.alpha*tps + (1-fc.alpha)*stats.tpsEMA
    }
    stats.lastUpdated = time.Now()
    stats.sampleCount++
}
```

#### 成功率

```go
func (fc *FeedbackCollector) RecordOutcome(modelID string, outcome Outcome) {
    successSample := 0.0
    if outcome == OutcomeSuccess { successSample = 1.0 }
    stats.successRate = fc.alpha*successSample + (1-fc.alpha)*stats.successRate
}
```

#### 成本

```go
func (fc *FeedbackCollector) RecordTokens(modelID string, inputTokens, outputTokens int) {
    cost := float64(inputTokens)/1000*pricing.InputPer1K +
            float64(outputTokens)/1000*pricing.OutputPer1K
    stats.costPerToken = fc.alpha*(cost/float64(inputTokens+outputTokens)) +
                        (1-fc.alpha)*stats.costPerToken
}
```

### 评分公式

```go
func (fc *FeedbackCollector) Score(modelID string, w ScoreWeights) float64 {
    latencyScore := 1.0 / (1.0 + stats.latencyEMA*10)
    successScore := stats.successRate
    costScore := 1.0 / (1.0 + stats.costPerToken*1000)
    return w.Latency*latencyScore + w.Success*successScore + w.Cost*costScore
}
```

默认权重：`latency=0.3, success=0.5, cost=0.2`

### 数据流

```
请求完成 → RecordLatency + RecordOutcome + RecordTokens
                     ↓
              内存 EMA 更新（O(1)）
                     ↓
下次路由 → AdaptiveStrategy.Select → Score 排序 → 返回最优 model
                     ↓ (同时)
              Prometheus 指标（观测/告警）
```

### 熔断器

```go
type CircuitBreaker struct {
    errorThreshold float64       // 默认 0.5
    cooldown       time.Duration // 默认 30s
    probeInterval  time.Duration // 默认 10s
    state          CBState       // closed | open | half-open
}
```

### 降级策略

| 故障场景 | 行为 |
|---------|------|
| 某 model 样本不足 | 使用静态配置，不参与动态排序 |
| 所有 model 无数据 | 完全回退静态策略 |
| 单个 provider 熔断 | 从候选池移除 |
| `feedback.enabled: false` | AdaptiveStrategy 不注册 |
| 评分 panic | recover 后返回 ErrNoMatch，下一个静态策略接管 |

### 配置扩展

```yaml
# configs/models/routing-rules.yaml
strategies:
  - name: adaptive
    mode: weighted-score       # weighted-score | ucb1 | thompson
    candidates: [deepseek-r1, claude-sonnet, gpt-4o]
    weights:
      latency: 0.3
      success: 0.5
      cost: 0.2
    ema_alpha: 0.1
    min_samples: 5
    stale_duration: 5m
    fallback: gpt-4o

providers:
  deepseek-r1:
    pricing:
      input_per_1k: 0.001
      output_per_1k: 0.002

feedback:
  enabled: true
  circuit_breaker:
    error_threshold: 0.5
    cooldown: 30s
    probe_interval: 10s
```

### Prometheus vs 路由决策

| 维度 | Prometheus（观测） | FeedbackCollector（决策） |
|------|-------------------|--------------------------|
| 存储 | 持久化 | 内存，进程重启丢失 |
| 时效 | 15-60s scrape | 实时（每次请求后） |
| 用途 | 告警/面板/SLO | 路由闭环 |

## 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 聚合算法 | EMA (alpha=0.1) | O(1) 内存，自然衰减 |
| V1 排序 | 加权评分 | 简单可解释 |
| 反馈存储 | 进程内存 | 避免外部依赖，延迟最低 |
| 降级通道 | 利用 strategies 数组顺序 | 零改动 Router 核心循环 |

## 风险

- 进程重启丢失所有 EMA 状态（可选从 Prometheus 预热）
- 单 Pod 指标不代表全局（多副本需要聚合，或各 Pod 独立决策）
- Alpha 参数需根据流量模式调优

## 关键代码位置

- `backend/pkg/modelrouter/router.go:106-143` — Route 主循环
- `backend/pkg/modelrouter/strategy.go:94-129` — 当前三策略
- `backend/pkg/modelrouter/config.go:20-47` — 配置结构
- `backend/pkg/observe/metrics.go:41-59` — 已有模型指标
