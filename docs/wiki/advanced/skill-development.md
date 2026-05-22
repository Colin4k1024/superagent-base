# 技能开发指南（Skill Development）

## 概述

**SkillsHub 技能系统**是 Superagent Base 提供的轻量级可扩展工具层。技能（Skill）是可被 Agent 调用的函数单元，与内置工具（`builtin/`）的区别在于：

- 内置工具由系统静态注册（web_search、http_request、code_execute）
- 技能支持本地注册（SkillFunc）和远程 HTTP 调用，通过统一接口调用

Agent 通过 `skill://<name>` URI 引用技能：

```yaml
tools:
  - ref: skill://calculator
  - ref: skill://datetime
```

---

## 架构概览

```
Agent YAML (skill://calculator)
    │
    ▼ AgentBuilder.resolveToolRef()
    │  scheme = "skill", target = "calculator"
    │
    ▼ skill.Manager.GetTool("calculator")
    │  包装为 Eino InvokableTool
    │
    ▼ skill.Manager → skill.SkillInvoker
    │
    ├── LocalInvoker：直接调用本地 SkillFunc
    ├── HTTPInvoker：POST <endpoint>/invoke
    └── CompositeInvoker：本地优先，回退到 HTTP
```

---

## SkillMeta（技能元数据）

每个技能都有描述其能力的元数据：

```go
type SkillMeta struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Description string            `json:"description"`
    InputSchema  map[string]any   `json:"input_schema"`
    OutputSchema map[string]any   `json:"output_schema"`
    Tags        []string          `json:"tags,omitempty"`
    Endpoint    string            `json:"endpoint,omitempty"` // HTTP 技能使用
}
```

| 字段 | 说明 |
|------|------|
| `name` | 技能唯一标识，对应 `skill://<name>` 中的 name |
| `version` | 语义版本号 |
| `description` | 技能功能描述（LLM 根据此决定是否调用） |
| `input_schema` | 输入参数的 JSON Schema |
| `output_schema` | 输出格式的 JSON Schema |
| `endpoint` | HTTP 技能的基础 URL |

---

## 内置技能

内置技能通过 `pkg/skill/builtin/builtin.go` 实现，由 `builtin.RegisterAll(invoker)` 批量注册：

### datetime（获取当前时间）

```json
// 输入（可选）
{
  "format": "2006-01-02 15:04:05",  // Go 时间格式字符串
  "timezone": "Asia/Shanghai"       // IANA 时区名称
}

// 输出
{
  "datetime": "2025-05-11 10:30:00",
  "timestamp": 1746944200,
  "timezone": "Asia/Shanghai"
}
```

**使用示例：**

```yaml
tools:
  - ref: skill://datetime
    config:
      timezone: "Asia/Shanghai"
```

### calculator（四则运算）

```json
// 输入（必填）
{
  "expression": "2 + 3 * 4"
}

// 输出
{
  "result": 14
}
```

**支持的运算符：** `+` `-` `*` `/` `%`（取模）`^`（幂）

**使用示例：**

```yaml
tools:
  - ref: skill://calculator
```

### uuid（生成 UUID v4）

```json
// 输入（无需参数）
{}

// 输出
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

**使用示例：**

```yaml
tools:
  - ref: skill://uuid
```

---

## 创建本地技能

本地技能通过 `LocalInvoker.Register(name, func)` 注册，函数签名为：

```go
type SkillFunc func(ctx context.Context, input map[string]any) (map[string]any, error)
```

### 步骤 1：实现 SkillFunc

```go
package myskills

import (
    "context"
    "fmt"
    "strings"
)

// TextTransformSkill 将文本进行大写/小写/反转转换
func TextTransformSkill(_ context.Context, input map[string]any) (map[string]any, error) {
    text, ok := input["text"].(string)
    if !ok || text == "" {
        return nil, fmt.Errorf("text-transform: 'text' field is required")
    }

    operation, _ := input["operation"].(string)

    var result string
    switch operation {
    case "upper":
        result = strings.ToUpper(text)
    case "lower":
        result = strings.ToLower(text)
    case "reverse":
        runes := []rune(text)
        for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
            runes[i], runes[j] = runes[j], runes[i]
        }
        result = string(runes)
    default:
        result = text
    }

    return map[string]any{
        "result":    result,
        "operation": operation,
        "length":    len([]rune(result)),
    }, nil
}
```

### 步骤 2：注册技能

在服务初始化时注册（例如在 `application/init.go`）：

```go
import (
    "github.com/superagent-ai/superagent-base/backend/pkg/skill"
    "your-module/myskills"
)

// 初始化 LocalInvoker 并注册技能
localInvoker := skill.NewLocalInvoker()
localInvoker.Register("text-transform", myskills.TextTransformSkill)

// 也注册内置技能
builtin.RegisterAll(localInvoker)

// 创建 Manager
hubClient := skill.NewHTTPHubClient(skillsHubURL)
manager := skill.NewManager(hubClient, localInvoker)
```

### 步骤 3：在 Agent YAML 中引用

```yaml
tools:
  - ref: skill://text-transform
    config:
      default_operation: upper
