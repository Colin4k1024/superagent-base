# 多 Agent 编排

## 编排模式

| 模式 | 描述 | 适用场景 |
|------|------|---------|
| `supervisor` | 主 Agent 协调子 Agent | 复杂任务分配 |
| `sequential` | 子 Agent 顺序执行 | 流水线处理 |
| `parallel` | 子 Agent 并发执行 | 独立分析合并 |

## Supervisor 模式

主 Agent 根据任务智能分配给子 Agent：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: project-manager
spec:
  type: supervisor
  model:
    primary: Qwen3-Coder-Next-4bit
  system_prompt: |
    You are a project manager. Coordinate sub-agents to complete tasks.
  sub_agents:
    - ref: research-agent
      role: Research and information gathering
    - ref: code-review-agent
      role: Code analysis and review
  orchestration:
    mode: supervisor
    max_rounds: 5
```

## Sequential 模式

子 Agent 按顺序执行，前一个的输出作为下一个的输入：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: content-pipeline
spec:
  type: sequential
  model:
    primary: Qwen3-Coder-Next-4bit
  sub_agents:
    - ref: research-agent
      role: 信息收集
    - ref: writer-agent
      role: 内容撰写
    - ref: editor-agent
      role: 审校润色
  orchestration:
    mode: sequential
```

## Parallel 模式

子 Agent 并发执行，结果合并返回：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: multi-analyst
spec:
  type: parallel
  model:
    primary: Qwen3-Coder-Next-4bit
  sub_agents:
    - ref: security-analyst
      role: 安全分析
    - ref: performance-analyst
      role: 性能分析
    - ref: architecture-analyst
      role: 架构评估
  orchestration:
    mode: parallel
```

## 注意事项

- 子 Agent 必须先定义为独立的 `chat_model_agent`
- 子 Agent 通过 `ref` 名称引用，运行时从 Registry 解析
- `max_rounds` 控制 Supervisor 的最大决策轮数
- Sequential 模式只流式输出最后一个 Agent 的结果
