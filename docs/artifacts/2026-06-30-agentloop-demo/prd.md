# PRD: AgentLoop Demo Page

> 状态：plan-complete | 主责：tech-lead | 日期：2026-06-30

## 背景

Superagent Base 后端已完整实现 `agentloop` agent 类型（自主多轮循环执行）和 A2UI 结构化 SSE 流式协议（10 种事件类型），但前端缺乏一个专用页面来完整展示 AgentLoop 的执行过程。当前通用 `ChatPage` 虽然能消费 A2UI 事件，但不区分 turn 边界、不展示 agentloop 特有的多轮迭代语义，也没有作为示例引导开发者理解平台能力。

**触发原因**：需要一个完整示例页面，让开发者/用户直观看到 AgentLoop 的思考链、工具调用、多轮迭代、最终完成等全流程，同时作为 A2UI 协议渲染的标准参考实现。

## 目标与成功标准

| 目标 | 成功标准 |
|------|----------|
| 展示 AgentLoop 完整执行流程 | 用户可看到 turn 边界、thinking、tool_call/tool_result、text、done 全链路 |
| 符合 A2UI 协议规范渲染 | 前端正确解析并渲染所有 10 种 A2UI 事件类型 |
| 可交互的 AgentLoop 示例 | 用户可选择 agentloop 类型 agent、发送消息、观察多轮自主执行 |
| 支持中断/抢占 | 用户可在 AgentLoop 执行中发送新消息，触发 preempt |
| 作为开发者参考实现 | 代码结构清晰，可被其他前端项目参考复用 |

## 用户故事

### US-1：查看 AgentLoop 多轮执行过程
- **作为** 开发者
- **我想要** 选择一个 agentloop 类型的 agent，发送任务请求
- **以便** 观察 agent 自主多轮迭代执行的完整过程（每轮思考、工具调用、中间结果、轮次切换）

**验收标准**：
1. 页面显示 agent 选择器，可筛选 agentloop 类型
2. 发送消息后，实时渲染 streaming 文本、thinking 折叠块、tool_call/tool_result 卡片
3. Turn 边界（`--- Turn N/maxTurns ---`）以可视化分隔线渲染，不混入正文
4. 完成时显示 `[DONE]` 标记或等效的完成状态指示

### US-2：查看工具调用详情
- **作为** 开发者
- **我想要** 展开每个工具调用，查看参数和返回结果
- **以便** 理解 agent 在每一轮中如何使用工具

**验收标准**：
1. `tool_call` 事件渲染为可折叠卡片，显示工具名、参数 JSON
2. `tool_result` 事件关联到对应的 tool_call，显示结果内容
3. 工具状态图标正确：spinner（调用中）、checkmark（成功）、X（失败）

### US-3：中断正在执行的 AgentLoop
- **作为** 用户
- **我想要** 在 AgentLoop 多轮执行过程中发送新消息
- **以便** 中断当前执行并提供新的指令

**验收标准**：
1. 发送新消息时，正在执行的流被 preempt
2. 被 preempt 的消息显示"已中断"标记
3. 新消息正常开始新的流式响应

### US-4：查看思考过程
- **作为** 开发者
- **我想要** 折叠/展开查看 agent 的 thinking 内容
- **以便** 理解 agent 在每轮中的推理过程

**验收标准**：
1. `thinking` 事件渲染为紫色折叠块，默认折叠
2. 支持流式显示（streaming 时有脉冲动画）
3. 思考内容支持富文本渲染

## 范围

### In Scope
1. **AgentLoop Demo 页面**（`/agentloop-demo`）— 独立路由页面
2. **AgentLoop 流可视化组件** — turn 分隔线（含轮次编号、max_turns、本轮耗时）、多轮状态指示
3. **A2UI 事件增强渲染** — 补充当前缺失的 `progress`、`agent_switch`、`code_block`、`interrupt` 事件渲染
4. **agentloop 类型 agent 过滤** — 选择器只显示 agentloop 类型
5. **侧边栏导航入口** — "AgentLoop Demo" 导航项

