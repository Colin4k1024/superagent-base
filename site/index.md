---
layout: home
hero:
  name: Superagent Base
  text: AI Agent 开发基座
  tagline: 基于 Eino 框架，YAML 声明式构建任意 AI Agent — 5 分钟上手
  actions:
    - theme: brand
      text: 5 分钟快速上手
      link: /guide/quickstart
    - theme: alt
      text: 完整文档
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/Colin4k1024/superagent-base

features:
  - icon: "\U0001F680"
    title: 14 个内置 Agent 开箱即用
    details: 覆盖 ReAct 工具调用、RAG 知识问答、自主编程、Supervisor 多 Agent、Plan-Execute、审批工作流、迭代写作等典型场景，克隆即用
    link: /guide/quickstart
    linkText: 开始上手
  - icon: "\U0001F4DD"
    title: YAML 声明式
    details: Kubernetes 风格 YAML 定义完整 Agent — 模型、工具、记忆、编排策略一步到位，fsnotify 热加载无需重启
    link: /guide/agent-yaml-spec
    linkText: 查看规范
  - icon: "\U0001F504"
    title: 多 Agent 编排
    details: Supervisor 委派、Sequential 流水线、Parallel 并行、Plan-Execute 规划执行四种编排模式
    link: /advanced/multi-agent
    linkText: 了解编排
  - icon: "\U0001F501"
    title: Agent Loop 自主循环
    details: 自主迭代执行模式 — Agent 自动分步推进任务，输出 [DONE] 终止循环，支持最大轮次保护和上下文累积
    link: /advanced/agentloop
    linkText: 了解循环
  - icon: "\U0001F3AF"
    title: Workflow DAG 引擎
    details: 拓扑排序 DAG 执行，5 种节点类型（llm_call / agent_call / tool_call / code / condition），React Flow 可视化编辑
    link: /advanced/workflow
    linkText: 查看工作流
  - icon: "\U000026A1"
    title: 模型路由
    details: 能力路由、成本路由、延迟路由 + 自动 Fallback，支持 OpenAI / Claude / Gemini / DeepSeek / Ark / Ollama / Qwen 七大供应商
    link: /guide/model-config
    linkText: 配置路由
  - icon: "\U0001F527"
    title: SkillHub + skills.sh 生态
    details: 内置 Skill + SkillHub API + skills.sh 开源市场三重来源，Agent 按需发现、安装和调用能力扩展
    link: /advanced/skill-development
    linkText: 探索技能
  - icon: "\U0001F50C"
    title: MCP 双向集成
    details: 作为 MCP Client（stdio/SSE 传输）调用外部工具，同时作为 MCP Server 暴露平台能力给第三方
    link: /advanced/mcp
    linkText: 查看集成
  - icon: "\U0001F9E0"
    title: 智能记忆
    details: 4 种 Memory 后端（builtin / Mem0 / Zep / Letta）一键切换，Agent 自动记住多轮对话上下文
    link: /guide/memory
    linkText: 配置记忆
  - icon: "\U0001F6E0"
    title: Tool 中间件链
    details: retry / timeout / rate-limit / cache 四层中间件 + 内置工具（web_search / http_request / code_execute）
    link: /guide/tools
    linkText: 使用工具
  - icon: "\U0001F3A8"
    title: A2UI 协议
    details: 10 种结构化事件（text / thinking / tool_call / tool_result / code_block / interrupt / error / done / progress / agent_switch）
    link: /advanced/a2ui-protocol
    linkText: 了解协议
  - icon: "\U0001F9EC"
    title: 经验自进化
    details: 本地 MySQL 存储执行经验，Eino Callback 自动收集信号 → 基因提炼 → 策略推荐注入 system prompt，零外部依赖
    link: /advanced/evolution
    linkText: 了解进化
  - icon: "\U000023F8"
    title: 中断与恢复
    details: 检测确认请求 → 保存 checkpoint（Redis）→ HTTP Resume API 恢复对话，支持超时自动过期
    link: /advanced/interrupt-resume
    linkText: 查看详情
  - icon: "\U0001F4CA"
    title: 全栈可观测性
    details: OpenTelemetry 分布式追踪 + Prometheus 指标 + Grafana 看板 + 实时 SSE 日志流，Eino Callback 自动上报
    link: /guide/architecture
    linkText: 查看架构
  - icon: "\U0001F4E1"
    title: 完整 Platform API（v2）
    details: 统一 /api/v2/ 命名空间，覆盖对话流式、会话 CRUD、消息历史、文件管理、长期记忆、Agent 状态 KV、Workflow 执行、Skills/Tools、Admin 全量管理
    link: /api/http-sse
    linkText: 查看 API
  - icon: "\U0001F510"
    title: RBAC 权限控制
    details: 三级角色（viewer / editor / admin）+ API Key 认证门禁，前端未授权自动跳转登录页
    link: /guide/getting-started
    linkText: 查看认证
  - icon: "\U0001F4AC"
    title: 企业级 Chat UI
    details: Markdown 渲染（代码高亮 + LaTeX）、深度思考可视化、工具调用卡片、智能滚动控制、停止生成，参考企业内部前端规范实现
    link: /guide/getting-started
    linkText: 查看 UI
  - icon: "\U0001F4BB"
    title: Web UI + CLI
    details: React + Vite + Tailwind 全功能前端（Agent 编辑器 / Workflow 画布 / 监控面板 / Skills 市场）+ sactl 命令行工具
    link: /api/cli
    linkText: 查看 CLI
