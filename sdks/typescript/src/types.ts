/**
 * A2UI event types matching backend pkg/a2ui/event.go.
 */
export enum A2UIEventType {
  Text = "text",
  Thinking = "thinking",
  ToolCall = "tool_call",
  ToolResult = "tool_result",
  CodeBlock = "code_block",
  Interrupt = "interrupt",
  Error = "error",
  Done = "done",
  Progress = "progress",
  AgentSwitch = "agent_switch",
}

/** Raw A2UI event envelope received over the SSE stream. */
export interface A2UIEvent {
  /** Discriminator for the event kind. */
  eventType: A2UIEventType;
  /** Event-specific payload. */
  data: Record<string, unknown>;
}

/** Agent metadata returned by list endpoints. */
export interface AgentInfo {
  name: string;
  type: string;
  description: string;
  status: string;
}

/** Options for constructing a SuperagentClient. */
export interface ClientOptions {
  /** Base URL of the Superagent server (e.g. "http://localhost:8888"). */
  baseUrl: string;
  /** Optional API key sent as the `Authorization: Bearer <key>` header. */
  apiKey?: string;
  /** Request timeout in milliseconds. Defaults to 30000. */
  timeout?: number;
}

/** Per-request options for chat calls. */
export interface ChatOptions {
  /** Session identifier for conversation history. */
  sessionId?: string;
}

/** Result returned by agent create/update operations. */
export interface ApplyResult {
  name: string;
  status: "created" | "updated";
  message: string;
}

/** Result returned by agent YAML validation. */
export interface ValidateResult {
  valid: boolean;
  errors: string[];
}
