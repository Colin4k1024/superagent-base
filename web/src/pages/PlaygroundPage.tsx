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

import { useState, useRef, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import {
  agentsApi,
  agentAdminApi,
  chatApi,
  modelConfigApi,
  type Agent,
  type AgentDetail,
  type ChatMessage,
  type ChatStreamCallbacks,
} from '../lib/api'
import { cn } from '../lib/cn'
import MessageBubble from '../components/chat/MessageBubble'
import ChatInput from '../components/chat/ChatInput'

// ---- Types ----

interface Session {
  id: string
  label: string
  messages: ChatMessage[]
  createdAt: number
}

// ---- Helpers ----

function generateSessionId() {
  return `pg-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---- Sub-components ----

function SessionListItem({
  session,
  isActive,
  onSelect,
  onDelete,
}: {
  session: Session
  isActive: boolean
  onSelect: () => void
  onDelete: () => void
}) {
  const [showDelete, setShowDelete] = useState(false)

  return (
    <div
      className={cn(
        'group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors text-sm',
        isActive
          ? 'bg-blue-50 text-blue-700 border border-blue-200'
          : 'hover:bg-gray-50 text-gray-600 border border-transparent',
      )}
      onClick={onSelect}
      onMouseEnter={() => setShowDelete(true)}
      onMouseLeave={() => setShowDelete(false)}
    >
      <svg className="w-4 h-4 flex-shrink-0 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
      <span className="truncate flex-1">{session.label}</span>
      {showDelete && (
        <button
          onClick={(e) => {
            e.stopPropagation()
            onDelete()
          }}
          className="flex-shrink-0 p-0.5 rounded text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
          title="Delete session"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  )
}

function DebugPanel({
  selectedAgent,
  sessionId,
  messageCount,
  agents,
  models,
}: {
  selectedAgent: string
  sessionId: string
  messageCount: number
  agents: AgentDetail[]
  models: { id: number; name: string; model: string }[]
}) {
  const agent = agents.find((a) => a.name === selectedAgent)

  return (
    <div className="h-full overflow-auto p-4 text-xs font-mono space-y-3">
      <div>
        <div className="text-gray-400 uppercase tracking-wider mb-1">Agent</div>
        <div className="text-gray-800">{selectedAgent || '(none)'}</div>
        {agent && (
          <div className="mt-1 space-y-0.5 text-gray-500">
            <div>type: {agent.type}</div>
            <div>status: {agent.status}</div>
          </div>
        )}
      </div>
      <div>
        <div className="text-gray-400 uppercase tracking-wider mb-1">Session</div>
        <div className="text-gray-800 break-all">{sessionId}</div>
      </div>
      <div>
        <div className="text-gray-400 uppercase tracking-wider mb-1">Messages</div>
        <div className="text-gray-800">{messageCount}</div>
      </div>
      <div>
        <div className="text-gray-400 uppercase tracking-wider mb-1">Models ({models.length})</div>
        {models.length === 0 && <div className="text-gray-400">No models configured</div>}
        {models.slice(0, 5).map((m) => (
          <div key={m.id} className="text-gray-600 truncate">{m.model || m.name}</div>
        ))}
      </div>
    </div>
  )
}

// ---- Main page ----

export default function PlaygroundPage() {
  const { t } = useTranslation()

  // ---- Data fetching ----

  const { data: agentDetails } = useQuery({
    queryKey: ['admin-agents'],
    queryFn: () => agentAdminApi.list(),
  })

  const { data: agentList } = useQuery({
    queryKey: ['agents'],
    queryFn: () => agentsApi.list(),
  })

  const { data: models } = useQuery({
    queryKey: ['models'],
    queryFn: () => modelConfigApi.list(),
  })

  const agents = agentDetails?.agents ?? []
  const simpleAgents: Agent[] = agentList ?? []
  const modelList = models ?? []

  // ---- State ----

  const [selectedAgent, setSelectedAgent] = useState('')
  const [sessions, setSessions] = useState<Session[]>([])
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null)
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [showDebug, setShowDebug] = useState(false)

  // ---- Refs ----

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const userScrollingRef = useRef(false)
  const requestIdRef = useRef(0)

  // ---- Derived ----

  const activeSession = sessions.find((s) => s.id === activeSessionId) ?? null
  const messages = activeSession?.messages ?? []

  // Auto-select first agent
  useEffect(() => {
    if (!selectedAgent && simpleAgents.length > 0) {
      setSelectedAgent(simpleAgents[0].name)
    }
  }, [simpleAgents, selectedAgent])

  // Auto-create first session
  useEffect(() => {
    if (sessions.length === 0) {
      const id = generateSessionId()
      setSessions([{ id, label: 'Session 1', messages: [], createdAt: Date.now() }])
      setActiveSessionId(id)
    }
  }, [sessions.length])

  // Auto-scroll
  useEffect(() => {
    if (!userScrollingRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages])

  // ---- Actions ----

  const handleCreateSession = useCallback(() => {
    const id = generateSessionId()
    const num = sessions.length + 1
    setSessions((prev) => [
      { id, label: `Session ${num}`, messages: [], createdAt: Date.now() },
      ...prev,
    ])
    setActiveSessionId(id)
  }, [sessions.length])

  const handleDeleteSession = useCallback(
    (id: string) => {
      setSessions((prev) => prev.filter((s) => s.id !== id))
      if (activeSessionId === id) {
        const remaining = sessions.filter((s) => s.id !== id)
        setActiveSessionId(remaining.length > 0 ? remaining[0].id : null)
      }
    },
    [activeSessionId, sessions],
  )

  const handleSelectSession = useCallback((id: string) => {
    setActiveSessionId(id)
    userScrollingRef.current = false
  }, [])

  const [showBackToBottom, setShowBackToBottom] = useState(false)

  const handleScroll = useCallback(() => {
    const el = scrollContainerRef.current
    if (!el) return
    const dist = el.scrollHeight - el.scrollTop - el.clientHeight
    userScrollingRef.current = dist > 100
    setShowBackToBottom(dist > 300)
  }, [])

  const scrollToBottom = useCallback(() => {
    userScrollingRef.current = false
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    setShowBackToBottom(false)
  }, [])

  const handleStop = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setIsLoading(false)
  }, [])

  const updateActiveSessionMessages = useCallback(
    (updater: (prev: ChatMessage[]) => ChatMessage[]) => {
      if (!activeSessionId) return
      setSessions((prev) =>
        prev.map((s) =>
          s.id === activeSessionId ? { ...s, messages: updater(s.messages) } : s,
        ),
      )
    },
    [activeSessionId],
  )

  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || !selectedAgent || !activeSessionId) return

    // Preempt if currently streaming
    if (isLoading) {
      updateActiveSessionMessages((prev) => {
        const updated = [...prev]
        const last = updated[updated.length - 1]
        if (last?.role === 'assistant') {
          updated[updated.length - 1] = { ...last, preempted: true }
        }
        return updated
      })
      chatApi.abort(selectedAgent, activeSessionId).catch(() => {})
      const oldCtrl = abortRef.current
      abortRef.current = null
      oldCtrl?.abort()
    }

    requestIdRef.current++
    const myRequestId = requestIdRef.current

    const userMsg: ChatMessage = { role: 'user', content: text }
    updateActiveSessionMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)
    userScrollingRef.current = false

    // Assistant placeholder
    updateActiveSessionMessages((prev) => [
      ...prev,
      { role: 'assistant', content: '' },
    ])

    const callbacks: ChatStreamCallbacks = {
      onToken: (token) => {
        if (requestIdRef.current !== myRequestId) return
        updateActiveSessionMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last?.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + token }
          }
          return updated
        })
      },
      onThinking: (text) => {
        if (requestIdRef.current !== myRequestId) return
        updateActiveSessionMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last?.role === 'assistant') {
            updated[updated.length - 1] = {
              ...last,
              thinking: (last.thinking || '') + text,
            }
          }
          return updated
        })
      },
      onToolCall: (name, args) => {
        if (requestIdRef.current !== myRequestId) return
        updateActiveSessionMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last?.role === 'assistant') {
            const toolCalls = [...(last.toolCalls || [])]
            toolCalls.push({ name, args, status: 'calling' })
            updated[updated.length - 1] = { ...last, toolCalls }
          }
          return updated
        })
      },
      onToolResult: (name, result) => {
        if (requestIdRef.current !== myRequestId) return
        updateActiveSessionMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last?.role === 'assistant' && last.toolCalls) {
            const toolCalls = [...last.toolCalls]
            let idx = -1
            for (let i = toolCalls.length - 1; i >= 0; i--) {
              if (toolCalls[i].name === name && toolCalls[i].status === 'calling') {
                idx = i
                break
              }
            }
            if (idx >= 0) {
              toolCalls[idx] = { ...toolCalls[idx], result, status: 'done' }
            }
            updated[updated.length - 1] = { ...last, toolCalls }
          }
          return updated
        })
      },
      onDone: () => {
        if (requestIdRef.current !== myRequestId) return
        setIsLoading(false)
        abortRef.current = null
      },
      onPreempted: () => {
        if (requestIdRef.current !== myRequestId) return
        setIsLoading(false)
        abortRef.current = null
      },
      onError: (err) => {
        if (requestIdRef.current !== myRequestId) return
        updateActiveSessionMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last?.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: `Error: ${err.message}` }
          }
          return updated
        })
        setIsLoading(false)
        abortRef.current = null
      },
    }

    abortRef.current = chatApi.sendMessage(selectedAgent, activeSessionId, text, callbacks)
  }, [input, selectedAgent, isLoading, activeSessionId, updateActiveSessionMessages])

  // ---- Render ----

  return (
    <div className="flex h-full bg-gray-50">
      {/* Sidebar: session list */}
      <aside className="w-64 flex-shrink-0 border-r border-gray-200 bg-white flex flex-col">
        <div className="px-4 py-3 border-b border-gray-100">
          <h2 className="text-sm font-semibold text-gray-900">{t('playground.sessions', 'Sessions')}</h2>
        </div>

        <div className="flex-1 overflow-auto p-2 space-y-1">
          {sessions.map((session) => (
            <SessionListItem
              key={session.id}
              session={session}
              isActive={session.id === activeSessionId}
              onSelect={() => handleSelectSession(session.id)}
              onDelete={() => handleDeleteSession(session.id)}
            />
          ))}
          {sessions.length === 0 && (
            <div className="px-3 py-8 text-center text-xs text-gray-400">
              {t('playground.noSessions', 'No sessions yet')}
            </div>
          )}
        </div>

        <div className="p-2 border-t border-gray-100">
          <button
            onClick={handleCreateSession}
            className="w-full flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-sm text-blue-600 hover:bg-blue-50 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            {t('playground.newSession', 'New Session')}
          </button>
        </div>
      </aside>

      {/* Main chat area */}
      <main className="flex-1 flex flex-col min-w-0">
        {/* Top bar: agent selector + debug toggle */}
        <header className="flex items-center justify-between px-4 py-2 bg-white border-b border-gray-200">
          <div className="flex items-center gap-3">
            <h1 className="text-sm font-semibold text-gray-900">
              {t('playground.title', 'Playground')}
            </h1>
            <select
              className="text-sm border border-gray-300 rounded-lg px-2.5 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
              value={selectedAgent}
              onChange={(e) => setSelectedAgent(e.target.value)}
            >
              {simpleAgents.length > 0
                ? simpleAgents.map((a) => (
                    <option key={a.name} value={a.name}>
                      {a.name}
                    </option>
                  ))
                : agents.map((a) => (
                    <option key={a.name} value={a.name}>
                      {a.name}
                    </option>
                  ))}
            </select>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowDebug((v) => !v)}
              className={cn(
                'flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs transition-colors',
                showDebug
                  ? 'bg-gray-900 text-white'
                  : 'text-gray-500 hover:bg-gray-100',
              )}
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              Debug
            </button>
          </div>
        </header>

        {/* Content: chat + optional debug panel */}
        <div className="flex-1 flex overflow-hidden">
          {/* Messages */}
          <div
            ref={scrollContainerRef}
            onScroll={handleScroll}
            className="flex-1 overflow-y-auto"
          >
            <div className="max-w-[840px] mx-auto px-4 py-6 space-y-4">
              {messages.length === 0 && (
                <div className="flex flex-col items-center justify-center min-h-[40vh] text-gray-400 gap-3">
                  <div className="w-14 h-14 rounded-full bg-gray-100 flex items-center justify-center">
                    <svg className="w-7 h-7 opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                    </svg>
                  </div>
                  <p className="text-sm">
                    {t('playground.emptyHint', 'Send a message to start testing your agent')}
                  </p>
                  {selectedAgent && (
                    <p className="text-xs text-gray-300">
                      Agent: <span className="font-mono">{selectedAgent}</span>
                    </p>
                  )}
                </div>
              )}

              {messages.map((msg, i) => (
                <div key={i}>
                  <MessageBubble
                    message={msg}
                    isStreaming={isLoading && i === messages.length - 1}
                    isLast={i === messages.length - 1}
                  />
                  {msg.preempted && (
                    <div className="flex justify-start mt-1 ml-1">
                      <span className="text-xs text-gray-400 italic">
                        {t('playground.interrupted', 'Interrupted')}
                      </span>
                    </div>
                  )}
                </div>
              ))}

              <div ref={messagesEndRef} />
            </div>

            {/* Back to bottom button */}
            {showBackToBottom && (
              <div className="sticky bottom-4 flex justify-center z-10 pointer-events-none">
                <button
                  onClick={scrollToBottom}
                  className="pointer-events-auto flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white border border-gray-200 shadow-lg text-xs text-gray-600 hover:bg-gray-50 transition-all"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                  </svg>
                  {t('playground.backToBottom', 'Back to bottom')}
                </button>
              </div>
            )}
          </div>

          {/* Debug panel */}
          {showDebug && (
            <aside className="w-72 flex-shrink-0 border-l border-gray-200 bg-gray-50">
              <DebugPanel
                selectedAgent={selectedAgent}
                sessionId={activeSessionId ?? ''}
                messageCount={messages.length}
                agents={agents}
                models={modelList}
              />
            </aside>
          )}
        </div>

        {/* Input */}
        <ChatInput
          value={input}
          onChange={setInput}
          onSend={handleSend}
          onStop={handleStop}
          isLoading={isLoading}
          disabled={!selectedAgent || !activeSessionId}
        />
      </main>
    </div>
  )
}
