# PRD — Observability Fix (P0 + P1)

**slug**: `observability-fix`
**状态**: draft
**日期**: 2026-05-13
**主责**: tech-lead
**阶段**: intake
**来源**: `docs/observability-issues-2026-05-13.md`

---

## 背景

代码审查发现可观测性层存在 10 个缺陷（P0×1, P1×3, P2×3, P3×3）。核心问题：Prometheus 指标已定义但 Model/Tool 层从未上报数据，Chat SSE 主链路无任何打点，OTel 追踪基础设施就绪但业务层零 Span。

**当前结果**：
- `superagent_model_tokens_total` = 0（永远）
- `superagent_tool_invocations_total` = 0（永远）
- `superagent_active_sessions` = 0（永远）
- 无分布式链路追踪能力
- 对话成功率、首 Token 延迟等核心指标完全缺失

## 目标与成功标准

| 目标 | 成功标准 |
|------|----------|
| P0: Model/Tool 指标上线 | 触发 chat 后 `/metrics` 中 `model_tokens_total > 0`、`tool_invocations_total > 0` |
| P1-A: ActiveSessions 实时 | 并发 chat 时 `active_sessions` 正确反映在线数 |
| P1-B: Chat SSE 核心打点 | chat 请求后 `/metrics` 可见请求计数、延迟直方图 |
| P1-C: OTel Span 生效 | `OTEL_ENABLED=true` 时 Agent/Model/Tool 三层 Span 可见 |
| 回归安全 | `make test` + E2E 全绿 |

## 用户故事

1. 作为 SRE，我需要在 Grafana 中看到每个模型的 token 消耗和错误率，以便成本控制和故障告警。
2. 作为开发者，我需要知道当前并发会话数，以便评估系统负载和扩容需求。
3. 作为平台运营，我需要对话成功率和首 Token 延迟指标，以便评估用户体验。
4. 作为 oncall 工程师，我需要分布式链路追踪，以便快速定位跨组件延迟瓶颈。

## 范围

### In Scope（P0 + P1）

| ID | 问题 | 文件 | 修复方向 | 复杂度 |
|----|------|------|----------|--------|
| OBS-001 | Eino Callback 未注册 | `pkg/observe/eino_callback.go`, `pkg/agentdef/builder.go` | 在 builder 构建 agent 时注入 `callbacks.InitCallbacks(ctx, NewEinoObserveCallback())` | **1 行核心改动** |
| OBS-002 | ActiveSessions 未更新 | `api/handler/coze/chat_sse.go` | `HandleChatStream` 入口 `Inc()`，defer `Dec()` | 2 行 |
| OBS-003 | OTel Span 无业务调用 | `api/handler/coze/chat_sse.go`, `pkg/observe/tracer.go` | 在 HandleChatStream 入口创建 Agent Span；Model/Tool Span 由 OBS-001 的 callback 覆盖 | 5 行 |
| OBS-004 | Chat SSE 无 Prometheus 打点 | `api/handler/coze/chat_sse.go` | 请求计数 + 延迟直方图（复用 `observe.AgentRequestsTotal` / `AgentRequestDuration`） | 10 行 |

### Out of Scope

- P2: AccessLog 误判、Workflow Trace 打通、K8s 监控栈（后续迭代）
- P3: errorType 分类、provider 解析、结构化日志（后续迭代）
- 前端 Monitor Dashboard 变更（已独立实现）
- Grafana Dashboard JSON 模板

## 关键假设

1. `callbacks.InitCallbacks` 是 Eino 框架的标准注入方式，注入后 callback 自动覆盖 model.Generate 和 tool.Invoke 调用。
2. OBS-001 修复后，`superagent_model_*` 和 `superagent_tool_*` 指标自动上报，无需额外改动 builder 逻辑。
3. OTel Span 仅在 `OTEL_ENABLED=true` 时实际上报到 Collector，默认关闭不影响性能。

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| Eino callback 注入位置不当导致重复计数 | 指标膨胀 | 验证每次 chat 只注入一次 ctx |
| OTel Span 开启后增加请求延迟 | 性能退化 | 默认关闭，通过 env 控制；benchmark 验证 overhead < 5ms |
| ActiveSessions `Dec()` 未触发（goroutine panic） | gauge 永增不减 | 用 `defer` 保证执行 |

## 待确认项

1. `callbacks.InitCallbacks` 是在每次 `Chat()` 调用时注入，还是在 builder 构建时一次性注入？需确认 Eino 框架行为。
2. OTel Span 的 `service.name` 是否需要统一为 `SERVICE_NAME` 环境变量值？
3. Chat SSE 打点是否需要区分 legacy mode 和 A2UI mode？

---

## 企业治理待确认项

- 开源项目，非企业内部应用
- 无合规 / 数据风险
- 无 private enterprise overlay 需求

## 领域技能包启用建议

| 技能 | 原因 |
|------|------|
| `golang-patterns` | Eino callback、context 注入模式 |
| `observability` (domain skill) | Prometheus + OTel 最佳实践 |

## UI 范围

无前端变更（Monitor Dashboard 已独立实现并展示 `/metrics` 数据）。

## 参与角色清单

| 角色 | 职责 |
|------|------|
| `tech-lead` | 确认 Eino callback 注入方式、收口验证 |
| `backend-engineer` | 实现 OBS-001 ~ OBS-004 |
| `qa-engineer` | 验证指标上报、E2E 回归 |

## 需求挑战会候选分组

| 分组 | 议题 | 参与者 |
|------|------|--------|
| 可观测性组 | OBS-001 callback 注入位置 + OBS-003 Span 粒度 | architect, backend-engineer, tech-lead |

---

## 修复计划摘要

```
Phase 1: P0 — OBS-001 (Eino callback 注册)
  └── 1 行代码，解锁全部 Model/Tool 指标 + OTel Span

Phase 2: P1 — OBS-002/003/004 (Chat SSE 可观测性)
  ├── OBS-002: ActiveSessions Inc/Dec (2 行)
  ├── OBS-003: Agent Span 创建 (5 行)
  └── OBS-004: 请求计数 + 延迟直方图 (10 行)

Phase 3: 验证
  ├── curl chat → 检查 /metrics 指标非零
  ├── OTEL_ENABLED=true → 验证 Span 输出
  └── make test + E2E 回归
```

**预估总工时**：< 2 小时（核心改动 < 20 行代码）

---

*已创建 `docs/artifacts/2026-05-13-observability-fix/prd.md`*
