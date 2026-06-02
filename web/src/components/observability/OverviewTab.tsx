import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type OverviewData } from '@/lib/monitor-api'
import MetricCard from './MetricCard'
import BarChart from './BarChart'

export default function OverviewTab() {
  const { t } = useTranslation()
  const [data, setData] = useState<OverviewData | null>(null)
  const [error, setError] = useState('')
  const [history, setHistory] = useState<number[]>([])

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const d = await monitorApi.getOverview()
        if (!active) return
        setData(d)
        setHistory((prev) => [...prev.slice(-19), d.total_requests])
        setError('')
      } catch (e) {
        if (active) setError((e as Error).message)
      }
    }
    load()
    const timer = setInterval(load, 10000)
    return () => { active = false; clearInterval(timer) }
  }, [])

  if (error) return <div className="text-red-500 text-sm">{error}</div>
  if (!data) return <div className="text-gray-400 text-sm">{t('common.loading')}</div>

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <MetricCard title={t('observability.metrics.requests')} value={data.total_requests} trend={history} />
        <MetricCard title={t('observability.metrics.sessions')} value={data.active_sessions} color="#10b981" />
        <MetricCard title={t('observability.metrics.inputTokens')} value={data.total_tokens.input} color="#8b5cf6" />
        <MetricCard title={t('observability.metrics.outputTokens')} value={data.total_tokens.output} color="#6366f1" />
        <MetricCard title={t('observability.metrics.toolCalls')} value={data.tool_invocations} color="#f59e0b" />
        <MetricCard title={t('observability.metrics.errors')} value={data.model_errors} color="#ef4444" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.overview.topAgents')}</h3>
          <BarChart
            data={(data.by_agent || []).map((a) => ({ label: a.agent_id, value: a.requests }))}
          />
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.overview.topModels')}</h3>
          <BarChart
            data={(data.by_model || []).map((m) => ({ label: m.model_id, value: m.tokens, color: '#8b5cf6' }))}
          />
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.overview.topTools')}</h3>
          <BarChart
            data={(data.by_tool || []).map((tool) => ({ label: tool.tool_name, value: tool.invocations, color: '#f59e0b' }))}
          />
        </div>
      </div>
    </div>
  )
}