### Out of Scope
- 后端 API 变更（已完整支持）
- AgentLoop agent 的创建/编辑（复用现有 AgentEditPage）
- 会话持久化/历史恢复（当前 ChatPage 也不支持）
- 多语言 i18n 新增 key 的完整翻译（先用中文 + 英文 fallback）

## 关键约束

- **后端零改动**：`POST /api/v1/chat/stream` + `X-A2UI: true` 已完全支持 agentloop agent
- **A2UI 事件完整覆盖**：当前前端只处理 `text`、`thinking`、`tool_call`、`tool_result`、`error`、`done`、`preempted` 7 种，需补充 `code_block`、`progress`、`agent_switch`、`interrupt` 4 种
- **复用现有组件**：`ThinkingBlock`、`ToolCallBlock`、`MarkdownRenderer`、`ChatInput` 已可用，需扩展而非重建
- **现有 agent**：`code-assistant`（agentloop, max_turns=15, 工具: code_execute + web_search）已配置可用

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| 需要模型 API key 才能实际测试 AgentLoop | 开发验证受限 | 提供 mock 模式或录制的 SSE 回放 |
| Turn 边界文本（`--- Turn N/maxTurns ---`）是纯文本混入流 | 前端需要文本模式匹配来识别 turn 边界 | 正则匹配 turn header，渲染为分隔线组件 |
| AgentLoop 长时间执行（15+ turns）时 UX | 用户可能觉得卡死 | 添加 turn 计数器、进度指示、耗时显示 |

## 待确认项

| # | 问题 | 决策 | 日期 |
|---|------|------|------|
| 1 | 是否需要 mock 模式？ | **否** — 依赖真实 agent（`code-assistant`） | 2026-06-30 |
| 2 | Turn 分隔线是否显示 token 消耗/耗时？ | **是** — 显示轮次编号、max_turns、本轮耗时 | 2026-06-30 |
| 3 | 路由方案 | **独立** `/agentloop-demo` | 2026-06-30 |

## 参与角色

| 角色 | 职责 | 输入缺口 |
|------|------|----------|
| tech-lead | intake 收口、方案仲裁 | 无 |
| frontend-engineer | 页面实现、A2UI 渲染增强 | 需确认 turn header 文本格式 |
| architect | 组件拆分、复用策略评审 | 无 |
| qa-engineer | A2UI 事件渲染验证、E2E 测试 | 需可用的 agentloop agent |

## 领域技能包启用建议

| 技能 | 触发原因 |
|------|----------|
| `frontend-engineering` | React 组件设计、状态管理 |
| `a2ui-streaming` | A2UI SSE 协议参考 |
| `agent-yaml-authoring` | 验证 agentloop YAML 配置 |

## UI 范围与质量门禁

- **目标端**：Web（桌面优先，响应式降级）
- **产品类型**：开发者工具 / 技术演示
- **关键页面**：AgentLoop Demo 页（chat 界面 + agent 选择器 + 执行状态面板）
- **设计约束**：复用现有 ChatPage 的布局模式，保持暗色主题一致性
- **可访问性基线**：键盘可操作、焦点可见、工具调用卡片有 aria 标签
- **性能基线**：流式渲染不卡顿（已有 streaming cursor 和 auto-scroll 机制）
- **一票否决项**：turn 边界不可辨识、工具调用状态不可见、无法中断执行

## 需求挑战会候选分组

| 分组 | 参与角色 | 议题 |
|------|----------|------|
| A2UI 事件覆盖 | frontend-engineer, architect | 现有 7 种事件 + 缺失 4 种的渲染策略 |
| Turn 可视化 | frontend-engineer, tech-lead | turn header 文本匹配 vs 后端结构化事件（第一期用文本匹配） |
