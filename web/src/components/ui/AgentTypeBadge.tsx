import type { LucideIcon } from 'lucide-react'
import { Bot, Workflow, GitBranch, Zap, Users, ArrowRightLeft } from 'lucide-react'

const agentTypeConfig: Record<string, { label: string; color: string; bg: string; icon: LucideIcon }> = {
  chat_model_agent: { label: 'Chat', color: 'text-blue-700', bg: 'bg-blue-50 border-blue-200', icon: Bot },
  agentloop:        { label: 'AgentLoop', color: 'text-violet-700', bg: 'bg-violet-50 border-violet-200', icon: Zap },
  supervisor:       { label: 'Supervisor', color: 'text-cyan-700', bg: 'bg-cyan-50 border-cyan-200', icon: Users },
  workflow:         { label: 'Workflow', color: 'text-amber-700', bg: 'bg-amber-50 border-amber-200', icon: Workflow },
  parallel:         { label: 'Parallel', color: 'text-emerald-700', bg: 'bg-emerald-50 border-emerald-200', icon: GitBranch },
  plan_execute:     { label: 'Plan-Execute', color: 'text-pink-700', bg: 'bg-pink-50 border-pink-200', icon: ArrowRightLeft },
  deep_agent:       { label: 'Deep', color: 'text-indigo-700', bg: 'bg-indigo-50 border-indigo-200', icon: Bot },
}

export function AgentTypeBadge({ type, showIcon = true }: { type: string; showIcon?: boolean }) {
  const config = agentTypeConfig[type] || { label: type, color: 'text-gray-700', bg: 'bg-gray-50 border-gray-200', icon: Bot }
  const Icon = config.icon

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full border ${config.color} ${config.bg}`}>
      {showIcon && <Icon className="w-3 h-3" />}
      {config.label}
    </span>
  )
}

export function getStatusColor(status: string): string {
  switch (status) {
    case 'loaded':
    case 'active':
    case 'success':
    case 'done':
      return 'bg-emerald-500'
    case 'streaming':
    case 'running':
      return 'bg-blue-500 animate-pulse'
    case 'error':
    case 'failed':
      return 'bg-red-500'
    case 'idle':
    case 'stopped':
      return 'bg-gray-400'
    default:
      return 'bg-gray-400'
  }
}
