import { A2UIEvent, A2UIEventType } from "./types.js";
import { StreamDisconnectedError } from "./errors.js";

type EventHandler = (data: Record<string, unknown>) => void;

/**
 * A2UIStream wraps a streaming HTTP response and exposes A2UI events via
 * two consumption patterns:
 *
 *   1. Async iterator: `for await (const event of stream) { ... }`
 *   2. Event listener: `stream.on('text', handler).on('done', handler)`
 *
 * Only one consumption pattern should be used per stream instance.
 */
export class A2UIStream implements AsyncIterable<A2UIEvent> {
  private readonly responsePromise: Promise<Response>;
  private readonly controller: AbortController;
  private readonly callbacks: Map<string, EventHandler[]> = new Map();

  constructor(responsePromise: Promise<Response>, controller: AbortController) {
    this.responsePromise = responsePromise;
    this.controller = controller;
  }

  // ---------------------------------------------------------------------------
  // Event listener API
  // ---------------------------------------------------------------------------

  /**
   * Register a handler for a specific A2UI event type.
   * Returns `this` to allow chaining.
   *
   * @example
   * stream
   *   .on(A2UIEventType.Text, ({ delta }) => process.stdout.write(String(delta)))
   *   .on('done', () => console.log('finished'));
   */
  on(
    event: A2UIEventType | "text" | "error" | "done",
    handler: EventHandler,
  ): this {
    const list = this.callbacks.get(event) ?? [];
    list.push(handler);
    this.callbacks.set(event, list);
    return this;
  }

  // ---------------------------------------------------------------------------
  // Async iterator API
  // ---------------------------------------------------------------------------

  [Symbol.asyncIterator](): AsyncIterator<A2UIEvent> {
    return this._iterate();
  }

  private async *_iterate(): AsyncGenerator<A2UIEvent> {
    const response = await this.responsePromise;
    if (!response.ok) {
      await this._throwHttpError(response);
    }
    if (!response.body) {
      throw new StreamDisconnectedError("Response body is null");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const frames = buffer.split("\n\n");
        // Keep the last (potentially incomplete) frame in the buffer.
        buffer = frames.pop() ?? "";

        for (const frame of frames) {
          const event = parseSSEFrame(frame);
          if (event !== null) {
            this._dispatch(event);
            yield event;
          }
        }
      }

      // Flush any remaining buffer content.
      if (buffer.trim()) {
        const event = parseSSEFrame(buffer);
        if (event !== null) {
          this._dispatch(event);
          yield event;
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  // ---------------------------------------------------------------------------
  // Convenience: consume all events and return concatenated text
  // ---------------------------------------------------------------------------

  /**
   * Iterates the stream to completion and returns all text delta content
   * concatenated into a single string.
   */
  async collect(): Promise<string> {
    let result = "";
    for await (const event of this) {
      if (event.eventType === A2UIEventType.Text) {
        const delta = event.data["delta"];
        if (typeof delta === "string") {
          result += delta;
        }
      }
    }
    return result;
  }

  // ---------------------------------------------------------------------------
  // Cancel
  // ---------------------------------------------------------------------------

  /** Abort the underlying HTTP request and close the stream. */
  abort(): void {
    this.controller.abort();
  }

  // ---------------------------------------------------------------------------
  // Internal helpers
  // ---------------------------------------------------------------------------

  /**
   * Start consuming the stream using the event listener callbacks.
   * Called automatically when `.on()` is used before iteration begins.
   */
  async _startCallbackLoop(): Promise<void> {
    try {
      for await (const event of this) {
        // Events are dispatched inside _iterate() via _dispatch().
        // Nothing extra needed here; the loop just drives consumption.
        void event;
      }
    } catch (err) {
      const handlers = this.callbacks.get("error") ?? [];
      const payload =
        err instanceof Error
          ? { message: err.message, code: "stream_error" }
          : { message: String(err), code: "stream_error" };
      for (const h of handlers) {
        h(payload);
      }
    }
  }

  private _dispatch(event: A2UIEvent): void {
    const handlers = this.callbacks.get(event.eventType) ?? [];
    for (const h of handlers) {
      h(event.data);
    }
  }

  private async _throwHttpError(response: Response): Promise<never> {
    let message = `HTTP ${response.status}`;
    try {
      const body = await response.text();
      const json = JSON.parse(body) as Record<string, unknown>;
      if (typeof json["msg"] === "string") message = json["msg"];
      else if (typeof json["error"] === "string") message = json["error"];
      else if (typeof json["message"] === "string") message = json["message"];
    } catch {
      // Ignore parse errors; keep the default message.
    }

    const { SuperagentError, AuthenticationError, InterruptConflictError } =
      await import("./errors.js");
    if (response.status === 401 || response.status === 403) {
      throw new AuthenticationError(message);
    }
    if (response.status === 409) {
      throw new InterruptConflictError(message);
    }
    throw new SuperagentError(response.status, "http_error", message);
  }
}

// ---------------------------------------------------------------------------
// SSE frame parser
// ---------------------------------------------------------------------------

/**
 * Parse a single SSE frame (the text between two `\n\n` separators).
 *
 * Expected wire format from the Superagent backend (A2UI mode):
 *   event: <A2UIEventType>
 *   data: <JSON string>
 *
 * Returns null for comment-only or empty frames.
 */
function parseSSEFrame(frame: string): A2UIEvent | null {
  let eventType: string | null = null;
  let dataLine: string | null = null;

  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) {
      eventType = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLine = line.slice(5).trim();
    }
    // Ignore id:, retry:, and comment lines (:).
  }

  if (!dataLine) return null;

  // Handle legacy plain-token mode where the backend emits `data: <token>`.
  if (eventType === null || eventType === "message") {
    if (dataLine === "[DONE]") {
      return {
        eventType: A2UIEventType.Done,
        data: {},
      };
    }
    return {
      eventType: A2UIEventType.Text,
      data: { delta: dataLine, content: dataLine },
    };
  }

  let parsed: Record<string, unknown> = {};
  try {
    const raw = JSON.parse(dataLine) as unknown;
    // The backend wraps the payload in an Event envelope: { type, timestamp, data }.
    // Extract the inner data field when present.
    if (
      raw !== null &&
      typeof raw === "object" &&
      "data" in (raw as object) &&
      typeof (raw as Record<string, unknown>)["data"] === "object" &&
      (raw as Record<string, unknown>)["data"] !== null
    ) {
      parsed = (raw as Record<string, unknown>)["data"] as Record<
        string,
        unknown
      >;
    } else if (raw !== null && typeof raw === "object") {
      parsed = raw as Record<string, unknown>;
    }
  } catch {
    // Non-JSON data; wrap as raw content.
    parsed = { raw: dataLine };
  }

  return {
    eventType: eventType as A2UIEventType,
    data: parsed,
  };
}
