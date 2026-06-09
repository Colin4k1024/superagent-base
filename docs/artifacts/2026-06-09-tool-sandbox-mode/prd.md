# PRD: Tool Sandbox Mode

**Slug**: tool-sandbox-mode
**状态**: draft
**主责**: tech-lead
**日期**: 2026-06-09

---

## 背景

当前 Superagent Base 平台的 Tool 执行管线（`pkg/tool/`）支持 retry、timeout、rate-limit、cache、log 等中间件，但**缺乏统一的沙盒隔离层**。现有沙盒能力仅限于：

1. `code_execute` 工具有一个基于 `infra/coderunner/impl/sandbox/` 的 Python 进程级隔离（文件/网络/环境变量白名单），但默认关闭（C-4 安全策略）。
2. 其他 builtin 工具（`web_search`、`http_request`）和 MCP/Skill 类工具**无任何隔离**，直接在主进程中执行。

用户（Agent 开发者）需要一种可声明式开启的沙盒模式，让**所有类型的 tool 调用**都能运行在受限环境中，防止恶意或意外的工具执行影响宿主系统。

### 触发原因

- `code_execute` 的 C-4 安全策略注释明确指出"no sandbox isolation, disabled by default"。
- MCP 工具通过 stdio/SSE 协议调用外部进程，当前无资源限制、网络策略或超时隔离。
- 企业客户要求 tool 在多租户环境中执行时互不影响。

---

## 目标与成功标准

### 业务目标

- 让 Agent 开发者通过 YAML 配置一键开启 tool 沙盒模式，无需修改工具代码。
- 提供安全隔离层，防止 tool 执行时的文件泄漏、网络滥用、资源耗尽。

### 用户价值

- **安全性**: 工具代码在隔离环境运行，即使工具有漏洞也不影响宿主。
- **可控性**: 通过白名单精细控制工具可访问的文件、网络、环境变量。
- **一致性**: 统一的沙盒策略覆盖 builtin/mcp/skill 所有类型工具。

### 成功指标

1. 所有 tool 类型（builtin、mcp、skill）均可通过配置启用沙盒模式。
2. 沙盒模式默认关闭，开启后不影响现有工具的功能正确性。
3. 沙盒提供至少：文件系统隔离、网络访问控制、内存/CPU 限制、执行超时。
4. Agent YAML 中可声明 `sandbox` 配置块，粒度支持全局和 per-tool。

---

## 用户故事

### US-1: Agent 开发者全局开启沙盒

**作为** Agent 开发者，
**我想** 在 Agent YAML 中添加 `sandbox.enabled: true`，
**以便** 该 Agent 的所有 tool 调用自动运行在沙盒中。

**验收标准**:
- YAML 中 `spec.sandbox.enabled: true` 生效后，所有 tool invoke 均经过沙盒层。
- 未配置时行为与当前一致（无隔离）。

### US-2: 按 tool 粒度配置沙盒策略

**作为** Agent 开发者，
**我想** 对不同的 tool 配置不同的沙盒权限（如 web_search 允许网络、code_execute 限制文件写入），
**以便** 精细控制每个工具的安全边界。

**验收标准**:
- `spec.tools[].config.sandbox` 可覆盖全局沙盒策略。
- 不同 tool 可拥有不同的 allow_net、allow_read、allow_write 配置。

### US-3: 沙盒资源限制

**作为** 平台运维，
**我想** 限制单次 tool 调用的 CPU 时间、内存和网络流量，
**以便** 防止恶意工具耗尽系统资源。

**验收标准**:
- 沙盒配置支持 `timeout_seconds`、`memory_limit_mb`、`cpu_limit` 参数。
- 超限时 tool 调用被终止并返回明确的超限错误。

### US-4: 沙盒模式下的 MCP 工具隔离

**作为** Agent 开发者，
**我想** MCP 协议的外部工具也能受沙盒策略约束，
**以便** 外部进程不能无限制地访问宿主文件系统和网络。

**验收标准**:
- MCP stdio 类工具的子进程运行在受限 namespace/cgroup 中。
- MCP SSE 类工具的网络请求受 allow_net 白名单约束。

---

## 范围

### In Scope

