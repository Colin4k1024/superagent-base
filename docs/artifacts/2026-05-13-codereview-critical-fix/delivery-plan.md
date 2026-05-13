# Delivery Plan — Code Review Critical Fix

**slug**: `codereview-critical-fix`
**状态**: handoff-ready
**日期**: 2026-05-13
**主责**: tech-lead
**阶段**: plan → handoff-ready

---

## 版本目标

- **里程碑**：Code Review BLOCK → PASS
- **范围**：修复 4 CRITICAL + 9 HIGH 级别问题
- **放行标准**：`make test` 全绿 + `go test -race` 无竞态 + E2E 47/47 + 二次 review 无 CRITICAL/HIGH

---

## 需求挑战会结论

### 假设与质疑

| # | 假设 | 质疑人 | 结论 |
|---|------|--------|------|
| 1 | C1 容器沙箱不影响现有 workflow 测试 | architect | **接受**：容器执行 = 进程执行语义不变，仅隔离环境；需新增 `docker` 依赖检测，不可用时 graceful fallback + log WARN |
| 2 | H8 增量合并热重载不引入新竞态 | backend-engineer | **接受**：改为 copy-on-write 模式，build 失败不替换，需 `-race` 测试验证 |
| 3 | 所有修复可在不改 YAML schema 前提下完成 | tech-lead | **接受**：本轮修复全部向后兼容，无 breaking change |

### 未决项

无。三个决策均已锁定（容器沙箱 / maxSize=1000 LRU / 告警）。

---

## Brownfield 上下文快照

| 模块 | 文件 | 当前状态 |
|------|------|----------|
| 代码执行 | `pkg/agentdef/workflow_builder.go:200-271` | 直接 `exec.Command`，无沙箱，`context.Background()` |
| EventStream | `pkg/a2ui/stream.go:36-42` | `mu.Lock` 内 channel send，channel 满时死锁 |
| 缓存 | `pkg/agentdef/builder.go:638-643` | SHA256 无分隔符拼接 |
| Interrupt | `pkg/agentdef/interrupt.go:92` | `map[string]*interruptEntry` 无上限 |
| Cache Agent | `pkg/agentdef/builder.go:586` | `map[string]cacheEntry` 无上限 |
| Runtime | `pkg/agentdef/runtime.go:186-189` | `rt.agents = built` 整体替换 |
| Memory builtin | `pkg/memory/builtin/builtin.go:196,261` | Delete 用错 key；DeleteState 非原子 |
| Memory letta | `pkg/memory/letta/client.go:155` | URL path 未 escape |
| Skill UUID | `pkg/skill/builtin/builtin.go:156` | `math/rand` 生成 UUID |

---

## Story Slice 列表

### Slice 1: CRITICAL — 最小安全修复 (C2, C3, C4)

**目标**：消除死锁、goroutine 泄漏和缓存碰撞
**Owner**: backend-engineer
**预估**: 2h
**验收标准**:
- C2: `executeCodeNode` 使用父 ctx 派生超时 (`context.WithTimeout(ctx, ...)`)
- C3: `cacheKey` 字段间写入 `\x00` 分隔符，附加碰撞测试
- C4: EventStream.Send channel send 移到锁外，附加并发测试
- `go test -race ./pkg/agentdef/... ./pkg/a2ui/...` 全绿

### Slice 2: CRITICAL — 容器沙箱 (C1)

**目标**：代码执行节点容器隔离
**Owner**: backend-engineer
**预估**: 4h
**依赖**: 无（独立于 Slice 1）
**验收标准**:
- `executeCodeNode` 通过 `docker run --rm --network=none --memory=128m --cpus=0.5` 执行代码
- 输入通过 stdin pipe 传递，消除变量注入风险
- Docker 不可用时 graceful 降级：拒绝执行 + 返回明确错误
- 新增单元测试覆盖：正常执行、超时、Docker 不可用三场景
- 环境变量 `WORKFLOW_CODE_SANDBOX=docker|none` 可配置

### Slice 3: HIGH — goroutine 生命周期治理 (H1, H3)

**目标**：全链路 ctx.Done guard + retry 响应取消
**Owner**: backend-engineer
**预估**: 2.5h
**依赖**: Slice 1 完成（C4 修复后 stream 发送模式稳定）
**验收标准**:
- 所有 goroutine 中 channel send 统一加 `select { case <-ctx.Done(): return }` guard
- `retryAgent` backoff 改用 `select` + `time.After` 替代 `time.Sleep`
- 涉及文件：`builder.go`, `orchestration.go`, `interrupt.go`, `workflow_builder.go`
- `go test -race ./pkg/agentdef/...` 全绿

### Slice 4: HIGH — 内存治理 (H2)

**目标**：Interrupt 和 Cache map 加 maxSize + LRU 淘汰
**Owner**: backend-engineer
**预估**: 1.5h
**依赖**: 无
**验收标准**:
- `InterruptableAgent.interrupts` 加 maxSize=1000，超限淘汰最旧条目
- `cacheAgent.cache` 加 maxSize=1000，超限淘汰最旧条目
- 定期清理过期条目（可在每次访问时惰性清理）
- 新增测试验证淘汰行为

