import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type OverviewData } from '@/lib/monitor-api'
import BarChart from './BarChart'

export default function ModelRoutingTab() {
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

  const models = data.by_model || []
  const decisions = data.route_decisions || []

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-4">{t('observability.models.tokenUsage')}</h3>
          <BarChart
            data={models.map((m) => ({ label: m.model_id, value: m.tokens, color: '#8b5cf6' }))}
            height={200}
          />
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-4">{t('observability.models.routeDecisions')}</h3>
          <BarChart
            data={decisions.map((d) => ({ label: d.strategy, value: d.count, color: '#06b6d4' }))}
            height={200}
          />
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.model')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.tokens')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.errors')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.errorRate')}</th>
            </tr>
          </thead>
          <tbody>
            {models.length === 0 ? (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-400">No model data</td></tr>
            ) : models.map((m) => (
              <tr key={m.model_id} className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-xs">{m.model_id}</td>
                <td className="px-4 py-2">{m.tokens.toLocaleString()}</td>
                <td className="px-4 py-2">{m.errors}</td>
                <td className="px-4 py-2">
                  <span className={`text-xs px-2 py-0.5 rounded ${m.errors > 0 ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                    {m.tokens > 0 ? ((m.errors / m.tokens) * 100).toFixed(2) : '0.00'}%
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {decisions.length > 0 && (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.strategy')}</th>
                <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.models.decisionCount')}</th>
              </tr>
            </thead>
            <tbody>
              {decisions.map((d) => (
                <tr key={d.strategy} className="border-b border-gray-100">
                  <td className="px-4 py-2 font-mono text-xs">{d.strategy}</td>
                  <td className="px-4 py-2">{d.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
