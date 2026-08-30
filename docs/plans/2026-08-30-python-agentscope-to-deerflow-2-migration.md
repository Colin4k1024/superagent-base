# Python AgentScope to DeerFlow 2.0 Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Python service's AgentScope 2.0 runtime with DeerFlow 2.0 while preserving the existing Superagent `/api/v2` HTTP/SSE contract, agent YAML compatibility, and matrix deployment behavior.

**Architecture:** Keep FastAPI and the Superagent public API as the product boundary. Add a small runtime port inside `superagent`, implement it with the embedded `DeerFlowClient`/compiled LangGraph agent, and translate DeerFlow stream/state events into the existing A2UI event schema. Use DeerFlow for model calls, tools, skills, MCP, sandbox, memory, checkpointing, and dynamic subagents; retain deterministic sequential/parallel/workflow semantics as thin LangGraph compositions over the DeerFlow-backed leaf agents.

**Tech Stack:** Python 3.12, FastAPI, DeerFlow 2.0 Harness, LangChain, LangGraph, Pydantic 2, `uv`, pytest/pytest-asyncio, SSE, SQLite for local checkpoints, PostgreSQL for production checkpoints.

---

## 1. Decisions and non-negotiable boundaries

1. **Embed the Harness; do not mount or fork the DeerFlow App.** `superagent.server` remains the only external API. The embedded `DeerFlowClient` is a process singleton and is reused across requests.
2. **Preserve external contracts.** Existing `/api/v2` routes, request/response envelopes, agent IDs, session IDs, and A2UI SSE names remain unchanged during the migration.
3. **Raise the Python floor from 3.11 to 3.12.** DeerFlow Harness requires Python 3.12+.
4. **Pin source immutably.** The official install page still says the public package is not released. Start with a Git dependency pinned to the exact DeerFlow `v2.0.0` commit/tag and its `backend/packages/harness` subdirectory. Do not track `main`. If implementation proves that current documented APIs only exist in a later 2.x release, record an ADR and pin that exact 2.x tag/commit instead.
5. **Use an anti-corruption layer.** No FastAPI route, SDK-facing schema, or product orchestration class should import `deerflow.*` or `langgraph.*` directly. Those imports stay under `superagent/runtime/deerflow/`.
6. **Do not replace deterministic workflows with probabilistic delegation.** `SequentialAgent`, `ParallelAgent`, and `WorkflowAgent` remain deterministic and move to LangGraph. `SupervisorAgent` and `AgentLoopAgent` map to DeerFlow lead-agent/subagent/plan-mode features.
7. **Dual runtime is temporary.** A feature flag exists only for shadow/canary verification. The definition of done includes deleting AgentScope, its imports, its event types, and the flag's AgentScope branch.

## 2. Compatibility map

| Current surface | DeerFlow 2 target | Migration rule |
|---|---|---|
| `Agent.reply()` / `reply_stream()` | `DeerFlowClient.ainvoke()` / `astream()` or compiled graph `ainvoke()` / `astream()` | Wrap behind `AgentRuntime` |
| AgentScope `AgentState` | LangGraph thread/checkpoint state | `session_id` maps 1:1 to `thread_id` |
| AgentScope message/event classes | LangChain messages + DeerFlow stream events | Normalize once, then map to A2UI |
| `OpenAIChatModel` | `langchain_openai.ChatOpenAI` | Preserve OpenAI-compatible `base_url` |
| `DashScopeChatModel` | a verified LangChain DashScope/Qwen provider | Decide through a contract spike; never silently route it through OpenAI |
| AgentScope `Toolkit` / `ToolBase` | LangChain tools + DeerFlow configured tools | Wrap current callables with typed schemas |
| AgentScope MCP client | DeerFlow MCP via `extensions_config.json` | Preserve Superagent admin endpoints through an adapter |
| Local `BuiltinMemory` / Redis memory | DeerFlow checkpointer + memory storage | Separate thread state from long-term memory |
| Manual `AgentLoopAgent` | DeerFlow lead agent with `plan_mode`, limits and middleware | Remove repeated prompt concatenation |
| `SupervisorAgent` | DeerFlow subagent registry/task delegation | Preserve declared child allow-list |
| Sequential/parallel/workflow agents | LangGraph `StateGraph` nodes calling DeerFlow leaf agents | Preserve ordering, fan-out, variables and error semantics |
| AgentScope permission events | DeerFlow guardrails + LangGraph interrupts | Translate to existing `interrupt` and resume API |

