import { useEffect, useState } from 'react'
import { adminApi, AdminStatus } from '../../lib/api'

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.floor(seconds % 60)}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function StatusBadge({ status }: { status: string }) {
  const s = status?.toLowerCase() ?? ''
  const base = 'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium'
  if (s === 'running' || s === 'healthy' || s === 'ok') {
    return <span className={`${base} bg-green-100 text-green-700`}>{status}</span>
  }
  if (s === 'degraded' || s === 'warn') {
    return <span className={`${base} bg-yellow-100 text-yellow-700`}>{status}</span>
  }
  if (s === 'error' || s === 'down' || s === 'unhealthy') {
    return <span className={`${base} bg-red-100 text-red-700`}>{status}</span>
  }
  return <span className={`${base} bg-gray-100 text-gray-600`}>{status}</span>
}

export default function StatusPanel() {
  const [status, setStatus] = useState<AdminStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  async function load() {
    try {
      setError(null)
      const data = await adminApi.getStatus()
      setStatus(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const interval = setInterval(load, 10_000)
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-40 text-sm text-gray-400">
        Loading status…
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-600">
        Failed to load status: {error}
      </div>
    )
  }

  if (!status) return null

  return (
    <div className="space-y-4">
      {/* Overview cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-xs text-gray-500 mb-1">Health</p>
          <StatusBadge status={status.health ?? 'unknown'} />
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-xs text-gray-500 mb-1">Ready</p>
          <span
            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
              status.ready ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
            }`}
          >
            {status.ready ? 'Yes' : 'No'}
          </span>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-xs text-gray-500 mb-1">Uptime</p>
          <p className="text-lg font-bold text-gray-900">{formatUptime(status.uptime_seconds)}</p>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <p className="text-xs text-gray-500 mb-1">Agents</p>
          <p className="text-lg font-bold text-gray-900">{status.agent_count}</p>
        </div>
      </div>

      {/* Time info */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
        <div>
          <span className="text-gray-500">Started: </span>
          <span className="text-gray-900">{formatDate(status.start_time)}</span>
        </div>
        <div>
          <span className="text-gray-500">Last Reload: </span>
          <span className="text-gray-900">{formatDate(status.last_reload_at)}</span>
        </div>
      </div>

      {/* Agent table */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b border-gray-100">
          <h2 className="text-sm font-semibold text-gray-900">Registered Agents</h2>
        </div>
        {status.agents && status.agents.length > 0 ? (
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 text-left text-xs text-gray-500">
                <th className="px-4 py-2 font-medium">Name</th>
                <th className="px-4 py-2 font-medium">Type</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {status.agents.map((agent) => (
                <tr key={agent.name} className="hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2 font-mono font-medium text-gray-900">{agent.name}</td>
                  <td className="px-4 py-2 text-gray-600">{agent.type}</td>
                  <td className="px-4 py-2">
                    <StatusBadge status={agent.status} />
                  </td>
                  <td className="px-4 py-2 text-gray-500 truncate max-w-xs">{agent.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="px-4 py-8 text-center text-sm text-gray-400">No agents registered</div>
        )}
      </div>
    </div>
  )
}
