# Superagent Python SDK

Python client library for the [Superagent](https://github.com/superagent-ai/superagent-base) AI Agent platform.

## Requirements

- Python 3.9+
- `httpx >= 0.25.0`
- `pydantic >= 2.0`

## Installation

```bash
pip install superagent-sdk
```

Or from source:

```bash
cd sdks/python
pip install -e .
```

## Quick Start

```python
import asyncio
from superagent import SuperagentClient

async def main():
    async with SuperagentClient(base_url="http://localhost:8888") as client:
        reply = await client.chat("research-agent", "What is quantum computing?")
        print(reply)

asyncio.run(main())
```

## Streaming

```python
import asyncio
from superagent import SuperagentClient, A2UIEventType

async def main():
    async with SuperagentClient(base_url="http://localhost:8888") as client:
        stream = client.chat_stream("research-agent", "Explain quantum entanglement")

        async for event in stream:
            if event.event_type == A2UIEventType.text:
                print(event.text_delta, end="", flush=True)
            elif event.event_type == A2UIEventType.done:
                print()

asyncio.run(main())
```

## Authentication

```python
client = SuperagentClient(
    base_url="http://localhost:8888",
    api_key="your-api-key",
)
```

## Agent Administration

```python
async with SuperagentClient() as client:
    # List all agents
    agents = await client.admin.list_agents()

    # Create an agent from YAML
    result = await client.admin.create_agent(yaml_content=open("agent.yaml").read())

    # Validate YAML before applying
    validation = await client.admin.validate_agent(yaml_content="...")
    if not validation.valid:
        print(validation.errors)

    # Hot-reload all agents
    await client.admin.reload()
```

## Interrupt / Resume

```python
async with SuperagentClient() as client:
    stream = client.chat_stream("my-agent", "Book a flight to Tokyo", session_id="s1")

    async for event in stream:
        if event.event_type == "interrupt":
            # Collect user input for the interrupt fields
            user_input = {"confirm": True, "date": "2025-12-01"}
            resume_stream = await client.resume("my-agent", "s1", user_input)
            async for resumed_event in resume_stream:
                if resumed_event.event_type == "text":
                    print(resumed_event.text_delta, end="")
            break
```

## Error Handling

```python
from superagent import (
    AuthenticationError,
    NotFoundError,
    ServerError,
    StreamDisconnectedError,
)

try:
    reply = await client.chat("unknown-agent", "Hello")
except NotFoundError:
    print("Agent not found")
except AuthenticationError:
    print("Check your API key")
except ServerError as e:
    print(f"Server error {e.status_code}: {e.message}")
except StreamDisconnectedError:
    print("Stream dropped — retrying manually")
```

## Event Types

| Event Type      | Description                                  |
|-----------------|----------------------------------------------|
| `text`          | Streaming text token from the agent          |
| `thinking`      | Internal reasoning / chain-of-thought token  |
| `tool_call`     | Agent is calling a tool                      |
| `tool_result`   | Result returned from a tool                  |
| `code_block`    | A complete or streaming code block           |
| `interrupt`     | Agent needs user input before continuing     |
| `error`         | Error during agent execution                 |
| `done`          | Stream completed successfully                |
| `progress`      | Step progress update for long-running tasks  |
| `agent_switch`  | Handoff between sub-agents                   |

## API Reference

### `SuperagentClient`

| Method | Description |
|--------|-------------|
| `chat(agent_id, message, session_id)` | Send a message, return full text response |
| `chat_stream(agent_id, message, session_id)` | Return an `A2UIStream` for live events |
| `resume(agent_id, session_id, input)` | Resume after an interrupt |
| `list_agents()` | List all live agents |
| `admin` | Access the `AdminClient` |

### `A2UIStream`

| Method | Description |
|--------|-------------|
| `async for event in stream` | Iterate events |
| `await stream.collect()` | Collect all text, return as string |
| `stream.on(event_type, callback)` | Register an event callback |

### `AdminClient`

| Method | Description |
|--------|-------------|
| `status()` | System status |
| `reload()` | Hot-reload agent YAML files |
| `list_agents()` | All registered agents |
| `get_agent(name)` | Single agent definition + YAML |
| `create_agent(yaml_content)` | Create a new agent |
| `update_agent(name, yaml_content)` | Update an existing agent |
| `delete_agent(name)` | Delete an agent |
| `validate_agent(yaml_content)` | Validate YAML without applying |
