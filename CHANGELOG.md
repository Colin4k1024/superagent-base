# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-05-11

### Added
- Project initialization based on Coze Studio fork (Apache 2.0)
- Agent YAML declarative definition system (`apiVersion: superagent/v1`)
- Agent Runtime Engine with real Eino + LLM integration
- ConfigMap Watch hot-reload for agent definitions
- Model Router with capability-based/cost-optimized/latency strategies
- MCP Client (stdio + SSE transport) for external tool consumption
- MCP Server for exposing platform capabilities
- Memory adapter system with 4 backends (builtin/Mem0/Zep/Letta)
- SkillsHub Client for external skill integration
- Tool system with middleware chain (retry/timeout/rate-limit/cache)
- Built-in tools: web_search, http_request, code_execute
- OpenTelemetry tracing + Prometheus metrics
- gRPC API (AgentService, ConversationService, ModelService, ToolService)
- HTTP SSE streaming endpoint (POST /api/v1/chat/stream)
- Web UI skeleton (React + TypeScript + Vite + Tailwind)
- Docker Compose development environment
- Helm chart for Kubernetes deployment
- CLI tool (sactl) for skill and agent management
- Multi-agent orchestration (Supervisor/Sequential/Parallel patterns)
- Complete Chinese documentation (architecture, YAML spec, model config, deployment)

### Infrastructure
- Go 1.24 + Eino framework
- MySQL 8.4 + Redis 7 + Elasticsearch 8 + Milvus 2.5
- Hertz HTTP + gRPC dual-protocol server
- 7 LLM providers (OpenAI/Claude/Gemini/DeepSeek/Ark/Ollama/Qwen)