---

<div class="vp-doc" style="padding: 0 24px;">

## 一分钟看懂核心流程

```
                    ┌─────────────────────────┐
                    │     YAML Agent 定义      │
                    │  (configs/agents/*.yaml) │
                    └────────────┬────────────┘
                                 │ fsnotify 热加载
                                 ▼
┌──────────┐    HTTP SSE    ┌─────────────┐    OpenAI API    ┌───────────┐
│  Client  │ ──────────────▶│  Agent 引擎  │ ──────────────▶ │  LLM 模型  │
│(curl/Web)│ ◀──────────────│  (port 8888) │ ◀────────────── │(多模型路由) │
└──────────┘   A2UI 事件流   └──────┬──────┘                  └───────────┘
                                   │
                    ┌──────────────┼──────────────┬───────────────┐
                    ▼              ▼              ▼               ▼
              ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐
              │  Memory   │  │  Tools   │  │  Skills  │  │  Evolution   │
              │(4 后端)   │  │(MCP/内置) │  │(3 来源)  │  │ (本地 MySQL) │
              └──────────┘  └──────────┘  └──────────┘  └──────────────┘
```

## 支持的 Agent 类型

| 类型 | 说明 |
|------|------|
| `chat_model_agent` | 标准对话 Agent，可挂载工具，Eino ReAct 自动调用 |
| `deep_agent` | 深度推理模式，支持多步规划 |
| `agentloop` | 自主循环，多轮迭代直到 `[DONE]` 或达到 max_turns |
| `supervisor` | 多 Agent 协调者，通过 LLM 决策分发给 sub_agents |
| `sequential` | 顺序执行 sub_agents，前一个输出作为下一个输入 |
| `parallel` | 并发执行所有 sub_agents，合并输出 |
| `plan_execute` | 先规划后执行的多 Agent 模式 |
| `workflow` | DAG 图执行，拓扑排序 + 变量映射 |
| `eino_graph` | 原生 Eino Graph，VS Code 插件可视化编排后注册 |

## 内置 Agent 案例一览（14 个）

