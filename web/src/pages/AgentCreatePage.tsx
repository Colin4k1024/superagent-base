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
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { MessageSquare, Zap, Users, ArrowRight, ArrowLeft, Check, Loader2 } from 'lucide-react'
import { agentAdminApi } from '@/lib/api'
import { cn } from '@/lib/cn'

const AGENT_TYPES = [
  { value: 'chat_model_agent', icon: MessageSquare, color: 'from-blue-500 to-cyan-500' },
  { value: 'deep_agent', icon: Zap, color: 'from-violet-500 to-purple-500' },
  { value: 'supervisor', icon: Users, color: 'from-orange-500 to-amber-500' },
  { value: 'sequential', icon: ArrowRight, color: 'from-green-500 to-emerald-500' },
  { value: 'parallel', icon: Users, color: 'from-pink-500 to-rose-500' },
] as const

const STEPS = ['type', 'info', 'model', 'tools', 'review'] as const

export default function AgentCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [agentType, setAgentType] = useState('chat_model_agent')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful assistant.')
  const [primaryModel, setPrimaryModel] = useState('gpt-4o-mini')
  const [selectedTools, setSelectedTools] = useState<string[]>([])

  const TOOLS = [
    { uri: 'builtin/web_search', name: 'Web Search' },
    { uri: 'builtin/code_execute', name: 'Code Execute' },
    { uri: 'builtin/http_request', name: 'HTTP Request' },
  ]

  const createMut = useMutation({
    mutationFn: () => {
      const yaml = `apiVersion: superagent/v1
kind: Agent
metadata:
  name: ${name}
  description: "${description}"
spec:
  type: ${agentType}
  system_prompt: "${systemPrompt.replace(/"/g, '\\"')}"
  model:
    primary: ${primaryModel}
${selectedTools.length > 0 ? `  tools:\n${selectedTools.map(t => `    - uri: "${t}"`).join('\n')}` : ''}
`
      return agentAdminApi.create(yaml)
    },
    onSuccess: () => {
      toast.success(t('agentCreate.created', 'Agent created'))
      navigate(`/agents/${name}/edit`)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const canNext = step === 0 ? true : step === 1 ? name.trim().length > 0 : true

  return (
    <div className="max-w-3xl mx-auto py-8 px-4">
      <h1 className="text-2xl font-bold text-gray-900 mb-8">{t('agentCreate.title', 'Create Agent')}</h1>

      {/* Progress */}
      <div className="flex items-center mb-8">
        {STEPS.map((s, i) => (
          <div key={s} className="flex items-center">
            <div className={cn('flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium transition-colors',
              i <= step ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500')}>
              {i < step ? <Check className="w-4 h-4" /> : i + 1}
            </div>
            <span className={cn('ml-2 text-sm', i <= step ? 'text-gray-900 font-medium' : 'text-gray-400')}>
              {t(`agentCreate.steps.${s}`, s)}
            </span>
            {i < STEPS.length - 1 && <div className={cn('w-12 h-0.5 mx-3', i < step ? 'bg-blue-600' : 'bg-gray-200')} />}
          </div>
        ))}
      </div>

      {/* Step content */}
      <div className="bg-white border border-gray-200 rounded-xl p-6 mb-6 min-h-[320px]">
        {step === 0 && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('agentCreate.agentType', 'Agent Type')}</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {AGENT_TYPES.map(({ value, icon: Icon, color }) => (
                <button key={value} onClick={() => setAgentType(value)}
                  className={cn('flex items-start gap-3 p-4 rounded-xl border-2 transition-all text-left',
                    agentType === value ? 'border-blue-500 bg-blue-50' : 'border-gray-100 hover:border-gray-200')}>
                  <div className={cn('w-10 h-10 rounded-lg bg-gradient-to-br flex items-center justify-center shrink-0', color)}>
                    <Icon className="w-5 h-5 text-white" />
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{t(`agentCreate.typeName.${value}`, value)}</p>
                    <p className="text-xs text-gray-500 mt-0.5">{t(`agentCreate.typeDesc.${value}`, '')}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {step === 1 && (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('agentCreate.name', 'Agent Name')} *</label>
              <input value={name} onChange={(e) => setName(e.target.value.replace(/[^a-zA-Z0-9_-]/g, '_'))}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="my-agent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <input value={description} onChange={(e) => setDescription(e.target.value)}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm" placeholder="What does this agent do?" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('agentCreate.systemPrompt', 'System Prompt')}</label>
              <textarea value={systemPrompt} onChange={(e) => setSystemPrompt(e.target.value)} rows={6}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono resize-none" />
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('agentCreate.primaryModel', 'Primary Model')}</h2>
            <div className="grid grid-cols-2 gap-3">
              {['gpt-4o', 'gpt-4o-mini', 'claude-3-5-sonnet', 'deepseek-chat', 'gemini-pro'].map((m) => (
                <button key={m} onClick={() => setPrimaryModel(m)}
                  className={cn('p-3 rounded-lg border-2 text-left transition-all',
                    primaryModel === m ? 'border-blue-500 bg-blue-50' : 'border-gray-100 hover:border-gray-200')}>
                  <p className="font-medium text-sm text-gray-900">{m}</p>
                </button>
              ))}
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('agentCreate.selectTools', 'Select Tools')}</h2>
            {TOOLS.map(({ uri, name: toolName }) => (
              <label key={uri} className="flex items-center gap-3 p-3 rounded-lg border border-gray-100 hover:bg-gray-50 cursor-pointer">
                <input type="checkbox" checked={selectedTools.includes(uri)}
                  onChange={(e) => setSelectedTools(e.target.checked ? [...selectedTools, uri] : selectedTools.filter(t => t !== uri))}
                  className="w-4 h-4 text-blue-600 rounded" />
                <div>
                  <p className="font-medium text-sm text-gray-900">{toolName}</p>
                  <p className="text-xs text-gray-500 font-mono">{uri}</p>
                </div>
              </label>
            ))}
          </div>
        )}

        {step === 4 && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Review</h2>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between py-2 border-b border-gray-100"><span className="text-gray-500">Type</span><span className="font-medium">{agentType}</span></div>
              <div className="flex justify-between py-2 border-b border-gray-100"><span className="text-gray-500">Name</span><span className="font-medium">{name}</span></div>
              <div className="flex justify-between py-2 border-b border-gray-100"><span className="text-gray-500">Model</span><span className="font-medium">{primaryModel}</span></div>
              <div className="flex justify-between py-2 border-b border-gray-100"><span className="text-gray-500">Tools</span><span className="font-medium">{selectedTools.length} selected</span></div>
              {description && <div className="py-2"><span className="text-gray-500">Description:</span><p className="mt-1">{description}</p></div>}
            </div>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <button onClick={() => setStep(step - 1)} disabled={step === 0}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 disabled:opacity-50">
          <ArrowLeft className="w-4 h-4" /> {t('agentCreate.back', 'Back')}
        </button>
        {step < STEPS.length - 1 ? (
          <button onClick={() => setStep(step + 1)} disabled={!canNext}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {t('agentCreate.next', 'Next')} <ArrowRight className="w-4 h-4" />
          </button>
        ) : (
          <button onClick={() => createMut.mutate()} disabled={createMut.isPending || !name.trim()}
            className="flex items-center gap-2 px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {createMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
            {createMut.isPending ? t('agentCreate.creating', 'Creating...') : t('agentCreate.create', 'Create Agent')}
          </button>
        )}
      </div>
    </div>
  )
}
