import { useEffect, useRef, useState } from 'react'
import { adminApi, AdminStatus } from '../../lib/api'
import { metricsApi } from '../../lib/api'
import { parsePrometheusText, getMetricValue } from '../../lib/prometheusParser'
import MiniChart from './MiniChart'

interface KpiCard {
  label: string
  value: string
  sub: string
  history: number[]
  color: string
}

const HISTORY_MAX = 20
const POLL_INTERVAL_MS = 10_000

function buildKpiCards(
  status: AdminStatus | null,
  activeSessions: number | null,
  agentCountHistory: number[],
  sessionHistory: number[],
): KpiCard[] {
  return [
    {
      label: 'Active Agents',
      value: status ? String(status.agent_count) : '—',
      sub: status?.health ? `health: ${status.health}` : '',
      history: agentCountHistory,
      color: '#3b82f6',
    },
    {
      label: 'Active Sessions',
      value: activeSessions !== null ? String(activeSessions) : '—',
      sub: 'superagent_active_sessions',
      history: sessionHistory,
      color: '#10b981',
    },
    {
      label: 'Uptime',
      value: status ? formatUptime(status.uptime_seconds) : '—',
      sub: status?.ready ? 'ready' : status ? 'not ready' : '',
      history: [],
      color: '#8b5cf6',
    },
    {
      label: 'Agents Ready',
      value:
        status && status.agents
          ? `${status.agents.filter((a) => ['running', 'ok'].includes(a.status?.toLowerCase() ?? '')).length} / ${status.agent_count}`
          : '—',
      sub: 'running agents',
      history: [],
      color: '#f59e0b',
    },
  ]
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

export default function MetricsPanel() {
  const [status, setStatus] = useState<AdminStatus | null>(null)
  const [activeSessions, setActiveSessions] = useState<number | null>(null)
  const [agentCountHistory, setAgentCountHistory] = useState<number[]>([])
  const [sessionHistory, setSessionHistory] = useState<number[]>([])
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const mountedRef = useRef(true)

  async function poll() {
    try {
      const [statusData, metricsRaw] = await Promise.all([
        adminApi.getStatus().catch(() => null),
        metricsApi.getRaw().catch(() => null),
      ])

      if (!mountedRef.current) return

      setError(null)

      if (statusData) {
        setStatus(statusData)
        setAgentCountHistory((prev) => {
          const next = [...prev, statusData.agent_count]
          return next.length > HISTORY_MAX ? next.slice(next.length - HISTORY_MAX) : next
        })
      }

      if (metricsRaw) {
        const parsed = parsePrometheusText(metricsRaw)
        const sessions = getMetricValue(parsed, 'superagent_active_sessions')
        setActiveSessions(sessions)
        setSessionHistory((prev) => {
          const val = sessions ?? 0
          const next = [...prev, val]
          return next.length > HISTORY_MAX ? next.slice(next.length - HISTORY_MAX) : next
        })
      }

      setLastUpdated(new Date())
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : String(err))
      }
    }
  }

  useEffect(() => {
    mountedRef.current = true
    poll()
    const interval = setInterval(poll, POLL_INTERVAL_MS)
    return () => {
      mountedRef.current = false
      clearInterval(interval)
    }
  }, [])

  const cards = buildKpiCards(status, activeSessions, agentCountHistory, sessionHistory)

  return (
    <div className="space-y-4">
      {/* Header row */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-700">System Metrics</h2>
        {lastUpdated && (
          <span className="text-xs text-gray-400">
            Updated {lastUpdated.toLocaleTimeString()}
          </span>
        )}
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-sm text-red-600">
          Metrics poll error: {error}
        </div>
      )}

      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map((card) => (
          <div key={card.label} className="bg-white rounded-lg border border-gray-200 p-4">
            <p className="text-xs text-gray-500 mb-1">{card.label}</p>
            <p className="text-2xl font-bold text-gray-900 mb-1">{card.value}</p>
            {card.sub && <p className="text-xs text-gray-400 mb-3">{card.sub}</p>}
            {card.history.length >= 2 && (
              <MiniChart
                data={card.history}
                width={160}
                height={40}
                color={card.color}
              />
            )}
          </div>
        ))}
      </div>

      {/* Agent count sparkline (full width) */}
      {agentCountHistory.length >= 2 && (
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-sm font-medium text-gray-700 mb-3">Agent Count over time</p>
          <MiniChart data={agentCountHistory} width={600} height={60} color="#3b82f6" />
          <p className="text-xs text-gray-400 mt-2">
            Sampled every {POLL_INTERVAL_MS / 1000}s — last {agentCountHistory.length} readings
          </p>
        </div>
      )}

      {/* Session history sparkline */}
      {sessionHistory.length >= 2 && (
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-sm font-medium text-gray-700 mb-3">Active Sessions over time</p>
          <MiniChart data={sessionHistory} width={600} height={60} color="#10b981" />
          <p className="text-xs text-gray-400 mt-2">
            From <code className="font-mono">superagent_active_sessions</code> gauge — last{' '}
            {sessionHistory.length} readings
          </p>
        </div>
      )}

      {/* Raw prometheus metrics (expandable) */}
      <PrometheusRawSection />
    </div>
  )
}

function PrometheusRawSection() {
  const [open, setOpen] = useState(false)
  const [raw, setRaw] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function fetchRaw() {
    setLoading(true)
    setError(null)
    try {
      const text = await metricsApi.getRaw()
      setRaw(text)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  function toggle() {
    if (!open && raw === null) fetchRaw()
    setOpen((v) => !v)
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <button
        onClick={toggle}
        className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
      >
        <span>Raw Prometheus Metrics</span>
        <span className="text-gray-400 text-xs">{open ? '▲ hide' : '▼ show'}</span>
      </button>

      {open && (
        <div className="border-t border-gray-100">
          {loading && (
            <p className="px-4 py-4 text-sm text-gray-400">Loading…</p>
          )}
          {error && (
            <p className="px-4 py-4 text-sm text-red-600">Error: {error}</p>
          )}
          {raw && (
            <pre className="p-4 text-xs font-mono text-gray-700 bg-gray-50 overflow-auto max-h-80 whitespace-pre-wrap">
              {raw}
            </pre>
          )}
          <div className="px-4 py-2 border-t border-gray-100 flex gap-2">
            <button
              onClick={fetchRaw}
              className="text-xs text-blue-600 hover:underline"
            >
              Refresh
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
