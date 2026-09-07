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
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { Puzzle, Plus, Trash2, Play, X, Code2, Send } from 'lucide-react'
import { pluginApi, type CozePlugin, type CozePluginAPI } from '@/lib/coze-api'
import { cn } from '@/lib/cn'

export default function PluginsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const [selectedPlugin, setSelectedPlugin] = useState<CozePlugin | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [showCreateAPI, setShowCreateAPI] = useState(false)
  const [debugAPI, setDebugAPI] = useState<CozePluginAPI | null>(null)
  const [debugParams, setDebugParams] = useState('{}')
  const [debugResult, setDebugResult] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'plugin' | 'api'; id: string; name: string } | null>(null)

  // Plugin list
  const { data: pluginsData, isLoading } = useQuery({
    queryKey: ['plugins'],
    queryFn: () => pluginApi.list(),
  })
  const plugins = pluginsData?.plugins || []

  // APIs for selected plugin
  const { data: apisData } = useQuery({
    queryKey: ['plugin-apis', selectedPlugin?.plugin_id],
    queryFn: () => pluginApi.getAPIs(selectedPlugin!.plugin_id),
    enabled: !!selectedPlugin,
  })
  const apis = apisData?.apis || []

  // Create plugin
  const createMut = useMutation({
    mutationFn: (params: { name: string; description: string }) => pluginApi.register(params),
    onSuccess: () => {
      toast.success(t('plugins.created', 'Plugin created'))
      qc.invalidateQueries({ queryKey: ['plugins'] })
      setShowCreate(false)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  // Delete plugin
  const deleteMut = useMutation({
    mutationFn: (id: string) => pluginApi.delete(id),
    onSuccess: () => {
      toast.success(t('plugins.deleted', 'Plugin deleted'))
      qc.invalidateQueries({ queryKey: ['plugins'] })
      if (selectedPlugin && confirmDelete?.id === selectedPlugin.plugin_id) setSelectedPlugin(null)
      setConfirmDelete(null)
    },
    onError: (e: Error) => { toast.error(e.message); setConfirmDelete(null) },
  })

  // Create API
  const createAPIMut = useMutation({
    mutationFn: (params: { plugin_id: string; name: string; description: string; method: string; url: string }) =>
      pluginApi.createAPI(params),
    onSuccess: () => {
      toast.success(t('plugins.apiCreated', 'API created'))
      qc.invalidateQueries({ queryKey: ['plugin-apis'] })
      qc.invalidateQueries({ queryKey: ['plugins'] })
      setShowCreateAPI(false)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  // Delete API
  const deleteAPIMut = useMutation({
    mutationFn: (id: string) => pluginApi.deleteAPI(id),
    onSuccess: () => {
      toast.success(t('plugins.apiDeleted', 'API deleted'))
      qc.invalidateQueries({ queryKey: ['plugin-apis'] })
      qc.invalidateQueries({ queryKey: ['plugins'] })
      setConfirmDelete(null)
    },
    onError: (e: Error) => { toast.error(e.message); setConfirmDelete(null) },
  })

  // Debug API
  const debugMut = useMutation({
    mutationFn: (api: CozePluginAPI) => {
      let params: Record<string, unknown> = {}
      try { params = JSON.parse(debugParams) } catch { /* ignore */ }
      return pluginApi.debugAPI({ api_id: api.api_id, parameters: params })
    },
    onSuccess: (data) => setDebugResult(JSON.stringify(data?.response ?? data, null, 2)),
    onError: (e: Error) => setDebugResult(`Error: ${e.message}`),
  })

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-gray-200 bg-white flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
          <Puzzle className="w-6 h-6 text-purple-500" />
          {t('plugins.title', 'Plugins')}
        </h1>
        <button onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700">
          <Plus className="w-4 h-4" /> {t('plugins.create', 'Create Plugin')}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {/* Loading */}
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
            </svg>
            {t('common.loading', 'Loading...')}
          </div>
        )}

        {/* Empty */}
        {!isLoading && plugins.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-gray-400">
            <Puzzle className="h-12 w-12 mb-3 opacity-40" />
            <p className="text-sm">{t('plugins.empty', 'No plugins yet. Create your first plugin!')}</p>
          </div>
        )}

        {/* Plugin cards grid */}
        {!isLoading && plugins.length > 0 && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {plugins.map((p) => (
              <div key={p.plugin_id}
                onClick={() => setSelectedPlugin(p)}
                className={cn(
                  'bg-white rounded-lg border p-4 cursor-pointer transition-all hover:shadow-md',
                  selectedPlugin?.plugin_id === p.plugin_id
                    ? 'border-purple-400 ring-2 ring-purple-400/20'
                    : 'border-gray-200 hover:border-gray-300',
                )}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-50 text-purple-600">
                      <Puzzle className="h-5 w-5" />
                    </div>
                    <div className="min-w-0">
                      <h3 className="text-sm font-medium text-gray-900 truncate">{p.name}</h3>
                      <p className="text-xs text-gray-500 mt-0.5">
                        {p.api_count ?? 0} {t('plugins.apis', 'APIs')}
                      </p>
                    </div>
                  </div>
                  <button onClick={(e) => { e.stopPropagation(); setConfirmDelete({ type: 'plugin', id: p.plugin_id, name: p.name }) }}
                    className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
                {p.description && <p className="text-xs text-gray-500 mt-2 line-clamp-2">{p.description}</p>}
              </div>
            ))}
          </div>
        )}

        {/* API list for selected plugin */}
        {selectedPlugin && (
          <div className="mt-8">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-gray-900 flex items-center gap-2">
                <Puzzle className="h-4 w-4 text-purple-500" />
                {selectedPlugin.name} — {t('plugins.apis', 'APIs')}
              </h2>
              <button onClick={() => setShowCreateAPI(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100">
                <Plus className="w-4 h-4" /> {t('plugins.addAPI', 'Add API')}
              </button>
            </div>

            {apis.length === 0 && (
              <p className="text-sm text-gray-400 italic py-4">{t('plugins.noAPIs', 'No APIs defined')}</p>
            )}

            {apis.length > 0 && (
              <div className="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100">
                {apis.map((api) => (
                  <div key={api.api_id} className="flex items-center gap-4 px-4 py-3">
                    <span className={cn('inline-flex items-center rounded px-2 py-0.5 text-xs font-mono font-semibold',
                      api.method === 'GET' ? 'bg-green-50 text-green-700' :
                      api.method === 'POST' ? 'bg-blue-50 text-blue-700' :
                      api.method === 'PUT' ? 'bg-amber-50 text-amber-700' :
                      api.method === 'DELETE' ? 'bg-red-50 text-red-700' :
                      'bg-gray-50 text-gray-700',
                    )}>
                      {api.method || 'POST'}
                    </span>
                    <div className="flex-1 min-w-0">
                      <span className="text-sm font-medium text-gray-900">{api.name}</span>
                      {api.url && <span className="ml-2 text-xs text-gray-400 font-mono truncate">{api.url}</span>}
                    </div>
                    <div className="flex items-center gap-1">
                      <button onClick={() => { setDebugAPI(api); setDebugParams('{}'); setDebugResult('') }}
                        className="rounded p-1.5 text-gray-400 hover:bg-green-50 hover:text-green-600 transition-colors"
                        title={t('plugins.debug', 'Debug')}>
                        <Play className="h-4 w-4" />
                      </button>
                      <button onClick={() => setConfirmDelete({ type: 'api', id: api.api_id, name: api.name })}
                        className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors"
                        title={t('common.delete', 'Delete')}>
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Debug panel */}
        {debugAPI && (
          <div className="mt-6 bg-white rounded-lg border border-gray-200 p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Code2 className="h-4 w-4 text-blue-600" />
                <span className="text-sm font-medium text-gray-900">
                  {t('plugins.debugAPI', 'Debug API')}: {debugAPI.name}
                </span>
                <span className="text-xs text-gray-400 font-mono">{debugAPI.method} {debugAPI.url}</span>
              </div>
              <button onClick={() => setDebugAPI(null)}
                className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">
                  {t('plugins.parameters', 'Parameters')} (JSON)
                </label>
                <textarea value={debugParams} onChange={(e) => setDebugParams(e.target.value)}
                  className="w-full h-32 rounded-md border border-gray-300 px-3 py-2 text-sm font-mono focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                  placeholder="{}" />
                <button onClick={() => debugMut.mutate(debugAPI)} disabled={debugMut.isPending}
                  className="mt-2 flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 disabled:opacity-50">
                  <Send className="w-4 h-4" />
                  {debugMut.isPending ? '...' : t('plugins.sendRequest', 'Send')}
                </button>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">
                  {t('plugins.response', 'Response')}
                </label>
                <pre className="w-full h-40 overflow-auto rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-xs font-mono text-gray-800 whitespace-pre-wrap">
                  {debugResult || t('plugins.runToSee', 'Click Send to see the response')}
                </pre>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Create Plugin Dialog */}
      {showCreate && <CreatePluginDialog onClose={() => setShowCreate(false)}
        onCreate={(name, desc) => createMut.mutate({ name, description: desc })} loading={createMut.isPending} />}

      {/* Create API Dialog */}
      {showCreateAPI && selectedPlugin && (
        <CreateAPIDialog onClose={() => setShowCreateAPI(false)}
          onCreate={(name, desc, method, url) =>
            createAPIMut.mutate({ plugin_id: selectedPlugin.plugin_id, name, description: desc, method, url })}
          loading={createAPIMut.isPending} />
      )}

      {/* Confirm Delete Dialog */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-96 space-y-4">
            <h3 className="text-lg font-semibold text-gray-900">{t('common.confirmDelete', 'Confirm Delete')}</h3>
            <p className="text-sm text-gray-700">
              {confirmDelete.type === 'plugin'
                ? t('plugins.confirmDeletePlugin', 'Are you sure you want to delete plugin "{{name}}"? This cannot be undone.', { name: confirmDelete.name })
                : t('plugins.confirmDeleteAPI', 'Are you sure you want to delete API "{{name}}"? This cannot be undone.', { name: confirmDelete.name })}
            </p>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setConfirmDelete(null)}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
                {t('common.cancel', 'Cancel')}
              </button>
              <button
                onClick={() => {
                  if (confirmDelete.type === 'plugin') deleteMut.mutate(confirmDelete.id)
                  else deleteAPIMut.mutate(confirmDelete.id)
                }}
                disabled={deleteMut.isPending || deleteAPIMut.isPending}
                className="px-4 py-2 text-sm text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50">
                {t('common.delete', 'Delete')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function CreatePluginDialog({ onClose, onCreate, loading }: {
  onClose: () => void; onCreate: (name: string, desc: string) => void; loading: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-xl shadow-xl p-6 w-96 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">{t('plugins.create', 'Create Plugin')}</h3>
        <input value={name} onChange={(e) => setName(e.target.value)}
          placeholder={t('plugins.namePlaceholder', 'Plugin Name')}
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)}
          placeholder={t('plugins.descPlaceholder', 'What does this plugin do?')}
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm h-20 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <div className="flex gap-3 justify-end">
          <button onClick={onClose}
            className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
            {t('common.cancel', 'Cancel')}
          </button>
          <button onClick={() => name && onCreate(name, desc)} disabled={!name || loading}
            className="px-4 py-2 text-sm text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {loading ? '...' : t('common.create', 'Create')}
          </button>
        </div>
      </div>
    </div>
  )
}

function CreateAPIDialog({ onClose, onCreate, loading }: {
  onClose: () => void; onCreate: (name: string, desc: string, method: string, url: string) => void; loading: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [method, setMethod] = useState('GET')
  const [url, setUrl] = useState('')
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-xl shadow-xl p-6 w-96 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">{t('plugins.createAPI', 'Add API')}</h3>
        <input value={name} onChange={(e) => setName(e.target.value)}
          placeholder={t('plugins.apiNamePlaceholder', 'API Name')}
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <select value={method} onChange={(e) => setMethod(e.target.value)}
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option><option>PATCH</option>
        </select>
        <input value={url} onChange={(e) => setUrl(e.target.value)}
          placeholder="https://api.example.com/v1/resource"
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)}
          placeholder={t('plugins.apiDescPlaceholder', 'Optional description')}
          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm h-16 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500" />
        <div className="flex gap-3 justify-end">
          <button onClick={onClose}
            className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
            {t('common.cancel', 'Cancel')}
          </button>
          <button onClick={() => name && url && onCreate(name, desc, method, url)} disabled={!name || !url || loading}
            className="px-4 py-2 text-sm text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {loading ? '...' : t('common.create', 'Create')}
          </button>
        </div>
      </div>
    </div>
  )
}
