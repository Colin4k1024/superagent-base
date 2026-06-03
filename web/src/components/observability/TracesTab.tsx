import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type TraceItem, type TraceListResponse } from '@/lib/monitor-api'
import TimeRangeSelector, { rangeToTimestamp } from './TimeRangeSelector'
import TraceDetail from './TraceDetail'

export default function TracesTab() {
  const { t } = useTranslation()
  const [traces, setTraces] = useState<TraceItem[]>([])
  const [meta, setMeta] = useState<TraceListResponse['meta'] | null>(null)
  const [page, setPage] = useState(1)
  const [range, setRange] = useState('24h')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    const load = async () => {
      setLoading(true)
      try {
        const resp = await monitorApi.listTraces({
          page,
          limit: 20,
          fromTimestamp: rangeToTimestamp(range),
        })
        if (!active) return
        setTraces(resp.data || [])
        setMeta(resp.meta)
        setError('')
      } catch (e) {
        if (active) setError((e as Error).message)
      } finally {
        if (active) setLoading(false)
      }
    }
    load()
    return () => { active = false }
  }, [page, range])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <TimeRangeSelector value={range} onChange={(v) => { setRange(v); setPage(1) }} />
        <span className="text-xs text-gray-400">
          {meta ? `${meta.totalItems} traces` : ''}
        </span>
      </div>

      {error && <div className="text-red-500 text-sm">{error}</div>}

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.traces.name')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.traces.time')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.traces.latency')}</th>
              <th className="text-left px-4 py-2 font-medium text-gray-600">{t('observability.traces.status')}</th>
            </tr>
          </thead>
          <tbody>
            {loading && traces.length === 0 ? (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-400">{t('common.loading')}</td></tr>
            ) : traces.length === 0 ? (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-400">No traces</td></tr>
            ) : (
              traces.map((trace) => (
                <tr
                  key={trace.id}
                  onClick={() => setSelectedId(trace.id)}
                  className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer"
                >
                  <td className="px-4 py-2 font-mono text-xs">{trace.name || trace.id.slice(0, 8)}</td>
                  <td className="px-4 py-2 text-gray-500">{trace.timestamp ? new Date(trace.timestamp).toLocaleString() : '-'}</td>
                  <td className="px-4 py-2">{trace.latency != null ? `${trace.latency.toFixed(2)}s` : '-'}</td>
                  <td className="px-4 py-2">
                    <span className={`inline-block px-2 py-0.5 rounded text-xs ${trace.status === 'ERROR' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                      {trace.status || 'OK'}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {meta && meta.totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="px-3 py-1 text-sm border border-gray-200 rounded disabled:opacity-40"
          >
            Prev
          </button>
          <span className="text-sm text-gray-600">{page} / {meta.totalPages}</span>
          <button
            disabled={page >= meta.totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="px-3 py-1 text-sm border border-gray-200 rounded disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}

      {/* Trace detail drawer */}
      {selectedId && (
        <TraceDetail traceId={selectedId} onClose={() => setSelectedId(null)} />
      )}
    </div>
  )
}