## 3. Target package layout

```text
python/
├── config/
│   ├── deerflow.yaml
│   └── extensions_config.json
├── src/superagent/
│   ├── runtime/
│   │   ├── protocol.py
│   │   ├── factory.py
│   │   └── deerflow/
│   │       ├── client.py
│   │       ├── config_mapper.py
│   │       ├── events.py
│   │       ├── models.py
│   │       ├── tools.py
│   │       ├── mcp.py
│   │       └── state.py
│   └── agents/
│       ├── base.py
│       ├── chat.py
│       ├── supervisor.py
│       ├── agentloop.py
│       └── graphs.py
└── tests/
    ├── contract/
    ├── runtime/
    └── integration/
```

## 4. Implementation tasks

### Task 1: Freeze the current behavior with contract tests

**Files:**
- Create: `python/tests/contract/test_agent_contract.py`
- Create: `python/tests/contract/test_a2ui_stream_contract.py`
- Create: `python/tests/contract/test_api_contract.py`
- Create: `python/tests/fixtures/runtime_events.py`
- Modify: `python/tests/test_server.py`

**Steps:**

1. Add tests for `BaseAgent.run`, `run_msg`, `run_stream`, `describe`, state get/set, and unsupported resume behavior.
2. Add golden tests for every SSE event currently consumed by the SDK/frontend: `progress`, `text_start`, `text`, `text_end`, `thinking_start`, `thinking`, `thinking_end`, `tool_call`, `tool_call_delta`, `tool_call_end`, `tool_result_start`, `tool_result_delta`, `tool_result`, `model_call_end`, `hint`, `interrupt`, `error`, and `done`.
3. Assert that one user turn invokes the runtime only once; this catches the current fallback path that can stream and then call `run()` again.
4. Add HTTP contract tests for `/health`, `/ready`, `/api/v2/agents`, `/api/v2/chat/stream`, `/api/v2/chat/resume`, state, sessions, skills, tools, and MCP admin routes.
5. Use fake runtime events; no live model, network, Docker, or API key is allowed in unit tests.
6. Run `cd python && uv run pytest tests/contract tests/test_server.py -q`. Expected before later tasks: all characterization tests pass against the AgentScope implementation.
7. Commit: `test(python): freeze agent and A2UI runtime contracts`.

### Task 2: Prove and lock the DeerFlow 2 dependency

**Files:**
- Create: `docs/adr/2026-08-30-deerflow-harness-version.md`
- Create: `python/scripts/probe_deerflow_api.py`
- Modify: `python/pyproject.toml`
- Create: `python/uv.lock`

**Steps:**

1. In a disposable `uv` environment, install `deerflow-harness` from the immutable DeerFlow `v2.0.0` source tag/commit and `backend/packages/harness` subdirectory.
2. The probe must import and exercise: configuration loading, `DeerFlowClient`, `create_deerflow_agent`, `RuntimeFeatures`, async invoke, async stream, thread IDs, checkpoint state, interrupts/resume, custom tools, and MCP initialization.
3. Capture the actual stream event shapes to a temporary JSON file and compare them with the assumptions in Task 1. Do not code the event adapter from documentation examples alone.
4. Record in the ADR: selected commit SHA, package name, import prefix, Python requirement, supported install method, discovered API gaps, and why v2.0.0 or a later pinned 2.x tag was chosen.
5. Change `requires-python` and Ruff/mypy targets from 3.11 to 3.12.
6. Replace `agentscope>=2.0.0` with the pinned DeerFlow Harness source. Add only direct provider/checkpointer packages actually required by the selected configuration.
7. Generate and commit `python/uv.lock`; run `cd python && uv sync --frozen --extra dev` in a clean environment. Expected: reproducible install with no undeclared transitive dependency reliance.
8. Run `cd python && uv run python scripts/probe_deerflow_api.py`. Expected: every required capability reports `PASS`; otherwise stop and revise the ADR before continuing.
9. Commit: `build(python): pin DeerFlow 2 harness runtime`.

### Task 3: Introduce a framework-neutral runtime port

**Files:**
- Create: `python/src/superagent/runtime/__init__.py`
- Create: `python/src/superagent/runtime/protocol.py`
- Create: `python/src/superagent/runtime/factory.py`
- Create: `python/tests/runtime/test_runtime_protocol.py`

