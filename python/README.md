# superagent-base-python

Python agent platform based on **AgentScope 2.0** — API-compatible with the Go base (`superagent-base`).

Part of the three-base architecture (Go / Python / Java).

## Quick Start

```bash
pip install -e ".[dev]"
uvicorn superagent.server:app --reload --port 8889
```

## Docker

```bash
docker build -t superagent-py .
docker run -p 8889:8889 --env-file .env superagent-py
```

## Project Structure

```
src/superagent/
├── server.py          # FastAPI app + SSE streaming endpoints
├── agents/            # Agent type implementations (chat, supervisor, etc.)
├── tools/             # Built-in tools + MCP client
├── memory/            # Memory backends (builtin, Redis)
├── models/            # Model registry + routing strategies
├── config/            # YAML loader + Pydantic schemas
├── harness/           # Workspace configuration
└── evolution/         # Signal collector (stub)
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/chat` | Send message to agent (SSE stream) |
| POST | `/api/v1/chat/resume` | Resume interrupted agent |
| GET | `/api/v1/agents` | List available agents |
| GET | `/api/v1/agents/{id}` | Get agent details |
| POST | `/api/v1/agents` | Create agent from YAML |
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |

## Development

```bash
make test        # Run tests
make lint        # Lint with ruff
make typecheck   # Type-check with mypy
```
