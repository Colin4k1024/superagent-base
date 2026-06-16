# Deployment Context: Agent 上下文管理 P0

| 字段 | 值 |
|------|------|
| 状态 | released |
| 主责 | devops-engineer |
| 日期 | 2026-06-16 |

---

## 环境清单

| 环境 | 用途 | 部署目标 |
|------|------|----------|
| dev | 开发验证 | `make dev` (Docker: MySQL + Redis + backend) |
| debug | 全栈调试 | `make debug` (Docker: full stack) |
| production | 线上 | K8s / Docker Compose (取决于部署方式) |

## 部署入口

| 入口 | 命令 |
|------|------|
| 主入口 | `make build && bin/superagent` 或 `make dev-server` |
| 回退入口 | `git revert` + 重新 build |
| 前置条件 | Go 1.24+, MySQL, Redis |

## 配置与密钥

本次变更不新增配置项或密钥。所有改动为代码逻辑层，不涉及：
- 环境变量
- `.env` 文件变更
- 数据库 schema 变更
- Redis 数据结构变更（子会话使用已有 list 结构，命名空间变化）

## 运行保障

| 维度 | 说明 |
|------|------|
| Feature flag | 无（P0 为 bug fix，直接生效） |
| 灰度控制 | 不适用（框架内核改动，无法灰度） |
| 监控 | Redis DBSIZE + backend 启动日志 |
| 告警 | 无新增告警规则 |
| 值守安排 | tech-lead 观察 24h |
| 观察窗口 | 合入后 24h |

## 恢复能力

| 维度 | 说明 |
|------|------|
| 回滚触发条件 | Agent 加载失败 / 运行时 panic / Redis 异常增长 |
| 回滚路径 | `git revert <commit-sha>` → `make build` → 重启 |
| 验证方法 | `make dev` 启动 + 14 agents 加载成功 |
| 数据清理 | 子会话 Redis keys 24h TTL 自动过期，无需手动清理 |

## 注意事项

1. **子会话 Redis key 格式变化**：隔离后子 Agent 使用 `parentID::qualifier::agentName` 格式的 session key。这些 key 24h TTL 自动过期。如果旧版本生成的 key 仍存在于 Redis 中，不会影响新版本行为（新版本使用不同的 key namespace）。

2. **AgentLoop memory 行为变化**：之前 agentloop 内部每轮都写 memory（N 条 user + N 条 assistant），升级后只写 1 条 user + 1 条 final assistant。如果有外部工具读取 agentloop 的 session messages 期望看到中间轮次，行为已变——但这属于 bug fix 而非 breaking change，因为中间轮次从未被设计为可见。

3. **Summarize 新增 LLM 调用**：仅当 Agent YAML 配置了 `result_aggregation: summarize` 时才会触发额外 LLM 调用。当前 14 个内置 Agent 中无一使用此配置，因此不会产生额外成本。