**Steps:**

1. Define immutable `RunContext(thread_id, agent_id, model_name, metadata)`, `RuntimeEvent(type, data)`, and `RuntimeResult(text, messages, usage, state)` models.
2. Define an `AgentRuntime` protocol with `ainvoke`, `astream`, `aresume`, `aget_state`, `aset_state`, and `aclose`.
3. Make `runtime.factory` select `agentscope` or `deerflow` from `SUPERAGENT_PY_RUNTIME`; default to `agentscope` until Task 12.
4. Add fake runtime tests that prove all agent classes can be tested without either framework installed.
5. Run `cd python && uv run pytest tests/runtime/test_runtime_protocol.py -q`. Expected: PASS.
6. Commit: `refactor(python): add framework-neutral agent runtime port`.

### Task 4: Add DeerFlow configuration without breaking shared agent YAML

**Files:**
- Create: `python/config/deerflow.yaml`
- Create: `python/config/extensions_config.json`
- Create: `python/src/superagent/runtime/deerflow/config_mapper.py`
- Modify: `python/src/superagent/config/schema.py`
- Modify: `python/src/superagent/config/loader.py`
- Create: `python/tests/runtime/test_deerflow_config_mapper.py`

**Steps:**

1. Add a minimal DeerFlow config with `config_version`, models, local sandbox, tools, skills, subagents, memory, summarization, token usage, checkpointer, and guardrails.
2. Default local sandbox to `allow_host_bash: false`. Production must use the container sandbox; host bash is never enabled by default.
3. Keep `configs/agents/*.yaml` as the shared product schema. Extend the Python Pydantic models to explicitly represent fields already present in those files (`description`, `tags`, model extras, memory, middleware, observability, evolution, `max_turns`, and workflow variables) instead of silently dropping them.
4. Implement a pure mapper from `AgentDefinition` to DeerFlow runtime settings: named agent, model, system prompt, tool groups, enabled skills, memory flag, plan mode, subagent allow-list, and execution limits.
5. Add table-driven tests for every checked-in agent YAML, including unsupported fields. Unsupported fields must fail validation with an actionable message; no silent loss of behavior.
6. Set `DEER_FLOW_CONFIG_PATH` explicitly during application startup; never depend on the process working directory.
7. Run `cd python && uv run pytest tests/runtime/test_deerflow_config_mapper.py -q`. Expected: all checked-in YAML definitions map or produce an explicit documented incompatibility.
8. Commit: `feat(python): map Superagent configuration to DeerFlow`.

### Task 5: Implement the DeerFlow client lifecycle and leaf-agent runtime

**Files:**
- Create: `python/src/superagent/runtime/deerflow/__init__.py`
- Create: `python/src/superagent/runtime/deerflow/client.py`
- Create: `python/src/superagent/runtime/deerflow/state.py`
- Create: `python/tests/runtime/test_deerflow_runtime.py`

**Steps:**

1. Load DeerFlow configuration exactly once in FastAPI lifespan.
2. Construct one reusable `DeerFlowClient` and protect initialization/reload with an async lock.
3. Implement `ainvoke`/`astream` with `thread_id=session_id`, `agent_name=agent_id`, selected model, runtime feature flags, and checkpointer config.
4. Normalize final assistant text from LangChain message objects and preserve usage/model metadata.
5. Implement state reads/writes through LangGraph checkpoint state, not a second in-memory dictionary.
6. Implement shutdown for client, MCP transports, sandbox provider, checkpointer, and background tasks.
7. Add fake-client unit tests for concurrent requests, same-thread continuity, different-thread isolation, initialization failure, cancellation, and clean shutdown.
8. Run `cd python && uv run pytest tests/runtime/test_deerflow_runtime.py -q`. Expected: PASS without a network key.
9. Commit: `feat(python): add embedded DeerFlow runtime`.

### Task 6: Replace AgentScope models and credentials

**Files:**
- Create: `python/src/superagent/runtime/deerflow/models.py`
- Modify: `python/src/superagent/models/registry.py`
- Modify: `python/src/superagent/models/router.py`
- Modify: `python/src/superagent/agents/chat.py`
- Create: `python/tests/runtime/test_deerflow_models.py`