开箱即用的典型 AI Agent 模板，参考 [cloudwego/eino-examples](https://github.com/cloudwego/eino-examples) 官方案例设计，覆盖所有主流 Agent 架构模式。

### 基础能力 — 单 Agent 场景

#### `research-agent` — 通用研究助手

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` |
| 工具 | 无（纯对话） |
| 场景 | 通用问答、信息整理、知识解释 |

```yaml
spec:
  type: chat_model_agent
  system_prompt: "You are a helpful research assistant..."
```

#### `react-tools-agent` — ReAct 工具调用

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` + 3 工具 |
| 工具 | `builtin/web_search` + `builtin/http_request` + `builtin/code_execute` |
| 场景 | 需要搜索、API 调用或代码执行的复杂任务 |
| 模式 | Think → Act → Observe → Respond 循环 |

```yaml
spec:
  type: chat_model_agent
  tools:
    - ref: builtin/web_search
    - ref: builtin/http_request
    - ref: builtin/code_execute
```

#### `rag-knowledge-agent` — RAG 知识问答

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` + search |
| 场景 | 基于文档检索回答问题，附带引用来源 |
| 特点 | 搜索 → 分析 → 综合 → 引用来源 |

```yaml
spec:
  type: chat_model_agent
  system_prompt: |
    Answer based on retrieved documents. Always cite sources.
    Never fabricate information not found in documents.
  tools:
    - ref: builtin/web_search
```

#### `code-assistant` — 自主编程循环

| 字段 | 值 |
|------|---|
| 类型 | `agentloop`（最多 15 轮） |
| 工具 | `builtin/code_execute` + `builtin/web_search` |
| 场景 | 自主编码：分析需求 → 编写代码 → 执行验证 → 修复问题 |
| 完成信号 | 输出 `[DONE]` 终止循环 |

```yaml
spec:
  type: agentloop
  max_turns: 15
  tools:
    - ref: builtin/code_execute
    - ref: builtin/web_search
```

#### `data-analyst` — 数据分析

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` + code_execute |
| 场景 | 统计分析、数据清洗、趋势识别 |
| 输出 | Summary → Key Findings → Details → Recommendations |

```yaml
spec:
  type: chat_model_agent
  model:
    temperature: 0.1
  tools:
    - ref: builtin/code_execute
    - ref: builtin/http_request
```

---

### 多 Agent 编排 — 协作场景

#### `team-supervisor` — 团队 Supervisor

| 字段 | 值 |
|------|---|
| 类型 | `supervisor`（最多 8 轮） |
| 子 Agent | `research-agent` + `code-assistant` + `react-tools-agent` |
| 场景 | 复杂任务自动分派给专业子 Agent 协作完成 |

```yaml
spec:
  type: supervisor
  sub_agents:
    - ref: research-agent
      role: Research and factual questions
    - ref: code-assistant
      role: Code writing and debugging
    - ref: react-tools-agent
      role: Web search and API calls
  orchestration:
    mode: supervisor
    max_rounds: 8
```

#### `plan-execute-agent` — 先规划后执行

| 字段 | 值 |
|------|---|
| 类型 | `plan_execute`（最多 10 轮） |
| 子 Agent | `react-tools-agent` + `research-agent` |
| 场景 | 制定 3-7 步计划 → 逐步执行 → 动态调整 |
| 特点 | 遇到新信息或阻塞时自动重新规划 |

```yaml
spec:
  type: plan_execute
  sub_agents:
    - ref: react-tools-agent
    - ref: research-agent
  orchestration:
    mode: plan_execute
    max_rounds: 10
```

#### `parallel-analysis` — 并行多视角分析

| 字段 | 值 |
|------|---|
| 类型 | `parallel` |
| 子 Agent | `research-agent`（事实分析）+ `react-tools-agent`（技术分析） |
| 场景 | 同时从多角度分析，综合得出结论 |

```yaml
spec:
  type: parallel
  sub_agents:
    - ref: research-agent
      role: "Factual analysis — data and evidence"
    - ref: react-tools-agent
      role: "Technical analysis — feasibility and trade-offs"
```

#### `sequential-pipeline` — 顺序流水线

| 字段 | 值 |
|------|---|
| 类型 | `workflow`（3 节点 DAG） |
| 流程 | 翻译 → 润色 → 校对 |
| 场景 | 每步输出作为下一步输入的固定流程 |

```yaml
spec:
  type: workflow
  workflow:
    nodes:
      - id: translate
        type: llm_call
        prompt: "Translate the text..."
      - id: polish
        type: llm_call
        prompt: "Polish the translation..."
      - id: proofread
        type: llm_call
        prompt: "Proofread for errors..."
    edges:
      - { from: START, to: translate }
      - { from: translate, to: polish }
      - { from: polish, to: proofread }
      - { from: proofread, to: END }
```

#### `research-workflow` — 研究报告流水线

| 字段 | 值 |
|------|---|
| 类型 | `workflow`（3 节点 DAG） |
| 流程 | 搜索 → 分析 → 格式化 Markdown 报告 |

---

### 人机协作 — 中断/审批场景

#### `approval-agent` — 安全确认

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` + interrupt |
| 场景 | 执行危险操作（删除、发送、支付）前暂停等待确认 |
| 恢复 | `POST /api/v1/chat/resume` 传入确认结果 |

#### `approval-workflow` — 审批工作流

| 字段 | 值 |
|------|---|
| 类型 | `chat_model_agent` + interrupt + tools |
| 工具 | `builtin/http_request` + `builtin/code_execute` |
| 场景 | 完整审批门禁：识别敏感操作 → 暂停 → 确认/取消 → 执行/放弃 |
| 超时 | 600 秒未确认自动过期 |

```yaml
spec:
  type: chat_model_agent
  tools:
    - ref: builtin/http_request
    - ref: builtin/code_execute
  interrupt:
    enabled: true
    checkpoint_backend: memory
    timeout_seconds: 600
```

#### `feedback-writer` — 迭代写作

| 字段 | 值 |
|------|---|
| 类型 | `agentloop`（最多 8 轮） |
| 场景 | 生成内容 → 等待用户反馈 → 根据反馈修改 → 循环直到满意 |
| 输出 | 每次修改说明变更原因和保留原因 |

```yaml
spec:
  type: agentloop
  model:
    temperature: 0.7
  max_turns: 8
```

---

> 所有 Agent 定义在 `backend/configs/agents/*.yaml`，支持 fsnotify **热加载**，修改后无需重启即生效。

## 快速体验

```bash
# 1. 克隆项目
git clone https://github.com/Colin4k1024/superagent-base.git && cd superagent-base

# 2. 配置模型（编辑 backend/.env 填入你的 LLM 地址和 Key）
cp .env.example backend/.env

# 3. 启动（MySQL + Redis + Backend）
make dev

# 4. 对话
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"hello"}'
```



## 技术栈

| 层 | 选型 |
|------|------|
| HTTP 框架 | Hertz (CloudWeGo) |
| LLM SDK | Eino (CloudWeGo) |
| 模型提供商 | OpenAI / Claude / Gemini / DeepSeek / Ark / Ollama / Qwen |
| ORM | GORM + MySQL 8.x |
| 缓存 | Redis 7 |
| 可观测性 | OpenTelemetry + Prometheus + Grafana |
| 前端 | React 18 + TypeScript + Vite 5 + Tailwind 3.4 |
| 部署 | Docker Compose / Kubernetes Helm |

</div>
