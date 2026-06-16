# Delivery Plan: Agent 上下文管理分类型方案

| 字段 | 值 |
|------|------|
| 状态 | draft |
| 主责 | tech-lead |
| 日期 | 2026-06-16 |
| 阶段 | plan (handoff-ready pending challenge session) |
| 关联 PRD | docs/artifacts/2026-06-16-context-management/prd.md |

---

## 一、需求挑战会结论

### 假设 1：存量 Agent 不依赖子 Agent 向主会话写入历史作为功能特性

| 字段 | 内容 |
|------|------|
| 质疑人 | architect |
| 质疑 | Supervisor 的 `mainAgent.Chat(ctx, sessionID, roundPrompt)` 把 enrichedPrompt 写入 memory，恢复会话时如果读到这些"上一轮协议文本"，是否被当作有用上下文？ |
| 结论 | **接受原假设**。enrichedPrompt 包含子 Agent 列表等固定协议文本，重复写入无信息增量。恢复场景下应从 system prompt 重建协议，而非从历史消息里"碰巧"读回。 |

### 假设 2：AgentLoop 底层 mainAgent 的 memory 可以安全关闭

| 字段 | 内容 |
|------|------|
| 质疑人 | project-manager |
| 质疑 | `code-assistant` 和 `feedback-writer` 这两个示例 Agent 是否有跨会话恢复需求？关闭底层 memory 后中断是否不可恢复？ |
| 结论 | **接受，加保护**。AgentLoop 目前没有中断/恢复设计（不像 InterruptableAgent）。如果未来需要恢复，应在 AgentLoop 层用 checkpoint 机制保存 `history` 快照，而不是依赖底层 memory 的副作用。P0 先关闭底层 memory，P2 设计 checkpoint 扩展点。 |

### 假设 3：token 近似估算 `len(content)/4` 足够 P1 使用

| 字段 | 内容 |
|------|------|
| 质疑人 | architect |
| 质疑 | 中文内容 token 密度远高于英文，`len()/4` 对中文场景会显著低估，导致窗口过大。 |
| 结论 | **修改假设**。改为 `len([]rune(content)) * 1.5` 作为保守估计（1 个中文字符约 1-2 token），同时在 `ContextSpec` 中预留 `tokenizer` 字段，P2 接入精确 tokenizer。 |

### 未决项

1. **子会话清理策略**：委派完成后的子会话数据是否自动 TTL 过期？还是只在主会话结束时批量清理？—— 留给 P1 详细设计。
2. **summarize 模型选择**：是否允许在 `OrchestrationSpec` 中配置 `summary_model`？还是统一用框架级默认小模型？—— 留给 arch-design 收口。

---

## 二、Brownfield 上下文快照

### 现有模块边界

```
pkg/agentdef/
├── schema.go           ← AgentSpec / MemorySpec / OrchestrationSpec 定义
├── builder.go          ← AgentBuilder.Build() 两阶段构建
├── chat_agents.go      ← SimpleChatAgent / ADKChatAgent（memory 读写 site A）
├── adk_stream.go       ← buildMessageHistory / persistUserMessage（memory 读写 site B）
├── agentloop.go        ← AgentLoopAgent.Chat（O(n²) 膨胀点）
├── orchestration.go    ← Supervisor / Sequential / Parallel（sessionID 复用点）
├── orchestration_delegate.go ← delegateTool.execute / aggregateResults
├── plan_execute.go     ← PlanExecuteAgent（sessionID 复用 + 无 step 上下文传递）
├── workflow_builder.go ← Workflow 节点（正确隔离，但缺 state 裁剪）
└── session_loop.go     ← SessionLoop 并发控制（不涉及本次改动）
```

### 关键数据流

```
User → HTTP Handler → agent.Chat(ctx, sessionID, message)
                         ↓
              ┌─ chat_agents.go: memBackend.GetMessages(Limit:20) → LLM
              │                  memBackend.AddMessage(user + assistant)
              │
              ├─ agentloop.go: 循环调用 mainAgent.Chat，history 累积
              │                mainAgent 内部再次 AddMessage（双重存储）
              │
              ├─ orchestration.go: supervisor 每轮 mainAgent.Chat(sessionID, enrichedPrompt+input)
              │                    delegate.execute → sub.Chat(sessionID, task)  ← 共享 session!
              │
              └─ plan_execute.go: mainAgent.Chat(sessionID, planPrompt)
                                  executor.Chat(sessionID, step)  ← 共享 session!
```