**Steps:**

1. Change `ModelRegistry.create_model()` to return a LangChain `BaseChatModel`.
2. Map OpenAI-compatible providers to `ChatOpenAI(model, api_key, base_url, timeout, max_retries)`.
3. Select and pin the verified LangChain provider for DashScope/Qwen; add a construction test using a fake key and no outbound request.
4. Preserve `MODEL_API_KEY_N`, `MODEL_BASE_URL_N`, `MODEL_NAME_N`, `MODEL_PROVIDER_N`, and `MODEL_ID_N` compatibility, while allowing DeerFlow-native model definitions.
5. Preserve routing metadata and fallback selection; pass the selected model name into DeerFlow per-run config.
6. Add secret-redaction tests for logs and `describe()` output.
7. Run `cd python && uv run pytest tests/runtime/test_deerflow_models.py -q`. Expected: PASS.
8. Commit: `refactor(python): replace AgentScope models with LangChain models`.

### Task 7: Replace ToolBase/Toolkit with DeerFlow and LangChain tools

**Files:**
- Create: `python/src/superagent/runtime/deerflow/tools.py`
- Modify: `python/src/superagent/tools/builtin/web_search.py`
- Modify: `python/src/superagent/tools/builtin/http_request.py`
- Modify: `python/src/superagent/tools/builtin/code_execute.py`
- Modify: `python/src/superagent/tools/__init__.py`
- Modify: `python/src/superagent/tools/middleware.py`
- Modify: `python/tests/test_tools.py`
- Create: `python/tests/runtime/test_deerflow_tools.py`

**Steps:**

1. Keep the existing business functions as plain typed async functions.
2. Delete AgentScope `TextBlock`, permission, `ToolBase`, `ToolChunk`, and `Toolkit` coupling.
3. Wrap product tools as LangChain structured tools with stable names and JSON schemas.
4. Map `builtin/web_search`, `builtin/http_request`, and `builtin/code_execute` to DeerFlow tool groups. Prefer DeerFlow sandbox execution for code; retain the old subprocess implementation only behind a migration flag until sandbox parity passes.
5. Map current middleware permission allow-lists to DeerFlow guardrails. Deny dangerous tools by default and test the denial path.
6. Add tests for success, validation errors, timeout, cancellation, error serialization, and sandbox path isolation.
7. Run `cd python && uv run pytest tests/test_tools.py tests/runtime/test_deerflow_tools.py -q`. Expected: PASS.
8. Commit: `refactor(python): migrate tools to DeerFlow runtime`.

### Task 8: Move MCP and skills onto DeerFlow extension mechanisms

**Files:**
- Create: `python/src/superagent/runtime/deerflow/mcp.py`
- Modify: `python/src/superagent/tools/mcp/client.py`
- Modify: `python/src/superagent/tools/mcp/registry.py`
- Modify: `python/src/superagent/skills/manager.py`
- Modify: `python/src/superagent/server.py`
- Create: `python/tests/integration/test_deerflow_mcp.py`
- Create: `python/tests/integration/test_deerflow_skills.py`

**Steps:**

1. Translate Superagent MCP server records into DeerFlow `extensions_config.json` entries using atomic writes and an async lock.
2. Keep existing MCP admin endpoints, but delegate connect/disconnect/list-tools behavior to DeerFlow initialization/cache invalidation.
3. Do not register an MCP filesystem server; use DeerFlow's thread-scoped sandbox file tools.
4. Map installed Superagent skills to standard `SKILL.md` directories and DeerFlow enable/disable state.
5. Keep endpoint response envelopes stable even if DeerFlow metadata contains more fields.
6. Test stdio MCP with an in-process fixture server; test skill discovery from a temporary directory. No public MCP server is used in CI.
7. Run `cd python && uv run pytest tests/integration/test_deerflow_mcp.py tests/integration/test_deerflow_skills.py -q`. Expected: PASS.
8. Commit: `feat(python): route MCP and skills through DeerFlow`.

### Task 9: Replace messages, events, and SSE translation

**Files:**
- Create: `python/src/superagent/runtime/deerflow/events.py`
- Modify: `python/src/superagent/message/types.py`
- Modify: `python/src/superagent/event/types.py`
- Modify: `python/src/superagent/server.py`
- Modify: `python/tests/contract/test_a2ui_stream_contract.py`

**Steps:**

