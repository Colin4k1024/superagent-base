import { AdminClient } from "./admin.js";
import {
  AuthenticationError,
  InterruptConflictError,
  SuperagentError,
} from "./errors.js";
import { A2UIStream } from "./streaming.js";
import { AgentInfo, ChatOptions, ClientOptions } from "./types.js";

/**
 * SuperagentClient is the primary entry point for interacting with a
 * Superagent server. It supports both synchronous and streaming chat,
 * interrupt/resume, agent listing, and an admin sub-client.
 *
 * @example
 * ```ts
 * const client = new SuperagentClient({ baseUrl: 'http://localhost:8888' });
 * const text = await client.chat('research-agent', 'What is quantum computing?');
 * ```
 */
export class SuperagentClient {
  private readonly baseUrl: string;
  private readonly apiKey: string | undefined;
  private readonly timeout: number;
  private readonly _admin: AdminClient;

  constructor(options: ClientOptions) {
    // Strip trailing slash so all path concatenations are uniform.
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.apiKey = options.apiKey;
    this.timeout = options.timeout ?? 30_000;
    this._admin = new AdminClient(this.request.bind(this));
  }

  // ---------------------------------------------------------------------------
  // Chat: synchronous
  // ---------------------------------------------------------------------------

  /**
   * Send a message to an agent and wait for the complete response.
   * Internally opens a streaming connection and concatenates all text deltas.
   *
   * @param agentId  - Agent name as defined in its YAML metadata.
   * @param message  - The user message to send.
   * @param options  - Optional session identifier.
   * @returns The full text response.
   */
  async chat(
    agentId: string,
    message: string,
    options?: ChatOptions,
  ): Promise<string> {
    return this.chatStream(agentId, message, options).collect();
  }

  // ---------------------------------------------------------------------------
  // Chat: streaming
  // ---------------------------------------------------------------------------

  /**
   * Send a message to an agent and receive a streaming A2UIStream.
   * The stream yields typed A2UI events as they arrive.
   *
   * @example
   * ```ts
   * const stream = client.chatStream('research-agent', 'Explain RAG');
   * for await (const event of stream) {
   *   if (event.eventType === A2UIEventType.Text) {
   *     process.stdout.write(String(event.data.delta));
   *   }
   * }
   * ```
   */
  chatStream(
    agentId: string,
    message: string,
    options?: ChatOptions,
  ): A2UIStream {
    return this.streamRequest("POST", "/api/v2/chat/stream", {
      agent_id: agentId,
      session_id: options?.sessionId ?? "default",
      message,
    });
  }

  // ---------------------------------------------------------------------------
  // Interrupt / resume
  // ---------------------------------------------------------------------------

  /**
   * Resume an interrupted agent session.
   * Returns a streaming response with the continued execution.
   *
   * @param agentId   - The agent that is currently interrupted.
   * @param sessionId - The session that holds the pending interrupt state.
   * @param input     - User-provided values for the interrupt form fields.
   */
  resume(
    agentId: string,
    sessionId: string,
    input: Record<string, unknown>,
  ): A2UIStream {
    return this.streamRequest("POST", "/api/v2/chat/resume", {
      agent_id: agentId,
      session_id: sessionId,
      input,
    });
  }

  // ---------------------------------------------------------------------------
  // Agent listing
  // ---------------------------------------------------------------------------

  /**
   * List all agents currently loaded by the runtime.
   * GET /api/v2/agents
   */
  async listAgents(): Promise<AgentInfo[]> {
    const resp = await this.request<{
      agents: Array<{ name: string; description: string }>;
    }>("GET", "/api/v2/agents");
    return (resp.agents ?? []).map((a) => ({
      name: a.name,
      type: "",
      description: a.description,
      status: "loaded",
    }));
  }

  // ---------------------------------------------------------------------------
  // Admin sub-client
  // ---------------------------------------------------------------------------

  /** Access the admin API for agent CRUD and runtime management. */
  get admin(): AdminClient {
    return this._admin;
  }

  // ---------------------------------------------------------------------------
  // Internal: HTTP primitives
  // ---------------------------------------------------------------------------

  /**
   * Perform a JSON HTTP request with auth headers and error mapping.
   * Throws a typed error on non-2xx responses.
   */
  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    let response: Response;
    try {
      response = await fetch(this.baseUrl + path, {
        method,
        headers: this.buildHeaders(),
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });
    } catch (err) {
      clearTimeout(timer);
      if (err instanceof Error && err.name === "AbortError") {
        throw new SuperagentError(0, "timeout", `Request timed out: ${path}`);
      }
      throw err;
    }
    clearTimeout(timer);

    if (!response.ok) {
      await this.throwHttpError(response);
    }

    const text = await response.text();
    if (!text) return undefined as unknown as T;
    return JSON.parse(text) as T;
  }

  /**
   * Create an A2UIStream for a streaming HTTP request.
   * The actual fetch is lazy — it starts when the stream is iterated.
   */
  streamRequest(method: string, path: string, body?: unknown): A2UIStream {
    const controller = new AbortController();
    const responsePromise = fetch(this.baseUrl + path, {
      method,
      headers: {
        ...this.buildHeaders(),
        // Request A2UI structured events from the backend.
        "X-A2UI": "true",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });
    return new A2UIStream(responsePromise, controller);
  }

  private buildHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.apiKey) {
      headers["Authorization"] = `Bearer ${this.apiKey}`;
    }
    return headers;
  }

  private async throwHttpError(response: Response): Promise<never> {
    let message = `HTTP ${response.status}`;
    try {
      const text = await response.text();
      const json = JSON.parse(text) as Record<string, unknown>;
      if (typeof json["msg"] === "string") message = json["msg"];
      else if (typeof json["error"] === "string") message = json["error"];
      else if (typeof json["message"] === "string") message = json["message"];
    } catch {
      // Keep default message on parse failure.
    }

    if (response.status === 401 || response.status === 403) {
      throw new AuthenticationError(message);
    }
    if (response.status === 409) {
      throw new InterruptConflictError(message);
    }
    throw new SuperagentError(response.status, "http_error", message);
  }
}
