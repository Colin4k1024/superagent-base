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

import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Bot, Edit, Trash2, Copy, ArrowLeft, Settings, MessageSquare, Workflow } from 'lucide-react'
import { agentAdminApi } from '@/lib/api'

export default function AgentDetailPage() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const qc = useQueryClient()

  const { data: agent, isLoading, error } = useQuery({
    queryKey: ['agent-detail', name],
    queryFn: () => agentAdminApi.get(name!),
    enabled: !!name,
  })

  const deleteMut = useMutation({
    mutationFn: () => agentAdminApi.delete(name!),
    onSuccess: () => {
      toast.success(t('agents.deleted', `Agent "${name}" deleted`, { name }))
      navigate('/agents')
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const duplicateMut = useMutation({
    mutationFn: async () => {
      const result = await agentAdminApi.get(name!)
      const newName = `${name}-copy`
      const newYaml = result.yaml.replace(/name:\s*.*/, `name: ${newName}`)
      return agentAdminApi.create(newYaml)
    },
    onSuccess: () => {
      toast.success('Agent duplicated')
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    )
  }

  if (error || !agent) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4">
        <p className="text-gray-500">{t('common.error', 'Agent not found')}</p>
        <button onClick={() => navigate('/agents')} className="text-blue-600 hover:underline text-sm">
          {t('editor.backToList', 'Back to list')}
        </button>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto py-6 px-4">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <button onClick={() => navigate('/agents')} className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-violet-500 flex items-center justify-center">
          <Bot className="w-6 h-6 text-white" />
        </div>
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold text-gray-900">{name}</h1>
          <p className="text-sm text-gray-500 truncate">YAML Agent</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => navigate(`/agents/${name}/edit`)}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700">
            <Edit className="w-4 h-4" /> {t('agents.edit', 'Edit')}
          </button>
          <button onClick={() => duplicateMut.mutate()} disabled={duplicateMut.isPending}
            className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
            <Copy className="w-4 h-4" /> {t('agents.duplicate', 'Duplicate')}
          </button>
          <button onClick={() => {
            if (confirm(t('agents.confirmDelete', `Delete agent "${name}"?`, { name }))) deleteMut.mutate()
          }} className="p-2 text-gray-400 hover:text-red-500 rounded-lg hover:bg-red-50">
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Quick actions */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        <button onClick={() => navigate(`/chat?agent=${name}`)}
          className="flex items-center gap-3 p-4 border border-gray-200 rounded-xl hover:border-blue-300 hover:bg-blue-50 transition-all">
          <div className="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center">
            <MessageSquare className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-left">
            <p className="font-medium text-gray-900 text-sm">Chat</p>
            <p className="text-xs text-gray-500">Test this agent in chat</p>
          </div>
        </button>
        <button onClick={() => navigate(`/agents/${name}/edit`)}
          className="flex items-center gap-3 p-4 border border-gray-200 rounded-xl hover:border-violet-300 hover:bg-violet-50 transition-all">
          <div className="w-10 h-10 rounded-lg bg-violet-100 flex items-center justify-center">
            <Settings className="w-5 h-5 text-violet-600" />
          </div>
          <div className="text-left">
            <p className="font-medium text-gray-900 text-sm">Configure</p>
            <p className="text-xs text-gray-500">Edit YAML configuration</p>
          </div>
        </button>
        <button onClick={() => navigate(`/agents/${name}/workflow`)}
          className="flex items-center gap-3 p-4 border border-gray-200 rounded-xl hover:border-green-300 hover:bg-green-50 transition-all">
          <div className="w-10 h-10 rounded-lg bg-green-100 flex items-center justify-center">
            <Workflow className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-left">
            <p className="font-medium text-gray-900 text-sm">Workflow</p>
            <p className="text-xs text-gray-500">Open graph editor</p>
          </div>
        </button>
      </div>

      {/* Agent YAML preview */}
      <div className="border border-gray-200 rounded-xl overflow-hidden">
        <div className="px-4 py-3 bg-gray-50 border-b border-gray-200">
          <h2 className="text-sm font-semibold text-gray-700">Agent Configuration (YAML)</h2>
        </div>
        <pre className="p-4 text-xs font-mono text-gray-800 bg-white overflow-auto max-h-96 whitespace-pre-wrap">
          {agent.yaml}
        </pre>
      </div>
    </div>
  )
}
