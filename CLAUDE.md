# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Superagent Base is an open-source AI Agent development platform built on Coze Studio. It provides a full backend for building, deploying, and managing AI agents with declarative YAML definitions, multi-model routing, tool calling, multi-agent orchestration, workflow DAG execution, interrupt/resume, A2UI protocol streaming, and observability.

**Stack**: Go 1.24 (backend), React 18 + Vite 5 + Tailwind 3.4 (frontend), MySQL + Redis + MinIO + ES + Milvus + NSQ/Kafka

## Build & Run Commands

### Development (recommended)

```bash
make dev                  # Start MySQL + Redis + backend server
make dev-middleware        # Start only MySQL + Redis via Docker
make dev-server           # Build and run backend (requires middleware running)
make dev-down             # Stop dev containers
make dev-clean            # Stop containers + delete data
```

### Build & Test

```bash
make build                # Compile to bin/superagent
make test                 # Run pkg/... tests (core packages only)
make test-all             # Run all Go tests in backend/
```

Run a single Go test:
```bash
cd backend && go test ./pkg/agentdef/... -run TestAgentDefBuild -count=1
cd backend && go test ./pkg/tool/... -v -count=1
```

### Frontend

```bash
cd web && npm run dev     # Dev server on port 3500, proxies /api -> :8888
cd web && npm run build   # TypeScript check + Vite build
```

### Full Debug Stack

```bash
make debug                # MySQL + Redis + ES + MinIO + NSQ + backend (requires docker/.env.debug)
make fe                   # Build frontend assets into bin/resources/static
```

### Proto Generation

```bash
make proto-gen            # Generate Go code from api/proto/ definitions
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
```

### E2E Tests

```bash
cd tests/e2e && bash run_tests.sh
```

## Architecture

The backend follows a layered DDD-inspired structure inherited from Coze Studio, with the `pkg/agentdef` declarative agent layer added on top:

```
backend/
  main.go            ← Entry: Init app, AgentBuilder, AgentRuntime, gRPC :50051, Hertz :8888
  api/               ← HTTP handlers + gRPC handlers + middleware + router
  application/       ← App services (conversation, workflow, plugin, memory)
  crossdomain/       ← Cross-domain service interfaces
  domain/            ← Domain logic (agent, app, connector, conversation, knowledge)
  infra/             ← Infrastructure adapters (cache/Redis, checkpoint, document, ES, ORM)
  pkg/               ← Core packages (the important new layer)
    agentdef/        ← Declarative agent runtime (schema, parser, builder, runtime, orchestration)
    a2ui/            ← A2UI protocol: SSE event types + encoder
    modelrouter/     ← Model routing strategies (capability/cost/latency) + fallback
    mcp/             ← MCP Client (stdio/SSE) + Server + Registry
    memory/          ← Memory backends (builtin, mem0, zep, letta)
    skill/           ← SkillsHub (Local/HTTP/Composite Invoker, built-in skills)
    tool/            ← Tool Manager + middleware chain (retry/timeout/ratelimit/cache)
    observe/         ← OpenTelemetry + Prometheus + Eino callback
  cmd/sactl/         ← CLI: skill search/install/uninstall, agent apply
  conf/              ← YAML configs (model templates, plugin defs, prompts)
```

**Application bootstrap**: `application.Init(ctx)` runs three phases (basicServices → primaryServices → complexServices), then wires cross-domain defaults. This progressive initialization with DI is the core startup pattern.

**Agent runtime two-pass build**: Leaf agents built first, then orchestration agents resolve sub-agent references from the registry. This order is critical — orchestration agents cannot reference agents that haven't been registered yet.

**Hot-reload**: Agent YAML files in `configs/agents/` are watched via fsnotify; changes propagate automatically without server restart.

**TurnLoop**: ADK-backed chat agents automatically use Eino TurnLoop for push/preempt/abort semantics. `TurnLoopManager` in `pkg/agentdef/turnloop.go` manages per-(agent, session) loops. Non-ADK agents fallback to `Agent.Chat()`.

## Key Design Patterns

- **Kubernetes-like YAML schema**: Agent definitions use `apiVersion: superagent/v1`, `kind: Agent`, `metadata`, `spec` — inspired by K8s resource manifests.
- **Tool URI schemes**: `builtin/<name>` for built-in tools, `mcp://server/tool` for MCP tools, `skill://name` for skills — consistent URI-based reference system.
- **A2UI event protocol**: Structured SSE streaming with typed events (text, thinking, tool_call, tool_result, code_block, interrupt, error, done, progress, agent_switch). See `docs/a2ui-protocol.md`.
- **Agent types**: `chat_model_agent`, `deep_agent`, `supervisor`, `sequential`, `parallel`, `plan_execute`, `workflow` — each has distinct execution semantics.
- **Interrupt/resume**: Agents with `spec.interrupt.enabled=true` save checkpoint state; resume via `POST /api/v1/chat/resume`.
- **Eino ReAct agent**: When tools are present, builder wraps ChatModel in `react.NewAgent` with `MaxStep: 10`.
- **Middleware pipeline**: ContextCache → RequestInspector → SetHost → SetLogID → CORS → AccessLog → OpenapiAuth → SessionAuth → I18n — applied in strict order.

## Configuration

- **Env-based**: `.env` files with `APP_ENV` suffix for environments. Copy `.env.example` to `backend/.env`.
- **Agent YAMLs**: `configs/agents/*.yaml` — declarative agent definitions loaded at startup.
- **Model routing**: `configs/models/routing-rules.yaml` — strategy definitions and provider configs.
- **Dev Docker**: `docker/docker-compose-dev.yml` (lightweight: MySQL + Redis only).
- **Debug Docker**: `docker/docker-compose-debug.yml` (full stack: MySQL + Redis + ES + MinIO + NSQ).

## Ports

| Service | Port |
|---------|------|
| Hertz HTTP | 8888 |
| gRPC | 50051 |
| Web UI (dev) | 3500 |
| MySQL | 3306 |
| Redis | 6379 |

## Conventions

- **Language**: Documentation and code comments are predominantly Chinese; README is in Chinese.
- **Go indentation**: Tabs, indent width 4 (per .editorconfig).
- **TS/JS indentation**: Spaces, indent width 2.
- **Go module path**: `github.com/superagent-ai/superagent-base/backend`
- **Frontend proxy**: Vite dev server forwards `/api` → `localhost:8888`, `/grpc` → `localhost:50051`.

## Key Files to Understand First

- `backend/main.go` — Server entry point and full bootstrap sequence
- `backend/pkg/agentdef/schema.go` — AgentDefinition struct (the core data model)
- `backend/pkg/agentdef/builder.go` — How agents are constructed from YAML definitions
- `backend/pkg/agentdef/runtime.go` — AgentRuntime lifecycle and two-pass build
- `backend/pkg/a2ui/types.go` — A2UI EventType enum and protocol definition
- `backend/pkg/modelrouter/` — Model routing strategies and fallback logic
- `configs/agents/research-agent.yaml` — Canonical agent YAML example