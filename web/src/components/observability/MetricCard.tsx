import MiniChart from '../monitor/MiniChart'

interface MetricCardProps {
  title: string
  value: string | number
  subtitle?: string
  trend?: number[]
  color?: string
}

export default function MetricCard({ title, value, subtitle, trend, color = '#3b82f6' }: MetricCardProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-2">
      <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">{title}</span>
      <div className="flex items-end justify-between">
        <span className="text-2xl font-semibold text-gray-900">{typeof value === 'number' ? value.toLocaleString() : value}</span>
        {trend && trend.length > 1 && (
          <MiniChart data={trend} width={80} height={32} color={color} />
        )}
      </div>
      {subtitle && <span className="text-xs text-gray-400">{subtitle}</span>}
    </div>
  )
}
