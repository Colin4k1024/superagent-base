/**
 * @superagent-ai/sdk — TypeScript SDK for Superagent Base
 *
 * Zero external dependencies; requires Node.js >= 18.
 */

export { SuperagentClient } from "./client.js";
export { AdminClient } from "./admin.js";
export { A2UIStream } from "./streaming.js";

// Error classes
export {
  SuperagentError,
  AuthenticationError,
  StreamDisconnectedError,
  InterruptConflictError,
  ValidationError,
} from "./errors.js";

// Types and enums
export {
  A2UIEventType,
  type A2UIEvent,
  type AgentInfo,
  type ClientOptions,
  type ChatOptions,
  type ApplyResult,
  type ValidateResult,
} from "./types.js";