1. Treat `superagent.message` as the stable storage/API DTO layer, not a framework protocol.
2. Implement a pure event normalizer using the recorded Task 2 event fixtures. Normalize message chunks, state snapshots, tool calls/results, usage, subagent progress, errors, interrupts, and completion.
3. Map normalized events to the exact existing A2UI SSE schema. Preserve tool call IDs and emit exactly one terminal `done` or `error` event.
4. Accumulate assistant text/messages from the stream and persist once. Remove the post-stream fallback that invokes the agent a second time.
5. Handle client disconnect by cancelling the DeerFlow/LangGraph run and releasing sandbox resources.
6. Run `cd python && uv run pytest tests/contract/test_a2ui_stream_contract.py -q`. Expected: all golden events pass unchanged.
7. Commit: `refactor(python): translate DeerFlow streams to A2UI`.

### Task 10: Adopt DeerFlow checkpointing, memory, interrupt, and resume

**Files:**
- Modify: `python/src/superagent/memory/backends/builtin.py`
- Modify: `python/src/superagent/memory/backends/redis.py`
- Modify: `python/src/superagent/runtime/deerflow/state.py`
- Modify: `python/src/superagent/server.py`
- Create: `python/tests/integration/test_deerflow_threads.py`
- Create: `python/tests/integration/test_deerflow_interrupts.py`

**Steps:**

1. Map `session_id` directly to DeerFlow `thread_id`; never create a second ID for the same conversation.
2. Configure SQLite checkpointer for local/single-process use and PostgreSQL for production/multi-process use. Do not use in-memory checkpointing outside tests.
3. Separate short-term thread history/checkpoints from long-term memory CRUD. Adapt long-term memory endpoints to a DeerFlow `MemoryStorage` implementation or retain the existing store behind that interface until data migration is complete.
4. Add an idempotent migration utility for existing stored messages/memory. Store a migration version and never duplicate records on re-run.
5. Translate LangGraph/DeerFlow interrupts into `_interrupt_states`-compatible API responses during transition, then make the checkpointer the source of truth.
6. Implement resume using the exact API verified in Task 2, preserving `/api/v2/chat/resume` request and response shapes.
7. Test restart persistence, concurrent turns on one thread, interrupt → process restart → resume, invalid resume, and deletion.
8. Run `cd python && uv run pytest tests/integration/test_deerflow_threads.py tests/integration/test_deerflow_interrupts.py -q`. Expected: PASS.
9. Commit: `feat(python): migrate sessions and resume to DeerFlow checkpoints`.

### Task 11: Rebuild multi-agent orchestration on DeerFlow and LangGraph

**Files:**
- Create: `python/src/superagent/agents/graphs.py`
- Modify: `python/src/superagent/agents/sequential.py`
- Modify: `python/src/superagent/agents/parallel.py`
- Modify: `python/src/superagent/agents/workflow.py`
- Modify: `python/src/superagent/agents/supervisor.py`
- Modify: `python/src/superagent/agents/agentloop.py`
- Modify: `python/tests/test_agents.py`
- Create: `python/tests/integration/test_deerflow_orchestration.py`

**Steps:**

1. Implement deterministic sequential, fan-out/fan-in, and DAG workflow graphs with LangGraph state models.
2. Preserve workflow template substitution, topological validation, condition evaluation, variable maps, node output naming, and failure behavior.
3. Run each LLM/agent node through the DeerFlow-backed runtime port; tool/code nodes use the same guarded tool registry.
4. Map `SupervisorAgent` to DeerFlow subagents with an explicit child-agent allow-list and bounded concurrency/turn limits.
5. Map `AgentLoopAgent` to one DeerFlow long-horizon run with plan mode; delete manual prompt concatenation and `[DONE]` polling after parity tests pass.
6. Add fake-runtime tests proving order, concurrency, deterministic joins, child failure handling, recursion/turn limits, and cancellation propagation.
7. Run `cd python && uv run pytest tests/test_agents.py tests/integration/test_deerflow_orchestration.py -q`. Expected: PASS.
8. Commit: `refactor(python): run orchestration on DeerFlow and LangGraph`.

### Task 12: Integrate the runtime into FastAPI and perform shadow verification

