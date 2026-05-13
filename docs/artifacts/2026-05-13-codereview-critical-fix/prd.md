# PRD — Code Review Critical Fix

**slug**: `codereview-critical-fix`
**状态**: draft
**日期**: 2026-05-13
**主责**: tech-lead
**阶段**: intake

---

## 背景

2026-05-12 code review (`docs/code-review-2026-05-12.md`) 结论为 **BLOCK**。
审查范围覆盖 `backend/pkg/agentdef/`、`backend/pkg/skill/`、`backend/pkg/memory/`、`backend/pkg/a2ui/`、`backend/cmd/examples/`、`configs/agents/`。

核心问题：
- 4 个 CRITICAL — 涉及任意命令执行、goroutine 泄漏、缓存碰撞、死锁
- 9 个 HIGH — 涉及内存泄漏、数据竞争、key 注入、热重载数据丢失
- E2E 通过率 90.4%（47/52），5 个 skipped 测试待启用

当前代码不可安全上线，必须完成 CRITICAL + HIGH 修复后才能放行。

## 目标与成功标准

| 目标 | 成功标准 |
|------|----------|
| 消除 CRITICAL 安全/稳定风险 | C1-C4 全部修复并附带回归测试 |
| 修复 HIGH 级别问题 | H1-H9 全部修复，无新增竞态/泄漏 |
| 回归验证通过 | `make test` 全绿 + E2E 47/47 通过（skipped 恢复为可选） |
| Code review 结论转为 PASS | 二次 review 无 CRITICAL/HIGH 残留 |

## 用户故事

1. 作为运维人员，我需要代码执行节点有沙箱隔离，以免恶意 YAML 导致宿主机被攻陷。
2. 作为开发者，我需要 goroutine 在 HTTP 断连后正确退出，避免服务内存持续增长。
3. 作为用户，我需要缓存 key 不碰撞，不会收到别人的会话响应。
4. 作为平台，我需要 EventStream 在高负载下不死锁，SSE 推送稳定可用。

## 范围

### In Scope

**第一优先（CRITICAL，立即修复）**

| ID | 问题 | 文件 | 修复方向 |
|----|------|------|----------|
| C1 | 代码执行节点无沙箱 | `workflow_builder.go:200-271` | `env -i` + 资源限制 + 输入转义加固 |
| C2 | `executeCodeNode` 丢弃调用方 context | `workflow_builder.go:248` | 改用 `context.WithTimeout(ctx, ...)` |
| C3 | `cacheKey` 哈希碰撞 | `builder.go:638-643` | 字段间插入 `\x00` 分隔符 |
| C4 | `EventStream.Send` 死锁 | `pkg/a2ui/stream.go:36-42` | channel send 移到锁外或用 atomic closed |

**第二优先（HIGH，合并前修复）**

| ID | 问题 | 文件 | 修复方向 |
|----|------|------|----------|
| H1 | goroutine 无 ctx.Done() guard | 多处 builder/orchestration/interrupt | 统一 select { case <-ctx.Done() } |
| H2 | 内存 map 无上限无清理 | `interrupt.go:92`, `builder.go:586` | 加 maxSize + TTL 淘汰 |
| H3 | retryAgent backoff 不响应 ctx | `builder.go:499` | select + timer 替代 time.Sleep |
| H4 | DeleteState 非原子读改写 | `pkg/memory/builtin/builtin.go:261` | 改用 HDel 原子操作 |
| H5 | Redis key 未转义注入风险 | `pkg/memory/builtin/builtin.go:94,132` | 对 sessionID/userID 做 sanitize |
| H6 | Letta URL path 注入 | `pkg/memory/letta/client.go:155` | `url.PathEscape(agentID)` |
| H7 | UUID 用 math/rand 而非 crypto/rand | `pkg/skill/builtin/builtin.go:156` | 改用 `crypto/rand` |
| H8 | two-pass buildAll 替换整个 map | `runtime.go:186` | 改为增量合并，失败不影响已有 agent |
| H9 | workflow model.primary 被全局覆盖 | `workflow_builder.go:134` | YAML 声明优先，全局为 fallback |

### Out of Scope

- MEDIUM / LOW 级别问题（后续迭代处理）
- 新增功能开发
- 前端变更
- E2E skipped 测试恢复（可选，非阻塞）

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| C1 沙箱改动影响现有 workflow 执行语义 | 行为变更 | 先加 env 隔离 + 白名单，不改 YAML schema |
| H8 热重载改为增量合并增加复杂度 | 新 bug | 配套单元测试覆盖 add/remove/update 三场景 |
| 修复涉及并发代码，可能引入新竞态 | 回归 | 跑 `go test -race ./pkg/...` |

## 决策记录（原待确认项）

1. **C1 沙箱方案 → 容器隔离**：直接上容器方案（Docker exec / nsjail），不做临时 `env -i` 过渡。
2. **H2 内存 map maxSize → 1000**：确认上限 1000 条，超限 LRU 淘汰。
3. **H8 热重载失败 → 需要告警**：失败时保留已有 agent 不替换 + 输出 WARN 日志 + 暴露 Prometheus counter。

---

## 企业治理待确认项

- 本项目为开源项目，非企业内部应用，无需判断 T1-T4 等级。
- 无数据合规 / 跨境风险。
- 无 private enterprise overlay 需求。

## 领域技能包启用建议

| 技能 | 原因 |
|------|------|
| `golang-patterns` | Go 并发/错误处理最佳实践 |
| `security-review` | C1 沙箱 + H5/H6 注入防御 |
| `tdd-workflow` | 所有修复需配套回归测试 |

## UI 范围

无前端变更，不涉及 UI 门禁。

## 参与角色清单

| 角色 | 职责 |
|------|------|
| `tech-lead` | 仲裁修复优先级、收口 review |
| `backend-engineer` | 执行 C1-C4、H1-H9 代码修复 |
| `qa-engineer` | 验证修复、跑 E2E + race 测试 |
| `architect` | 评审 C1 沙箱方案（容器 vs 最小隔离）|

## 需求挑战会候选分组

| 分组 | 议题 | 参与者 |
|------|------|--------|
| 安全组 | C1 沙箱深度 + H5/H6 注入防御策略 | architect, backend-engineer, tech-lead |
| 稳定性组 | C4 死锁 + H1 goroutine 生命周期治理 | backend-engineer, tech-lead |

---

## 修复计划摘要

```
Phase 1: CRITICAL 修复（预计 1 天）
  ├── C2 context 传递（5 min，最简单）
  ├── C3 cacheKey 分隔符（15 min）
  ├── C4 EventStream 死锁（1h）
  └── C1 代码执行沙箱（2-4h，含测试）

Phase 2: HIGH 修复（预计 1-2 天）
  ├── H3 retry context-aware backoff（30 min）
  ├── H4 DeleteState 原子操作（30 min）
  ├── H5 Redis key sanitize（30 min）
  ├── H6 URL path escape（15 min）
  ├── H7 crypto/rand UUID（15 min）
  ├── H9 workflow model 优先级（30 min）
  ├── H1 goroutine ctx.Done guard（2h，多文件）
  ├── H2 内存 map 上限 + TTL（1h）
  └── H8 热重载增量合并（1-2h）

Phase 3: 验证（预计半天）
  ├── make test（全绿）
  ├── go test -race ./pkg/...
  ├── E2E 回归
  └── 二次 code review → PASS
```

---

*已创建 `docs/artifacts/2026-05-13-codereview-critical-fix/prd.md`*
