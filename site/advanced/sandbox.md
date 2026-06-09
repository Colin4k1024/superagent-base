# Tool 沙盒模式

## 概述

沙盒模式为 Agent 的所有工具调用提供安全隔离层。开启后，工具执行将在受限环境中运行，防止恶意代码影响宿主系统、泄漏文件或滥用网络资源。

**支持的隔离后端：**

| 后端 | 隔离级别 | 依赖 | 适用场景 |
|------|----------|------|----------|
| `docker` | 容器级（网络/文件系统/内存/CPU） | Docker daemon | 生产环境、多租户 |
| `process` | 进程级（环境变量/超时） | 无 | 开发环境、Docker 不可用时 |

> 当指定 `docker` 但 Docker 不可用时，系统自动降级为 `process` 模式。

---

## 快速启用

在 Agent YAML 中添加 `spec.sandbox` 块即可开启：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: safe-code-agent
spec:
  type: chat_model_agent
  model:
    primary: qwen-max
  sandbox:
    enabled: true
  tools:
    - ref: builtin/code_execute
    - ref: builtin/web_search
```

仅需 `sandbox.enabled: true`，其余使用默认值（Docker 后端、30s 超时、256MB 内存限制、网络禁用）。

---

## 完整配置

```yaml
spec:
  sandbox:
    enabled: true                          # 开启沙盒
    backend: docker                        # docker | process
    timeout_seconds: 30                    # 单次工具执行超时（秒）
    memory_limit_mb: 256                   # 内存上限（MB）
    allow_net:                             # 网络白名单（glob 模式）
      - "*.openai.com"
      - "*.googleapis.com"
    allow_read:                            # 可读文件路径白名单
      - "/tmp/agent-workspace"
      - "/data/knowledge-base"
    allow_write:                           # 可写文件路径白名单
      - "/tmp/agent-workspace"
    allow_env:                             # 环境变量白名单
      - "API_KEY"
      - "MODEL_ENDPOINT"
```

### 配置项说明

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `enabled` | bool | `false` | 总开关，false 时所有 tool 正常执行 |
| `backend` | string | `docker` | 隔离后端，支持 `docker` 和 `process` |
| `timeout_seconds` | int | `30` | 工具在沙盒内的最大执行时间 |
| `memory_limit_mb` | int | `256` | 容器/进程内存上限 |
| `allow_net` | []string | `[]`（禁网） | 允许访问的网络目标，支持 glob |
| `allow_read` | []string | `[]` | 允许读取的宿主路径 |
| `allow_write` | []string | `[]` | 允许写入的路径（Docker 用 tmpfs 挂载） |
| `allow_env` | []string | `[]` | 传入沙盒的环境变量名 |

---

## Per-Tool 策略覆盖

不同工具可配置不同的沙盒策略，在 `tools[].config.sandbox` 中覆盖全局设定：

```yaml
spec:
  sandbox:
    enabled: true
    backend: docker
    timeout_seconds: 30
    memory_limit_mb: 256
    allow_net: []                           # 全局禁网

  tools:
    - ref: builtin/web_search
      config:
        sandbox:
          allow_net: ["*"]                  # 搜索工具允许全网访问
          timeout_seconds: 15

    - ref: builtin/code_execute
      config:
        sandbox:
          allow_net: []                     # 代码执行严格禁网
          memory_limit_mb: 128             # 更严格的内存限制
          allow_write: ["/tmp/sandbox"]

    - ref: builtin/http_request
      config:
        sandbox:
          allow_net:                        # 仅允许特定域名
            - "api.example.com"
            - "*.internal.corp"
          timeout_seconds: 10
```

**优先级**：`per-tool config > 全局 sandbox spec > 系统默认值`

---

## 工作原理

### 执行流程

```
用户消息 → Agent → LLM → Tool Call
                              ↓
                    SandboxMiddleware 拦截
                              ↓
              ┌─────────────────────────────────┐
              │ code_execute 类工具？             │
              │   YES → 委托 Sandbox Backend     │
              │         (Docker/Process 内执行)  │
              │   NO  → 包裹原始 endpoint        │
              │         (timeout + panic recovery│
              │          + output 限制)          │
              └─────────────────────────────────┘
                              ↓
                         返回结果
