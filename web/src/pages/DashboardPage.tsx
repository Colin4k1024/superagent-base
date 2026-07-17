import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { adminApi, agentAdminApi, type AgentDetail, type AdminStatus } from '@/lib/api'
import { AgentTypeBadge } from '@/components/ui/AgentTypeBadge'
import { CardGridSkeleton } from '@/components/ui/Skeleton'
import {
  Bot, MessageSquare, Zap, Activity, ArrowRight,
  CheckCircle2, AlertCircle, Clock, Server,
} from 'lucide-react'

export default function DashboardPage() {
  const { t } = useTranslation()
  const [agents, setAgents] = useState<AgentDetail[]>([])
  const [status, setStatus] = useState<AdminStatus | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      agentAdminApi.list().catch(() => ({ agents: [] })),
      adminApi.getStatus().catch(() => null),
    ]).then(([agentData, statusData]) => {
      setAgents(agentData.agents)
      setStatus(statusData)
      setLoading(false)
    })
  }, [])

  const agentTypes = agents.reduce((acc, a) => {
    acc[a.type] = (acc[a.type] || 0) + 1
    return acc
  }, {} as Record<string, number>)

  const loopAgents = agents.filter((a) => a.type === 'agentloop')

  if (loading) {
    return (
      <div className="p-6 max-w-[1400px] mx-auto space-y-6">
        <div className="space-y-2">
          <div className="skeleton h-8 w-48" />
          <div className="skeleton h-4 w-96" />
        </div>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => <div key={i} className="skeleton h-28 rounded-card" />)}
        </div>
        <CardGridSkeleton count={6} />
      </div>
    )
  }

  return (
    <div className="p-6 max-w-[1400px] mx-auto space-y-6 animate-fade-in">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">{t('dashboard.title', 'Dashboard')}</h1>
        <p className="text-sm text-gray-500 mt-1">{t('dashboard.subtitle', 'Overview of your AI Agent platform')}</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KPICard
          icon={Bot}
          label={t('dashboard.agents', 'Agents')}
          value={agents.length}
          detail={Object.entries(agentTypes).map(([k, v]) => `${k}: ${v}`).join(' / ')}
          color="blue"
        />
        <KPICard
          icon={Zap}
          label={t('dashboard.agentloop', 'AgentLoop')}
          value={loopAgents.length}
          detail={t('dashboard.agentloopDetail', 'Autonomous multi-turn agents')}
          color="violet"
        />
        <KPICard
          icon={Activity}
          label={t('dashboard.health', 'Health')}
          value={status?.health === 'ok' ? 'OK' : 'Error'}
          detail={status?.ready ? t('dashboard.ready', 'All systems ready') : t('dashboard.notReady', 'Not ready')}
          color={status?.health === 'ok' ? 'green' : 'red'}
        />
        <KPICard
          icon={Server}
          label={t('dashboard.uptime', 'Uptime')}
          value={formatUptime(status?.uptime_seconds || 0)}
          detail={`${agents.length} ${t('dashboard.agentsLoaded', 'agents loaded')}`}
          color="cyan"
        />
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* AgentLoop Quick Start */}
        <div className="lg:col-span-2 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">{t('dashboard.loopAgents', 'AgentLoop Agents')}</h2>
            <Link
              to="/agents"
              className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
            >
              {t('dashboard.viewAll', 'View all')} <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          {loopAgents.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {loopAgents.map((agent) => (
                <Link
                  key={agent.name}
                  to={`/agentloop-demo?agent=${agent.name}`}
                  className="card p-4 hover:shadow-card-hover hover:-translate-y-0.5 transition-all duration-200 group"
                >
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 rounded-lg bg-violet-50 flex items-center justify-center">
                        <Zap className="w-4 h-4 text-violet-600" />
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-gray-900">{agent.name}</h3>
                        <AgentTypeBadge type={agent.type} showIcon={false} />
                      </div>
                    </div>
                    <ArrowRight className="w-4 h-4 text-gray-300 group-hover:text-blue-500 transition-colors" />
                  </div>
                  <p className="text-xs text-gray-500 line-clamp-2">{agent.description || 'No description'}</p>
                </Link>
              ))}
            </div>
          ) : (
            <div className="card p-8 text-center">
              <Zap className="w-10 h-10 text-gray-300 mx-auto mb-3" />
              <p className="text-sm text-gray-500">{t('dashboard.noLoopAgents', 'No AgentLoop agents configured')}</p>
            </div>
          )}
        </div>

        {/* System Status */}
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">{t('dashboard.systemStatus', 'System Status')}</h2>
          <div className="card p-4 space-y-3">
            <StatusRow label="Backend" status={status ? 'ok' : 'error'} />
            <StatusRow label="MySQL" status="ok" />
            <StatusRow label="Redis" status="ok" />
            <StatusRow
              label={t('dashboard.agentCount', 'Agents')}
              status="ok"
              detail={`${agents.length} loaded`}
            />
            {status?.start_time && (
              <div className="pt-2 border-t border-gray-100">
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Clock className="w-3.5 h-3.5" />
                  <span>{t('dashboard.startedAt', 'Started')}: {new Date(status.start_time).toLocaleString()}</span>
                </div>
              </div>
            )}
          </div>

          {/* Agent Type Distribution */}
          <div className="card p-4">
            <h3 className="text-sm font-medium text-gray-700 mb-3">{t('dashboard.typeDistribution', 'Agent Types')}</h3>
            <div className="space-y-2">
              {Object.entries(agentTypes)
                .sort(([, a], [, b]) => b - a)
                .map(([type, count]) => (
                  <div key={type} className="flex items-center gap-2">
                    <AgentTypeBadge type={type} />
                    <div className="flex-1 h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-blue-500 rounded-full transition-all duration-500"
                        style={{ width: `${(count / agents.length) * 100}%` }}
                      />
                    </div>
                    <span className="text-xs text-gray-500 w-5 text-right">{count}</span>
                  </div>
                ))}
            </div>
          </div>
        </div>
      </div>

      {/* All Agents Grid */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">{t('dashboard.allAgents', 'All Agents')}</h2>
          <Link
            to="/agents/new"
            className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
          >
            {t('dashboard.createAgent', 'Create agent')} <ArrowRight className="w-3.5 h-3.5" />
          </Link>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {agents.map((agent) => (
            <Link
              key={agent.name}
              to={`/agents/${agent.name}/edit`}
              className="card p-3 hover:shadow-card-hover transition-all duration-200 group"
            >
              <div className="flex items-center gap-2 mb-1.5">
                <div className="w-6 h-6 rounded bg-gray-100 flex items-center justify-center">
                  <Bot className="w-3.5 h-3.5 text-gray-500" />
                </div>
                <span className="text-sm font-medium text-gray-900 truncate">{agent.name}</span>
              </div>
              <AgentTypeBadge type={agent.type} showIcon={false} />
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}

// --- Sub-components ---

function KPICard({ icon: Icon, label, value, detail, color }: {
  icon: typeof Bot
  label: string
  value: string | number
  detail: string
  color: string
}) {
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-600',
    violet: 'bg-violet-50 text-violet-600',
    green: 'bg-emerald-50 text-emerald-600',
    red: 'bg-red-50 text-red-600',
    cyan: 'bg-cyan-50 text-cyan-600',
  }

  return (
    <div className="card p-4">
      <div className="flex items-center gap-3 mb-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colorMap[color] || colorMap.blue}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
      <p className="text-[11px] text-gray-400 truncate">{detail}</p>
    </div>
  )
}

function StatusRow({ label, status, detail }: { label: string; status: string; detail?: string }) {
  const isOk = status === 'ok'
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        {isOk ? (
          <CheckCircle2 className="w-4 h-4 text-emerald-500" />
        ) : (
          <AlertCircle className="w-4 h-4 text-red-500" />
        )}
        <span className="text-sm text-gray-700">{label}</span>
      </div>
      {detail && <span className="text-xs text-gray-400">{detail}</span>}
    </div>
  )
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
  return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`
}
