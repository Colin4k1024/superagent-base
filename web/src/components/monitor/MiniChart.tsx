interface MiniChartProps {
  data: number[]
  width?: number
  height?: number
  color?: string
  /** Fill the area under the line with a gradient */
  fill?: boolean
}

/**
 * Pure SVG sparkline / mini-chart. No external dependencies.
 * Renders a polyline connecting the data points with an optional gradient fill.
 */
export default function MiniChart({
  data,
  width = 200,
  height = 48,
  color = '#3b82f6',
  fill = true,
}: MiniChartProps) {
  if (data.length < 2) {
    return (
      <svg width={width} height={height} className="block">
        <line
          x1={0}
          y1={height / 2}
          x2={width}
          y2={height / 2}
          stroke={color}
          strokeWidth={1.5}
          strokeOpacity={0.3}
          strokeDasharray="4 4"
        />
      </svg>
    )
  }

  const padV = 4
  const effectiveHeight = height - padV * 2
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1

  // Map data points to SVG coordinates
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * width
    const y = padV + effectiveHeight - ((v - min) / range) * effectiveHeight
    return [x, y] as [number, number]
  })

  const polylinePoints = points.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')

  // Area fill path: line points + close back along the bottom
  const fillPath =
    `M ${points[0][0].toFixed(1)},${points[0][1].toFixed(1)} ` +
    points
      .slice(1)
      .map(([x, y]) => `L ${x.toFixed(1)},${y.toFixed(1)}`)
      .join(' ') +
    ` L ${points[points.length - 1][0].toFixed(1)},${(height - padV).toFixed(1)}` +
    ` L ${points[0][0].toFixed(1)},${(height - padV).toFixed(1)} Z`

  const gradientId = `mini-chart-grad-${color.replace('#', '')}`

  return (
    <svg width={width} height={height} className="block overflow-visible">
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.25} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>

      {fill && <path d={fillPath} fill={`url(#${gradientId})`} />}

      <polyline
        points={polylinePoints}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* Dot on the last point */}
      <circle
        cx={points[points.length - 1][0]}
        cy={points[points.length - 1][1]}
        r={2.5}
        fill={color}
      />
    </svg>
  )
}
