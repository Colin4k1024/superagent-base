# Superagent Base — AI Agent 开发基座

> 基于 Eino 框架，YAML 声明式构建任意 AI Agent — 5 分钟上手

**[5 分钟快速上手](guide/quickstart)** | **[完整文档](guide/getting-started)** | **[GitHub](https://github.com/Colin4k1024/superagent-base)**

---

## 核心特性

| 特性 | 说明 |
|------|------|
| 🚀 14 个内置 Agent | ReAct、RAG、编程、Supervisor、Plan-Execute、审批工作流等典型场景，克隆即用 |
| 📝 YAML 声明式 | Kubernetes 风格 YAML 定义完整 Agent，fsnotify 热加载无需重启 |
| 🔄 多 Agent 编排 | Supervisor 委派、Sequential 流水线、Parallel 并行、Plan-Execute 四种模式 |
| 🔁 Agent Loop | 自主迭代执行，输出 `[DONE]` 终止，支持最大轮次保护 |
| 🎯 Workflow DAG | 拓扑排序 DAG，5 种节点类型，React Flow 可视化编辑 |
| ⚡ 模型路由 | 能力/成本/延迟路由 + 自动 Fallback，支持 OpenAI / Claude / Gemini / DeepSeek / Ark / Ollama / Qwen |
| 🔧 SkillHub 生态 | 内置 Skill + SkillHub API + skills.sh 开源市场三重来源 |
| 🔌 MCP 双向集成 | 作为 MCP Client 调用外部工具，同时作为 MCP Server 暴露能力 |
| 🧠 智能记忆 | 内置 / Mem0 / Zep / Letta 多种记忆后端，对话历史自动管理 |
| 🔒 Interrupt/Resume | 人工介入审批检查点，Redis 持久化，断点续跑 |
| 📊 全链路可观测 | OpenTelemetry Traces + Prometheus Metrics + Grafana 仪表盘 |
| 🖥️ A2UI 协议 | 结构化 SSE 流式输出，前端实时渲染工具调用、思考过程、进度 |

---

## 快速导航

- 📖 [快速上手](guide/quickstart)
- 🏗️ [架构总览](guide/architecture)
- 📋 [Agent YAML 规范](guide/agent-yaml-spec)
- ⚙️ [模型配置](guide/model-config)
- 🚢 [部署指南](guide/deployment)
- 🧰 [工具系统](guide/tools)
- 🧠 [记忆系统](guide/memory)

**进阶专题**
- [A2UI 流式协议](advanced/a2ui-protocol)
- [Agent Loop 自主循环](advanced/agentloop)
- [Interrupt / Resume](advanced/interrupt-resume)
- [Workflow DAG 引擎](advanced/workflow)
- [多 Agent 编排](advanced/multi-agent)
- [MCP 集成](advanced/mcp)
- [Skill 开发](advanced/skill-development)

**API 参考**
- [HTTP + SSE API](api/http-sse)
- [gRPC API](api/grpc)
- [CLI (sactl)](api/cli)
