/**
 * Base error class for all Superagent SDK errors.
 */
export class SuperagentError extends Error {
  constructor(
    public readonly statusCode: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "SuperagentError";
    // Restore prototype chain for instanceof checks across transpilation boundaries.
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * Raised when the server returns 401 Unauthorized or 403 Forbidden.
 */
export class AuthenticationError extends SuperagentError {
  constructor(message = "Authentication failed") {
    super(401, "authentication_error", message);
    this.name = "AuthenticationError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * Raised when the SSE connection drops unexpectedly before a `done` event.
 */
export class StreamDisconnectedError extends SuperagentError {
  constructor(message = "Stream disconnected unexpectedly") {
    super(0, "stream_disconnected", message);
    this.name = "StreamDisconnectedError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * Raised when trying to resume a session that has no pending interrupt (HTTP 409).
 */
export class InterruptConflictError extends SuperagentError {
  constructor(message = "No pending interrupt for this session") {
    super(409, "interrupt_conflict", message);
    this.name = "InterruptConflictError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * Raised when agent YAML validation fails (HTTP 400).
 */
export class ValidationError extends SuperagentError {
  constructor(message = "Validation failed") {
    super(400, "validation_error", message);
    this.name = "ValidationError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
