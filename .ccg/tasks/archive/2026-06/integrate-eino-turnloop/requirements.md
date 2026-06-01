# 集成 Eino ADK TurnLoop

## 目标
- 现有 `/api/v1|v2/chat/stream` 保持请求和 SSE 输出兼容。
- 同一 `agent_id + session_id` 的新消息抢占旧回答。
- 停止按钮调用后端中止，使用 `Stop(WithImmediate())`。
- 保留现有 `/chat/resume` 与 `Interruptable` 行为。

## 实施默认值
- 新消息抢占：`WithPreemptTimeout(AfterToolCalls, 10s)`。
- 非 ADK agent 走旧 `Agent.Chat`。
- 当前 Eino 版本保持 `github.com/cloudwego/eino v0.9.0-beta.1`。

## CCG 说明
- M 复杂度按规则应进行双模型分析/审查。
- 本机 `~/.claude/bin/codeagent-wrapper` 不存在，无法执行指定 wrapper；实施以本地代码阅读、测试和人工审查替代。
