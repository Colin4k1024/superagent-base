// REST API client for the Superagent backend.
// All requests go to /api which the Vite dev proxy forwards to localhost:8888.

import { getToken, handleLogin } from './auth'
import type { InterruptField } from './agentloop-types'

const API_BASE = '/api/v1'

function authHeaders(): Record<string, string> {
  const token = getToken()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) {
    headers['Access-Token'] = token
  }
  return headers
}

function handleAuthError(res: Response): void {
  if (res.status === 401 || res.status === 403) {
    handleLogin({ invalidateToken: true })
    throw new Error('Session expired')
  }
}

export interface Agent {
  name: string
  description: string
}

export interface FileAttachment {
  name: string
  size?: number
  type?: string
  url?: string
}

export interface CardData {
  title?: string
  content: string
}

export interface Reference {
  title: string
  url?: string
  source?: string
}

export interface ToolCallInfo {
  name: string
  args?: string
  result?: string
  status?: 'calling' | 'done' | 'error'
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  type?: 'text' | 'card' | 'file' | 'thinking'
  thinking?: string
  thinkingSteps?: string[]
  files?: FileAttachment[]
  card?: CardData
  references?: Reference[]
  toolCalls?: ToolCallInfo[]
  /** True when the turn was cancelled by a preempt or abort before completion. */
  preempted?: boolean
}

export interface AdminAgent {
  name: string
  type: string
  status: string
  description: string
}

export interface AdminStatus {
  uptime_seconds: number
  agent_count: number
  agents: AdminAgent[]
  health: string
  ready: boolean
  readiness_checks?: Record<string, string>
  start_time: string
  last_reload_at: string
}

export interface ReloadResult {
  message: string
  agent_count: number
  agents: AdminAgent[]
}

