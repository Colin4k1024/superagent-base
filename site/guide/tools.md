# 工具使用

## 工具引用格式

在 Agent YAML 的 `spec.tools[]` 中引用工具：

```yaml
spec:
  tools:
    - ref: builtin/web_search          # 内置工具
    - ref: builtin/http_request        # 内置 HTTP 客户端
    - ref: builtin/code_execute        # 代码执行
    - ref: mcp://filesystem/read_file  # MCP Server 工具
    - ref: skill://datetime            # SkillsHub 技能
```

## 内置工具

| 工具 | 描述 | 输入 |
|------|------|------|
| `web_search` | 网页搜索 | query, max_results |
| `http_request` | HTTP 请求 | url, method, headers, body |
| `code_execute` | 代码执行 | language, code |

## 工具中间件

所有工具调用自动经过中间件链：

| 中间件 | 功能 |
|--------|------|
| Retry | 失败重试（可配次数和退避策略） |
| Timeout | 执行超时控制 |
| RateLimit | RPM 限流 |
| Cache | 结果缓存（幂等工具） |
| Log | 调用日志 |

## MCP 工具

连接外部 MCP Server 获取工具：

```yaml
# 在运行时配置 MCP Server
# 通过 mcp://server-name/tool-name 引用
spec:
  tools:
    - ref: mcp://filesystem/read_file
    - ref: mcp://github/search_repos
```

## Skill 工具

从 SkillsHub 安装和使用技能：

```bash
# 安装
sactl skill install datetime@1.0.0

# 在 Agent 中引用
spec:
  tools:
    - ref: skill://datetime
    - ref: skill://calculator
```

详见 [Skill 开发指南](/advanced/skill-development)。