### 影响面评估

| 改动目标 | 直接影响文件 | 测试文件 | 风险 |
|----------|-------------|---------|------|
| 会话隔离 | orchestration.go, orchestration_delegate.go, plan_execute.go | orchestration_test.go (911行) | 中：需调整 mock sessionID 断言 |
| 双重存储 | agentloop.go, builder.go (buildAgentLoop) | agentloop_test.go (148行) | 低：逻辑简单 |
| summarize 实现 | orchestration_delegate.go | orchestration_test.go | 低：新增独立函数 |
| ContextSpec | schema.go, adk_stream.go, chat_agents.go | adk_stream_test.go (210行) | 中：需要新测试 |

---

## 三、Story Slice 列表

### Slice 1 (P0): 编排类型子会话隔离

| 字段 | 内容 |
|------|------|
| 目标 | Supervisor / Sequential / Parallel / PlanExecute 调用子 Agent 时使用独立子会话 ID |
| 验收标准 | 1) 单测验证子 Agent 收到的 sessionID ≠ 主 sessionID；2) 子 Agent 的 AddMessage 不出现在主会话的 GetMessages 结果中 |
| 改动范围 | orchestration.go L64,241,307; orchestration_delegate.go L129; plan_execute.go L56,120 |
| 子会话命名规则 | `{parentSessionID}::sub::{agentName}::r{round}` (supervisor); `{parentSessionID}::seq::{stepIndex}` (sequential); `{parentSessionID}::par::{branchIndex}` (parallel); `{parentSessionID}::plan::{stepIndex}` (plan_execute) |
| 依赖 | 无 |
| Owner | backend-engineer |
| Handoff 终点 | 代码合入 + 单测通过 + 无回归 |

### Slice 2 (P0): AgentLoop 停止双重存储

| 字段 | 内容 |
|------|------|
| 目标 | AgentLoop 的底层 mainAgent 不再自行读写 memory；上下文完全由外层循环控制 |
| 验收标准 | 1) 15 轮 agentloop 执行后，memory backend 中只有 1 条 user + 1 条 assistant（初始输入和最终输出）；2) 第 15 轮发给模型的内容长度 < 第 1 轮的 5 倍 |
| 改动范围 | builder.go (buildAgentLoop)：构建 mainAgent 时强制 memBackend=nil；agentloop.go：循环开始/结束时写入 outer memory |
| 依赖 | 无 |
| Owner | backend-engineer |
| Handoff 终点 | 代码合入 + 单测通过 |

### Slice 3 (P0): aggregateResults summarize 真正调用模型

| 字段 | 内容 |
|------|------|
| 目标 | `result_aggregation: summarize` 配置生效时，用小模型压缩多 Agent 结果为精简摘要 |
| 验收标准 | 1) 单测：输入 3 个各 500 字结果，summarize 输出 < 200 字；2) 接口：新增 `SummarizeFunc` 类型注入，测试可 mock |
| 改动范围 | orchestration_delegate.go L227; builder.go (buildSupervisor)：注入 summarize 能力 |
| 设计要点 | 注入 `func(ctx, []string) (string, error)` 而非硬编码模型调用，保持可测试性 |
| 依赖 | 无 |
| Owner | backend-engineer |
| Handoff 终点 | 代码合入 + 单测通过 |

### Slice 4 (P1): ContextSpec + 统一 BuildContext

| 字段 | 内容 |
|------|------|
| 目标 | 新增 `AgentSpec.Context` 声明式配置，替换硬编码 `Limit:20`，支持 stateless / sliding_window / token_budget 三种策略 |
| 验收标准 | 1) 不配置 `spec.context` 时行为 = 现状（sliding_window, max_messages=20）；2) 配置 `token_budget: 8000` 时按 token 裁剪；3) 配置 `stateless` 时不读取 memory |
| 改动范围 | schema.go (新增 ContextSpec)；adk_stream.go L182-201 (替换为 BuildContext 调用)；chat_agents.go L76-87 (替换为 BuildContext 调用) |
| 依赖 | Slice 1-3 合入后基础稳定 |
| Owner | backend-engineer + architect (接口设计) |
| Handoff 终点 | 代码合入 + 单测覆盖三种策略 + 存量 YAML 回归通过 |

### Slice 5 (P2): AgentLoop 滚动摘要 + Supervisor 协议移入 system prompt

