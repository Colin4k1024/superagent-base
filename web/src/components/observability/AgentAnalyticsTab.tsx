import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type OverviewData } from '@/lib/monitor-api'
import BarChart from './BarChart'

export default function AgentAnalyticsTab() {
  const { t } = useTranslation()
  const [data, setData] = useState<OverviewData | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    monitorApi.getOverview()
      .then((d) => { if (active) setData(d) })
      .catch((e) => { if (active) setError((e as Error).message) })
    return () => { active = false }
  }, [])

  if (error) return <div className="text-red-500 text-sm">{error}</div>
  if (!data) return <div className="text-gray-400 text-sm">{t('common.loading')}</div>

  const agents = data.by_agent || []

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-4">{t('observability.agents.requests')}</h3>
          <BarChart
            data={agents.map((a) => ({ label: a.agent_id, value: a.requests }))}
            height={200}
          />
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-4">{t('observability.agents.latency')}</h3>
          <BarChart
            data={agents.map((a) => ({ label: a.agent_id, value: Math.round(a.avg_latency_ms), color: '#6366f1' }))}
            height={200}
          />
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-2 font-medium text-gray-600">Agent</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.agents.requestCount')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.agents.avgLatency')}</th>
            </tr>
          </thead>
          <tbody>
            {agents.length === 0 ? (
              <tr><td colSpan={3} className="px-4 py-8 text-center text-gray-400">No agent data</td></tr>
            ) : agents.map((a) => (
              <tr key={a.agent_id} className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-xs">{a.agent_id}</td>
                <td className="px-4 py-2">{a.requests}</td>
                <td className="px-4 py-2">{a.avg_latency_ms > 0 ? `${a.avg_latency_ms.toFixed(0)}ms` : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
