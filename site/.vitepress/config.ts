import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Superagent Base',
  description: 'AI Agent 开发基座 — 基于 Eino 框架',
  lang: 'zh-CN',
  base: '/superagent-base/',
  ignoreDeadLinks: true,

  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: '指南', link: '/guide/getting-started' },
      { text: 'API', link: '/api/http-sse' },
      { text: '高级', link: '/advanced/a2ui-protocol' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '5 分钟快速上手', link: '/guide/quickstart' },
            { text: '完整安装指南', link: '/guide/getting-started' },
            { text: 'Windows 本地开发', link: '/guide/windows-dev' },
            { text: '架构概览', link: '/guide/architecture' },
            { text: '部署指南', link: '/guide/deployment' }
          ]
        },
        {
          text: 'Agent 开发',
          items: [
            { text: 'YAML 规范', link: '/guide/agent-yaml-spec' },
            { text: '模型配置', link: '/guide/model-config' },
            { text: '工具使用', link: '/guide/tools' },
            { text: '记忆系统', link: '/guide/memory' }
          ]
        },
        {
          text: '多语言 SDK',
          items: [
            { text: 'Go SDK', link: '/guide/go-sdk' },
            { text: 'Python SDK', link: '/guide/python-sdk' },
            { text: 'Java SDK', link: '/guide/java-sdk' }
          ]
        }
      ],
      '/api/': [
        {
          text: 'API 参考',
          items: [
            { text: 'HTTP SSE', link: '/api/http-sse' },
            { text: 'gRPC', link: '/api/grpc' },
            { text: 'CLI (sactl)', link: '/api/cli' }
          ]
        }
      ],
      '/advanced/': [
        {
          text: '高级特性',
          items: [
            { text: 'A2UI 协议', link: '/advanced/a2ui-protocol' },
            { text: '中断与恢复', link: '/advanced/interrupt-resume' },
            { text: '工作流编排', link: '/advanced/workflow' },
            { text: 'Tool 沙盒模式', link: '/advanced/sandbox' },
            { text: 'Skill 开发', link: '/advanced/skill-development' },
            { text: '多 Agent 编排', link: '/advanced/multi-agent' },
            { text: 'MCP 集成', link: '/advanced/mcp' },
            { text: '经验自进化', link: '/advanced/evolution' },
            { text: 'Agent Loop 自主循环', link: '/advanced/agentloop' },
            { text: 'TurnLoop 抢占与中止', link: '/advanced/turnloop' },
            { text: 'Langfuse 可观测性', link: '/advanced/langfuse' }
          ]
        }
      ]
    },

    socialLinks: [],

    footer: {
      message: 'Released under the Apache 2.0 License.',
      copyright: 'Copyright 2026 Superagent AI'
    },

    search: {
      provider: 'local'
    }
  }
})
