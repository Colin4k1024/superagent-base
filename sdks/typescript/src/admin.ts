import { AgentInfo, ApplyResult, ValidateResult } from "./types.js";

/** Internal request function type used by AdminClient. */
type RequestFn = <T>(
  method: string,
  path: string,
  body?: unknown,
) => Promise<T>;

/**
 * AdminClient provides access to the Superagent admin API endpoints.
 * Obtain an instance via `client.admin`.
 */
export class AdminClient {
  constructor(private readonly _request: RequestFn) {}

  /**
   * Returns runtime status including uptime, agent count, and health checks.
   * GET /api/v2/admin/status
   */
  async status(): Promise<Record<string, unknown>> {
    return this._request<Record<string, unknown>>("GET", "/api/v2/admin/status");
  }

  /**
   * Triggers a hot-reload of all agent YAML definitions.
   * POST /api/v2/admin/reload
   */
  async reload(): Promise<{ message: string }> {
    return this._request<{ message: string }>("POST", "/api/v2/admin/reload");
  }

  /**
   * Returns all agents known to the runtime with their status.
   * GET /api/v2/admin/agents
   */
  async listAgents(): Promise<AgentInfo[]> {
    const resp = await this._request<{ agents: AgentInfo[] }>(
      "GET",
      "/api/v2/admin/agents",
    );
    return resp.agents ?? [];
  }

  /**
   * Returns the definition and raw YAML for a specific agent.
   * GET /api/v2/admin/agents/:name
   */
  async getAgent(name: string): Promise<Record<string, unknown>> {
    return this._request<Record<string, unknown>>(
      "GET",
      `/api/v2/admin/agents/${encodeURIComponent(name)}`,
    );
  }

  /**
   * Creates a new agent from YAML content.
   * POST /api/v2/admin/agents
   */
  async createAgent(yamlContent: string): Promise<ApplyResult> {
    const raw = await this._request<{ name: string; message: string }>(
      "POST",
      "/api/v2/admin/agents",
      { yaml: yamlContent },
    );
    return {
      name: raw.name,
      status: "created",
      message: raw.message,
    };
  }

  /**
   * Overwrites an existing agent YAML definition.
   * PUT /api/v2/admin/agents/:name
   */
  async updateAgent(name: string, yamlContent: string): Promise<ApplyResult> {
    const raw = await this._request<{ name: string; message: string }>(
      "PUT",
      `/api/v2/admin/agents/${encodeURIComponent(name)}`,
      { yaml: yamlContent },
    );
    return {
      name: raw.name,
      status: "updated",
      message: raw.message,
    };
  }

  /**
   * Deletes an agent YAML file and unloads it from the runtime.
   * DELETE /api/v2/admin/agents/:name
   */
  async deleteAgent(name: string): Promise<void> {
    await this._request<unknown>(
      "DELETE",
      `/api/v2/admin/agents/${encodeURIComponent(name)}`,
    );
  }

  /**
   * Validates agent YAML without writing it to disk.
   * POST /api/v2/admin/agents/validate
   */
  async validateAgent(yamlContent: string): Promise<ValidateResult> {
    const raw = await this._request<{
      valid: boolean;
      error?: string;
      errors?: string[];
    }>("POST", "/api/v2/admin/agents/validate", { yaml: yamlContent });

    const errors: string[] = [];
    if (!raw.valid) {
      if (raw.error) errors.push(raw.error);
      if (raw.errors) errors.push(...raw.errors);
    }

    return { valid: raw.valid, errors };
  }
}
