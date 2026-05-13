# 可观测性问题清单 — 2026-05-13

> 基于代码审查，梳理当前监控、日志、Agent 会话采集等可观测性能力的现状与缺口。

---

## 整体评估

| 能力 | 状态 |
|------|------|
| Agent 请求计数 / 延迟 | ✅ 正常上报 |
| Agent 热重载失败计数 | ✅ 正常上报 |
| HTTP 访问日志 | ✅ 正常工作 |
| OTel 分布式追踪基础设施 | ⚠️ 就绪但默认关闭，业务层无 Span |
| Model Token / 延迟 / 错误指标 | ❌ 定义但未接入 |
| Tool 调用指标 | ❌ 定义但未接入 |
| 活跃会话数 | ❌ 定义但未接入 |
| Agent 会话级链路追踪 | ❌ 缺失 |

---

## 问题清单

### P0 — 阻塞级

#### [OBS-001] Eino Callback 从未注册，Model / Tool 层全部可观测性失效

- **文件**：`pkg/observe/eino_callback.go`
- **现象**：`NewEinoObserveCallback()` 在整个代码库中只有定义，没有任何调用点。导致以下指标和 Span 永远为零：
  - `superagent_model_tokens_total`
  - `superagent_model_request_duration_seconds`
  - `superagent_model_errors_total`
  - `superagent_tool_invocations_total`
  - `superagent_tool_invocation_duration_seconds`
  - OTel `model.invoke` / `tool.invoke` Span
- **修复方向**：在 `pkg/agentdef/builder.go` 构建 Eino graph 时注入 callback：
  ```go
  ctx = callbacks.InitCallbacks(ctx, observe.NewEinoObserveCallback())
  ```
- **影响**：无法观测模型费用、调用失败率、工具性能，是生产运营的盲区。

---

### P1 — 高优先级

#### [OBS-002] `ActiveSessions` Gauge 定义但从未更新

- **文件**：`pkg/observe/metrics.go:81`，`api/handler/coze/chat_sse.go`
- **现象**：`superagent_active_sessions` Gauge 已定义，但 `chat_sse.go` 中的 `HandleChatStream` 和 `HandleChatResume` 没有在请求开始/结束时调用 `Inc()` / `Dec()`。
- **修复方向**：
  ```go
  // HandleChatStream 入口
  observe.ActiveSessions.Inc()
  defer observe.ActiveSessions.Dec()
  ```
- **影响**：无法监控并发会话负载，无法设置过载告警。

#### [OBS-003] OTel 追踪默认关闭，业务层无任何 Span

- **文件**：`pkg/observe/tracer.go`，`pkg/observe/config.go`
- **现象**：`OTEL_ENABLED` 默认为 false（no-op shutdown）。即使启用，`StartAgentSpan` / `StartModelSpan` / `StartToolSpan` 三个工厂方法在业务代码中均无调用点。
- **修复方向**：
  1. 生产环境将 `OTEL_ENABLED=true` 写入部署配置。
  2. 在 `HandleChatStream` 入口调用 `observe.StartAgentSpan`。
  3. Eino callback（见 OBS-001）负责 model/tool 层 Span，修复 OBS-001 后自动覆盖。
- **影响**：无分布式链路，无法排查跨组件延迟问题。

#### [OBS-004] Chat SSE 层无任何可观测性接入点

- **文件**：`api/handler/coze/chat_sse.go`
- **现象**：`HandleChatStream`（对话主链路）和 `HandleChatResume`（中断恢复）中没有：
  - Prometheus 计数器 / 直方图
  - OTel Span
  - 请求级结构化日志（只依赖全局 `AccessLogMW` 的粗粒度日志）
- **影响**：对话成功率、首 Token 延迟、流式完成率等核心业务指标全部缺失。

---

### P2 — 中优先级

#### [OBS-005] `AccessLogMW` 对静态文件和有扩展名路径跳过 body 日志，但判断逻辑依赖文件扩展名，存在误判

- **文件**：`api/middleware/log.go:76`
- **现象**：
  ```go
  if requestAuthType != int32(RequestAuthTypeStaticFile) && filepath.Ext(path) == "" {
  ```
  逻辑为：非静态文件 **且** 路径无扩展名才记录详情。带 `.json` 后缀的 API 路径（如果存在）会被误判为静态文件跳过记录。
- **修复方向**：改为仅用 `RequestAuthType` 判断，去掉 `filepath.Ext` 条件。

#### [OBS-006] Workflow Trace 与 OTel 两套体系未打通

- **文件**：`api/handler/coze/workflow_service.go:628`
- **现象**：Workflow 执行追踪（`ListRootSpans` / `GetTraceSDK`）数据存在 MySQL，是 Coze Studio 原有机制，与 OTel Collector 完全独立，无法在统一的 Trace 视图中关联 Agent → Workflow → Model 的完整调用链。
- **影响**：排障时需要在两个系统间人工对照，效率低。

#### [OBS-007] K8s 监控栈仅存在于 e2e 目录，生产 Helm chart 无监控配置

- **文件**：`k8s/e2e/monitoring.yaml`，`helm/charts/superagent/values.yaml`
- **现象**：Prometheus + Grafana 部署文件只在 `k8s/e2e/` 下（测试用途），Helm chart 的 `values.yaml` 中引用了 `prometheus.io/scrape` annotation 但没有配套的 Prometheus / Grafana chart dependency。
- **影响**：生产部署无开箱即用的监控栈，需手动搭建。

---

### P3 — 低优先级 / 改进项

#### [OBS-008] `errorType` 函数分类过于粗糙

- **文件**：`pkg/observe/eino_callback.go:165`
- **现象**：`errorType(err)` 只返回 `"error"` 或 `"none"`，无法区分超时、限流、上下文取消、模型服务不可用等类型。
- **修复方向**：使用 `errors.Is` 对常见错误类型分支分类，提升告警精度。

#### [OBS-009] `resolveModelInfo` 无法提取 provider 信息

- **文件**：`pkg/observe/eino_callback.go:154`
- **现象**：`provider` 硬编码返回 `"unknown"`，`model_id` 依赖 `info.Name`，当 Eino 组件未设置 Name 时会为空字符串。
- **影响**：即使修复 OBS-001 后，按 provider 维度的模型费用和错误率分析仍不可用。

#### [OBS-010] 日志未集成结构化字段，LogID 仅通过 context value 传递

- **文件**：`pkg/logs/logger.go`，`api/middleware/log.go:86`
- **现象**：`pkg/logs` 基于 Hertz 默认 logger，`CtxInfof` 等方法没有自动从 context 提取 LogID 并附加到日志行。LogID 只通过 `X-Log-ID` 响应头返回，日志本身无法与请求关联。
- **修复方向**：替换为 `zerolog` / `zap` 等结构化 logger，在 `CtxInfof` 中自动提取并附加 `log_id` 字段。

---

## 修复优先级建议

```
P0: OBS-001 (Eino callback 注册) — 一行代码，解锁全部 Model/Tool 指标
P1: OBS-002 (ActiveSessions), OBS-003 (OTel 启用), OBS-004 (Chat SSE 打点)
P2: OBS-005, OBS-006, OBS-007
P3: OBS-008, OBS-009, OBS-010
```

P0 修复成本极低，收益最大，建议优先处理。