### Slice 5: HIGH — 数据安全修复 (H4, H5, H6, H7)

**目标**：原子操作 + 注入防御 + 安全随机
**Owner**: backend-engineer
**预估**: 1.5h
**依赖**: 无
**验收标准**:
- H4: `DeleteState` 改用 `HDel` 原子命令
- H5: Redis key 中 `sessionID`/`userID` 做 sanitize（仅允许 `[a-zA-Z0-9_-]`）
- H6: Letta client URL path 中 `agentID` 做 `url.PathEscape`
- H7: UUID 生成改用 `crypto/rand`
- 各项附带单元测试

### Slice 6: HIGH — 热重载与 workflow 模型 (H8, H9)

**目标**：增量合并 + 模型优先级修正
**Owner**: backend-engineer
**预估**: 2h
**依赖**: 无
**验收标准**:
- H8: `buildAll` 改为增量合并（新 build 成功才替换对应 key），失败时保留旧 agent + WARN 日志 + Prometheus counter `agentdef_reload_failures_total`
- H9: workflow `createNodeModel` 优先使用 `spec.model.primary`，全局 `modelCfg.ModelID` 作为 fallback
- `go test -race ./pkg/agentdef/...` 全绿

### Slice 7: 验证与二次 Review

**目标**：全量回归 + 二次 code review
**Owner**: qa-engineer + tech-lead
**预估**: 3h
**依赖**: Slice 1-6 全部完成
**验收标准**:
- `make test` 全绿
- `cd backend && go test -race ./pkg/...` 无竞态
- E2E `tests/e2e/run_tests.sh` 47/47 通过
- 二次 code review 结论为 PASS（无 CRITICAL/HIGH 残留）

---

## 工作拆解与执行顺序

```
         ┌── Slice 1 (C2,C3,C4) ──┐
         │                          │
Start ───┤── Slice 2 (C1 容器) ────┤── Slice 3 (H1,H3) ──┐
         │                          │                       │
         ├── Slice 4 (H2 LRU) ────┤                       ├── Slice 7 (验证)
         │                          │                       │
         └── Slice 5 (H4-H7) ─────┤── Slice 6 (H8,H9) ──┘
                                    │
                                    └──────────────────────┘
```

- Slice 1, 2, 4, 5 可并行执行
- Slice 3 依赖 Slice 1（C4 修复后确定 stream 模式）
- Slice 6 独立
- Slice 7 依赖全部完成

---

## 角色分工

| 角色 | 职责 | 交接 |
|------|------|------|
| `tech-lead` | 计划仲裁、二次 review、放行决策 | → qa-engineer (验证) → 放行 |
| `backend-engineer` | Slice 1-6 代码实现 | → tech-lead (code review) |
| `qa-engineer` | Slice 7 验证执行 | → tech-lead (放行结论) |
| `architect` | C1 容器沙箱方案评审 | → backend-engineer (实现) |

---

## 风险与依赖

| 风险 | 影响 | 缓解措施 | Owner |
|------|------|----------|-------|
| 容器沙箱需要 Docker daemon | 开发/测试环境无 Docker 时 C1 测试失败 | 提供 `WORKFLOW_CODE_SANDBOX=none` 降级开关 + 测试 skip | backend-engineer |
| goroutine guard 改动范围大 | 遗漏导致隐藏泄漏 | `go test -race` + golangci-lint `goroutineleak` | qa-engineer |
| 热重载增量合并逻辑 | 新竞态 | 专项 `-race` 测试 + 并发热重载场景 | backend-engineer |

---

## 节点检查

| 节点 | 通过标准 | 角色 |
|------|----------|------|
| 方案评审 | 本 delivery-plan 通过 | tech-lead |
| 开发完成 | Slice 1-6 代码提交 + 单元测试全绿 | backend-engineer |
| 测试完成 | Slice 7 验证通过 | qa-engineer |
| 发布准备 | 二次 review PASS | tech-lead |

---

## 技能装配清单

| 技能 | 类型 | 触发原因 | 主责 |
|------|------|----------|------|
| `golang-patterns` | domain | 并发/错误处理 | backend-engineer |
| `security-review` | quality-gate | C1 沙箱 + 注入防御 | architect |
| `tdd-workflow` | process | 修复需配套回归测试 | backend-engineer |
| `systematic-debugging` | process | 竞态/死锁定位 | backend-engineer |

---

## Implementation-Readiness 结论

| 维度 | 状态 | 说明 |
|------|------|------|
| 需求挑战 | PASS | 3 假设已验证，无未决 |
| 方案设计 | PASS | 每个 slice 修复方向明确 |
| 依赖确认 | PASS | Docker daemon 为可选依赖 |
| 接口契约 | N/A | 无新增 API |
| 前端 | N/A | 无前端变更 |
| ADR | 不需要 | 修复不涉及架构决策变更 |

**结论**：`handoff-ready`，可进入 `/team-execute`。

---

## 应用等级 / 企业内控

- 开源项目，非企业内部应用
- 无 T1-T4 等级要求
- 无组件偏离需要记录

---

*已创建 `docs/artifacts/2026-05-13-codereview-critical-fix/delivery-plan.md`*
