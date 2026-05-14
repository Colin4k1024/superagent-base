import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import Header from '../components/Header'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Dialog } from '../components/ui/dialog'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'
import { modelConfigApi, CreateModelRequest } from '../lib/api'

// ─── Types ────────────────────────────────────────────────────────────────────

interface ModelRecord {
  id: number
  name: string
  model: string
  model_class: string
  base_url: string
  api_key?: string
  status?: string
}

// ─── Form schema ──────────────────────────────────────────────────────────────

const addModelSchema = z.object({
  model_class: z.string().min(1, 'Provider is required'),
  name: z.string().min(1, 'Model name is required'),
  model: z.string().min(1, 'Model ID is required'),
  base_url: z.string().url('Must be a valid URL'),
  api_key: z.string().min(1, 'API key is required'),
})

type AddModelFormValues = z.infer<typeof addModelSchema>

const PROVIDERS = [
  { value: 'OpenAI', label: 'OpenAI', baseUrl: 'https://api.openai.com/v1' },
  { value: 'Claude', label: 'Anthropic (Claude)', baseUrl: 'https://api.anthropic.com' },
  { value: 'DeepSeek', label: 'DeepSeek', baseUrl: 'https://api.deepseek.com/v1' },
  { value: 'Gemini', label: 'Google Gemini', baseUrl: 'https://generativelanguage.googleapis.com/v1beta' },
  { value: 'Ollama', label: 'Ollama (local)', baseUrl: 'http://localhost:11434/v1' },
  { value: 'Qwen', label: 'Alibaba Qwen', baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1' },
  { value: 'Ark', label: 'ByteDance Ark', baseUrl: 'https://ark.cn-beijing.volces.com/api/v3' },
]

const PROVIDER_COLORS: Record<string, string> = {
  OpenAI: 'bg-green-100 text-green-700',
  Claude: 'bg-orange-100 text-orange-700',
  DeepSeek: 'bg-blue-100 text-blue-700',
  Gemini: 'bg-purple-100 text-purple-700',
  Ollama: 'bg-gray-100 text-gray-700',
  Qwen: 'bg-red-100 text-red-700',
  Ark: 'bg-indigo-100 text-indigo-700',
}

function providerColor(modelClass: string): string {
  return PROVIDER_COLORS[modelClass] ?? 'bg-slate-100 text-slate-700'
}

function maskApiKey(key?: string): string {
  if (!key || key.length < 8) return '••••••••'
  return `${'•'.repeat(Math.min(key.length - 4, 12))}${key.slice(-4)}`
}

// ─── Add Model Dialog ─────────────────────────────────────────────────────────

interface AddModelDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

function AddModelDialog({ open, onClose, onSuccess }: AddModelDialogProps) {
  const { t } = useTranslation()
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<AddModelFormValues>({
    resolver: zodResolver(addModelSchema),
    defaultValues: { model_class: 'OpenAI', base_url: PROVIDERS[0].baseUrl },
  })

  const selectedClass = watch('model_class')

  const handleProviderChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const provider = PROVIDERS.find((p) => p.value === e.target.value)
    setValue('model_class', e.target.value)
    if (provider) setValue('base_url', provider.baseUrl)
  }

  const onSubmit = async (values: AddModelFormValues) => {
    try {
      const payload: CreateModelRequest = { ...values }
      await modelConfigApi.create(payload)
      toast.success(t('settings.modelAdded'))
      reset()
      onSuccess()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to add model')
    }
  }

  const handleClose = () => {
    reset()
    onClose()
  }

  return (
    <Dialog open={open} onClose={handleClose} title="Add Model">
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700">Provider</label>
          <select
            value={selectedClass}
            onChange={handleProviderChange}
            className="h-9 w-full rounded-md border border-gray-300 bg-white px-3 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          >
            {PROVIDERS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </div>

        <Input
          label="Model Name"
          placeholder="e.g. GPT-4o"
          error={errors.name?.message}
          {...register('name')}
        />

        <Input
          label="Model ID"
          placeholder="e.g. gpt-4o"
          error={errors.model?.message}
          {...register('model')}
        />

        <Input
          label="Base URL"
          placeholder="https://api.openai.com/v1"
          error={errors.base_url?.message}
          {...register('base_url')}
        />

        <Input
          label="API Key"
          type="password"
          placeholder="sk-..."
          error={errors.api_key?.message}
          {...register('api_key')}
        />

        <p className="text-xs text-gray-500">
          Clicking "Add &amp; Test" will validate the connection by sending a test request.
        </p>

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="ghost" onClick={handleClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Testing...' : 'Add & Test'}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

// ─── Delete Confirm Dialog ────────────────────────────────────────────────────

interface DeleteConfirmDialogProps {
  open: boolean
  model: ModelRecord | null
  onClose: () => void
  onConfirm: () => void
  loading: boolean
}

function DeleteConfirmDialog({ open, model, onClose, onConfirm, loading }: DeleteConfirmDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} title="Delete Model">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-gray-600">
          Are you sure you want to delete{' '}
          <span className="font-semibold">{model?.name}</span>? This cannot be undone.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={loading}>
            {loading ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}

// ─── Models Tab ───────────────────────────────────────────────────────────────

function ModelsTab() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<ModelRecord | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['models'],
    queryFn: modelConfigApi.list,
  })

  // Coze legacy API wraps results in data.data or data.model_configs — handle both
  const models: ModelRecord[] = (() => {
    if (!data) return []
    if (Array.isArray(data)) return data
    if (Array.isArray(data.data)) return data.data
    if (Array.isArray(data.model_configs)) return data.model_configs
    if (data.data && Array.isArray(data.data.model_configs)) return data.data.model_configs
    return []
  })()

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    setDeleteLoading(true)
    try {
      await modelConfigApi.delete(deleteTarget.id)
      toast.success(t('settings.modelDeleted'))
      queryClient.invalidateQueries({ queryKey: ['models'] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete model')
    } finally {
      setDeleteLoading(false)
      setDeleteTarget(null)
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">
          Configure LLM providers and models for the Model Router.
        </p>
        <Button size="sm" onClick={() => setAddOpen(true)}>
          + {t('settings.addModel')}
        </Button>
      </div>

      {isLoading && (
        <div className="rounded-lg border border-gray-200 bg-white px-4 py-6 text-center text-sm text-gray-400">
          Loading models...
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">
          Failed to load models: {error instanceof Error ? error.message : 'Unknown error'}
        </div>
      )}

      {!isLoading && !isError && models.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-4 py-8 text-center">
          <p className="text-sm text-gray-400">No models configured yet.</p>
          <button
            onClick={() => setAddOpen(true)}
            className="mt-2 text-sm text-blue-600 hover:underline"
          >
            Add your first model
          </button>
        </div>
      )}

      {models.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white divide-y divide-gray-100">
          {models.map((m) => (
            <div key={m.id} className="flex items-center gap-3 px-4 py-3">
              {/* Provider badge */}
              <span
                className={`inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full text-xs font-bold ${providerColor(m.model_class)}`}
              >
                {(m.model_class ?? 'M').charAt(0).toUpperCase()}
              </span>

              {/* Model info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{m.name}</p>
                <p className="text-xs text-gray-500 truncate">
                  {m.model_class} · {m.model}
                </p>
              </div>

              {/* Masked key */}
              <span className="hidden sm:block text-xs font-mono text-gray-400">
                {maskApiKey(m.api_key)}
              </span>

              {/* Status */}
              <span className="text-xs rounded-full bg-green-50 px-2 py-0.5 text-green-700 font-medium">
                Active
              </span>

              {/* Delete */}
              <button
                onClick={() => setDeleteTarget(m)}
                className="ml-2 rounded px-2 py-1 text-xs text-red-500 hover:bg-red-50 hover:text-red-700 transition-colors"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      )}

      <AddModelDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['models'] })}
      />

      <DeleteConfirmDialog
        open={deleteTarget !== null}
        model={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        loading={deleteLoading}
      />
    </div>
  )
}

// ─── MCP Servers Tab ──────────────────────────────────────────────────────────

function McpTab() {
  return (
    <div className="space-y-4 max-w-2xl">
      <p className="text-sm text-gray-500">
        MCP servers are configured in agent YAML files via{' '}
        <code className="rounded bg-gray-100 px-1 py-0.5 font-mono text-xs">mcp://</code> tool
        references.
      </p>

      <div className="rounded-lg border border-gray-200 bg-white p-4">
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">
          Example — agent YAML
        </p>
        <pre className="overflow-x-auto rounded bg-gray-50 p-3 text-xs leading-relaxed text-gray-700">
{`spec:
  tools:
    - mcp://filesystem/read_file
    - mcp://filesystem/write_file
    - mcp://brave-search/search`}
        </pre>
      </div>

      <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700">
        Dynamic MCP server management coming soon. For now, define MCP server connections in
        your agent YAML definitions.
      </div>
    </div>
  )
}

// ─── Page ──────────────────────────────────────────────────────────────────────

export default function SettingsPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState('models')

  return (
    <div className="flex flex-col h-full">
      <Header title={t('settings.title')} />

      <div className="flex-1 overflow-auto p-6">
        <Tabs value={tab} onChange={setTab}>
          <TabsList className="mb-6">
            <TabsTrigger value="models">{t('settings.models')}</TabsTrigger>
            <TabsTrigger value="mcp">{t('settings.mcpServers')}</TabsTrigger>
          </TabsList>

          <TabsContent value="models">
            <ModelsTab />
          </TabsContent>

          <TabsContent value="mcp">
            <McpTab />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
