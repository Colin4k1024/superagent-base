# Execute Log: AgentLoop Demo Page

> 状态：execute | 主责：frontend-engineer | 日期：2026-06-30
> 关联 Plan：`docs/artifacts/2026-06-30-agentloop-demo/delivery-plan.md`

## 计划 vs 实际

| Slice | 计划 | 实际 | 偏差 |
|-------|------|------|------|
| S1: EventAgent turn header | ~15 行后端改动 | 30 行（含 parseTurnHeader 函数 + 测试） | 略多，增加了安全的格式校验 |
| S2: A2UI 解析扩展 + 类型系统 | api.ts 扩展 + 2 个新文件 | 完成，含 reducer 单元逻辑 | 无偏差 |
| S3: 页面骨架 + 路由 | 3 个文件修改 | 完成 | 无偏差 |
| S4: TurnSeparator + StatusPanel | 2 个新组件 | 完成 | 无偏差 |
| S5: MessageList + MessageItem | 2 个新组件 | 完成 | 无偏差 |
| S6: InterruptForm + Preempt | 1 个新组件 + api.ts resume | 完成 | 无偏差 |

## 关键决定

1. **Turn header pattern**：使用 `--- Turn N/M ---` 的精确格式匹配（TrimSpace 后），包含 `fmt.Sscanf` 数值校验，避免 LLM 输出误匹配
2. **ToolCallBlock 接口适配**：发现 ToolCallBlock 接受单个工具调用 props 而非数组，改为逐个渲染
3. **ChatInput 状态管理**：发现 ChatInput 需要 `value`/`onChange` 受控模式，增加 `input` 状态
4. **Agent 过滤**：优先使用 `agentAdminApi.list()` 返回的 `AgentDetail.type` 过滤，fallback 到硬编码列表

## 阻塞与解决

| 阻塞 | 解决 |
|------|------|
| ChatInput 接口不匹配（需受控 value/onChange） | 增加 input state，onSend 包装为 `() => handleSend(input)` |
| ToolCallBlock 不接受数组 | 改为 map 单个渲染 |
| agentAdminApi 可能需要认证 | 增加 fallback 到硬编码 agent 列表 |

## 影响面

### 后端（1 文件）
- `backend/pkg/agentdef/event_agent.go` — 增加 `parseTurnHeader` + progress event 发送

### 前端（新增 8 文件，修改 4 文件）

**新增**：
- `web/src/lib/agentloop-types.ts` — 类型定义
- `web/src/lib/agentloop-reducer.ts` — 状态 reducer
- `web/src/pages/AgentLoopChatPage.tsx` — 页面主组件
- `web/src/components/agentloop/TurnSeparator.tsx` — Turn 分隔线
- `web/src/components/agentloop/ExecutionStatusPanel.tsx` — 执行状态面板
- `web/src/components/agentloop/InterruptForm.tsx` — 中断表单
- `web/src/components/agentloop/AgentLoopMessageItem.tsx` — 消息渲染
- `web/src/components/agentloop/AgentLoopMessageList.tsx` — 消息列表
- `web/src/components/agentloop/index.ts` — barrel export

**修改**：
- `web/src/lib/api.ts` — ChatStreamCallbacks 增加 onProgress/onInterrupt，新增 chatApi.resume
- `web/src/router.tsx` — 增加 /agentloop-demo 路由
- `web/src/components/Sidebar.tsx` — 增加导航入口
- `web/src/styles/globals.css` — 增加 fadeInSlideDown 动画

## 自测结论

- [x] 后端编译通过 (`go build ./pkg/agentdef/...`)
- [x] 后端 4 个 agentloop 测试全部通过
- [x] 前端 HMR 无编译错误
- [x] `/agentloop-demo` 页面 HTTP 200
- [x] Sidebar 导航入口可见

## 未完成项

- Slice 7（E2E 集成验证）— 需 QA 执行
- Interrupt/resume 端到端联调 — 需要 agent 支持 interrupt 的配置
- `progress` 事件实际端到端验证 — 需要 agentloop agent 实际执行多轮
