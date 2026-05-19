// REST API client for the Superagent backend.
// All requests go to /api which the Vite dev proxy forwards to localhost:8888.

import { getApiKey, clearApiKey } from './auth'

const API_BASE = '/api/v1'

function authHeaders(): Record<string, string> {
  const key = getApiKey()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (key !== null) {
    // Send header even for empty string — dev mode backend matches empty ADMIN_API_KEY
    headers['X-Admin-Key'] = key
  }
  return headers
}

function handleAuthError(res: Response): void {
  if (res.status === 401 || res.status === 403) {
    clearApiKey()
    window.location.href = '/login'
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

export interface CreateModelRequest {
  model_class: string  // e.g., "GPT", "Claude", "DeepSeek"
  name: string         // display name
  base_url: string     // API endpoint
  api_key: string      // API key
  model: string        // model ID (e.g., "gpt-4o")
}

export const modelConfigApi = {
  list: async (): Promise<any> => {
    const res = await fetch('/api/admin/config/model/list', { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Failed to fetch models')
    return res.json()
  },
  create: async (data: CreateModelRequest): Promise<any> => {
    const res = await fetch('/api/admin/config/model/create', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(data),
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

export interface ChatStreamCallbacks {
  onToken: (token: string) => void
  onThinking?: (text: string) => void
  onToolCall?: (name: string, args: string) => void
  onToolResult?: (name: string, result: string) => void
  onDone: () => void
  onError: (err: Error) => void
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
    const { onToken, onThinking, onToolCall, onToolResult, onDone, onError } = callbacks
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
        const decoder = new TextDecoder()

        if (!reader) throw new Error('No readable stream on response')

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
                  const tcName = event.data?.name || event.name || ''
                  const tcArgs = event.data?.arguments
                  onToolCall?.(tcName, typeof tcArgs === 'object' ? JSON.stringify(tcArgs, null, 2) : String(tcArgs || ''))
                  break
                }
                case 'tool_result': {
                  const trName = event.data?.name || event.name || ''
                  const trResult = event.data?.result || event.data?.content || content
                  onToolResult?.(trName, typeof trResult === 'object' ? JSON.stringify(trResult, null, 2) : String(trResult))
                }
                  break
                case 'error':
                  onError(new Error(content || 'Unknown error'))
                  return
                case 'done':
                  onDone()
                  return
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
