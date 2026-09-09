# ADK Dev Visual Orchestration Toolchain — Replacement Evaluation

> **Issue:** #15
> **Date:** 2026-09-07
> **Status:** Proposed
> **Owner:** architect / tech-lead

## 1. Background

ADK Dev is the visual orchestration + debugging toolchain shipped with
`google/adk-go`. It provides a canvas-style workflow editor, a node/edge
graph debugger, step-by-step tracing, and a dry-run surface tightly coupled
to the Google ADK runtime.

The Go backend has completed its migration to Google ADK Go as the
LLM framework (see `backend/pkg/agentdef/builder.go`, `adk_runner.go`,
`adk_stream.go`). With Google ADK now the primary framework, ADK Dev loses
its main value proposition: framework-native debugging. This document evaluates
what — if anything — should replace it.

## 2. Current state of in-repo orchestration assets

The project **already owns** the pieces ADK Dev used to provide externally:

| Capability | In-repo location | Notes |
|---|---|---|
| Custom workflow DAG engine | `backend/pkg/agentdef/workflow_builder.go` | `WorkflowAgent` executes a topologically-sorted DAG of nodes/edges with `{{.var}}` template variable passing and a concurrency-safe state map (`workflow_state.go`). |
| Canvas serialization schema | `backend/domain/workflow/entity/vo/canvas.go` | `Canvas{Nodes,Edges,Versions}` — the frontend-facing workflow definition. |
| Canvas ↔ runtime adaptor | `backend/domain/workflow/internal/canvas/adaptor/` | `from_node.go`, `to_schema.go` translate canvas nodes to `WorkflowNode`s and back. |
| Validation | `backend/domain/workflow/internal/canvas/validate/` | `canvas_validate.go` enforces structural rules. |
| Example fixtures | `backend/domain/workflow/internal/canvas/examples/` | 20+ JSON fixtures covering LLM, code, loop, interrupt, aggregator, chatflow. |
| Visual editor (frontend) | `web/src/pages/WorkflowEditorPage.tsx`, `web/src/components/workflow/` | Node palette (LLM/Code/Agent/Condition/Tool), property panel, serializer. |
| Runtime tracing hook | `backend/pkg/observe/adk_callback.go`, trace helpers | OpenTelemetry-style callbacks already emit per-node spans. |

**Conclusion:** the project does not lack a visual editor or a canvas format —
it lacks the **dev/debugging surface** ADK Dev layered on top (live trace
visualization, dry-run, node inspection). That gap is small and in-repo.

## 3. Candidate replacements

| Candidate | Fit | Verdict |
|---|---|---|
| **Keep ADK Dev** | Tightly coupled to legacy eino compose; we left eino as the primary framework. Would require maintaining a now-orphaned eino dependency for tooling only. | ❌ Reject — drags a deprecated framework back in. |
| **Dify** | Full opinionated platform with its own runtime, LLM gateway, and data layer. Replaces far more than the editor; conflicts with our ADK Go runtime and `/api/v2` contract. | ❌ Reject — scope sprawl, runtime conflict. |
| **Langflow** | LangChain/Python visual builder. Own Python runtime; misaligned with a Go-first backend and our DeerFlow Python track. | ❌ Reject — wrong language ecosystem. |
| **Flowise** | Node.js/LangChain visual builder. Own runtime; duplicates our Go DAG engine and adds a second JS service. | ❌ Reject — redundant runtime. |
| **n8n** | General-purpose workflow automation, not LLM-agent native. Weak fit for agent loop / tool-calling / SSE semantics. | ❌ Reject — wrong domain. |
| **Build in-repo dev surface (recommended)** | Extend the existing canvas + DAG engine with the missing dev/debug features. No new runtime, no new framework, preserves the `/api/v2` contract. | ✅ Recommended. |

## 4. Recommendation — invest in the in-repo dev surface

Adopting an external orchestration platform would import a second runtime that
competes with the ADK Go agent runtime and the Python DeerFlow track. The
cheapest, lowest-risk path is to fill the specific ADK Dev gap on top of assets
we already own.

### 4.1 Scope of the in-repo dev surface

1. **Live trace visualization.** The `adk_callback.go` / OpenTelemetry spans
   already emit per-node timing and I/O. Wire these spans into the existing
   `WorkflowEditorPage` canvas so a running execution highlights nodes, streams
   intermediate variables, and shows errors inline — mirroring ADK Dev's
   debug view without its legacy coupling.

2. **Dry-run / single-node replay.** Add a `?dry_run=1` execution mode to
   `WorkflowAgent` that runs the DAG against a user-supplied input fixture
   (reuse the `examples/` JSON set) and returns per-node outputs without
   persisting side effects. This replaces ADK Dev's step debugger.

3. **Node inspector.** Expose `safeState` snapshots per node via a read-only
   admin endpoint so the property panel can render variable resolution at each
   hop. No new schema — read what the DAG engine already computes.

4. **Canvas versioning.** Reuse the P2 "Agent version management" roadmap item
   (`configs/agents/.history/`) to version workflow canvases, giving the
   editor undo/history parity with ADK Dev.

### 4.2 Explicit non-goals

- Do **not** introduce Dify/Flowise/Langflow as a runtime.
- Do **not** reintroduce an eino dependency for tooling.
- Do **not** build a new canvas format — the existing `Canvas` VO + adaptor
  layer is the canonical format.

## 5. Timeline

| Phase | Work | Effort |
|---|---|---|
| Phase 1 (1 wk) | Span→canvas highlight wiring in `WorkflowEditorPage`; dry-run flag in `WorkflowAgent`. | S |
| Phase 2 (1–2 wk) | Node inspector endpoint + property panel integration; example-fixture dry-run harness. | M |
| Phase 3 (ongoing) | Canvas versioning (rides P2 roadmap); trace export to existing OTel backend. | M |

## 6. Risks

| Risk | Mitigation |
|---|---|
| Editor bloat from trace wiring | Keep trace as an overlay mode, not a separate page. |
| Dry-run diverges from live semantics | Dry-run shares the exact `WorkflowAgent.Execute` path; only side-effect persistence is gated. |
| External platform temptation resurfaces | Record this ADR; require a new evaluation to overturn it. |

## 7. Decision

Replace ADK Dev with the in-repo dev surface described in §4. Do not adopt
an external visual orchestration platform. The project already owns the DAG
engine, canvas schema, and editor — only the debug/trace layer is missing,
and it is small enough to build in-repo.