export const agentsApi = {
  async list(): Promise<Agent[]> {
    const res = await fetch(`${API_BASE}/agents`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    return data.agents || []
  },
}

export const adminApi = {
  async getStatus(): Promise<AdminStatus> {
    const res = await fetch(`${API_BASE}/admin/status`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async triggerReload(): Promise<ReloadResult> {
    const res = await fetch(`${API_BASE}/admin/reload`, {
      method: 'POST',
      headers: authHeaders(),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },
}

export interface AgentDetail {
  name: string
  type: string
  description: string
  status: string
  file: string
}

export const agentAdminApi = {
  list: async (): Promise<{ agents: AgentDetail[] }> => {
    const res = await fetch(`${API_BASE}/admin/agents`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  get: async (name: string): Promise<{ agent: unknown; yaml: string }> => {
    const res = await fetch(`${API_BASE}/admin/agents/${name}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  create: async (yaml: string): Promise<unknown> => {
    const res = await fetch(`${API_BASE}/admin/agents`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ yaml }),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  update: async (name: string, yaml: string): Promise<unknown> => {
    const res = await fetch(`${API_BASE}/admin/agents/${name}`, {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify({ yaml }),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  delete: async (name: string): Promise<unknown> => {
    const res = await fetch(`${API_BASE}/admin/agents/${name}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  validate: async (yaml: string): Promise<{ valid: boolean; error?: string; agent?: unknown }> => {
    const res = await fetch(`${API_BASE}/admin/agents/validate`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ yaml }),
    })
    handleAuthError(res)
    return res.json()
  },
}

export const metricsApi = {
  async getRaw(): Promise<string> {
    const res = await fetch('/metrics')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.text()
  },
}

export interface SkillInfo {
  name: string
  version: string
  description: string
  type?: string
  installed?: boolean
  // Marketplace fields (populated by skills.sh)
  installs?: number
  source?: string
  author?: string
  install_cmd?: string
  url?: string
}

export const skillsApi = {
  list: async (): Promise<{ skills: SkillInfo[] }> => {
    const res = await fetch(`${API_BASE}/skills`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Failed to fetch skills')
    return res.json()
  },
  search: async (query: string): Promise<{ skills: SkillInfo[] }> => {
    const res = await fetch(`${API_BASE}/skills/search?q=${encodeURIComponent(query)}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Search failed')
    return res.json()
  },
  install: async (name: string, version = 'latest'): Promise<unknown> => {
    const res = await fetch(`${API_BASE}/skills/install`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ name, version }),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  uninstall: async (name: string): Promise<unknown> => {
    const res = await fetch(`${API_BASE}/skills/${name}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
}

// ModelClass integer values from backend developer_api.ModelClass enum
export const MODEL_CLASS_MAP: Record<string, number> = {
  OpenAI:   1,   // ModelClass_GPT
  Claude:   3,   // ModelClass_Claude
  DeepSeek: 19,  // ModelClass_DeekSeek
  Gemini:   11,  // ModelClass_Gemini
  Ollama:   20,  // ModelClass_Llama
  Qwen:     15,  // ModelClass_QWen
  Ark:      2,   // ModelClass_SEED
}

export const MODEL_CLASS_LABELS: Record<number, string> = {
  1:  'OpenAI',
  2:  'Ark',
  3:  'Claude',
  11: 'Gemini',
  15: 'Qwen',
  19: 'DeepSeek',
  20: 'Ollama',
}

export interface CreateModelRequest {
  model_class: string  // display name key e.g., "OpenAI", "Claude"
  name: string         // display name
  base_url: string
  api_key: string
  model: string        // model ID e.g., "gpt-4o"
}

export interface ModelRecord {
  id: number
  name: string
  model: string
  model_class_id: number
  model_class: string
  base_url: string
  api_key?: string
}

export const modelConfigApi = {
  list: async (): Promise<ModelRecord[]> => {
    const res = await fetch('/api/admin/config/model/list', { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Failed to fetch models')
    const data = await res.json()
    // Backend returns {provider_model_list: [{provider, model_list: [Model]}]}
    const providerList: any[] = data?.provider_model_list ?? []
    const models: ModelRecord[] = []
    for (const pl of providerList) {
      for (const m of (pl.model_list ?? [])) {
        const classId: number = m.provider?.model_class ?? 0
        models.push({
          id: m.id,
          name: m.display_info?.name ?? '',
          model: m.connection?.base_conn_info?.model ?? '',
          model_class_id: classId,
          model_class: MODEL_CLASS_LABELS[classId] ?? String(classId),
          base_url: m.connection?.base_conn_info?.base_url ?? '',
          api_key: m.connection?.base_conn_info?.api_key,
        })
      }
    }
    return models
  },
  create: async (data: CreateModelRequest): Promise<any> => {
    const classId = MODEL_CLASS_MAP[data.model_class] ?? 1
    const payload = {
      model_class: classId,
      model_name: data.name,
      connection: {
        base_conn_info: {
          base_url: data.base_url,
          api_key: data.api_key,
          model: data.model,
        },
      },
    }
    const res = await fetch('/api/admin/config/model/create', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(payload),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  delete: async (modelId: number): Promise<any> => {
    const res = await fetch('/api/admin/config/model/delete', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ id: modelId }),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
}

// MCP server management API  (/api/v1/admin/mcp/servers)
export interface McpServerItem {
  name: string
  transport: string
  status: string
  tools_count: number
}

export interface McpConnectRequest {
  name: string
  transport: 'stdio' | 'sse'
  command?: string
  args?: string[]
  url?: string
  env?: Record<string, string>
}

export const mcpAdminApi = {
  list: async (): Promise<McpServerItem[]> => {
    const res = await fetch(`${API_BASE}/admin/mcp/servers`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Failed to fetch MCP servers')
    const data = await res.json()
    return data.servers ?? []
  },
  connect: async (req: McpConnectRequest): Promise<any> => {
    const res = await fetch(`${API_BASE}/admin/mcp/servers`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(req),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  disconnect: async (name: string): Promise<any> => {
    const res = await fetch(`${API_BASE}/admin/mcp/servers/${encodeURIComponent(name)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  listTools: async (name: string): Promise<any[]> => {
    const res = await fetch(`${API_BASE}/admin/mcp/servers/${encodeURIComponent(name)}/tools`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    return data.tools ?? []
  },
}

export interface ChatStreamCallbacks {
  onToken: (token: string) => void
  onThinking?: (text: string) => void
  onToolCall?: (name: string, args: string) => void
  onToolResult?: (name: string, result: string) => void
  onDone: () => void
  onPreempted?: () => void
  onError: (err: Error) => void
  onProgress?: (data: { agent_name?: string; step?: string; total: number; current: number }) => void
  onInterrupt?: (data: { reason: string; fields: InterruptField[] }) => void
}

/**
 * Shared SSE stream processor for A2UI JSON events and legacy raw text.
 * Reads from a Response body reader and dispatches parsed events via callbacks.
 */
async function processSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  callbacks: ChatStreamCallbacks,
): Promise<void> {
  const { onToken, onThinking, onToolCall, onToolResult, onDone, onPreempted, onError, onProgress, onInterrupt } = callbacks
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      const data = line.slice(6)
      if (data === '[DONE]') {
        onDone()
        return
      }

      // Try A2UI JSON format first
      try {
        const event = JSON.parse(data)
        // A2UI format: {"type":"text","data":{"delta":"token"}} or {"type":"text","content":"token"}
        const content = event.data?.delta || event.data?.content || event.content || ''
        switch (event.type) {
          case 'text':
            onToken(content)
            break
          case 'thinking':
            onThinking?.(content)
            break
          case 'tool_call': {
            const name = event.data?.name || event.name || ''
            const args = event.data?.arguments
            onToolCall?.(name, typeof args === 'object' ? JSON.stringify(args, null, 2) : String(args || ''))
            break
          }
          case 'tool_result': {
            const name = event.data?.name || event.name || ''
            const result = event.data?.result || event.data?.content || content
            onToolResult?.(name, typeof result === 'object' ? JSON.stringify(result, null, 2) : String(result))
            break
          }
          case 'error':
            onError(new Error(content || 'Unknown error'))
            return
          case 'done':
            onDone()
            return
          case 'preempted':
            onPreempted?.()
            return
          case 'progress':
            onProgress?.({
              agent_name: event.data?.agent_name,
              step: event.data?.step,
              total: event.data?.total ?? 0,
              current: event.data?.current ?? 0,
            })
            break
          case 'interrupt':
            onInterrupt?.({
              reason: event.data?.reason || '',
              fields: event.data?.fields || [],
            })
            break
          default:
            if (content) onToken(content)
        }
      } catch {
        // Legacy format: raw text token
        onToken(data)
      }
    }
  }
  onDone()
}

export const chatApi = {
  /**
   * Send a message and receive streaming response.
   * Supports both A2UI JSON events and legacy raw text format.
   * Returns an AbortController for cancellation.
   */
  sendMessage(
    agentId: string,
    sessionId: string,
    message: string,
    callbacks: ChatStreamCallbacks,
  ): AbortController {
    const { onDone, onError } = callbacks
    const controller = new AbortController()

    const run = async () => {
      try {
        const res = await fetch(`${API_BASE}/chat/stream`, {
          method: 'POST',
          headers: { ...authHeaders(), 'X-A2UI': 'true' },
          body: JSON.stringify({ agent_id: agentId, session_id: sessionId, message }),
          signal: controller.signal,
        })

        handleAuthError(res)
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }

        const reader = res.body?.getReader()
        if (!reader) throw new Error('No readable stream on response')

        await processSSEStream(reader, callbacks)
      } catch (err) {
        if ((err as Error).name === 'AbortError') {
          onDone()
          return
        }
        onError(err instanceof Error ? err : new Error(String(err)))
      }
    }

    run()
    return controller
  },

  /**
   * Tell the backend to abort the active turn for a session (fire-and-forget).
   */
  async abort(agentId: string, sessionId: string): Promise<void> {
    try {
      await fetch(`${API_BASE}/chat/abort`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ agent_id: agentId, session_id: sessionId }),
      })
    } catch {
      // best-effort; ignore errors
    }
  },

  /**
   * Resume an interrupted agent conversation.
   */
  async resume(
    agentId: string,
    sessionId: string,
    input: string,
    callbacks: ChatStreamCallbacks,
  ): Promise<AbortController> {
    const { onDone, onError } = callbacks
    const controller = new AbortController()

    const run = async () => {
      try {
        const res = await fetch(`${API_BASE}/chat/resume`, {
          method: 'POST',
          headers: { ...authHeaders(), 'X-A2UI': 'true' },
          body: JSON.stringify({ agent_id: agentId, session_id: sessionId, input }),
          signal: controller.signal,
        })
        handleAuthError(res)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)

        const reader = res.body?.getReader()
        if (!reader) throw new Error('No readable stream on response')

        await processSSEStream(reader, callbacks)
      } catch (err) {
        if ((err as Error).name === 'AbortError') { onDone(); return }
        onError(err instanceof Error ? err : new Error(String(err)))
      }
    }
    run()
    return controller
  },

  /**
   * Check if a session has a pending interrupt.
   */
  async interruptState(agentId: string, sessionId: string): Promise<{ interrupted: boolean; state?: unknown }> {
    try {
      const res = await fetch(`${API_BASE}/chat/interrupt_state?agent_id=${encodeURIComponent(agentId)}&session_id=${encodeURIComponent(sessionId)}`, {
        headers: authHeaders(),
      })
      if (!res.ok) return { interrupted: false }
      return res.json()
    } catch {
      return { interrupted: false }
    }
  },
}

// Evolution API
export interface EvolutionStats {
  enabled: boolean
  experience_url: string
  hub_url: string
  sender_id: string
  peer_nodes: number
  min_confidence: number
  max_suggestions: number
}

export interface GeneItem {
  id: string
  strategy: unknown
  confidence: number
  quality_score: number
  use_count: number
  success_count: number
  success_rate: number
  contributor_id: string
  created_at: string
}

export const evolutionApi = {
  async getStats(): Promise<EvolutionStats> {
    const res = await fetch(`${API_BASE}/admin/evolution/stats`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async listGenes(params?: { q?: string; min_confidence?: number; limit?: number }): Promise<{ enabled: boolean; genes: GeneItem[]; total: number }> {
    const qs = new URLSearchParams()
    if (params?.q) qs.set('q', params.q)
    if (params?.min_confidence !== undefined) qs.set('min_confidence', String(params.min_confidence))
    if (params?.limit !== undefined) qs.set('limit', String(params.limit))
    const res = await fetch(`${API_BASE}/admin/evolution/genes?${qs}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async federatedSearch(q: string, minConfidence = 0.5, limit = 10): Promise<{ results: unknown[]; total: number }> {
    const qs = new URLSearchParams({ q, min_confidence: String(minConfidence), limit: String(limit) })
    const res = await fetch(`${API_BASE}/admin/evolution/federated?${qs}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },
}
