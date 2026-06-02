import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type DailyMetric } from '@/lib/monitor-api'
import TimeRangeSelector, { rangeToTimestamp } from './TimeRangeSelector'
import LineChart from './LineChart'
import BarChart from './BarChart'

export default function TokenUsageTab() {
  const { t } = useTranslation()
  const [metrics, setMetrics] = useState<DailyMetric[]>([])
  const [range, setRange] = useState('7d')
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const resp = await monitorApi.getDailyMetrics(rangeToTimestamp(range))
        if (!active) return
        setMetrics(resp.data || [])
        setError('')
      } catch (e) {
        if (active) setError((e as Error).message)
      }
    }
    load()
    return () => { active = false }
  }, [range])

  const labels = metrics.map((m) => m.date?.slice(5, 10) || '')
  const inputTokens = metrics.map((m) => m.usage?.[0]?.inputUsage || 0)
  const outputTokens = metrics.map((m) => m.usage?.[0]?.outputUsage || 0)
  const traceCounts = metrics.map((m) => m.countTraces || 0)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-gray-700">{t('observability.tokens.title')}</h2>
        <TimeRangeSelector value={range} onChange={setRange} />
      </div>

      {error && <div className="text-red-500 text-sm">{error}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-600 mb-3">{t('observability.tokens.trend')}</h3>
          <LineChart
            series={[
              { label: 'Input', data: inputTokens, color: '#8b5cf6' },
              { label: 'Output', data: outputTokens, color: '#3b82f6' },
            ]}
            labels={labels}
          />
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-600 mb-3">{t('observability.tokens.dailyTraces')}</h3>
          <BarChart
            data={metrics.map((m) => ({
              label: m.date?.slice(5, 10) || '',
              value: m.countTraces || 0,
              color: '#10b981',
            }))}
          />
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-4">
        <h3 className="text-sm font-medium text-gray-600 mb-3">{t('observability.tokens.summary')}</h3>
        <div className="grid grid-cols-3 gap-4 text-center">
          <div>
            <div className="text-2xl font-semibold text-gray-900">{inputTokens.reduce((a, b) => a + b, 0).toLocaleString()}</div>
            <div className="text-xs text-gray-500">Total Input Tokens</div>
          </div>
          <div>
            <div className="text-2xl font-semibold text-gray-900">{outputTokens.reduce((a, b) => a + b, 0).toLocaleString()}</div>
            <div className="text-xs text-gray-500">Total Output Tokens</div>
          </div>
          <div>
            <div className="text-2xl font-semibold text-gray-900">{traceCounts.reduce((a, b) => a + b, 0).toLocaleString()}</div>
            <div className="text-xs text-gray-500">Total Traces</div>
          </div>
        </div>
      </div>
    </div>
  )
}