```

---

## 创建 HTTP 技能服务

HTTP 技能是独立运行的微服务，通过标准 HTTP API 暴露技能能力。

### 协议规范

**端点：** `POST <base_url>/invoke`

**请求体：**

```json
{
  "skill": "my-skill-name",
  "input": {
    "param1": "value1",
    "param2": 42
  }
}
```

**响应体（成功）：**

```json
{
  "output": {
    "result": "...",
    "metadata": {}
  }
}
```

**响应体（失败）：**

```json
{
  "error": "错误描述信息"
}
```

HTTP 状态码：`200 OK` 表示成功，其他状态码表示失败。

### Go HTTP 技能示例

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
)

type InvokeRequest struct {
    Skill string         `json:"skill"`
    Input map[string]any `json:"input"`
}

func invokeHandler(w http.ResponseWriter, r *http.Request) {
    var req InvokeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
        return
    }

    // 根据技能名分发
    var output map[string]any
    var err error

    switch req.Skill {
    case "my-skill":
        output, err = mySkillLogic(req.Input)
    default:
        http.Error(w, fmt.Sprintf(`{"error":"unknown skill: %s"}`, req.Skill), http.StatusNotFound)
        return
    }

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"output": output})
}

func main() {
    http.HandleFunc("/invoke", invokeHandler)
    log.Println("Skill server listening on :9090")
    log.Fatal(http.ListenAndServe(":9090", nil))
}
```

### Python HTTP 技能示例（FastAPI）

```python
from fastapi import FastAPI
from pydantic import BaseModel
from typing import Any

app = FastAPI()

class InvokeRequest(BaseModel):
    skill: str
    input: dict[str, Any]

@app.post("/invoke")
async def invoke(req: InvokeRequest) -> dict:
    if req.skill == "sentiment-analysis":
        text = req.input.get("text", "")
        # 你的技能逻辑
        sentiment = analyze_sentiment(text)
        return {"output": {"sentiment": sentiment, "text": text}}
    
    return {"error": f"Unknown skill: {req.skill}"}

def analyze_sentiment(text: str) -> str:
    # 实现你的情感分析逻辑
    return "positive" if "good" in text.lower() else "neutral"
```

### 注册 HTTP 技能

```go
// 方式 1：直接使用 HTTPInvoker
httpInvoker := skill.NewHTTPInvoker()
httpInvoker.Register("sentiment-analysis", "http://localhost:9090")

// 方式 2：使用 CompositeInvoker（本地优先，回退 HTTP）
compositeInvoker := skill.NewCompositeInvoker(localInvoker, httpInvoker)

// 方式 3：通过 Manager 从 HubClient 安装
manager.Install(ctx, "sentiment-analysis", "latest")
```

---

## 技能调用流程（内部机制）

```go
// skill.Manager.GetTool("calculator") 返回包装后的 Eino InvokableTool
func (m *Manager) GetTool(name string) (tool.InvokableTool, bool) {
    inst, ok := m.cache.Get(name)
    if !ok {
        return nil, false
    }
    return NewSkillTool(inst.Meta, m.invoker), true
}

// SkillTool 实现 Eino InvokableTool 接口
// 调用时：m.invoker.Invoke(ctx, name, input)
// invoker 可以是 LocalInvoker / HTTPInvoker / CompositeInvoker
```

---

## CompositeInvoker 回退逻辑

```
CompositeInvoker.Invoke("calculator", input)
  │
  ▼ LocalInvoker.Invoke("calculator", input)
  │   └── 已注册？→ 直接执行本地函数，返回结果
  │   └── 未注册？→ 返回 error "local skill not registered"
  │
  └── （本地失败时）
      ▼ HTTPInvoker.Invoke("calculator", input)
          └── 已注册 endpoint？→ POST <endpoint>/invoke，返回结果
          └── 未注册？→ 返回 error "skill not registered"
```

---

## 技能管理 API（sactl）

CLI 工具 `sactl` 提供技能生命周期管理命令：

```bash
# 列出已安装技能
sactl skill list

# 安装技能（从 SkillsHub）
sactl skill install sentiment-analysis@1.0.0

# 卸载技能
sactl skill uninstall sentiment-analysis

# 查看技能详情
sactl skill info calculator
```

---

## 最佳实践

1. **幂等性**：技能函数应尽量无副作用或幂等，便于重试
2. **超时处理**：HTTP 技能默认超时 30 秒，通过 `HTTPInvoker` 的 `httpClient.Timeout` 配置
3. **错误信息**：返回清晰的错误信息（不包含敏感数据），LLM 可根据错误信息决定是否重试
4. **输入校验**：在技能函数入口校验必填字段，提前返回错误
5. **输出格式**：输出应为扁平的 `map[string]any`，避免深层嵌套，便于 LLM 理解
6. **注册顺序**：先注册内置技能（`builtin.RegisterAll`），再注册自定义技能

---

## 与内置工具的对比

| 特性 | 内置工具（`builtin/`） | 技能（`skill://`） |
|------|----------------------|------------------|
| 引用方式 | `builtin/web_search` | `skill://datetime` |
| 注册方式 | 系统静态注册 | LocalInvoker / HTTPInvoker 动态注册 |
| 扩展性 | 需修改代码重新编译 | 无需重新编译，HTTP 技能可热部署 |
| 适用场景 | 核心系统工具 | 业务特定、领域专用工具 |
| 远程调用 | 不支持 | 支持（HTTPInvoker） |
| 技能市场 | 不适用 | 支持 SkillsHub 安装 |
