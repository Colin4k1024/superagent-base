# Java SDK (superagent-base-java)

基于 **AgentScope Java 2.0** 的 Java Agent 基座，API 与 Go/Python 基座完全对等。

## 快速开始

### 前置条件

- JDK 17+
- Maven 3.9+

### 启动

```bash
cd java
mvn spring-boot:run
```

### Docker

```bash
docker build -t superagent-java -f java/Dockerfile java/
docker run -p 8890:8890 superagent-java
```

## 核心能力

### Agent 类型

| 类型 | 说明 | 类名 |
|------|------|------|
| `chat_model_agent` | ReAct Agent | `ChatModelAgent` |
| `supervisor` | 多 Agent 协调者 | `SupervisorAgent` |
| `sequential` | 顺序流水线 | `SequentialAgent` |
| `parallel` | 并发执行 | `ParallelAgent` |
| `workflow` | DAG 工作流 | `WorkflowAgent` |
| `agentloop` | 自主循环 | `AgentLoopAgent` |

### Harness 架构

Java 基座完整实现了 AgentScope Harness 架构：

```java
// 使用 HarnessAgentBuilder 构建生产级 Agent
var agent = new HarnessAgentBuilder()
    .name("coder")
    .workspace(Paths.get(".agentscope/workspace"))
    .stateStore(new FileAgentStateStore())
    .memory(new HarnessMemory())
    .compaction(new CompactionManager(80000))
    .skillRepository(new SkillRepository())
    .enablePlanMode()
    .build();
```

#### 核心组件

| 组件 | 功能 |
|------|------|
| `Workspace` | AGENTS.md + MEMORY.md + tools.json + skills/ + subagents/ |
| `AgentStateStore` | 状态持久化（File / Redis） |
| `SessionManager` | .log.jsonl 会话日志 |
| `HarnessMemory` | 双层记忆：短期对话 + 长期 MEMORY.md |
| `CompactionManager` | 上下文压缩 + 溢出检测 |
| `ToolResultEviction` | 大结果卸载（>80K） |
| `SubagentManager` | 子 agent 声明 + 同步/后台委派 |
| `SandboxManager` | Docker 沙箱隔离执行 |
| `SkillRepository` | 四层技能加载 |
| `PlanModeManager` | 只读思考 + HITL 退出 |
| `Channel` | 会话路由 + 并发控制 + SSE 流式 |

### MCP 集成

```java
@Autowired
MCPRegistry mcpRegistry;

// 连接 MCP 服务器
mcpRegistry.connect(MCPRegistry.ServerConfig.builder()
    .name("filesystem")
    .endpoint("http://localhost:3000")
    .build());

// 获取客户端
MCPClient client = mcpRegistry.getClient("filesystem");

// 列出工具
List<MCPClient.ToolDefinition> tools = client.listTools();

// 调用工具
MCPClient.ToolCallResult result = client.callTool("read_file", 
    Map.of("path", "/tmp/test.txt"));
```

### Skills 系统

```java
@Autowired
SkillManager skillManager;

// 注册本地技能
skillManager.registerLocal("datetime", input -> 
    Map.of("date", LocalDate.now().toString()));

// 从 Hub 安装
skillManager.install("web-search", "1.0.0");

// 调用技能
Map<String, Object> result = skillManager.invoke("datetime", Map.of());
```

### Tool 中间件链

```java
import io.superagent.tools.ToolMiddleware;

// 创建中间件
ToolMiddleware.Middleware retry = ToolMiddleware.retry(3, Duration.ofSeconds(1));
ToolMiddleware.Middleware timeout = ToolMiddleware.timeout(Duration.ofSeconds(30));
ToolMiddleware.Middleware cache = ToolMiddleware.cache(Duration.ofMinutes(5));
ToolMiddleware.Middleware log = ToolMiddleware.log(logger);

// 组合中间件
ToolMiddleware.Middleware pipeline = ToolMiddleware.chain(retry, timeout, cache, log);

// 应用到工具调用
ToolMiddleware.ToolInvoker wrapped = pipeline.apply(invoker);
```

### 流式事件

```java
@Autowired
ChatController chatController;

// SSE 流式对话
Flux<Map<String, Object>> events = chatController.chatStream(
    Map.of("agent_id", "my-agent", "message", "Hello"));

events.subscribe(event -> {
    String type = (String) event.get("type");
    switch (type) {
        case "text" -> System.out.print(event.get("delta"));
        case "tool_call" -> System.out.println("[Calling " + event.get("name") + "]");
        case "done" -> System.out.println("[Done]");
    }
});
```

### 上下文注入

```java
import io.superagent.context.ContextInjectionMiddleware;

ContextInjectionMiddleware middleware = new ContextInjectionMiddleware(
    true,   // injectTimestamp
    true,   // injectSessionMetadata
    "You are a helpful assistant."  // staticContext
);

// 注入上下文
List<ChatMessage> injected = middleware.inject(messages);
```

### AgentLoop

```java
import io.superagent.agents.AgentLoopAgent;

AgentLoopAgent loopAgent = new AgentLoopAgent(
    "loop-1",
    "autonomous-agent",
    chatAgent,
    25  // maxTurns
);

// 自主循环执行
Map<String, Object> result = loopAgent.run(Map.of("message", "Research quantum computing"));
```

## API 端点

所有端点与 Go/Python 基座完全对等：

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v2/chat/stream` | SSE 流式对话 |
| `POST` | `/api/v2/chat/resume` | 恢复中断对话 |
| `GET` | `/api/v2/chat/interrupt_state` | 查询中断状态 |
| `POST` | `/api/v2/chat/abort` | 中止对话 |
| `GET` | `/api/v2/agents` | Agent 列表 |
| `GET` | `/api/v2/conversations` | 会话列表 |
| `POST` | `/api/v2/conversations` | 创建会话 |
| `GET` | `/api/v2/tools` | 工具列表 |
| `GET` | `/api/v2/skills` | 技能列表 |
| `GET` | `/api/v2/mcp/servers` | MCP 服务器列表 |
| `GET` | `/health` | 健康检查 |
| `GET` | `/ready` | 就绪检查 |
| `GET` | `/metrics` | Prometheus 指标 |

## 配置

### application.yml

```yaml
server:
  port: 8890

superagent:
  agents-dir: configs/agents
  agent:
    max-steps: 10
    timeout-seconds: 120

spring:
  redis:
    url: redis://localhost:6379
```

### Agent YAML

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
  system_prompt: "You are a helpful assistant."
  tools:
    - ref: builtin/web_search
    - ref: mcp://filesystem/read_file
```

## 依赖

```xml
<dependencies>
    <dependency>
        <groupId>io.agentscope</groupId>
        <artifactId>agentscope-harness</artifactId>
        <version>2.0.0-RC3</version>
    </dependency>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-webflux</artifactId>
    </dependency>
    <dependency>
        <groupId>io.micrometer</groupId>
        <artifactId>micrometer-registry-prometheus</artifactId>
    </dependency>
</dependencies>
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 框架 | Spring Boot 3.3 + WebFlux |
| Agent 框架 | AgentScope Java 2.0.0-RC3 |
| SSE 流式 | Reactor Flux |
| Redis | Spring Data Redis Reactive |
| 监控 | Micrometer + Prometheus |
| 构建 | Maven |
