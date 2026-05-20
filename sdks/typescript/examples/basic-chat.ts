/**
 * basic-chat.ts — Synchronous chat example.
 *
 * Sends a single message and waits for the complete response.
 *
 * Run with:
 *   npx tsx examples/basic-chat.ts
 */

import { SuperagentClient } from "../src/index.js";

const client = new SuperagentClient({
  baseUrl: process.env["SUPERAGENT_URL"] ?? "http://localhost:8888",
  // apiKey: process.env['SUPERAGENT_API_KEY'],
});

const question = process.argv[2] ?? "What is quantum computing?";

console.log(`Asking: ${question}\n`);

const response = await client.chat("research-agent", question);

console.log("Response:");
console.log(response);