```

### 两类隔离策略

| 工具类型 | 隔离方式 | 说明 |
|----------|----------|------|
| `code_execute` | **完全隔离** | 代码在独立容器/进程中执行，与宿主完全隔离 |
| `web_search`、`http_request` 等 | **约束包裹** | 原始工具正常执行，但施加 timeout、output 限制、panic 恢复 |

### Docker 后端细节

Docker 后端为每次 `code_execute` 调用创建一个临时容器：

- **镜像**：`python:3.11-slim`
- **网络**：默认 `--network=none`（除非 `allow_net` 非空）
- **文件系统**：`--read-only`，通过 `--tmpfs` 提供可写区域
- **资源**：`--memory` + `--memory-swap` 限制，`--pids-limit=64`
- **CPU**：50% CPU 配额（`--cpu-quota=50000`）
- **生命周期**：`--rm` 执行后自动销毁

### Process 后端细节

Process 后端在受限子进程中执行代码：

- **环境变量**：仅传入 `PATH` + `allow_env` 白名单中的变量
- **超时**：通过 `context.WithTimeout` 强制终止
- **无容器依赖**：适合开发和 CI 环境

---

## 使用场景

### 场景 1：安全代码执行 Agent

让 Agent 可以运行用户代码，但不暴露宿主环境：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: code-runner
spec:
  type: chat_model_agent
  model:
    primary: qwen-max
  system_prompt: |
    你是一个代码执行助手。用户可以让你执行 Python 代码来验证想法。
    安全地执行代码并返回结果。
  sandbox:
    enabled: true
    backend: docker
    timeout_seconds: 60
    memory_limit_mb: 512
    allow_net: []
    allow_write: ["/tmp/workspace"]
  tools:
    - ref: builtin/code_execute
```

### 场景 2：受限研究 Agent

研究 Agent 可以搜索和请求网页，但仅限特定域名：

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: restricted-researcher
spec:
  type: chat_model_agent
  model:
    primary: qwen-max
  sandbox:
    enabled: true
    timeout_seconds: 15
    allow_net:
      - "*.google.com"
      - "*.wikipedia.org"
      - "api.openai.com"
  tools:
    - ref: builtin/web_search
    - ref: builtin/http_request
```

### 场景 3：多租户平台

为不同租户的 Agent 配置独立的沙盒边界：

```yaml
spec:
  sandbox:
    enabled: true
    backend: docker
    timeout_seconds: 30
    memory_limit_mb: 128
    allow_read: ["/data/tenant-123/knowledge"]
    allow_write: ["/tmp/tenant-123"]
    allow_env: ["TENANT_API_KEY"]
  tools:
    - ref: builtin/code_execute
    - ref: builtin/web_search
      config:
        sandbox:
          allow_net: ["api.tenant-123.example.com"]
```

---

## 环境变量快捷开关

除了 YAML 配置，也可通过环境变量全局控制：

| 环境变量 | 说明 |
|----------|------|
| `SANDBOX_BACKEND` | 覆盖默认后端选择（`docker` / `process`） |
| `CODE_EXECUTE_ENABLED=true` | 启用 code_execute 工具（独立于沙盒） |

---

## 与现有功能的关系

| 功能 | 关系 |
|------|------|
| Tool Middleware (retry/timeout/cache) | 沙盒在中间件链**之前**生效，timeout 独立控制 |
| code_execute C-4 安全策略 | 沙盒模式开启后，code_execute 安全隔离有了实质保障 |
| MCP 工具 | MCP stdio 工具的子进程也受沙盒 timeout 约束 |
| Interrupt/Resume | 沙盒不影响 checkpoint，中断恢复正常工作 |

---

## 故障排查

### Docker 不可用

```
WARN: docker sandbox: docker not available, falling back to process backend
```

**解决**：安装 Docker 并确保当前用户有权限运行 `docker info`，或显式设置 `backend: process`。

### 执行超时

```
[sandbox] code_execute: execution timed out after 30s
```

**解决**：增大 `timeout_seconds`，或优化工具代码的执行效率。

### 内存超限

```
[sandbox error] code_execute: container error: OOMKilled
```

**解决**：增大 `memory_limit_mb`，或减少工具执行时的内存消耗。

### 网络访问被拒

如果工具需要网络但返回连接错误，检查 `allow_net` 是否包含目标域名。
默认为空（禁网），需显式添加允许的域名模式。

---

## 限制与注意事项

- **Docker 后端**仅在 Linux/macOS 上可用，Windows 需要 WSL2 + Docker Desktop。
- **Process 后端**的隔离强度低于 Docker，不适合处理不可信代码。
- 沙盒的文件路径白名单使用精确路径匹配，不支持通配符。
- 每次 `code_execute` 调用都会创建和销毁容器，高频调用时有约 200-500ms 冷启动开销。
- 沙盒默认关闭，需显式启用；未配置沙盒的 Agent 行为与之前完全一致。
