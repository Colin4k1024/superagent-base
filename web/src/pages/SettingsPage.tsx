/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
import { modelConfigApi, mcpAdminApi, CreateModelRequest, ModelRecord, McpServerItem } from '../lib/api'

// ─── Helpers ──────────────────────────────────────────────────────────────────

const PROVIDERS = [
  { value: 'OpenAI',   label: 'OpenAI',            baseUrl: 'https://api.openai.com/v1' },
  { value: 'Claude',   label: 'Anthropic (Claude)', baseUrl: 'https://api.anthropic.com' },
  { value: 'DeepSeek', label: 'DeepSeek',           baseUrl: 'https://api.deepseek.com/v1' },
  { value: 'Gemini',   label: 'Google Gemini',      baseUrl: 'https://generativelanguage.googleapis.com/v1beta' },
  { value: 'Ollama',   label: 'Ollama (local)',      baseUrl: 'http://localhost:11434/v1' },
  { value: 'Qwen',     label: 'Alibaba Qwen',        baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1' },
  { value: 'Ark',      label: 'ByteDance Ark',       baseUrl: 'https://ark.cn-beijing.volces.com/api/v3' },
]

const PROVIDER_COLORS: Record<string, string> = {
  OpenAI:   'bg-green-100 text-green-700',
  Claude:   'bg-orange-100 text-orange-700',
  DeepSeek: 'bg-blue-100 text-blue-700',
  Gemini:   'bg-purple-100 text-purple-700',
  Ollama:   'bg-gray-100 text-gray-700',
  Qwen:     'bg-red-100 text-red-700',
  Ark:      'bg-indigo-100 text-indigo-700',
}

function providerColor(modelClass: string): string {
  return PROVIDER_COLORS[modelClass] ?? 'bg-slate-100 text-slate-700'
}

function maskApiKey(key?: string): string {
  if (!key || key.length < 8) return '••••••••'
  return `${'•'.repeat(Math.min(key.length - 4, 12))}${key.slice(-4)}`
}

// ─── Add Model Dialog ─────────────────────────────────────────────────────────

const addModelSchema = z.object({
  model_class: z.string().min(1, 'Provider is required'),
  name: z.string().min(1, 'Model name is required'),
  model: z.string().min(1, 'Model ID is required'),
  base_url: z.string().url('Must be a valid URL'),
  api_key: z.string().min(1, 'API key is required'),
})

type AddModelFormValues = z.infer<typeof addModelSchema>

interface AddModelDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

function AddModelDialog({ open, onClose, onSuccess }: AddModelDialogProps) {
  const { t } = useTranslation()
  const {
    register, handleSubmit, reset, setValue, watch,
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

  const handleClose = () => { reset(); onClose() }

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
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </select>
        </div>

        <Input label="Model Name" placeholder="e.g. GPT-4o" error={errors.name?.message} {...register('name')} />
        <Input label="Model ID" placeholder="e.g. gpt-4o" error={errors.model?.message} {...register('model')} />
        <Input label="Base URL" placeholder="https://api.openai.com/v1" error={errors.base_url?.message} {...register('base_url')} />
        <Input label="API Key" type="password" placeholder="sk-..." error={errors.api_key?.message} {...register('api_key')} />

        <p className="text-xs text-gray-500">
          Clicking "Add &amp; Test" will validate the connection by sending a test request.
        </p>

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="ghost" onClick={handleClose} disabled={isSubmitting}>Cancel</Button>
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
  name: string
  onClose: () => void
  onConfirm: () => void
  loading: boolean
}

function DeleteConfirmDialog({ open, name, onClose, onConfirm, loading }: DeleteConfirmDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} title="Confirm Delete">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-gray-600">
          Are you sure you want to delete <span className="font-semibold">{name}</span>? This cannot be undone.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose} disabled={loading}>Cancel</Button>
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

  const { data: models = [], isLoading, isError, error } = useQuery({
    queryKey: ['models'],
    queryFn: modelConfigApi.list,
  })

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
        <p className="text-sm text-gray-500">Configure LLM providers and models for the Model Router.</p>
        <Button size="sm" onClick={() => setAddOpen(true)}>+ {t('settings.addModel')}</Button>
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
          <button onClick={() => setAddOpen(true)} className="mt-2 text-sm text-blue-600 hover:underline">
            Add your first model
          </button>
        </div>
      )}

      {models.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white divide-y divide-gray-100">
          {models.map((m) => (
            <div key={m.id} className="flex items-center gap-3 px-4 py-3">
              <span className={`inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full text-xs font-bold ${providerColor(m.model_class)}`}>
                {(m.model_class ?? 'M').charAt(0).toUpperCase()}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{m.name}</p>
                <p className="text-xs text-gray-500 truncate">{m.model_class} · {m.model}</p>
              </div>
              <span className="hidden sm:block text-xs font-mono text-gray-400">{maskApiKey(m.api_key)}</span>
              <span className="text-xs rounded-full bg-green-50 px-2 py-0.5 text-green-700 font-medium">Active</span>
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

      <AddModelDialog open={addOpen} onClose={() => setAddOpen(false)} onSuccess={() => queryClient.invalidateQueries({ queryKey: ['models'] })} />

      <DeleteConfirmDialog
        open={deleteTarget !== null}
        name={deleteTarget?.name ?? ''}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        loading={deleteLoading}
      />
    </div>
  )
}

// ─── Add MCP Server Dialog ────────────────────────────────────────────────────

interface AddMcpDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

function AddMcpDialog({ open, onClose, onSuccess }: AddMcpDialogProps) {
  const [transport, setTransport] = useState<'stdio' | 'sse'>('stdio')
  const {
    register, handleSubmit, reset,
    formState: { isSubmitting },
  } = useForm<any>({
    defaultValues: { transport: 'stdio', name: '', command: '', args: '', url: '' },
  })

  const onSubmit = async (values: any) => {
    try {
      const req: any = {
        name: values.name,
        transport,
      }
      if (transport === 'stdio') {
        req.command = values.command
        if (values.args?.trim()) {
          req.args = values.args.trim().split(/\s+/)
        }
      } else {
        req.url = values.url
      }
      await mcpAdminApi.connect(req)
      toast.success(`MCP server "${values.name}" connected`)
      reset()
      onSuccess()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to connect MCP server')
    }
  }

  const handleClose = () => { reset(); onClose() }

  return (
    <Dialog open={open} onClose={handleClose} title="Connect MCP Server">
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700">Transport</label>
          <div className="flex gap-2">
            {(['stdio', 'sse'] as const).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setTransport(t)}
                className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
                  transport === t
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-gray-300 text-gray-600 hover:border-gray-400'
                }`}
              >
                {t === 'stdio' ? 'stdio (local process)' : 'SSE (HTTP)'}
              </button>
            ))}
          </div>
        </div>

        <Input label="Server Name" placeholder="e.g. filesystem" {...register('name')} />

        {transport === 'stdio' && (
          <>
            <Input label="Command" placeholder="e.g. npx" {...register('command')} />
            <Input label="Arguments (space-separated)" placeholder="e.g. -y @modelcontextprotocol/server-filesystem /tmp" {...register('args')} />
          </>
        )}

        {transport === 'sse' && (
          <Input label="URL" placeholder="http://localhost:3000/sse" {...register('url')} />
        )}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="ghost" onClick={handleClose} disabled={isSubmitting}>Cancel</Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Connecting...' : 'Connect'}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

// ─── MCP Servers Tab ──────────────────────────────────────────────────────────

function McpTab() {
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = useState(false)
  const [disconnectTarget, setDisconnectTarget] = useState<McpServerItem | null>(null)
  const [disconnectLoading, setDisconnectLoading] = useState(false)

  const { data: servers = [], isLoading, isError, error } = useQuery({
    queryKey: ['mcp-servers'],
    queryFn: mcpAdminApi.list,
    refetchInterval: 30_000,
  })

  const handleDisconnect = async () => {
    if (!disconnectTarget) return
    setDisconnectLoading(true)
    try {
      await mcpAdminApi.disconnect(disconnectTarget.name)
      toast.success(`MCP server "${disconnectTarget.name}" disconnected`)
      queryClient.invalidateQueries({ queryKey: ['mcp-servers'] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to disconnect')
    } finally {
      setDisconnectLoading(false)
      setDisconnectTarget(null)
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">
          Manage live MCP server connections. Servers can also be referenced via{' '}
          <code className="rounded bg-gray-100 px-1 py-0.5 font-mono text-xs">mcp://</code> in agent YAML.
        </p>
        <Button size="sm" onClick={() => setAddOpen(true)}>+ Connect Server</Button>
      </div>

      {isLoading && (
        <div className="rounded-lg border border-gray-200 bg-white px-4 py-6 text-center text-sm text-gray-400">
          Loading MCP servers...
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">
          Failed to load MCP servers: {error instanceof Error ? error.message : 'Unknown error'}
        </div>
      )}

      {!isLoading && !isError && servers.length === 0 && (
        <div className="rounded-lg border border-gray-200 bg-white px-4 py-8 text-center">
          <p className="text-sm text-gray-400">No MCP servers connected.</p>
          <button onClick={() => setAddOpen(true)} className="mt-2 text-sm text-blue-600 hover:underline">
            Connect your first MCP server
          </button>
        </div>
      )}

      {servers.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white divide-y divide-gray-100">
          {servers.map((s) => (
            <div key={s.name} className="flex items-center gap-3 px-4 py-3">
              <span className="inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-cyan-100 text-cyan-700 text-xs font-bold">
                M
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{s.name}</p>
                <p className="text-xs text-gray-500">{s.transport} · {s.tools_count} tools</p>
              </div>
              <span className={`text-xs rounded-full px-2 py-0.5 font-medium ${
                s.status === 'connected'
                  ? 'bg-green-50 text-green-700'
                  : 'bg-red-50 text-red-600'
              }`}>
                {s.status}
              </span>
              <button
                onClick={() => setDisconnectTarget(s)}
                className="ml-2 rounded px-2 py-1 text-xs text-red-500 hover:bg-red-50 hover:text-red-700 transition-colors"
              >
                Disconnect
              </button>
            </div>
          ))}
        </div>
      )}

      <AddMcpDialog open={addOpen} onClose={() => setAddOpen(false)} onSuccess={() => queryClient.invalidateQueries({ queryKey: ['mcp-servers'] })} />

      <DeleteConfirmDialog
        open={disconnectTarget !== null}
        name={disconnectTarget?.name ?? ''}
        onClose={() => setDisconnectTarget(null)}
        onConfirm={handleDisconnect}
        loading={disconnectLoading}
      />
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
