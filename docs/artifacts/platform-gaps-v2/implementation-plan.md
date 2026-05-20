# 平台能力 Gap V2 — 实施计划

## 总体排期

基于依赖关系和复杂度，推荐以下实施顺序：

```
Wave 1 (基础，无依赖)
├── Gap 3: ReAct Memory 集成      [2天] — 改动最小，立即见效
├── Gap 4: Workflow 并行执行       [3天] — 独立模块，无外部依赖
└── Gap 5: K8s 部署修正           [2天] — 修正现有 Helm chart

Wave 2 (核心，依赖 Wave 1 验证)
├── Gap 2: Model Router 实时反馈   [4天] — 需要新增组件
└── Gap 1: Supervisor V2          [5天] — 最复杂，依赖 tool 系统稳定

总计：~16 人天
```

---

## Gap 3: ReAct Memory 集成 [2天]

### Phase 1: V1 首尾保存 [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 3.1 | 修改 `einoReactAgent.Chat` 接收 sessionID | `builder.go:1238` | 编译通过 |
| 3.2 | 启动时加载历史消息（复制 einoChatAgent 模式） | `builder.go:1238-1250` | 单测：mock memBackend，验证 GetMessages 被调用 |
| 3.3 | 流结束后保存 assistant 响应 | `builder.go:1260-1275` | 单测：验证 AddMessage 被调用 |
| 3.4 | 保存 user 消息到 memory | `builder.go:1245` | 同上 |

### Phase 2: 集成测试 [1天]

| # | 任务 | 验证 |
|---|------|------|
| 3.5 | 编写集成测试：多轮对话场景 | ReAct Agent 第二轮能引用第一轮内容 |
| 3.6 | 验证与 interrupt/checkpoint 的兼容 | 中断后 resume 历史不丢失 |
| 3.7 | 更新 research-agent.yaml 示例添加 memory 配置 | Agent YAML 示例完整 |

### 验收标准

- [ ] ReAct Agent 多轮对话能引用之前的上下文
- [ ] Memory 后端可通过 YAML `spec.memory` 配置
- [ ] 不影响现有 einoChatAgent 行为
- [ ] 测试覆盖：加载/保存/空历史/memory 禁用

---

## Gap 4: Workflow 并行执行 [3天]

### Phase 1: 基础并行化 [1.5天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 4.1 | 实现 `safeState` 并发安全封装 | 新建 `workflow_state.go` | 单测：并发读写无 race |
| 4.2 | 实现 `topologicalLevels()` 按层输出 | `workflow_builder.go` | 单测：DAG 层级正确 |
| 4.3 | 实现 `executeLevel()` 并行执行 | `workflow_builder.go` | 单测：并行节点确实并行 |
| 4.4 | 改造 `WorkflowAgent.Chat()` 使用 level-based 执行 | `workflow_builder.go:74-96` | 现有测试仍通过 |

### Phase 2: 错误处理与配置 [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 4.5 | Schema 扩展：`WorkflowExecution` 类型 | `schema.go` | YAML 解析正确 |
| 4.6 | 实现 fail_fast 策略（context cancel） | `workflow_builder.go` | 单测：一个失败取消其余 |
| 4.7 | 实现 best_effort 策略 | `workflow_builder.go` | 单测：收集所有错误 |
| 4.8 | A2UI progress 事件上报 | `workflow_builder.go` | 客户端收到并行节点事件 |

### Phase 3: Checkpoint 兼容 [0.5天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 4.9 | 实现 `WorkflowCheckpoint` 结构 | `workflow_builder.go` | 序列化/反序列化正确 |
| 4.10 | 实现 resume 跳过已完成 level/节点 | `workflow_builder.go` | 中断后 resume 跳过已完成 |

### 验收标准

- [ ] 同层无依赖节点并行执行（`go test -race` 无报告）
- [ ] `max_parallelism` 配置生效
- [ ] fail_fast/best_effort 策略正确
- [ ] Checkpoint 可恢复并行中断的 workflow
- [ ] 现有串行 workflow 行为不变（`max_parallelism=1` 等价串行）

---

## Gap 5: K8s 部署修正 [2天]

### Phase 1: 修正现有问题 [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 5.1 | readinessProbe 改为 `/ready` | `helm/.../deployment.yaml` | helm template 输出正确 |
| 5.2 | 增加 terminationGracePeriodSeconds: 60 | `deployment.yaml` | 同上 |
| 5.3 | 增加 preStop lifecycle hook | `deployment.yaml` | 同上 |
| 5.4 | 补充 gRPC Ingress 模板 | 新建 `grpc-ingress.yaml` | helm template 生成 |

### Phase 2: 增强 [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 5.5 | 补充 ServiceMonitor 模板 | 新建 `servicemonitor.yaml` | 条件渲染正确 |
| 5.6 | HPA 增加自定义指标支持 | `hpa.yaml` + `values.yaml` | 配置可选启用 |
| 5.7 | 补充 existingSecret 支持 | `secret.yaml` + `values.yaml` | 有 existingSecret 时不创建 |
| 5.8 | 补充 git-sync sidecar 配置（可选） | `deployment.yaml` + `values.yaml` | `gitSync.enabled=true` 时注入 |
| 5.9 | 更新 values.yaml 文档注释 | `values.yaml` | 每个字段有说明 |

### 验收标准

- [ ] `helm template` 输出正确无报错
- [ ] `helm lint` 通过
- [ ] readinessProbe 指向 `/ready`
- [ ] gRPC 和 HTTP 分别有 Ingress
- [ ] ServiceMonitor 可选启用
- [ ] 文档化所有 values 字段

---

## Gap 2: Model Router 实时反馈 [4天]

