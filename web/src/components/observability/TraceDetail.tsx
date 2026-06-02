import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { monitorApi, type TraceItem, type TraceObservation } from '@/lib/monitor-api'
import WaterfallChart from './WaterfallChart'

interface TraceDetailProps {
  traceId: string
  onClose: () => void
}

export default function TraceDetail({ traceId, onClose }: TraceDetailProps) {
  const { t } = useTranslation()
  const [trace, setTrace] = useState<TraceItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showRaw, setShowRaw] = useState(false)

  useEffect(() => {
    let active = true
    setLoading(true)
    monitorApi.getTrace(traceId)
      .then((d) => { if (active) { setTrace(d); setError('') } })
      .catch((e) => { if (active) setError((e as Error).message) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [traceId])

  const spans: TraceObservation[] = trace?.observations || []
  const traceStart = trace?.timestamp ? new Date(trace.timestamp).getTime() : 0
  const traceDuration = trace?.latency ? trace.latency / 1000 : 0

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <div
        className="w-full max-w-2xl bg-white h-full overflow-auto shadow-xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">
              {trace?.name || traceId.slice(0, 8)}
            </h2>
            <p className="text-xs text-gray-400 font-mono mt-0.5">{traceId}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-2xl leading-none">&times;</button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6 space-y-6">
          {loading && <div className="text-sm text-gray-400">{t('common.loading')}</div>}
          {error && <div className="text-sm text-red-500">{error}</div>}

          {trace && !loading && (
            <>
              {/* Meta cards */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <MetaItem label={t('observability.traceDetail.status')} value={trace.status || 'OK'} badge={trace.status === 'ERROR' ? 'red' : 'green'} />
                <MetaItem label={t('observability.traceDetail.latency')} value={traceDuration > 0 ? `${traceDuration.toFixed(0)}ms` : '-'} />
                <MetaItem label={t('observability.traceDetail.spans')} value={String(spans.length)} />
                <MetaItem label={t('observability.traceDetail.cost')} value={trace.totalCost != null ? `$${trace.totalCost.toFixed(4)}` : '-'} />
              </div>

              {/* Timestamp */}
              {trace.timestamp && (
                <div className="text-xs text-gray-500">
                  {t('observability.traceDetail.startedAt')}: {new Date(trace.timestamp).toLocaleString()}
                </div>
              )}

              {/* Waterfall */}
              {spans.length > 0 && (
                <section>
                  <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.traceDetail.waterfall')}</h3>
                  <div className="border border-gray-200 rounded-lg p-3 bg-gray-50">
                    <WaterfallChart spans={spans} traceStart={traceStart} traceDuration={traceDuration} />
                  </div>
                </section>
              )}

              {/* Span details table */}
              {spans.length > 0 && (
                <section>
                  <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.traceDetail.spanList')}</h3>
                  <div className="border border-gray-200 rounded-lg overflow-hidden">
                    <table className="w-full text-xs">
                      <thead className="bg-gray-50 border-b border-gray-200">
                        <tr>
                          <th className="text-left px-3 py-2 font-medium text-gray-600">Name</th>
                          <th className="text-left px-3 py-2 font-medium text-gray-600">Level</th>
                          <th className="text-left px-3 py-2 font-medium text-gray-600">Model</th>
                          <th className="text-left px-3 py-2 font-medium text-gray-600">Duration</th>
                        </tr>
                      </thead>
                      <tbody>
                        {spans.map((span) => {
                          const start = span.startTime ? new Date(span.startTime).getTime() : 0
                          const end = span.endTime ? new Date(span.endTime).getTime() : start
                          const dur = end - start
                          return (
                            <tr key={span.id} className="border-b border-gray-100">
                              <td className="px-3 py-2 font-mono">{span.name}</td>
                              <td className="px-3 py-2">
                                <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
                                  span.level === 'ERROR' ? 'bg-red-100 text-red-700'
                                    : span.level === 'GENERATION' ? 'bg-purple-100 text-purple-700'
                                    : 'bg-gray-100 text-gray-600'
                                }`}>
                                  {span.level || 'SPAN'}
                                </span>
                              </td>
                              <td className="px-3 py-2 text-gray-500">{span.model || '-'}</td>
                              <td className="px-3 py-2 text-gray-500">{dur > 0 ? `${dur}ms` : '-'}</td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                </section>
              )}

              {/* Input / Output */}
              {(trace.input || trace.output) && (
                <section>
                  <h3 className="text-sm font-medium text-gray-700 mb-3">{t('observability.traceDetail.io')}</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {trace.input != null ? (
                      <div>
                        <span className="text-xs font-medium text-gray-500 mb-1 block">Input</span>
                        <pre className="text-[11px] bg-gray-50 border border-gray-200 rounded p-2 overflow-auto max-h-40">
                          {typeof trace.input === 'string' ? trace.input : JSON.stringify(trace.input, null, 2)}
                        </pre>
                      </div>
                    ) : null}
                    {trace.output != null ? (
                      <div>
                        <span className="text-xs font-medium text-gray-500 mb-1 block">Output</span>
                        <pre className="text-[11px] bg-gray-50 border border-gray-200 rounded p-2 overflow-auto max-h-40">
                          {typeof trace.output === 'string' ? trace.output : JSON.stringify(trace.output, null, 2)}
                        </pre>
                      </div>
                    ) : null}
                  </div>
                </section>
              )}

              {/* Raw JSON toggle */}
              <section>
                <button
                  onClick={() => setShowRaw((v) => !v)}
                  className="text-xs text-blue-600 hover:text-blue-800 font-medium"
                >
                  {showRaw ? t('observability.traceDetail.hideRaw') : t('observability.traceDetail.showRaw')}
                </button>
                {showRaw && (
                  <pre className="mt-2 text-[11px] bg-gray-50 border border-gray-200 rounded p-3 overflow-auto max-h-80">
                    {JSON.stringify(trace, null, 2)}
                  </pre>
                )}
              </section>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function MetaItem({ label, value, badge }: { label: string; value: string; badge?: 'green' | 'red' }) {
  return (
    <div className="bg-gray-50 rounded-lg border border-gray-200 p-3">
      <div className="text-[10px] font-medium text-gray-500 uppercase tracking-wide">{label}</div>
      <div className="mt-1">
        {badge ? (
          <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
            badge === 'red' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'
          }`}>
            {value}
          </span>
        ) : (
          <span className="text-sm font-semibold text-gray-900">{value}</span>
        )}
      </div>
    </div>
  )
}
