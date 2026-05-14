// REST API client for the Superagent backend.
// All requests go to /api which the Vite dev proxy forwards to localhost:8888.

import { getApiKey, clearApiKey } from './auth'

const API_BASE = '/api/v1'

function authHeaders(): Record<string, string> {
  const key = getApiKey()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (key) {
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

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
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
}

export const skillsApi = {
  list: async (): Promise<{ skills: SkillInfo[] }> => {
    const res = await fetch(`${API_BASE}/skills`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error('Failed to fetch skills')
    return res.json()
  },
  search: async (query: string): Promise<{ results: SkillInfo[] }> => {
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

export const chatApi = {
  async sendMessage(
    agentId: string,
    sessionId: string,
    message: string,
    onToken: (token: string) => void,
    onDone: () => void,
    onError: (err: Error) => void,
  ): Promise<void> {
    try {
      const res = await fetch(`${API_BASE}/chat/stream`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ agent_id: agentId, session_id: sessionId, message }),
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
          if (line.startsWith('data: ')) {
            const data = line.slice(6)
            if (data === '[DONE]') {
              onDone()
              return
            }
            onToken(data)
          }
        }
      }
      onDone()
    } catch (err) {
      onError(err instanceof Error ? err : new Error(String(err)))
    }
  },
}