| 字段 | 内容 |
|------|------|
| 目标 | AgentLoop 超过 K 轮后自动摘要历史轮次；Supervisor 的 enrichedPrompt 移入 system message 不再每轮重复 |
| 验收标准 | 1) 15 轮 agentloop，第 10 轮后早期轮次自动压缩为摘要行；2) Supervisor 的 memory 中不再出现重复的子 Agent 列表文本 |
| 依赖 | Slice 4 (复用 ContextSpec 的 summary_buffer 策略) |
| Owner | backend-engineer |
| Handoff 终点 | 代码合入 + 单测通过 + token 消耗对比基线降低 ≥50% |

### Slice 6 (P2): Workflow state 变量裁剪

| 字段 | 内容 |
|------|------|
| 目标 | `resolveTemplate` 注入下游节点前对 state 变量做 token 裁剪保护 |
| 验收标准 | 单测：state 中某变量超过 2000 token 时，注入到下游 prompt 的内容被截断/摘要为 < 500 token |
| 依赖 | Slice 4 (复用 token 估算函数) |
| Owner | backend-engineer |
| Handoff 终点 | 代码合入 + 单测通过 |

---

## 四、角色分工

| 角色 | 职责 | 产出 |
|------|------|------|
| tech-lead | Intake 收口、挑战会主持、方案仲裁、放行 | delivery-plan.md, 放行决策 |
| architect | ContextSpec 接口设计、子会话命名规范、token 估算策略 | arch-design.md, 接口契约 |
| backend-engineer | P0 ~ P2 全部 Slice 实现 | 代码 + 单测 |
| qa-engineer | 编排隔离回归验证、token 消耗基线对比 | test-plan.md, 验收报告 |

---

## 五、风险与依赖清单

| 风险 | 概率 | 影响 | 缓解措施 | Owner |
|------|------|------|----------|-------|
| orchestration_test.go 大量测试依赖 sessionID 精确匹配 | 高 | P0 合入后测试失败 | 先跑现有测试建立 baseline，再逐个调整 | backend-engineer |
| summarize 调模型引入延迟 | 中 | supervisor 响应变慢 100-500ms | 用最快模型 (haiku)、设 max_tokens=150、加超时 | backend-engineer |
| token 估算对中文不准 | 中 | 窗口偏大超模型限制 | 用 rune 计数 * 1.5 保守估计 | architect |
| AgentLoop 关闭底层 memory 后无恢复能力 | 低 | 未来需求时需补设计 | P2 预留 checkpoint 扩展点 | architect |

---

## 六、应用等级与治理

- 非企业内部应用，无需应用等级 / 技术架构等级判断。
- 无私有 overlay / 组件偏离。
- 不需要 ADR（纯内核优化，不引入新外部依赖或架构范式变更）。

---

## 七、技能装配清单

| 技能 | 触发原因 | 主责角色 |
|------|----------|----------|
| golang-patterns | Go 接口设计、注入模式 | architect + backend-engineer |
| tdd-workflow | P0 先补测试再改代码 | backend-engineer |
| api-design | ContextSpec YAML schema 设计 | architect |

---

## 八、前端交付物

无前端变更，不涉及 UI 门禁。

---

## 九、Implementation Readiness 结论

| 检查项 | 状态 |
|--------|------|
| PRD 完成 | ✅ |
| 需求挑战会完成 | ✅ 3 个假设已挑战并收敛 |
| Brownfield 快照 | ✅ 模块边界、数据流、影响面已梳理 |
| Story Slices 定义 | ✅ 6 个 Slice，P0→P1→P2 依赖明确 |
| 角色分工确认 | ✅ |
| 风险已识别 | ✅ 4 条，含缓解措施 |
| 阻塞项 | 无 |
| 未决项 | 2 条（子会话清理策略、summarize 模型选择），不阻塞 P0 |

**结论：P0 (Slice 1-3) implementation-ready，可进入 `/team-execute`。**

---

## 十、执行节奏建议

```
Week 1: Slice 1 + Slice 2 + Slice 3 (P0, 并行)
         → 合入后跑全量回归
Week 2: Slice 4 (P1, architect 先出接口设计)
         → 合入后在 2-3 个内置 Agent YAML 中试配 token_budget
Week 3: Slice 5 + Slice 6 (P2, 依赖 Slice 4)
         → token 消耗对比基线验收
```
