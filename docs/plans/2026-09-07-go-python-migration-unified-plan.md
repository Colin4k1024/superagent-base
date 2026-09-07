# Go + Python Migration — Unified Direction Plan

> **Issue:** #16
> **Date:** 2026-09-07
> **Status:** Proposed
> **Owner:** tech-lead / architect

## 1. Current state

| Track | Status | Evidence |
|---|---|---|
| Go backend: eino → Google ADK Go | **Done.** Dual-framework flag in `backend/pkg/agentdef/builder.go` (`"eino"` default / `"adk"` Google ADK Go); ADK runner/stream paths live in `adk_runner.go`, `adk_stream.go`; eino remains only as a fallback path. | In-repo code |
| Python side: AgentScope → DeerFlow 2.0 | **Planned.** Full migration plan exists at `docs/plans/2026-08-30-python-agentscope-to-deerflow-2-migration.md`: embed DeerFlow Harness behind an anti-corruption layer, preserve `/api/v2` + A2UI SSE contract, Python 3.12 floor, LangGraph for deterministic agents. | Existing plan |

Both tracks independently chose to leave their original framework. The two
migrations have **not** been reconciled into a single product direction. This
document does that.

## 2. The three options

| Option | Description | Trade-off |
|---|---|---|
| **A — Go long-term** | Consolidate all agent + workflow runtime on Go (ADK Go); sunset the Python side. | Loses the Python ML/research ecosystem (DeerFlow, LangGraph, data tools) that justifies the DeerFlow migration. Forfeits the investment already scoped. |
| **B — Python long-term** | Consolidate on Python (DeerFlow/LangGraph); sunset the Go backend. | Discards the completed ADK Go migration and Go's throughput/concurrency advantage for the high-volume API gateway path. |
| **C — Coexistence (recommended)** | Go and Python each own a clear lane, bound by a stable HTTP/SSE contract. | Requires interface discipline and a contract boundary; avoids forcing one ecosystem to do the other's weakest job. |

## 3. Recommendation — Option C (coexistence)

Split responsibilities by where each ecosystem is strongest, not by politics.

- **Go (ADK Go) owns:** the public API gateway, the high-throughput agent
  runtime, request fan-out, RBAC, MCP/tool registry, and the workflow DAG
  engine (`pkg/agentdef/workflow_builder.go`). Go's typed concurrency and low
  p99 latency suit the synchronous chat/tool-call hot path.
- **Python (DeerFlow/LangGraph) owns:** data pipelines, long-horizon research
  workflows, notebook-style experimentation, and any capability that leans on
  the Python ML stack (retrieval pipelines, evaluation harnesses, custom
  research agents).

The boundary is the existing `/api/v2` HTTP/SSE contract plus A2UI event
schema — **not** shared in-process state. Neither side imports the other's
internals; they communicate over the contract that both migrations already
commit to preserving.

### Why not A or B

- A discards the DeerFlow plan and the Python ecosystem's research strength.
- B discards a completed Go migration and regresses gateway throughput.
- C is the only option that keeps both completed/scoped investments productive.

## 4. Boundary contract (non-negotiable)

1. Go is the **only** externally exposed entrypoint (`/api/v2`).
2. Python services are **internal**, reached via a private Go-side proxy or
   internal HTTP; never exposed directly to clients.
3. The A2UI SSE event schema is the canonical interop surface — both sides
   emit/translate to it, never to each other's internal types.
4. No shared mutable state; session/checkpoint namespaces are partitioned per
   the DeerFlow plan's §6 isolation rules.
5. Feature selection per request is explicit (agent YAML declares
   `runtime: go|python`); no silent cross-runtime delegation.

## 5. Timeline recommendation

| Phase | Window | Work |
|---|---|---|
| **P1 — Stabilize boundary** | Now – 2 wk | Freeze `/api/v2` + A2UI schema as the contract; add contract tests asserting both runtimes emit identical event shapes; document the `runtime:` selector in agent YAML. |
| **P2 — Execute Python migration** | 2–8 wk | Run the existing DeerFlow migration plan to completion; Go remains the gateway, Python becomes the research-workflow runtime behind it. |
| **P3 — Lanes harden** | 8–12 wk | Python research pipelines callable from Go workflows as a typed tool node (one new `WorkflowNode` type calling the internal Python endpoint); OTel spans cross both runtimes. |
| **P4 — Steady state** | 12 wk+ | Routine iteration within lanes; revisit only if one lane's workload clearly fits the other better (track via the P1 contract metrics). |

## 6. Decision

Adopt Option C. Go and Python coexist with split lanes, bound by the frozen
`/api/v2` + A2UI contract. Proceed with the DeerFlow migration as the Python
lane's next step; keep Go as the permanent gateway + high-throughput runtime.
