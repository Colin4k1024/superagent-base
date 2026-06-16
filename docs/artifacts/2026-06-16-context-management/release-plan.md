# Release Plan: Agent 上下文管理 P0

| 字段 | 值 |
|------|------|
| 状态 | released |
| 主责 | devops-engineer |
| 日期 | 2026-06-16 |

---

## 发布信息

| 维度 | 内容 |
|------|------|
| 发布标题 | fix(agentdef): session isolation + agentloop double-storage + real summarize |
| 变更类型 | Bug fix (P0) |
| 影响范围 | `backend/pkg/agentdef/` — 6 文件，+201/-34 行 |
| 风险等级 | 中（改动核心编排逻辑，但向后兼容，有完整测试覆盖） |
| 发布负责人 | devops-engineer |
| 值守人 | tech-lead |

## 变更与风险

### 变更清单

| 文件 | 变更说明 |
|------|----------|
| orchestration.go | Supervisor/Sequential/Parallel 子会话隔离 + delegation scope via context |
| orchestration_delegate.go | SubSessionID()、SummarizeFunc 类型、aggregateResults 重写、context-based scope |
| plan_execute.go | PlanExecute planner/executor 子会话隔离 |
| agentloop.go | 移除双重存储，外层管理 memory，AddMessage 移入 goroutine |
| builder.go | buildAgentLoop 不传 memory；buildSupervisor 注入 SummarizeFunc |
| orchestration_test.go | 适配新 API 签名 + 3 条 summarize 新测试 |

### 风险评估

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| 存量 Agent 因子会话隔离行为变化 | 低 | 中 | 108 测试 + race 全过；隔离不影响功能语义 |
| AgentLoop 恢复场景无记忆 | 低 | 低 | AgentLoop 无 interrupt 特性，不影响现有流程 |
| Summarize 新增 LLM 调用延迟 | 中 | 低 | 仅配置 `summarize` 的 supervisor 触发；默认 concat 无变化 |

## 执行步骤

### 发布前检查

- [x] `go build ./...` PASS
- [x] `go test ./pkg/agentdef/... -race` 108 PASS
- [x] `go vet ./pkg/agentdef/...` No issues
- [x] Code review: 4 HIGH → all fixed
- [x] Security review: 2 HIGH → all fixed
- [x] Launch acceptance: accepted

### 发布执行

1. 提交代码到 feature 分支
2. 创建 PR 到 main
3. CI 自动执行 `make test`
4. Code review 通过后 merge

### 发布后验证

- [ ] `make dev` 启动无报错
- [ ] 内置 14 个 Agent YAML 加载成功
- [ ] 基本对话流程正常（research-agent smoke）
- [ ] Redis key 数量监控基线建立

## 验证与监控

| 项目 | 方法 | 阈值 |
|------|------|------|
| 启动健康 | `make dev-server` 日志无 FATAL/ERROR | 0 errors |
| Agent 加载 | 启动日志 "registered N agents" | N = 14 |
| Redis keys | `redis-cli DBSIZE` | 发布后 24h 内无异常增长 |
| Token 消耗 | 对比 supervisor 15-round 场景 | 应下降 ≥30% |

## 回滚方案

| 触发条件 | 回滚路径 | 验证方法 |
|----------|----------|----------|
| 启动失败 | `git revert <commit>` | `make dev` 正常启动 |
| Agent 加载异常 | 同上 | 启动日志确认 14 agents |
| 运行时 panic | 同上 + 检查 stack trace | 无 panic 日志 |

## 放行结论

| 决定 | 说明 |
|------|------|
| **放行** | 所有门禁通过，无阻塞项 |
| 观察窗口 | 合入后 24h |
| 后续动作 | P1 (ContextSpec) 进入下一迭代 |
