import Header from '../components/Header'

const placeholderMetrics = [
  { name: 'Active Agents', value: '3', trend: '+1 today' },
  { name: 'Total Requests', value: '1,284', trend: '+42 last hour' },
  { name: 'Avg Latency', value: '312ms', trend: '-18ms vs yesterday' },
  { name: 'Error Rate', value: '0.4%', trend: 'stable' },
]

export default function MonitorPage() {
  return (
    <div className="flex flex-col h-full">
      <Header title="Monitor" />

      <div className="flex-1 overflow-auto p-6 space-y-6">
        <p className="text-sm text-gray-500">
          Real-time metrics and traces. Data will stream from the backend Prometheus/OpenTelemetry endpoints.
        </p>

        {/* KPI cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {placeholderMetrics.map((m) => (
            <div key={m.name} className="bg-white rounded-lg border border-gray-200 p-4">
              <p className="text-xs text-gray-500 mb-1">{m.name}</p>
              <p className="text-2xl font-bold text-gray-900">{m.value}</p>
              <p className="text-xs text-gray-400 mt-1">{m.trend}</p>
            </div>
          ))}
        </div>

        {/* Chart placeholder */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <p className="text-sm font-medium text-gray-700 mb-3">Request Rate (last 1h)</p>
          <div className="h-40 bg-gray-50 rounded flex items-center justify-center text-sm text-gray-400">
            Chart will render here (connect to /metrics endpoint)
          </div>
        </div>

        {/* Trace list placeholder */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <p className="text-sm font-medium text-gray-700 mb-3">Recent Traces</p>
          <div className="space-y-2">
            {['agent.run', 'tool.invoke', 'model.completion'].map((span) => (
              <div
                key={span}
                className="flex items-center justify-between text-xs text-gray-600 bg-gray-50 px-3 py-2 rounded"
              >
                <span className="font-mono">{span}</span>
                <span className="text-gray-400">–</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
