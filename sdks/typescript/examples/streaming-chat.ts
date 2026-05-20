/**
 * streaming-chat.ts — Streaming chat examples.
 *
 * Demonstrates two patterns for consuming A2UI events:
 *   1. `for await...of` async iterator
 *   2. `.on()` event listener
 *
 * Run with:
 *   npx tsx examples/streaming-chat.ts
 */

import { SuperagentClient, A2UIEventType } from "../src/index.js";

const client = new SuperagentClient({
  baseUrl: process.env["SUPERAGENT_URL"] ?? "http://localhost:8888",
});

const question = process.argv[2] ?? "Explain how large language models work.";

// ---------------------------------------------------------------------------
// Pattern 1: async iterator
// ---------------------------------------------------------------------------
console.log("=== Pattern 1: async iterator ===\n");
console.log(`Q: ${question}\nA: `);

const stream = client.chatStream("research-agent", question, {
  sessionId: "example-session-1",
});

for await (const event of stream) {
  switch (event.eventType) {
    case A2UIEventType.Text: {
      const delta = event.data["delta"];
      if (typeof delta === "string") {
        process.stdout.write(delta);
      }
      break;
    }

    case A2UIEventType.ToolCall: {
      const name = event.data["name"];
      console.log(`\n[tool call: ${String(name)}]`);
      break;
    }

    case A2UIEventType.ToolResult: {
      const toolName = event.data["name"];
      const isError = event.data["is_error"];
      console.log(`[tool result: ${String(toolName)} ${isError ? "ERROR" : "OK"}]`);
      break;
    }

    case A2UIEventType.Thinking: {
      const delta = event.data["delta"];
      if (typeof delta === "string" && delta) {
        process.stdout.write(`<think>${delta}</think>`);
      }
      break;
    }

    case A2UIEventType.Done:
      console.log("\n\n[stream complete]");
      break;

    case A2UIEventType.Error: {
      const msg = event.data["message"];
      console.error(`\n[error: ${String(msg)}]`);
      break;
    }

    default:
      // Progress, AgentSwitch, etc. — ignore in this example.
      break;
  }
}

// ---------------------------------------------------------------------------
// Pattern 2: .on() event listener
// ---------------------------------------------------------------------------
console.log("\n\n=== Pattern 2: .on() event listener ===\n");
console.log(`Q: ${question}\nA: `);

const stream2 = client.chatStream("research-agent", question, {
  sessionId: "example-session-2",
});

stream2
  .on(A2UIEventType.Text, (data) => {
    if (typeof data["delta"] === "string") {
      process.stdout.write(data["delta"]);
    }
  })
  .on(A2UIEventType.ToolCall, (data) => {
    console.log(`\n[tool call: ${String(data["name"])}]`);
  })
  .on("done", () => {
    console.log("\n\n[stream complete via .on()]");
  })
  .on("error", (data) => {
    console.error(`\n[error: ${String(data["message"])}]`);
  });

// Drive the stream: the .on() pattern requires manual iteration.
await stream2._startCallbackLoop();
