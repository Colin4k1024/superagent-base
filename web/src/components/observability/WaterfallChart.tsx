import { type TraceObservation } from '@/lib/monitor-api'

interface WaterfallChartProps {
  spans: TraceObservation[]
  traceStart: number
  traceDuration: number
}

function levelColor(level?: string): string {
  switch (level) {
    case 'ERROR': return '#ef4444'
    case 'WARNING': return '#f59e0b'
    case 'GENERATION': return '#8b5cf6'
    case 'SPAN': return '#3b82f6'
    default: return '#6b7280'
  }
}

export default function WaterfallChart({ spans, traceStart, traceDuration }: WaterfallChartProps) {
  if (!spans.length || traceDuration <= 0) {
    return <div className="text-sm text-gray-400 py-4 text-center">No span data</div>
  }

  const rowH = 32
  const labelW = 200
  const barPad = 8
  const chartW = 500
  const totalW = labelW + chartW + 20
  const totalH = spans.length * rowH + 20

  return (
    <div className="overflow-x-auto">
      <svg width={totalW} height={totalH} className="text-xs">
        {/* Time axis */}
        {[0, 0.25, 0.5, 0.75, 1].map((pct) => (
          <g key={pct}>
            <line
              x1={labelW + pct * chartW}
              y1={0}
              x2={labelW + pct * chartW}
              y2={totalH}
              stroke="#e5e7eb"
              strokeDasharray="2,2"
            />
            <text x={labelW + pct * chartW} y={totalH - 2} fill="#9ca3af" fontSize={10} textAnchor="middle">
              {(traceDuration * pct).toFixed(0)}ms
            </text>
          </g>
        ))}

        {spans.map((span, i) => {
          const start = span.startTime ? new Date(span.startTime).getTime() - traceStart : 0
          const end = span.endTime ? new Date(span.endTime).getTime() - traceStart : start + 100
          const duration = end - start
          const x = labelW + (start / traceDuration) * chartW
          const w = Math.max((duration / traceDuration) * chartW, 2)
          const y = i * rowH + barPad

          return (
            <g key={span.id}>
              {/* Label */}
              <text
                x={4}
                y={y + (rowH - barPad * 2) / 2 + 4}
                fill="#374151"
                fontSize={11}
                className="font-mono"
              >
                {span.name.length > 24 ? span.name.slice(0, 24) + '…' : span.name}
              </text>

              {/* Bar */}
              <rect
                x={x}
                y={y}
                width={w}
                height={rowH - barPad * 2}
                rx={3}
                fill={levelColor(span.level)}
                opacity={0.85}
              />

              {/* Duration label */}
              {duration > 0 && (
                <text
                  x={x + w + 4}
                  y={y + (rowH - barPad * 2) / 2 + 4}
                  fill="#6b7280"
                  fontSize={10}
                >
                  {duration.toFixed(0)}ms
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}
