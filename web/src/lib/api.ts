// REST API client for the Superagent backend.
// All requests go to /api which the Vite dev proxy forwards to localhost:8888.

const API_BASE = '/api/v1'

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
    const res = await fetch(`${API_BASE}/agents`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    return data.agents || []
  },
}

export const adminApi = {
  async getStatus(): Promise<AdminStatus> {
    const res = await fetch(`${API_BASE}/admin/status`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async triggerReload(): Promise<ReloadResult> {
    const res = await fetch(`${API_BASE}/admin/reload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
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
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, session_id: sessionId, message }),
      })

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