### Phase 1: FeedbackCollector [1.5天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 2.1 | 实现 `ProviderStats` 结构 | 新建 `modelrouter/feedback.go` | 单测：EMA 计算正确 |
| 2.2 | 实现 `FeedbackCollector`（RecordLatency/Outcome/Tokens） | `feedback.go` | 单测：并发安全、冷启动 |
| 2.3 | 实现 `Score()` 加权评分 | `feedback.go` | 单测：权重影响排序 |
| 2.4 | 实现 `CircuitBreaker` 熔断逻辑 | 新建 `modelrouter/circuit_breaker.go` | 单测：open/half-open/closed 状态转换 |

### Phase 2: AdaptiveStrategy [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 2.5 | 实现 `AdaptiveStrategy` 满足 Strategy 接口 | 新建 `modelrouter/adaptive_strategy.go` | 单测：选择最高分 model |
| 2.6 | 降级逻辑：数据不足回退 fallback | `adaptive_strategy.go` | 单测：冷启动走 fallback |
| 2.7 | 集成到 `DefaultRouter`（作为 strategies 数组元素） | `router.go` | 现有 Route 逻辑不变 |

### Phase 3: 数据采集集成 [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 2.8 | 在 LLM 调用完成处注入 Record 调用 | `builder.go` 或 Eino callback | 调用后 stats 更新 |
| 2.9 | Config 扩展：解析 `feedback` + `pricing` 配置 | `config.go` | YAML 解析正确 |
| 2.10 | 更新 `routing-rules.yaml` 示例 | `configs/models/` | 文档完整 |

### Phase 4: 验证 [0.5天]

| # | 任务 | 验证 |
|---|------|------|
| 2.11 | 集成测试：模拟多 provider，验证动态路由 | 高延迟 provider 被降权 |
| 2.12 | 压力测试：高并发下 FeedbackCollector 性能 | 无 race，无明显锁争抢 |

### 验收标准

- [ ] 延迟高的 provider 自动降权
- [ ] 错误率超阈值触发熔断
- [ ] 冷启动回退静态规则
- [ ] `feedback.enabled: false` 时完全禁用
- [ ] 现有静态路由行为不受影响
- [ ] `go test -race` 无报告

---

## Gap 1: Supervisor V2 [5天]

### Phase 1: delegate 工具 [1.5天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 1.1 | 定义 `DelegateToolInput/Output` 结构 | 新建 `orchestration_delegate.go` | 编译通过 |
| 1.2 | 实现 `delegateTool` 满足 Eino InvokableTool 接口 | `orchestration_delegate.go` | 单测：调用子 Agent 并返回结果 |
| 1.3 | 注册 delegate tool 到 supervisor mainAgent | `builder.go:862-899` | 构建后 supervisor 有 delegate 工具 |

### Phase 2: 多轮循环 [2天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 1.4 | 重写 `SupervisorAgent.Chat()` 为多轮循环 | `orchestration.go` | 单测：执行 delegation + final_answer |
| 1.5 | 实现并行 delegation 执行（goroutine + sem） | `orchestration.go` | 单测：parallel_max 生效 |
| 1.6 | 实现结果聚合（concat/summarize/structured） | `orchestration.go` | 单测：三种模式输出正确 |
| 1.7 | A2UI 事件：agent_switch + progress | `orchestration.go` | 客户端收到调度事件 |

### Phase 3: 错误恢复与 Schema [1天]

| # | 任务 | 文件 | 验证 |
|---|------|------|------|
| 1.8 | Schema 扩展：`DelegationConfig` | `schema.go` | YAML 解析正确 |
| 1.9 | 实现 timeout/retry/fallback_strategy | `orchestration.go` | 单测：超时触发 fallback |
| 1.10 | Checkpoint 集成：每轮保存状态 | `orchestration.go` | 中断后 resume 从正确 round 继续 |

### Phase 4: 验证与示例 [0.5天]

| # | 任务 | 验证 |
|---|------|------|
| 1.11 | 编写 supervisor 示例 YAML（project-manager） | 可正常加载运行 |
| 1.12 | 集成测试：多子 Agent 协作场景 | Supervisor 成功调度并汇总 |
| 1.13 | 验证与热重载的兼容性 | 子 Agent 热更后 supervisor 引用正确 |

### 验收标准

- [ ] Supervisor 通过 tool_call 委派子 Agent
- [ ] 多轮循环正常执行直到 final_answer 或 maxRounds
- [ ] 并行 delegation 生效
- [ ] 超时/重试/fallback 策略正确
- [ ] 中断/resume 可恢复多轮状态
- [ ] 示例 YAML 可开箱即用

---

## 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| Eino ReAct 对历史消息格式敏感 | Gap 3 可能需要调整消息格式 | 先用简单 user/assistant 历史测试 |
| 并行 workflow 引入 race condition | Gap 4 稳定性 | 严格 `go test -race`，Level-based 保证隔离 |
| Supervisor tool_call 依赖模型能力 | Gap 1 弱模型不支持 | 提供文本解析 fallback 路径 |
| FeedbackCollector 重启丢失状态 | Gap 2 冷启动 | minSamples 阈值兜底 |
| Helm chart 改动影响现有部署 | Gap 5 兼容性 | 所有新特性默认 disabled |

## 测试策略

每个 Gap 的测试分层：

1. **单元测试**：每个新函数/方法独立测试
2. **集成测试**：组件间交互（如 ReAct + Memory、Router + Feedback）
3. **Race 检测**：所有并发改动必须 `go test -race`
4. **回归测试**：现有 `make test` 全量通过
5. **E2E**：关键场景端到端验证（Supervisor 调度、Workflow 并行）
