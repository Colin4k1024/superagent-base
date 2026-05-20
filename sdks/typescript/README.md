# @superagent-ai/sdk

TypeScript SDK for [Superagent Base](https://github.com/superagent-ai/superagent-base).

Zero external dependencies — uses only Node.js 18+ native `fetch`.

## Installation

```bash
npm install @superagent-ai/sdk
```

## Quick start

```ts
import { SuperagentClient } from '@superagent-ai/sdk';

const client = new SuperagentClient({ baseUrl: 'http://localhost:8888' });

// Synchronous: wait for full response
const answer = await client.chat('research-agent', 'What is quantum computing?');
console.log(answer);
```

## Streaming

### Async iterator

```ts
import { SuperagentClient, A2UIEventType } from '@superagent-ai/sdk';

const client = new SuperagentClient({ baseUrl: 'http://localhost:8888' });
const stream = client.chatStream('research-agent', 'Explain RAG');

for await (const event of stream) {
  if (event.eventType === A2UIEventType.Text) {
    process.stdout.write(String(event.data.delta));
  }
}
```

### Event listener

```ts
const stream = client.chatStream('research-agent', 'Hello');

stream
  .on(A2UIEventType.Text, ({ delta }) => process.stdout.write(String(delta)))
  .on('done', () => console.log('\ndone'));

await stream._startCallbackLoop();
```

## Interrupt / resume

```ts
// Resume an interrupted agent session
const stream = client.resume('approval-agent', 'session-1', { confirm: true });
for await (const event of stream) { /* ... */ }
```

## Admin API

```ts
// List agents
const agents = await client.admin.listAgents();

// Create agent from YAML
await client.admin.createAgent(`
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
spec:
  type: chat_model_agent
  model: gpt-4o-mini
  systemPrompt: You are a helpful assistant.
`);

// Validate YAML without writing
const result = await client.admin.validateAgent(yaml);
if (!result.valid) console.error(result.errors);

// Hot-reload all agents
await client.admin.reload();
```

## API reference

### `SuperagentClient`

| Method | Description |
|--------|-------------|
| `chat(agentId, message, opts?)` | Send message, return full text response |
| `chatStream(agentId, message, opts?)` | Return `A2UIStream` for streaming |
| `resume(agentId, sessionId, input)` | Resume interrupted session |
| `listAgents()` | List all loaded agents |
| `admin` | Access `AdminClient` |

### `A2UIStream`

| Member | Description |
|--------|-------------|
| `[Symbol.asyncIterator]()` | Iterate events with `for await...of` |
| `on(eventType, handler)` | Register event callback, returns `this` |
| `collect()` | Consume stream, return concatenated text |
| `abort()` | Cancel the request |

### `AdminClient`

| Method | Description |
|--------|-------------|
| `status()` | Runtime status |
| `reload()` | Hot-reload agent definitions |
| `listAgents()` | All agents with status |
| `getAgent(name)` | Agent definition + raw YAML |
| `createAgent(yaml)` | Create from YAML |
| `updateAgent(name, yaml)` | Update existing agent |
| `deleteAgent(name)` | Delete agent |
| `validateAgent(yaml)` | Validate YAML without writing |

## Requirements

- Node.js >= 18
- No external dependencies
