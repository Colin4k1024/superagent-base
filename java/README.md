# Superagent Base Java

Java implementation of the Superagent agent platform, built on **AgentScope Java 2.0** with **Spring Boot 3 WebFlux** for reactive SSE streaming.

Part of the three-base architecture (Go / Python / Java) — API-compatible with the existing bases.

## Quick Start

```bash
# Prerequisites: JDK 17+, Maven 3.9+

# Run directly
cd java
mvn spring-boot:run

# Or build and run JAR
mvn clean package -DskipTests
java -jar target/superagent-base-java-0.1.0-SNAPSHOT.jar
```

Server starts on **http://localhost:8890**.

## Docker

```bash
# Build image
docker build -t superagent-base-java -f java/Dockerfile java/

# Run
docker run -p 8890:8890 \
    -v ./configs/agents:/app/configs \
    superagent-base-java
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v2/chat` | Synchronous chat |
| `POST` | `/api/v2/chat/stream` | Streaming chat (SSE) |
| `POST` | `/api/v2/chat/resume` | Resume interrupted conversation |
| `GET` | `/api/v2/chat/interrupt_state` | Query interrupt state |
| `GET` | `/api/v2/agents` | List loaded agents |
| `GET` | `/api/v2/admin/status` | System status |
| `POST` | `/api/v2/admin/reload` | Trigger agent hot-reload |
| `GET` | `/api/v2/admin/agents` | List agents (admin) |
| `GET` | `/api/v2/admin/agents/{name}` | Get agent detail |
| `POST` | `/api/v2/admin/agents` | Create/update agent |
| `DELETE` | `/api/v2/admin/agents/{name}` | Delete agent |
| `GET` | `/api/v2/admin/mcp/servers` | List MCP servers |
| `GET` | `/api/v2/admin/logs` | SSE log stream |
| `GET` | `/health` | Liveness check |
| `GET` | `/ready` | Readiness check |
| `GET` | `/actuator/prometheus` | Prometheus metrics |

## Project Structure

```
java/
├── src/main/java/io/superagent/
│   ├── SuperagentApplication.java       # Spring Boot main
│   ├── agents/
│   │   ├── BaseAgent.java               # Abstract base (run/describe)
│   │   ├── ChatModelAgent.java          # ReAct agent wrapper
│   │   ├── SupervisorAgent.java         # Routes to children
│   │   ├── SequentialAgent.java         # Pipeline execution
│   │   ├── ParallelAgent.java           # Fan-out with CompletableFuture
│   │   └── WorkflowAgent.java           # DAG with topological sort
│   ├── tools/
│   │   ├── Tool.java                    # Tool interface
│   │   ├── McpToolWrapper.java          # MCP server tool adapter
│   │   └── builtin/
│   │       ├── WebSearchTool.java
│   │       ├── HttpRequestTool.java
│   │       └── CodeExecuteTool.java
│   ├── memory/
│   │   ├── MemoryStore.java             # Interface
│   │   ├── BuiltinMemory.java           # ConcurrentHashMap impl
│   │   └── RedisMemory.java             # Reactive Redis impl
│   ├── models/
│   │   ├── ModelRegistry.java           # Model config registry
│   │   └── ModelRouter.java             # Routing with fallback
│   ├── config/
│   │   ├── AgentDefinition.java         # YAML schema record
│   │   ├── YamlAgentLoader.java         # YAML file scanner
│   │   └── AgentBuilderFactory.java     # Two-pass agent builder
│   ├── server/
│   │   ├── ChatController.java          # REST + SSE chat
│   │   ├── AdminController.java         # Admin management
│   │   └── HealthController.java        # Health/readiness
│   └── harness/
│       └── WorkspaceConfig.java         # Path configuration
├── src/main/resources/
│   ├── application.yml                  # App configuration
│   └── agents/                          # Agent YAML definitions
├── src/test/java/io/superagent/
│   ├── agents/                          # Agent unit tests
│   ├── server/                          # Controller tests
│   └── tools/                           # Tool tests
├── pom.xml                              # Maven build
├── Dockerfile                           # Multi-stage Docker build
└── README.md
```

## Configuration

Edit `src/main/resources/application.yml`:

```yaml
server:
  port: 8890

superagent:
  agents-dir: configs/agents    # Path to agent YAML files
  agent:
    max-steps: 10
    timeout-seconds: 120

spring:
  redis:
    url: redis://localhost:6379
```

## Agent YAML Definitions

Place agent YAML files in the `configs/agents/` directory:

```yaml
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
    fallback: deepseek-r1
  system_prompt: "You are a helpful assistant."
  tools:
    - ref: builtin/web_search
    - ref: builtin/http_request
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Runtime | Java 17+ (Eclipse Temurin) |
| Framework | Spring Boot 3.3 + WebFlux |
| Agent Framework | AgentScope Java 2.0 |
| SSE Streaming | Reactor Flux |
| Redis | Spring Data Redis Reactive |
| Metrics | Micrometer + Prometheus |
| Build | Maven |
| Container | Docker (multi-stage) |

## Development Status

This is a **skeleton project** — all classes are stubs with TODO markers.
Implementation will proceed in phases:

- [ ] Phase 1: YAML loading + agent builder wiring
- [ ] Phase 2: ChatModelAgent with AgentScope ReAct
- [ ] Phase 3: Model routing + fallback chains
- [ ] Phase 4: MCP tool integration
- [ ] Phase 5: SSE streaming with A2UI protocol
- [ ] Phase 6: Redis memory + checkpoint persistence
- [ ] Phase 7: Observability (OTel traces + Prometheus metrics)
