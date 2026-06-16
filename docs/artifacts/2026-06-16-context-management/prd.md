# PRD: Agent 上下文管理分类型方案

| 字段 | 值 |
|------|------|
| 状态 | draft |
| 主责 | tech-lead |
| 日期 | 2026-06-16 |
| 阶段 | intake |

---

## 背景

Superagent Base 框架支持 8 种 Agent 类型（chat_model_agent / deep_agent / agentloop / supervisor / sequential / parallel / plan_execute / workflow），但上下文管理机制高度单一：

1. **全局硬编码窗口**：所有 chat 类 Agent 共用 `GetMessages(Limit:20)`，无 token 预算、无摘要策略。
2. **编排类型 sessionID 复用**：Supervisor / Sequential / Parallel / PlanExecute 把主 `sessionID` 直接传给子 Agent，导致中间决策写入用户可见的对话历史，子 Agent 记忆互相污染。
3. **AgentLoop 双重累积**：外层 `history.String()` 拼全量 + 底层 chat agent 又把同样内容持久化再读回，上下文呈 O(n²) 增长。
4. **Supervisor 协议文本重复持久化**：每轮 `enrichedPrompt` 被当作 user 消息存入 memory，轮数越多冗余越大。
5. **`summarize` 名不副实**：代码直接 `fallthrough` 到 `concat`，不调用模型。

唯一正确实现隔离的是 Workflow 节点（`sessionID+"-"+node.ID`），但缺乏 state 变量长度裁剪保护。

## 目标与成功标准

### 业务目标

- 降低多轮/多 Agent 场景的 token 消耗，避免无效上下文膨胀。
- 消除编排类型 Agent 之间的"上下文串话"问题。
- 为不同 Agent 类型提供声明式、可调节的上下文策略。

### 成功指标

- P0 合并后：supervisor/sequential/parallel/plan_execute 子 Agent 不再向主会话写入中间产物（可通过单测验证 memory 隔离）。
- P0 合并后：agentloop 15 轮执行，总 token 消耗对比现状降低 ≥50%。
- P1 合并后：所有 chat 类 Agent 可在 YAML 中声明 `spec.context.strategy` 并生效。
- P1 合并后：存量 14 个内置 Agent YAML 不需要修改即可保持现有行为（向后兼容）。

## 用户故事

### US-1: 作为平台开发者，我想消除编排类型 Agent 的上下文串话

- 验收标准：Supervisor 的委派消息不会出现在用户会话的消息历史中；Sequential 各步骤使用独立子会话。

### US-2: 作为平台开发者，我想让 AgentLoop 不再 O(n²) 膨胀

- 验收标准：15 轮 agentloop 执行，第 15 轮发送给模型的 token 数 < 第 1 轮的 5 倍（而非 15 倍）。

### US-3: 作为 Agent 定义者，我想在 YAML 中声明上下文管理策略

- 验收标准：`spec.context.strategy: token_budget` + `max_tokens: 8000` 配置后，BuildContext 按 token 预算裁剪历史，而非固定 20 条。

### US-4: 作为 Agent 定义者，我想让 `result_aggregation: summarize` 真正压缩结果

- 验收标准：配置 `summarize` 模式后，aggregateResults 调用模型输出精简摘要，而非原文拼接。

## 范围

### In Scope

| 优先级 | 改动 | 影响文件 |
|--------|------|----------|
| P0 | 编排类型子 Agent 会话隔离 | orchestration.go, orchestration_delegate.go, plan_execute.go |
| P0 | AgentLoop 停止双重存储 | agentloop.go, builder.go |
| P0 | `aggregateResults` summarize 真正调用模型 | orchestration_delegate.go |
| P1 | 新增 `AgentSpec.Context` + 统一 `BuildContext` | schema.go, adk_stream.go, chat_agents.go, turnloop.go |
| P1 | RAG 检索内容与对话历史分开核算预算 | builder.go (RAG 构建逻辑) |
| P2 | AgentLoop 滚动摘要策略 | agentloop.go |
| P2 | Supervisor 协议文本移入 system prompt | orchestration.go |
| P2 | Workflow state 变量裁剪 | workflow_builder.go |

### Out of Scope

- 对外部 memory backend（mem0 / zep / letta）协议层的改动。
- 前端 UI 展示变更。
- Agent YAML schema 的 breaking change（必须向后兼容）。
- 模型 tokenizer 的精确实现（P1 阶段先用字符数估算，后续再接入 tiktoken）。

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| P0 会话隔离改动可能破坏依赖共享 session 的存量测试 | 测试失败 | 先审查 orchestration_test.go 24KB 的测试用例 |
| summarize 真正调模型会引入额外延迟和成本 | Supervisor 响应变慢 | 使用小模型(haiku/deepseek) + 设置 token 上限 |
| token 估算不精确（字符数 vs 真实 token） | 窗口可能偏大或偏小 | P1 先用 `len(content)/4` 近似，P2 接入 tiktoken |
| AgentLoop 改为 stateless 后可能失去跨会话恢复能力 | 中断后无法续跑 | 在 AgentStateMemory 中保存 checkpoint |

## 待确认项

1. P0 会话隔离后，是否需要提供"子会话可查询"的 API（用于调试/审计）？
2. summarize 模式使用哪个模型？是否由 YAML 配置还是框架全局默认？
3. token 预算的默认值是否需要根据模型自动推断（从 ModelSpec 获取模型上下文窗口大小）？
4. 是否需要为 P0 改动提供数据迁移脚本（清理历史脏数据）？
