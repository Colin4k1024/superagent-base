# Superagent Base

开源 AI Agent 开发平台，基于 Coze Studio 构建。

## 核心能力

- **声明式 Agent 定义** — Kubernetes 风格 YAML 定义 Agent
- **多模型路由** — 按能力/成本/延迟智能路由
- **Tool 调用** — builtin / MCP / Skill 三种工具体系
- **多 Agent 编排** — supervisor / sequential / parallel / plan_execute
- **Workflow DAG** — 可视化工作流引擎
- **中断/恢复** — 人机协作，断点续传
- **A2UI 协议** — 结构化 SSE 流式输出
- **可观测性** — OpenTelemetry + Prometheus

## 快速开始

```bash
make dev          # 启动 MySQL + Redis + 后端
cd web && npm run dev  # 启动前端
```

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.24, Hertz, gRPC |
| 前端 | React 18, Vite 5, Tailwind |
| 存储 | MySQL, Redis, MinIO, ES, Milvus |
| 消息 | NSQ / Kafka |

## 文档导航

- [系统架构](architecture.md)
- [Agent YAML 规范](agent-yaml-spec.md)
- [部署指南](deployment.md)
- [Skill 开发](skill-development.md)
