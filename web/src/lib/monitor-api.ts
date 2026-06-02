import { getToken, handleLogin } from './auth'

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

export interface OverviewData {
  total_requests: number
  active_sessions: number
  model_errors: number
  tool_invocations: number
  total_tokens: { input: number; output: number }
  by_agent: { agent_id: string; requests: number; avg_latency_ms: number }[]
  by_model: { model_id: string; tokens: number; errors: number }[]
  by_tool: { tool_name: string; invocations: number; success_rate: number }[]
  route_decisions: { strategy: string; model_id: string; count: number }[]
}

export interface TraceObservation {
  id: string
  name: string
  startTime?: string
  endTime?: string
  level?: string
  model?: string
  input?: unknown
  output?: unknown
  metadata?: Record<string, unknown>
  completionTokens?: number
  promptTokens?: number
  totalTokens?: number
}

export interface TraceItem {
  id: string
  name?: string
  timestamp?: string
  latency?: number
  status?: string
  totalCost?: number
  input?: unknown
  output?: unknown
  metadata?: Record<string, unknown>
  observations?: TraceObservation[]
}

export interface TraceListResponse {
  data: TraceItem[]
  meta: { page: number; limit: number; totalItems: number; totalPages: number }
}

export interface DailyMetric {
  date: string
  countTraces?: number
  countObservations?: number
  totalCost?: number
  usage?: { inputUsage: number; outputUsage: number; totalUsage: number }[]
}

export interface DailyMetricsResponse {
  data: DailyMetric[]
  meta: Record<string, unknown>
}

export interface TraceListParams {
  page?: number
  limit?: number
  orderBy?: string
  name?: string
  fromTimestamp?: string
  toTimestamp?: string
}

export const monitorApi = {
  async getOverview(): Promise<OverviewData> {
    const res = await fetch(`${API_BASE}/admin/monitor/overview`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async listTraces(params: TraceListParams = {}): Promise<TraceListResponse> {
    const qs = new URLSearchParams()
    if (params.page) qs.set('page', String(params.page))
    if (params.limit) qs.set('limit', String(params.limit))
    if (params.orderBy) qs.set('orderBy', params.orderBy)
    if (params.name) qs.set('name', params.name)
    if (params.fromTimestamp) qs.set('fromTimestamp', params.fromTimestamp)
    if (params.toTimestamp) qs.set('toTimestamp', params.toTimestamp)
    const res = await fetch(`${API_BASE}/admin/monitor/traces?${qs}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async getTrace(id: string): Promise<TraceItem> {
    const res = await fetch(`${API_BASE}/admin/monitor/traces/${id}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async getDailyMetrics(from?: string, to?: string): Promise<DailyMetricsResponse> {
    const qs = new URLSearchParams()
    if (from) qs.set('fromTimestamp', from)
    if (to) qs.set('toTimestamp', to)
    const res = await fetch(`${API_BASE}/admin/monitor/metrics/daily?${qs}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },

  async getSessions(page = 1, limit = 20): Promise<unknown> {
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) })
    const res = await fetch(`${API_BASE}/admin/monitor/sessions?${qs}`, { headers: authHeaders() })
    handleAuthError(res)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json()
  },
}
