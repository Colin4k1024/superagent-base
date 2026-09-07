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
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { Search, Plus, Database, Trash2 } from 'lucide-react'
import { knowledgeApi, type CozeDataset } from '../lib/coze-api'
import Header from '../components/Header'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog } from '../components/ui/dialog'

function DatasetCard({
  dataset,
  onClick,
  onDelete,
}: {
  dataset: CozeDataset
  onClick: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()

  return (
    <div
      onClick={onClick}
      className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-3 hover:shadow-sm transition-shadow cursor-pointer group"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center">
            <Database className="w-5 h-5 text-blue-600" />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-gray-900 text-sm truncate">{dataset.name}</h3>
          </div>
        </div>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          className="rounded p-1 text-gray-400 opacity-0 group-hover:opacity-100 hover:bg-red-50 hover:text-red-600 transition-all"
          aria-label={t('common.delete')}
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>

      {dataset.description && (
        <p className="text-xs text-gray-500 leading-relaxed line-clamp-2">{dataset.description}</p>
      )}

      <div className="mt-auto flex items-center justify-between">
        <span className="text-xs text-gray-400">
          {t('knowledge.docCount', { count: dataset.document_count ?? 0 })}
        </span>
        <span className="text-xs text-blue-600 font-medium opacity-0 group-hover:opacity-100 transition-opacity">
          {t('knowledge.view')} →
        </span>
      </div>
    </div>
  )
}

export default function KnowledgePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()

  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<CozeDataset | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['knowledge-datasets'],
    queryFn: () => knowledgeApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: () => knowledgeApi.create({ name: newName.trim(), description: newDesc.trim() || undefined }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['knowledge-datasets'] })
      toast.success(t('knowledge.created'))
      setCreateOpen(false)
      setNewName('')
      setNewDesc('')
    },
    onError: (err: Error) => {
      toast.error(`${t('knowledge.createFailed')}: ${err.message}`)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => knowledgeApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['knowledge-datasets'] })
      toast.success(t('knowledge.deleted', { name: deleteTarget?.name }))
      setDeleteTarget(null)
    },
    onError: (err: Error) => {
      toast.error(`${t('knowledge.deleteFailed')}: ${err.message}`)
    },
  })

  const datasets = (data?.datasets ?? []).filter((d) =>
    search.trim() === '' ? true : d.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="flex flex-col h-full">
      <Header
        title={t('knowledge.title')}
        actions={
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder={t('knowledge.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="h-8 w-56 rounded-md border border-gray-300 bg-white pl-8 pr-3 text-sm placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="w-4 h-4" />
              {t('knowledge.newDataset')}
            </Button>
          </div>
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
            {t('knowledge.loadFailed')}: {error instanceof Error ? error.message : String(error)}
          </div>
        )}

        {!isLoading && !error && datasets.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
            <Database className="w-12 h-12 text-gray-300" />
            <p className="text-sm text-gray-500">
              {search.trim() ? t('knowledge.noResults') : t('knowledge.empty')}
            </p>
            {!search.trim() && (
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="w-4 h-4" />
                {t('knowledge.newDataset')}
              </Button>
            )}
          </div>
        )}

        {!isLoading && datasets.length > 0 && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {datasets.map((ds) => (
              <DatasetCard
                key={ds.dataset_id}
                dataset={ds}
                onClick={() => navigate(`/knowledge/${ds.dataset_id}`)}
                onDelete={() => setDeleteTarget(ds)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Create dialog */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} title={t('knowledge.createTitle')}>
        <div className="flex flex-col gap-4">
          <Input
            label={t('knowledge.nameLabel')}
            placeholder={t('knowledge.namePlaceholder')}
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <Input
            label={t('knowledge.descLabel')}
            placeholder={t('knowledge.descPlaceholder')}
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
          />
          <div className="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={() => setCreateOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              size="sm"
              loading={createMutation.isPending}
              disabled={!newName.trim()}
              onClick={() => createMutation.mutate()}
            >
              {t('common.confirm')}
            </Button>
          </div>
        </div>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('knowledge.deleteTitle')}>
        <p className="text-sm text-gray-700 mb-6">
          {t('knowledge.confirmDelete', { name: deleteTarget?.name })}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={deleteMutation.isPending}
            onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.dataset_id)}
          >
            {t('common.delete')}
          </Button>
        </div>
      </Dialog>
    </div>
  )
}
