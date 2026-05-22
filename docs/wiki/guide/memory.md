# 记忆系统

## 概述

Superagent Base 提供三层记忆和四种后端：

| 层级 | 范围 | 用途 |
|------|------|------|
| 短期记忆 | Session | 当前对话上下文 |
| 长期记忆 | User | 跨 Session 的知识积累 |
| Agent 状态 | Agent | Agent 内部持久状态 |

## 配置

```yaml
spec:
  memory:
    backend: mem0      # builtin | mem0 | zep | letta
    config:
      session_window: 20
      endpoint: http://mem0-server:8080
      api_key: your-key
```

## 后端对比

| 后端 | 短期 | 长期 | 状态 | 特点 |
|------|------|------|------|------|
| **builtin** | Redis List | Redis Hash | Redis Hash | 零依赖，开箱即用 |
| **mem0** | Mem0 API | Mem0 搜索 | Mem0 元数据 | 自动事实抽取，三级范围 |
| **zep** | Session 消息 | 事实搜索 | Session 元数据 | 企业级长期记忆 |
| **letta** | Agent 管理 | Archival Memory | Core Memory | 自主记忆管理 (MemGPT) |

## Builtin（默认）

无需额外配置，使用系统自带的 Redis：

```yaml
spec:
  memory:
    backend: builtin
```

## Mem0

```yaml
spec:
  memory:
    backend: mem0
    config:
      endpoint: http://localhost:8080
      api_key: your-mem0-key
```

## Zep

```yaml
spec:
  memory:
    backend: zep
    config:
      endpoint: http://localhost:8000
      api_key: your-zep-key
```

## Letta

```yaml
spec:
  memory:
    backend: letta
    config:
      endpoint: http://localhost:8283
      api_key: your-letta-key
```
