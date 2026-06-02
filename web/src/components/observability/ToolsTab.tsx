import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type OverviewData } from '@/lib/monitor-api'
import BarChart from './BarChart'

export default function ToolsTab() {
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

  const tools = data.by_tool || []

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-lg border border-gray-200 p-4">
        <h3 className="text-sm font-medium text-gray-700 mb-4">{t('observability.tools.invocations')}</h3>
        <BarChart
          data={tools.map((tool) => ({ label: tool.tool_name, value: tool.invocations, color: '#f59e0b' }))}
          height={200}
        />
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.tools.name')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.tools.count')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.tools.successRate')}</th>
            </tr>
          </thead>
          <tbody>
            {tools.length === 0 ? (
              <tr><td colSpan={3} className="px-4 py-8 text-center text-gray-400">No tool data</td></tr>
            ) : tools.map((tool) => (
              <tr key={tool.tool_name} className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-xs">{tool.tool_name}</td>
                <td className="px-4 py-2">{tool.invocations}</td>
                <td className="px-4 py-2">
                  <div className="flex items-center gap-2">
                    <div className="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
                      <div
                        className="h-full rounded-full"
                        style={{ width: `${tool.success_rate * 100}%`, background: tool.success_rate > 0.9 ? '#10b981' : '#f59e0b' }}
                      />
                    </div>
                    <span className="text-xs text-gray-600">{(tool.success_rate * 100).toFixed(0)}%</span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