**Files:**
- Modify: `python/src/superagent/agents/base.py`
- Modify: `python/src/superagent/config/loader.py`
- Modify: `python/src/superagent/server.py`
- Create: `python/src/superagent/runtime/shadow.py`
- Create: `python/tests/integration/test_server_deerflow.py`

**Steps:**

1. Change `BaseAgent` to depend only on `AgentRuntime`; remove AgentScope types from its public methods and state.
2. Initialize/reload/close the runtime in FastAPI lifespan and preserve atomic agent registry reload.
3. Enable `SUPERAGENT_PY_RUNTIME=deerflow` in integration tests.
4. Add opt-in shadow mode that executes a sampled request on both runtimes but returns only AgentScope output. Redact secrets and do not duplicate side-effecting tool calls; shadow only read-only/test fixtures.
5. Compare final text presence, event ordering invariants, tool names, terminal status, latency, token usage, and checkpoint continuity. Do not require byte-identical model text.
6. Set go/no-go thresholds: 100% contract tests; zero duplicate side effects; zero missing terminal SSE events; zero cross-thread state leakage; p95 latency regression no worse than the agreed budget; no unbounded task/sandbox leak.
7. Run `cd python && SUPERAGENT_PY_RUNTIME=deerflow uv run pytest tests -q`. Expected: full Python suite PASS.
8. Commit: `feat(python): switch FastAPI integration to DeerFlow`.

### Task 13: Upgrade container and matrix deployment

**Files:**
- Modify: `python/Dockerfile`
- Modify: `docker/docker-compose-matrix.yml`
- Modify: `.env.example`
- Modify: `Makefile`
- Modify: relevant CI workflow under `.github/workflows/`
- Create: `python/.dockerignore`

**Steps:**

1. Change base image to Python 3.12 and install from the frozen `uv.lock` in a multi-stage build.
2. Copy source before installing the local wheel, fixing the current Docker ordering where `pip install .` runs before `src/` is copied.
3. Copy/mount `deerflow.yaml`, `extensions_config.json`, named agent configs, skills, and the persistent thread-data directory explicitly.
4. Add environment variables for `DEER_FLOW_CONFIG_PATH`, checkpoint DSN, sandbox mode, DeerFlow root/data directory, and the temporary runtime selector.
5. Use container sandbox in production profiles; keep local sandbox with host bash disabled for developer mode.
6. Add readiness checks for configuration, checkpoint database, and runtime initialization, while keeping `/health` lightweight.
7. Add Make targets: `python-sync`, `python-test`, `python-lint`, `python-typecheck`, `python-integration`, and `python-lock-check`.
8. Build and run: `docker compose -f docker/docker-compose-matrix.yml build python-backend && docker compose -f docker/docker-compose-matrix.yml up -d python-backend`.
9. Verify: `curl -fsS http://localhost:8889/health`, `curl -fsS http://localhost:8889/ready`, then run `make matrix-test`.
10. Commit: `build(python): deploy DeerFlow runtime on Python 3.12`.

### Task 14: Remove AgentScope and temporary compatibility code

**Files:**
- Modify: every file returned by `rg -l -i 'agentscope' python`
- Delete: obsolete AgentScope-only event/permission/tool adapters after consumers are migrated
- Modify: `python/pyproject.toml`
- Modify: `python/uv.lock`

**Steps:**

1. Make DeerFlow the only runtime factory result and remove the AgentScope branch and shadow executor.
2. Remove `agentscope` from dependencies and lockfile.
3. Remove AgentScope imports, type names, docstrings, comments, and compatibility-only tests.
4. Keep Superagent message/event DTOs only where they serve API/storage compatibility.
5. Run `rg -n -i 'agentscope' python`. Expected: no matches.
6. Run `cd python && uv sync --frozen --extra dev && uv run ruff check . && uv run mypy src && uv run pytest -q`. Expected: all commands PASS.
7. Run a clean-container import check to prove the service does not receive AgentScope transitively.
8. Commit: `chore(python): remove AgentScope runtime`.

### Task 15: End-to-end acceptance, rollout, and documentation

**Files:**
- Modify: `python/README.md`
- Modify: root `README.md`
- Modify: `docs/superpowers/specs/2026-07-01-matrix-deployment-design.md`
- Modify: `docs/superpowers/plans/2026-07-01-matrix-deployment.md`
- Create: `docs/runbooks/deerflow-python-runtime.md`
- Create: `docs/migrations/agentscope-to-deerflow-2.md`