1. **SandboxSpec 数据模型** — 在 `AgentSpec` 中新增 `Sandbox` 配置块。
2. **SandboxMiddleware** — 新增 tool middleware，在 tool invoke 前后设置/销毁沙盒。
3. **Builtin tool 适配** — `code_execute`、`web_search`、`http_request` 适配沙盒接口。
4. **MCP tool 适配** — stdio/SSE 协议的工具进程受沙盒约束。
5. **Docker/nsjail 沙盒后端** — 提供至少一种生产可用的隔离后端。
6. **Agent YAML schema 扩展** — 支持全局和 per-tool 的 sandbox 配置。
7. **环境变量控制** — `SANDBOX_ENABLED=true` 作为全局开关的快速通道。

### Out of Scope

- WebUI 上的沙盒配置面板（后续迭代）。
- Windows 平台沙盒支持（仅 Linux/macOS）。
- 计费和用量配额（属于平台运营层面，不在本期）。
- 已有的 `infra/coderunner/impl/sandbox` 的重构（可复用，不重写）。

---

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| Docker/nsjail 依赖增加部署复杂度 | 中 | 提供 fallback 到进程级隔离，Docker 为可选增强 |
| 沙盒开销导致 tool 调用延迟增加 | 中 | 基准测试，轻量隔离（namespace）优先 |
| MCP stdio 工具的子进程隔离在 macOS 上受限 | 低 | macOS 使用进程级 + 文件权限降级方案 |
| 现有 coderunner sandbox 和新方案的集成边界 | 低 | 新方案复用现有 sandbox.Config，扩展能力 |

### 关键依赖

- Linux namespaces / cgroups v2（生产环境）
- 现有 `infra/coderunner/impl/sandbox` 代码可复用
- `pkg/tool/middleware.go` 中间件链机制

### 待确认项

1. 生产环境 Docker 权限是否允许创建嵌套容器/namespace？
2. 沙盒后端优先选择 Docker、nsjail、还是纯 Linux namespace？
3. Per-tool 沙盒策略是否需要支持"继承+覆盖"语义？
4. MCP SSE 类工具的网络隔离是否需要走 iptables/proxy 方案？

---

## 技术方向初判

### 架构层次

```
Agent YAML (spec.sandbox)
       ↓
AgentBuilder — 解析 SandboxSpec, 注入 SandboxMiddleware
       ↓
Tool Manager — middleware chain 中新增 SandboxMiddleware
       ↓
SandboxMiddleware — 根据策略选择后端:
  ├── ProcessSandbox (namespace + seccomp, Linux)
  ├── DockerSandbox  (container per-invocation, 可选)
  └── LightSandbox   (仅 timeout + 文件权限, macOS fallback)
```

### YAML 设计草案

```yaml
spec:
  sandbox:
    enabled: true
    backend: process          # process | docker | light
    defaults:
      timeout_seconds: 30
      memory_limit_mb: 256
      allow_net: ["*.openai.com", "*.google.com"]
      allow_read: ["/tmp/agent-workspace"]
      allow_write: ["/tmp/agent-workspace"]
      allow_env: ["API_KEY", "MODEL_*"]
  tools:
    - ref: builtin/code_execute
      config:
        sandbox:
          allow_net: []           # 覆盖: 代码执行禁网
          memory_limit_mb: 128
    - ref: builtin/web_search
      config:
        sandbox:
          allow_net: ["*"]        # 覆盖: 搜索允许全网
```

---

## 参与角色清单

| 角色 | 职责 |
|------|------|
| tech-lead | intake 收口、方案仲裁、放行决策 |
| architect | 沙盒架构设计、隔离后端选型、接口契约 |
| backend-engineer | SandboxMiddleware 实现、tool 适配、测试 |
| qa-engineer | 安全验证、资源限制验证、回归测试 |

---

## 需求挑战会候选分组

| 分组 | 关注点 | 参与角色 |
|------|--------|----------|
| 隔离后端选型 | Docker vs nsjail vs namespace 的 trade-off | architect, backend-engineer |
| YAML schema 设计 | 全局/per-tool 继承语义、向后兼容 | tech-lead, architect |
| MCP 工具隔离策略 | stdio 进程隔离 vs SSE 网络隔离的不同路径 | architect, backend-engineer |

---

## 领域技能包启用建议

- `docker-patterns` — 沙盒后端涉及容器化
- `security-review` — 隔离方案需安全审计
- `golang-patterns` — 中间件实现

---

## UI 范围

本期无 UI 变更。沙盒配置通过 YAML 声明式管理。

---

## 企业治理待确认项

- 非企业内部应用，无需应用等级判定。
- 开源项目，无集团组件约束。
