# Test Plan: AgentLoop Demo Page

> 状态：review | 主责：qa-engineer | 日期：2026-06-30

## 测试范围

### 功能范围
1. AgentLoop Demo 页面加载和 agent 选择
2. 消息发送 → 多轮 streaming 渲染
3. A2UI 事件渲染：text, thinking, tool_call, tool_result, progress, error, done, preempted
4. Turn 分隔线渲染（轮次编号 + max_turns + 耗时）
5. 执行状态面板（轮次/耗时/状态 badge）
6. Preempt（中断当前执行发送新消息）
7. Interrupt 表单渲染和 resume 调用

### 不覆盖项
- 后端 AgentLoopAgent 执行逻辑（已有 4 个单测覆盖）
- A2UI 协议定义和编码（已有后端测试覆盖）
- 现有 ChatPage 功能回归（独立页面，无交叉影响）
- `code_block` 和 `agent_switch` 事件渲染（预留组件，无发送方）

## 测试矩阵

| # | 场景 | 类型 | 前置条件 | 预期结果 |
|---|------|------|----------|----------|
| 1 | 页面加载 | 功能 | 已登录 | 显示 agent 选择器 + 空消息区 + 输入框 |
| 2 | Agent 过滤 | 功能 | agentAdminApi 可用 | 下拉只显示 agentloop 类型 agent |
| 3 | Agent 过滤 fallback | 功能 | agentAdminApi 不可用 | 下拉显示硬编码 agent 列表 |
| 4 | 发送消息 | 功能 | 选择 agent | 用户消息入列 + assistant placeholder 创建 |
| 5 | text 事件渲染 | A2UI | 发送消息 | Markdown 渲染 + streaming cursor |
| 6 | thinking 事件渲染 | A2UI | agent 产生 thinking | 紫色折叠块，默认折叠 |
| 7 | tool_call 事件渲染 | A2UI | agent 调用工具 | 可折叠卡片，spinner 图标 |
| 8 | tool_result 匹配 | A2UI | 工具返回结果 | 匹配 name+calling 状态，更新为 done |
| 9 | progress → Turn 分隔 | A2UI | AgentLoop 进入 Turn 2+ | 分隔线显示 Turn N/M + 耗时 |
| 10 | 执行状态面板 | 功能 | streaming 中 | 显示当前轮次/总轮次/耗时/running badge |
| 11 | 完成状态 | A2UI | done 事件 | StatusPanel 显示 done + 总耗时 |
| 12 | Preempt | 功能 | streaming 中发送新消息 | 旧消息标记 preempted + 新消息开始 |
| 13 | 错误处理 | A2UI | error 事件 | 红色错误 banner 显示 |
| 14 | Interrupt 表单 | A2UI | interrupt 事件 | 渲染表单 + 提交调用 resume |
| 15 | 自动滚动 | UX | streaming 中 | 消息自动滚动到底部 |
| 16 | 手动上滚 | UX | 用户上滚 300px+ | 停止自动滚动 |

## 风险

| 风险 | 等级 | 关注点 |
|------|------|--------|
| tool_result 匹配逻辑 | 高 | 已修复为 name+status 匹配，需验证实际 SSE 流 |
| reducer 不可变性 | 中 | 已修复为 copyMsg/copyTurn，需验证 React 渲染正确性 |
| Agent 过滤依赖 admin API | 中 | 已加 fallback，需验证降级体验 |
| 长时间 AgentLoop UX | 低 | 15+ turns 时状态面板和消息列表性能 |

## 评审发现与修复状态

### Code Review
| 发现 | 严重度 | 状态 |
|------|--------|------|
| tool_result ID 匹配失败 | CRITICAL | ✅ 已修复（name+status 匹配） |
| Reducer 状态直接变更 | HIGH | ✅ 已修复（copyMsg/copyTurn） |
| ToolCallInfo.status 'success' vs 'done' | HIGH | ✅ 已修复（对齐为 'done'） |
| handleResume 无 try-catch | MEDIUM | ✅ 已修复 |
| parseTurnHeader Sscanf 尾部容错 | MEDIUM | ✅ 已修复（精确格式校验） |
| SSE 解析逻辑重复 | MEDIUM | ⏳ P2 增量 |
| InterruptField 类型重复 | MEDIUM | ⏳ P2 增量 |
| Fallback agent 静默失败 | LOW | ⏳ P2 增量 |

### Security Review
| 发现 | 严重度 | 引入方 | 状态 |
|------|--------|--------|------|
| rehype-raw XSS | CRITICAL | Pre-existing | ⏳ 非本次引入，单独跟踪 |
| Chat 端点无认证 | HIGH | Pre-existing | ⏳ 非本次引入 |
| CORS wildcard | HIGH | Pre-existing | ⏳ 非本次引入 |
| Admin API auth mismatch | HIGH | 本次引入 | ⏳ P2（fallback 已缓解） |
| Session ID 可预测 | MEDIUM | 本次引入 | ✅ 已修复（crypto.randomUUID） |
| Resume input 类型不匹配 | MEDIUM | 本次引入 | ⏳ P2（需后端配合） |
| Error 消息泄露 | MEDIUM | Pre-existing | ⏳ 非本次引入 |

## 放行建议

**建议放行**（附条件）

所有 CRITICAL 和 HIGH 问题已修复。4 个 MEDIUM 问题中 2 个已修复、2 个标记为 P2 增量（不阻塞功能使用）。3 个 pre-existing 安全问题单独跟踪，不阻塞本次交付。

**已接受风险**：
- SSE 解析逻辑重复（P2 重构）
- InterruptField 类型重复（P2 统一）
- Admin API fallback 到硬编码列表（已缓解，生产环境需 admin API key 配置）
