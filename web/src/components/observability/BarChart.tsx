interface BarItem {
  label: string
  value: number
  color?: string
}

interface BarChartProps {
  data: BarItem[]
  height?: number
  maxBars?: number
}

export default function BarChart({ data, height = 160, maxBars = 8 }: BarChartProps) {
  const items = data.slice(0, maxBars)
  const maxVal = Math.max(...items.map((d) => d.value), 1)
  const barWidth = 100 / (items.length * 2 + 1)

  if (items.length === 0) {
    return (
      <div className="flex items-center justify-center text-gray-400 text-sm" style={{ height }}>
        No data
      </div>
    )
  }

  return (
    <svg width="100%" height={height} className="block">
      {items.map((item, i) => {
        const x = barWidth * (i * 2 + 1)
        const barH = (item.value / maxVal) * (height - 24)
        const y = height - 20 - barH
        return (
          <g key={item.label}>
            <rect
              x={`${x}%`}
              y={y}
              width={`${barWidth}%`}
              height={barH}
              rx={3}
              fill={item.color || '#3b82f6'}
              opacity={0.85}
            />
            <text
              x={`${x + barWidth / 2}%`}
              y={height - 4}
              textAnchor="middle"
              className="fill-gray-500"
              fontSize={10}
            >
              {item.label.length > 8 ? item.label.slice(0, 7) + '..' : item.label}
            </text>
            <text
              x={`${x + barWidth / 2}%`}
              y={y - 4}
              textAnchor="middle"
              className="fill-gray-700"
              fontSize={10}
              fontWeight={500}
            >
              {item.value >= 1000 ? `${(item.value / 1000).toFixed(1)}k` : item.value}
            </text>
          </g>
        )
      })}
    </svg>
  )
}