**Steps:**

1. Document installation, Python 3.12, pinned Harness source, configuration files, model providers, sandbox security, checkpoint storage, MCP, skills, and troubleshooting.
2. Document every intentional behavior difference and all YAML fields that were renamed, rejected, or reinterpreted.
3. Run API/SDK contract tests, matrix tests, frontend chat E2E, interrupt/resume E2E, MCP fixture test, skill discovery, file/sandbox test, workflow test, and a 30-minute concurrency soak.
4. Canary with DeerFlow on a small percentage of Python traffic. Monitor error rate, SSE completion rate, tool failures, checkpoint latency, sandbox leaks, model token usage, and p95/p99 latency.
5. Increase traffic in stages only when Task 12 thresholds hold. Keep the prior image digest and data backup for rollback.
6. After the agreed observation window, remove the temporary migration toggles and archive shadow metrics.
7. Commit: `docs(python): complete DeerFlow 2 migration runbook`.

## 5. Required acceptance matrix

| Capability | Required proof |
|---|---|
| Basic chat | Final text and exactly one `done` event |
| Streaming | Stable A2UI ordering and no duplicate invocation |
| Tool call | Stable name/ID, arguments, result, error and timeout mapping |
| Session continuity | Same `session_id` sees prior turn after process restart |
| Isolation | Different sessions cannot see each other's messages/files/state |
| Interrupt/resume | Interrupt survives restart and resumes exactly once |
| Abort/disconnect | Runtime task and sandbox work are cancelled and cleaned up |
| Skills | Enabled custom skill is discovered; disabled skill is unavailable |
| MCP | Dynamic enable/disable and cached tool refresh work without restart |
| Sandbox | Host bash denied by default; thread files remain scoped |
| Sequential | Child output feeds the next child in order |
| Parallel | Children run concurrently and results join deterministically |
| Workflow | DAG ordering, conditions and variable mapping remain compatible |
| Supervisor | Only declared subagents can be delegated to |
| Agent loop | DeerFlow plan/long-horizon run respects configured limits |
| Observability | Request, model, tool, subagent, token and error metrics are emitted |
| Packaging | Clean Python 3.12 container installs from frozen lock and starts |

## 6. Data migration and rollback

1. Before canary, back up Redis/SQLite/PostgreSQL state and the thread workspace directory.
2. The migration utility writes new DeerFlow checkpoints without deleting old AgentScope message/memory records.
3. During dual-runtime rollout, use separate checkpoint namespaces/tables so rollback does not read partially converted state.
4. Rollback means routing Python traffic to the previous image digest and old namespace; it does not mean downgrading the new checkpoint schema in place.
5. Do not remove old state until the observation window ends and a restore drill succeeds.

## 7. Main risks and mitigations

| Risk | Mitigation |
|---|---|
| Harness packaging/API drift | Immutable Git SHA, API probe, lockfile, ADR |
| Python 3.12 ecosystem conflicts | Clean-image install and lock check before code migration |
| SSE event-shape mismatch | Recorded DeerFlow fixtures plus golden A2UI contract tests |
| Duplicate side effects during shadowing | Shadow only pure/read-only tools; never duplicate writes |
| Session semantics differ | One-to-one `session_id`/`thread_id`, restart tests, namespaced checkpoints |
| Local sandbox grants host access | Host bash disabled; production container sandbox; guardrails |
| Current YAML silently drops fields | Explicit schemas and fail-fast compatibility report |
| Deterministic workflows become nondeterministic | Keep them as LangGraph graphs, not subagent prompts |
| MCP filesystem path mismatch | Use DeerFlow sandbox file tools, not filesystem MCP |
| Large transitive dependency footprint | Pin only required extras and test a clean wheel/container |

## 8. Definition of done

- `rg -n -i 'agentscope' python` returns no matches.
- Python runs on 3.12+ with a frozen, reproducible dependency graph.
- All Python unit, integration, contract, matrix, SDK, and relevant frontend E2E tests pass.
- Existing `/api/v2` consumers require no coordinated breaking change.
- Checkpointed conversations and interrupt/resume survive restart.
- Production uses an isolated sandbox and persistent checkpointer.
- Canary metrics meet the agreed thresholds and rollback has been drilled.
- The old AgentScope runtime, compatibility flag, and migration-only shadow code are removed.
