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
      { text: '高级', link: '/advanced/a2ui-protocol' },
      { text: 'GitLab', link: 'https://hgit.haier.net/s04795/superagent-base' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '5 分钟快速上手', link: '/guide/quickstart' },
            { text: '完整安装指南', link: '/guide/getting-started' },
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
            { text: 'Skill 开发', link: '/advanced/skill-development' },
            { text: '多 Agent 编排', link: '/advanced/multi-agent' },
            { text: 'MCP 集成', link: '/advanced/mcp' },
            { text: '经验自进化', link: '/advanced/evolution' },
            { text: 'Agent Loop 自主循环', link: '/advanced/agentloop' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'gitlab', link: 'https://hgit.haier.net/s04795/superagent-base' }
    ],

    footer: {
      message: 'Released under the Apache 2.0 License.',
      copyright: 'Copyright 2026 Superagent AI'
    },

    search: {
      provider: 'local'
    }
  }
})
