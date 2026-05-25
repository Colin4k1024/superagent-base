# 双端分支治理模型

## 分支模型

```
GitHub (origin)          GitLab (haier-origin)
    │                         │
    main ──────────────── internal (推送为 main)
    │                         │
    ├─ 通用框架代码             ├─ 通用框架代码 (继承自 main)
    ├─ web_search/http/code    ├─ + haier_knowledge_qa
    ├─ .github/workflows/      ├─ + hiagent_rag + pkg/hiagent/
    ├─ VitePress (GitHub链接)  ├─ + .gitlab-ci.yml (Pages)
    ├─ scripts/sync-branches   ├─ + VitePress (GitLab链接+内部文档)
    └─ 开源通用文档             ├─ + harbor 镜像 Dockerfile
                               └─ + 内部专属文档 (haier-knowledge-qa.md)
```

## 内容归属规则

| 分类 | 内容 | 归属 |
|------|------|------|
| 通用代码 | agentdef, modelrouter, mcp, memory, skill, tool 框架 | 双端同步 (main) |
| 通用工具 | web_search, http_request, code_execute | 双端同步 (main) |
| 内部工具 | haier_knowledge_qa, hiagent_rag, pkg/hiagent/ | 仅 internal → GitLab |
| GitHub CI | .github/workflows/ | 仅 main → GitHub |
| GitLab CI | .gitlab-ci.yml | 仅 internal → GitLab |
| 文档站配置 | VitePress config（链接指向不同） | 各端独立维护 |
| 内部文档 | haier-knowledge-qa.md 等 | 仅 internal → GitLab |

## 日常操作

| 场景 | 命令 |
|------|------|
| 在 main 上做了通用改动 | `./scripts/sync-branches.sh merge` 然后 `./scripts/sync-branches.sh push` |
| 只改内部代码 | 切到 `internal` 直接改，push 用 `git push haier-origin internal:main` |
| 查看双端差异 | `./scripts/sync-branches.sh status` |
| 一键推送双端 | `./scripts/sync-branches.sh push` |

## 同步脚本

路径：`scripts/sync-branches.sh`

```bash
./scripts/sync-branches.sh merge    # 将 main 的新 commit 合并到 internal
./scripts/sync-branches.sh push     # 推送 main→GitHub, internal→GitLab
./scripts/sync-branches.sh status   # 查看两分支差异
```

## Remote 配置

| Remote | URL | 用途 |
|--------|-----|------|
| origin | git@github.com:Colin4k1024/superagent-base.git | GitHub 公开仓库 |
| haier-origin | ssh://git@hgit.haier.net:2289/s04795/superagent-base.git | 海尔内部 GitLab |

## 注意事项

- main 分支绝不包含内部专属代码（haier/hiagent 工具、内部 CI、内部文档）
- internal 分支通过 merge main 保持与公开代码同步
- 在 internal 上开发内部功能时，确保不修改通用代码；如需修改通用部分，先在 main 上改好再 merge
- 解决冲突时，internal 的内部文件保留 internal 版本，通用文件保留 main 版本
- GitLab Pages 由 internal 分支的 .gitlab-ci.yml 驱动，推送时用 `internal:main`

## 环境变量（内部工具）

haier_knowledge_qa 工具需要以下环境变量才会注册：

```bash
HAIER_RAG_BASE_URL=https://rag-test.haier.net   # 或 https://rag.haier.net (生产)
HAIER_RAG_ACCESS_TOKEN=<access-token>
HAIER_RAG_APP_TOKEN=<app-access-token>
HAIER_RAG_K_CODE=<kcode>
```

hiagent_rag 工具需要：

```bash
HIAGENT_API_URL=<hiagent-api-url>
HIAGENT_API_KEY=<hiagent-api-key>
```
