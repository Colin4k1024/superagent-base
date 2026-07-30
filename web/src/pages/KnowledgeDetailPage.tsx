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
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Upload, Trash2, FileText, ChevronDown, ChevronRight, Database } from 'lucide-react'
import {
  knowledgeApi,
  knowledgeDocumentApi,
  knowledgeSliceApi,
  type CozeDocument,
} from '../lib/coze-api'
import Header from '../components/Header'
import { Button } from '../components/ui/button'
import { Dialog } from '../components/ui/dialog'

function DocStatusBadge({ status }: { status?: number }) {
  const { t } = useTranslation()
  if (status === undefined || status === null) return null
  const map: Record<number, { label: string; cls: string }> = {
    0: { label: t('knowledge.docStatus.pending'), cls: 'bg-yellow-100 text-yellow-700' },
    1: { label: t('knowledge.docStatus.processing'), cls: 'bg-blue-100 text-blue-700' },
    2: { label: t('knowledge.docStatus.ready'), cls: 'bg-green-100 text-green-700' },
    3: { label: t('knowledge.docStatus.failed'), cls: 'bg-red-100 text-red-700' },
  }
  const entry = map[status] ?? { label: `#${status}`, cls: 'bg-gray-100 text-gray-600' }
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${entry.cls}`}>
      {entry.label}
    </span>
  )
}

function SliceList({ documentId }: { documentId: string }) {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['knowledge-slices', documentId],
    queryFn: () => knowledgeSliceApi.list({ document_id: documentId }),
  })

  if (isLoading) {
    return (
      <div className="px-4 py-3 text-xs text-gray-400">{t('common.loading')}</div>
    )
  }

  const slices = data?.slices ?? []
  if (slices.length === 0) {
    return (
      <div className="px-4 py-3 text-xs text-gray-400 italic">{t('knowledge.noSlices')}</div>
    )
  }

  return (
    <div className="divide-y divide-gray-100">
      {slices.map((slice) => (
        <div key={slice.slice_id} className="px-4 py-3">
          <p className="text-xs text-gray-700 leading-relaxed whitespace-pre-wrap line-clamp-4">
            {slice.content}
          </p>
          <div className="mt-1 flex items-center gap-2">
            <span className="text-[10px] text-gray-400 font-mono">{slice.slice_id.slice(0, 8)}…</span>
            {slice.status !== undefined && <DocStatusBadge status={slice.status} />}
          </div>
        </div>
      ))}
    </div>
  )
}

function DocumentRow({
  doc,
  onDelete,
}: {
  doc: CozeDocument
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div
        className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-50 transition-colors"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? (
          <ChevronDown className="w-4 h-4 text-gray-400 flex-shrink-0" />
        ) : (
          <ChevronRight className="w-4 h-4 text-gray-400 flex-shrink-0" />
        )}
        <FileText className="w-4 h-4 text-gray-500 flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 truncate">{doc.name}</p>
        </div>
        <DocStatusBadge status={doc.status} />
        <span className="text-xs text-gray-400 flex-shrink-0">
          {t('knowledge.sliceCount', { count: doc.slice_count ?? 0 })}
        </span>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors flex-shrink-0"
          aria-label={t('common.delete')}
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
      {expanded && (
        <div className="border-t border-gray-100 bg-gray-50/50">
          <SliceList documentId={doc.document_id} />
        </div>
      )}
    </div>
  )
}

export default function KnowledgeDetailPage() {
  const { datasetId: id } = useParams<{ datasetId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()

  const [deleteTarget, setDeleteTarget] = useState<CozeDocument | null>(null)

  const { data: dataset, isLoading: dsLoading } = useQuery({
    queryKey: ['knowledge-dataset', id],
    queryFn: () => knowledgeApi.detail(id!),
    enabled: !!id,
  })

  const { data: docData, isLoading: docsLoading } = useQuery({
    queryKey: ['knowledge-documents', id],
    queryFn: () => knowledgeDocumentApi.list({ dataset_id: id! }),
    enabled: !!id,
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) =>
      knowledgeDocumentApi.create({ dataset_id: id!, name: file.name, file }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['knowledge-documents', id] })
      queryClient.invalidateQueries({ queryKey: ['knowledge-datasets'] })
      toast.success(t('knowledge.docUploaded'))
    },
    onError: (err: Error) => {
      toast.error(`${t('knowledge.uploadFailed')}: ${err.message}`)
    },
  })

  const deleteDocMutation = useMutation({
    mutationFn: (docId: string) => knowledgeDocumentApi.delete(docId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['knowledge-documents', id] })
      queryClient.invalidateQueries({ queryKey: ['knowledge-datasets'] })
      toast.success(t('knowledge.docDeleted', { name: deleteTarget?.name }))
      setDeleteTarget(null)
    },
    onError: (err: Error) => {
      toast.error(`${t('knowledge.deleteFailed')}: ${err.message}`)
    },
  })

  function handleFileUpload() {
    const input = document.createElement('input')
    input.type = 'file'
    input.multiple = false
    input.accept = '.txt,.md,.pdf,.doc,.docx,.csv,.json'
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file) uploadMutation.mutate(file)
    }
    input.click()
  }

  const documents = docData?.documents ?? []

  return (
    <div className="flex flex-col h-full">
      <Header
        title={
          <div className="flex items-center gap-3">
            <button
              onClick={() => navigate('/knowledge')}
              className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
              aria-label={t('knowledge.back')}
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
            {dsLoading ? (
              <span className="text-gray-400">{t('common.loading')}</span>
            ) : (
              <div className="flex items-center gap-2">
                <Database className="w-5 h-5 text-blue-600" />
                <span>{dataset?.name ?? id}</span>
              </div>
            )}
          </div>
        }
        actions={
          <Button size="sm" onClick={handleFileUpload} loading={uploadMutation.isPending}>
            <Upload className="w-4 h-4" />
            {t('knowledge.uploadDoc')}
          </Button>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        {/* Dataset description */}
        {dataset?.description && (
          <p className="text-sm text-gray-500 mb-4">{dataset.description}</p>
        )}

        {docsLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span className="w-2 h-2 rounded-full bg-gray-400 animate-pulse" />
            {t('common.loading')}
          </div>
        )}

        {!docsLoading && documents.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
            <FileText className="w-12 h-12 text-gray-300" />
            <p className="text-sm text-gray-500">{t('knowledge.noDocuments')}</p>
            <Button size="sm" onClick={handleFileUpload} loading={uploadMutation.isPending}>
              <Upload className="w-4 h-4" />
              {t('knowledge.uploadDoc')}
            </Button>
          </div>
        )}

        {!docsLoading && documents.length > 0 && (
          <div className="flex flex-col gap-2">
            {documents.map((doc) => (
              <DocumentRow
                key={doc.document_id}
                doc={doc}
                onDelete={() => setDeleteTarget(doc)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Delete document confirmation */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('knowledge.deleteDocTitle')}>
        <p className="text-sm text-gray-700 mb-6">
          {t('knowledge.confirmDeleteDoc', { name: deleteTarget?.name })}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={deleteDocMutation.isPending}
            onClick={() => deleteTarget && deleteDocMutation.mutate(deleteTarget.document_id)}
          >
            {t('common.delete')}
          </Button>
        </div>
      </Dialog>
    </div>
  )
}
