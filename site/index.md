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
    title: 5 分钟上手
    details: 从零到运行只需 5 步 — 克隆、配置、启动、创建 Agent、多 Agent 协同
    link: /guide/quickstart
    linkText: 开始上手
  - icon: "\U0001F4DD"
    title: YAML 声明式
    details: 一个 YAML 文件定义完整 Agent — 模型、工具、记忆、编排策略一步到位
    link: /guide/agent-yaml-spec
    linkText: 查看规范
  - icon: "\U0001F504"
    title: 多 Agent 编排
    details: Supervisor 委派、Sequential 流水线、Parallel 并行、Workflow DAG 四种模式
    link: /advanced/multi-agent
    linkText: 了解编排
  - icon: "\U0001F527"
    title: SkillHub 生态
    details: 内置 Skill + SkillHub 远程市场，Agent 按需安装和调用能力扩展
    link: /advanced/skill-development
    linkText: 探索技能
  - icon: "\U0001F50C"
    title: MCP 双向集成
    details: 作为 MCP Client 调用外部工具，同时作为 MCP Server 暴露能力
    link: /advanced/mcp
    linkText: 查看集成
  - icon: "\U0001F9E0"
    title: 智能记忆
    details: 4 种 Memory 后端一键切换，Agent 自动记住多轮对话上下文
    link: /guide/memory
    linkText: 配置记忆
  - icon: "\U000026A1"
    title: 模型路由
    details: 能力路由、成本路由、延迟路由 + 自动 Fallback，一套系统管理多模型
    link: /guide/model-config
    linkText: 配置路由
  - icon: "\U0001F3A8"
    title: A2UI 协议
    details: 结构化事件流，前端可渲染工具调用、思考过程、代码块等 UI 组件
    link: /advanced/a2ui-protocol
    linkText: 了解协议
  - icon: "\U0001F9EC"
    title: 经验自进化
    details: 本地 MySQL 存储执行经验，自动收集信号 → 基因提炼 → 策略推荐，零外部依赖
    link: /advanced/evolution
    linkText: 了解进化
---

<div class="vp-doc" style="padding: 0 24px;">

## 一分钟看懂核心流程

```
                    ┌─────────────────────────┐
                    │     YAML Agent 定义      │
                    │  (configs/agents/*.yaml) │
                    └────────────┬────────────┘
                                 │ 热加载
                                 ▼
┌──────────┐    HTTP SSE    ┌─────────────┐    OpenAI API    ┌───────────┐
│  Client  │ ──────────────▶│  Agent 引擎  │ ──────────────▶ │  LLM 模型  │
│(curl/Web)│ ◀──────────────│  (port 8888) │ ◀────────────── │ (port 8000)│
└──────────┘   流式 Token   └──────┬──────┘                  └───────────┘
                                   │
                    ┌──────────────┼──────────────┬───────────────┐
                    ▼              ▼              ▼               ▼
              ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐
              │  Memory   │  │  Tools   │  │  Skills  │  │  Evolution   │
              │  (Redis)  │  │ (MCP/内置)│  │(SkillHub)│  │ (本地 MySQL) │
              └──────────┘  └──────────┘  └──────────┘  └──────────────┘
```

## 快速体验

```bash
# 1. 克隆项目
git clone https://github.com/Colin4k1024/superagent-base.git && cd superagent-base

# 2. 配置模型（编辑 backend/.env 填入你的 LLM 地址和 Key）
cp .env.example backend/.env

# 3. 启动
make dev

# 4. 对话
curl -N -X POST http://localhost:8888/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"s1","message":"hello"}'
```

</div>
