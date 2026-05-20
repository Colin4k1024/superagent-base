---
name: observability
description: >
  Set up and query Superagent's observability stack: OpenTelemetry traces,
  Prometheus metrics, and Grafana dashboards. TRIGGER when: debugging latency or
  errors via traces, adding custom metrics, configuring OTel collector, reading
  Grafana dashboards, or when the user asks "如何查看 trace", "metrics 怎么看",
  "observability 怎么接入", "add a span to my code".
  DO NOT TRIGGER when: working on agent logic, model routing, or frontend UI.
origin: learned
tags: [observability, otel, tracing, prometheus, grafana, metrics, logs]
---

# Observability

Source: `backend/pkg/observe/`. Three pillars: traces (OTel), metrics (Prometheus), logs (structured).

## Start the Full Stack

```bash
make debug   # includes OTel Collector, Prometheus, Grafana + backend
```

| Service | URL |
|---------|-----|
| Grafana | http://localhost:3001 (admin/admin) |
| Prometheus | http://localhost:9090 |
| OTel Collector | grpc :4317, http :4318 |
| Backend metrics | http://localhost:8888/metrics |

## Key Dashboards (Grafana)

- **Agent Overview** — request rate, p50/p95/p99 latency, error rate per agent
- **Tool Calls** — invocation counts, latency, retry rate per tool
- **Model Routing** — requests per model, tokens used, fallback frequency
- **Streaming** — SSE connection count, event throughput, interrupt rate

## Traces (OpenTelemetry)

Every agent invocation creates a root span. Child spans cover:
- Model calls (with `model`, `tokens_in`, `tokens_out` attributes)
- Tool calls (with `tool.name`, `tool.duration_ms`)
- Checkpoint save/load
- MCP server calls

### Add a Custom Span (Go)

```go
ctx, span := observe.Tracer().Start(ctx, "my.operation",
    trace.WithAttributes(
        attribute.String("agent.name", agentName),
        attribute.Int("items.count", len(items)),
    ),
)
defer span.End()

// Mark error
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### Query Traces (Grafana Tempo / Jaeger)

```
# Find slow agent calls
{service.name="superagent"} | duration > 5s

# Find failed tool calls
{span.name=~"tool.*"} | status=error
```

## Metrics (Prometheus)

Key metrics exposed at `/metrics`:

| Metric | Type | Labels |
|--------|------|--------|
| `superagent_requests_total` | Counter | `agent`, `status` |
| `superagent_request_duration_seconds` | Histogram | `agent` |
| `superagent_tool_calls_total` | Counter | `tool`, `status` |
| `superagent_model_tokens_total` | Counter | `model`, `type` (prompt/completion) |
| `superagent_active_sessions` | Gauge | — |

### Add a Custom Metric (Go)

```go
var myCounter = observe.MustRegisterCounter("superagent_my_events_total",
    "Total my events", []string{"event_type"})

// Increment
myCounter.WithLabelValues("webhook").Inc()
```

## Structured Logging

```go
// Use the context-aware logger (includes trace_id automatically)
log := observe.LoggerFromCtx(ctx)
log.Info("agent started", "agent", name, "session", sessionID)
log.Error("tool failed", "tool", toolName, "error", err)
```

Log fields auto-correlated to traces via `trace_id` and `span_id`.

## OTel Collector Config

```yaml
# docker/otel-collector-config.yaml
exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  otlp/tempo:
    endpoint: "tempo:4317"
  loki:
    endpoint: "http://loki:3100/loki/api/v1/push"
```
