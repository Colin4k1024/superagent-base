import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { agentAdminApi, type AgentDetail } from '../lib/api'
import Header from '../components/Header'
import { Button } from '../components/ui/button'
import { Dialog } from '../components/ui/dialog'

// ---- Agent type badge (mirrors AdminPanel style) ----
function AgentTypeTag({ type }: { type: string }) {
  const base = 'inline-block px-2 py-0.5 rounded text-xs font-medium'
  const t = type?.toLowerCase() ?? ''
  if (t.includes('supervisor') || t.includes('orchestrat'))
    return <span className={`${base} bg-purple-100 text-purple-700`}>{type}</span>
  if (t.includes('chat') || t.includes('model'))
    return <span className={`${base} bg-blue-100 text-blue-700`}>{type}</span>
  if (t.includes('workflow') || t.includes('plan'))
    return <span className={`${base} bg-orange-100 text-orange-700`}>{type}</span>
  if (t.includes('parallel') || t.includes('sequential'))
    return <span className={`${base} bg-teal-100 text-teal-700`}>{type}</span>
  return <span className={`${base} bg-gray-100 text-gray-600`}>{type}</span>
}

// ---- Dropdown menu ----
interface DropdownItem {
  label: string
  onClick: () => void
  danger?: boolean
}

function CardMenu({ items }: { items: DropdownItem[] }) {
  const [open, setOpen] = useState(false)
  return (
    <div className="relative">
      <button
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v) }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
        aria-label="Options"
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path d="M10 6a2 2 0 110-4 2 2 0 010 4zm0 2a2 2 0 100 4 2 2 0 000-4zm0 6a2 2 0 110 4 2 2 0 010-4z" />
        </svg>
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-1 w-36 rounded-md border border-gray-200 bg-white py-1 shadow-lg text-sm">
          {items.map((item) => (
            <button
              key={item.label}
              onClick={() => { setOpen(false); item.onClick() }}
              className={`block w-full px-3 py-1.5 text-left hover:bg-gray-50 transition-colors ${
                item.danger ? 'text-red-600 hover:bg-red-50' : 'text-gray-700'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// ---- Agent card ----
function AgentCard({
  agent,
  onEdit,
  onWorkflow,
  onDuplicate,
  onDelete,
}: {
  agent: AgentDetail
  onEdit: () => void
  onWorkflow?: () => void
  onDuplicate: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const isWorkflow = agent.type?.toLowerCase().includes('workflow')

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-3 hover:shadow-sm transition-shadow">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="font-semibold text-gray-900 text-sm truncate">{agent.name}</h3>
          <div className="mt-1">
            <AgentTypeTag type={agent.type} />
          </div>
        </div>
        <CardMenu
          items={[
            { label: t('agents.edit'), onClick: onEdit },
            ...(isWorkflow && onWorkflow ? [{ label: t('agents.graphEditor'), onClick: onWorkflow }] : []),
            { label: t('agents.duplicate'), onClick: onDuplicate },
            { label: t('agents.delete'), onClick: onDelete, danger: true },
          ]}
        />
      </div>
      {agent.description && (
        <p className="text-xs text-gray-500 leading-relaxed line-clamp-2">{agent.description}</p>
      )}
      <div className="mt-auto flex items-center justify-between">
        <span className={`inline-flex items-center gap-1 text-xs ${
          agent.status === 'active' ? 'text-green-600' : 'text-gray-400'
        }`}>
          <span className={`w-1.5 h-1.5 rounded-full ${
            agent.status === 'active' ? 'bg-green-500' : 'bg-gray-300'
          }`} />
          {agent.status || 'unknown'}
        </span>
        <button
          onClick={onEdit}
          className="text-xs text-blue-600 hover:text-blue-700 font-medium transition-colors"
        >
          Edit →
        </button>
      </div>
    </div>
  )
}

// ---- Main page ----
export default function AgentsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()
  const [deleteTarget, setDeleteTarget] = useState<AgentDetail | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['agents'],
    queryFn: () => agentAdminApi.list(),
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => agentAdminApi.delete(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      toast.success(t('agents.deleted', { name: deleteTarget?.name }))
      setDeleteTarget(null)
    },
    onError: (err: unknown) => {
      toast.error(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    },
  })

  async function handleDuplicate(agent: AgentDetail) {
    try {
      const { yaml } = await agentAdminApi.get(agent.name)
      // Rename in YAML so it doesn't collide
      const newYaml = yaml.replace(
        /^(\s*name:\s*)(.+)$/m,
        `$1${agent.name}-copy`,
      )
      navigate('/agents/new', { state: { yaml: newYaml } })
    } catch (err) {
      toast.error(`Failed to duplicate: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  const agents = data?.agents ?? []

  return (
    <div className="flex flex-col h-full">
      <Header
        title={t('agents.title')}
        actions={
          <Button size="sm" onClick={() => navigate('/agents/new')}>
            + {t('agents.newAgent')}
          </Button>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span className="w-2 h-2 rounded-full bg-gray-400 animate-pulse" />
            {t('common.loading')}
          </div>
        )}

        {error && (
          <div className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
            Failed to load agents: {error instanceof Error ? error.message : String(error)}
          </div>
        )}

        {!isLoading && !error && agents.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
            <div className="text-4xl">🤖</div>
            <p className="text-sm text-gray-500">{t('agents.empty')}</p>
            <Button size="sm" onClick={() => navigate('/agents/new')}>
              + {t('agents.newAgent')}
            </Button>
          </div>
        )}

        {!isLoading && agents.length > 0 && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {agents.map((agent) => (
              <AgentCard
                key={agent.name}
                agent={agent}
                onEdit={() => navigate(`/agents/${agent.name}/edit`)}
                onWorkflow={() => navigate(`/agents/${agent.name}/workflow`)}
                onDuplicate={() => handleDuplicate(agent)}
                onDelete={() => setDeleteTarget(agent)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Delete confirmation dialog */}
      <Dialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title={t('agents.delete')}
      >
        <p className="text-sm text-gray-700 mb-6">
          {t('agents.confirmDelete', { name: deleteTarget?.name })}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={deleteMutation.isPending}
            onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.name)}
          >
            {t('common.delete')}
          </Button>
        </div>
      </Dialog>
    </div>
  )
}
