interface Series {
  label: string
  data: number[]
  color: string
}

interface LineChartProps {
  series: Series[]
  labels?: string[]
  height?: number
}

export default function LineChart({ series, labels, height = 180 }: LineChartProps) {
  if (series.length === 0 || series[0].data.length === 0) {
    return (
      <div className="flex items-center justify-center text-gray-400 text-sm" style={{ height }}>
        No data
      </div>
    )
  }

  const padX = 40
  const padY = 20
  const svgW = 600
  const svgH = height
  const plotW = svgW - padX * 2
  const plotH = svgH - padY * 2

  const allVals = series.flatMap((s) => s.data)
  const minV = Math.min(...allVals)
  const maxV = Math.max(...allVals)
  const range = maxV - minV || 1

  const pointCount = series[0].data.length

  function toX(i: number) {
    return padX + (i / Math.max(pointCount - 1, 1)) * plotW
  }
  function toY(v: number) {
    return padY + plotH - ((v - minV) / range) * plotH
  }

  return (
    <div>
      <svg viewBox={`0 0 ${svgW} ${svgH}`} className="w-full" style={{ height }}>
        {/* Y-axis grid */}
        {[0, 0.25, 0.5, 0.75, 1].map((pct) => {
          const y = padY + plotH * (1 - pct)
          const val = minV + range * pct
          return (
            <g key={pct}>
              <line x1={padX} y1={y} x2={svgW - padX} y2={y} stroke="#e5e7eb" strokeWidth={1} />
              <text x={padX - 6} y={y + 3} textAnchor="end" fontSize={9} className="fill-gray-400">
                {val >= 1000 ? `${(val / 1000).toFixed(0)}k` : val.toFixed(0)}
              </text>
            </g>
          )
        })}

        {/* Lines */}
        {series.map((s) => {
          const points = s.data.map((v, i) => `${toX(i).toFixed(1)},${toY(v).toFixed(1)}`).join(' ')
          return (
            <polyline
              key={s.label}
              points={points}
              fill="none"
              stroke={s.color}
              strokeWidth={2}
              strokeLinejoin="round"
            />
          )
        })}

        {/* X-axis labels */}
        {labels && labels.map((l, i) => (
          <text key={i} x={toX(i)} y={svgH - 2} textAnchor="middle" fontSize={9} className="fill-gray-400">
            {l}
          </text>
        ))}
      </svg>

      {/* Legend */}
      {series.length > 1 && (
        <div className="flex gap-4 mt-1 px-2">
          {series.map((s) => (
            <div key={s.label} className="flex items-center gap-1 text-xs text-gray-600">
              <span className="w-3 h-[2px] inline-block rounded" style={{ background: s.color }} />
              {s.label}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
