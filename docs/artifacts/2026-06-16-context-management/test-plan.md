# Test Plan: Agent 上下文管理 P0

| 字段 | 值 |
|------|------|
| 状态 | complete |
| 主责 | qa-engineer |
| 日期 | 2026-06-16 |

---

## 测试范围

### 功能范围

| 场景 | 验证方式 | 覆盖 |
|------|----------|------|
| Supervisor 子 Agent 会话隔离 | 单测：子 Agent 收到的 sessionID ≠ 主 sessionID | ✅ 已有 |
| Sequential 步骤会话隔离 | 单测：各步骤 sessionID 包含 `seq.step{i}` | ✅ 已有 |
| Parallel 分支会话隔离 | 单测 + race detector：分支 ID 唯一且无竞态 | ✅ 已有 |
| PlanExecute planner/executor 隔离 | 单测：planner 和 step 使用不同子会话 | ✅ 已有 |
| AgentLoop 无双重存储 | 单测：15 轮后 memory 仅 1 user + 1 assistant | ✅ 覆盖 |
| aggregateResults summarize 成功 | 单测：mockSummarize 返回压缩结果 | ✅ 新增 |
| aggregateResults summarize 降级 | 单测：summarizeFn 返回 error 时 fallback 到 concat | ✅ 新增 |
| aggregateResults summarize nil func | 单测：summarizeFn=nil 时 fallback 到 concat | ✅ 新增 |

### 非功能范围

| 场景 | 验证方式 | 覆盖 |
|------|----------|------|
| 并发安全（race condition） | `go test -race` 全量通过 | ✅ |
| 向后兼容 | 存量 105 个测试无回归 | ✅ |
| 性能回归 | 无额外 LLM 调用（除 summarize 配置场景） | ✅ by design |

### 不覆盖项

- 跨会话恢复（P2 scope）
- 子会话 TTL/清理（P1 scope，低优先级）
- 真实 Redis backend 集成测试（需 e2e 环境）
- Prompt injection 防护（security hardening，非 P0）

## 测试矩阵

| Agent 类型 | 隔离 | 双重存储 | Summarize | Race-safe |
|-----------|------|---------|-----------|-----------|
| supervisor | ✅ | N/A | ✅ | ✅ |
| sequential | ✅ | N/A | N/A | ✅ |
| parallel | ✅ | N/A | N/A | ✅ |
| plan_execute | ✅ | N/A | N/A | ✅ |
| agentloop | N/A | ✅ | N/A | ✅ |
| chat_model_agent | N/A | N/A | N/A | ✅ |

## 风险

| 风险 | 等级 | 说明 |
|------|------|------|
| 子会话 Redis key 堆积 | LOW | 24h TTL mitigates；P1 补清理逻辑 |
| Summarize prompt injection | MEDIUM | 子 Agent 输出可注入摘要 prompt；P2 加 fence markers |
| Sub-session collision on rerun | LOW | 仅影响 resume-from-interrupt 场景（当前 agentloop 无此特性） |

## 放行建议

**建议放行**。

- 4 个 HIGH 问题已在本轮 review 中全部修复并验证通过。
- 剩余 2 个 MEDIUM + 2 个 LOW 问题不阻塞 P0，记录为 P1/P2 待改进项。
- 108 单测 + race detector 全量通过。
- 向后兼容验证通过（存量测试无回归）。
