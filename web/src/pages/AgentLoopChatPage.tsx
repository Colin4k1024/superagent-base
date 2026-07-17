import { useCallback, useEffect, useReducer, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { agentAdminApi, chatApi, type AgentDetail } from '@/lib/api'
import type { ChatStreamCallbacks } from '@/lib/api'
import type { AgentLoopMessage, AgentLoopAction } from '@/lib/agentloop-types'
import { agentLoopMessageReducer, executionStateReducer, initialExecutionState } from '@/lib/agentloop-reducer'
import { AgentLoopMessageList, ExecutionStatusPanel } from '@/components/agentloop'
import ChatInput from '@/components/chat/ChatInput'

// AgentLoop agents are filtered from the full agent list by type.
const AGENTLOOP_AGENTS_FALLBACK = ['code-assistant', 'code-agent', 'feedback-writer']

export default function AgentLoopChatPage() {
  const { t } = useTranslation()
  const [agents, setAgents] = useState<AgentDetail[]>([])
  const [selectedAgent, setSelectedAgent] = useState('')
  const [messages, dispatch] = useReducer(agentLoopMessageReducer, [] as AgentLoopMessage[])
  const [execState, execDispatch] = useReducer(executionStateReducer, initialExecutionState)
  const [isLoading, setIsLoading] = useState(false)
  const [input, setInput] = useState('')
  const [pendingInterrupt, setPendingInterrupt] = useState<{ reason: string; fields: { name: string; type: string; label: string; required: boolean; options?: string[] }[] } | null>(null)

  const sessionIdRef = useRef(`session-${crypto.randomUUID()}`)
  const abortRef = useRef<AbortController | null>(null)
  const requestIdRef = useRef(0)

  // Load agentloop agents on mount
  useEffect(() => {
    agentAdminApi.list().then(({ agents: all }) => {
      const loopAgents = all.filter((a) => a.type === 'agentloop')
      setAgents(loopAgents)
      if (loopAgents.length > 0) {
        setSelectedAgent((prev) => prev || loopAgents[0].name)
      }
    }).catch(() => {
      setAgents(AGENTLOOP_AGENTS_FALLBACK.map((name) => ({
        name, type: 'agentloop', description: '', status: 'active', file: '',
      })))
      setSelectedAgent(AGENTLOOP_AGENTS_FALLBACK[0])
    })
  }, [])

  // Check for pending interrupt on mount (e.g., after page refresh)
  useEffect(() => {
    if (!selectedAgent) return
    chatApi.interruptState(selectedAgent, sessionIdRef.current).then((result) => {
      if (result.interrupted && result.state) {
        const state = result.state as { reason?: string; fields?: { name: string; type: string; label: string; required: boolean; options?: string[] }[] }
        setPendingInterrupt({
          reason: state.reason || 'Agent is waiting for your input',
          fields: state.fields || [{ name: 'response', type: 'text', label: 'Your response', required: true }],
        })
      }
    })
  }, [selectedAgent])

  // --- TurnLoop: Send (with preempt support) ---

  const handleSend = useCallback(async (text: string) => {
    if (!selectedAgent || !text.trim()) return

    // Clear any pending interrupt
    setPendingInterrupt(null)

    // Preempt if already streaming
    if (isLoading && abortRef.current) {
      dispatch({ type: 'PREEMPTED' })
      execDispatch({ type: 'PREEMPTED' })
      chatApi.abort(selectedAgent, sessionIdRef.current)
      abortRef.current.abort()
      abortRef.current = null
      setIsLoading(false)
    }

    setInput('')

    const myRequestId = ++requestIdRef.current

    dispatch({ type: 'USER_MESSAGE', content: text })
    execDispatch({ type: 'USER_MESSAGE', content: text })

    const callbacks = buildCallbacks(myRequestId)

    setIsLoading(true)
    const controller = chatApi.sendMessage(selectedAgent, sessionIdRef.current, text, callbacks)
    abortRef.current = controller
  }, [selectedAgent, isLoading])

  // --- TurnLoop: Resume (after interrupt) ---

  const handleResume = useCallback(async (values: Record<string, string>) => {
    if (!selectedAgent) return
    const inputText = values.response || JSON.stringify(values)
    const myRequestId = ++requestIdRef.current

    setPendingInterrupt(null)
    dispatch({ type: 'TEXT_DELTA', delta: '\n' })
    setIsLoading(true)

    try {
      const callbacks = buildCallbacks(myRequestId)
      const controller = await chatApi.resume(selectedAgent, sessionIdRef.current, inputText, callbacks)
      abortRef.current = controller
    } catch (err) {
      dispatch({ type: 'ERROR', code: 'RESUME_ERROR', message: err instanceof Error ? err.message : String(err) })
      execDispatch({ type: 'ERROR', code: '', message: String(err) })
      setIsLoading(false)
    }
  }, [selectedAgent])

  // --- TurnLoop: Abort (explicit stop) ---

  const handleStop = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    if (selectedAgent) chatApi.abort(selectedAgent, sessionIdRef.current)
    dispatch({ type: 'PREEMPTED' })
    execDispatch({ type: 'PREEMPTED' })
    setIsLoading(false)
  }, [selectedAgent])

  // --- TurnLoop: New Session ---

  const handleNewSession = useCallback(() => {
    // Abort any active stream
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    if (selectedAgent && isLoading) {
      chatApi.abort(selectedAgent, sessionIdRef.current)
    }
    // Reset everything
    requestIdRef.current++
    sessionIdRef.current = `session-${crypto.randomUUID()}`
    setIsLoading(false)
    setPendingInterrupt(null)
    setInput('')
    // Force re-render by dispatching a reset-like action
    // (messages are cleared by the new sessionId context)
    window.location.reload()
  }, [selectedAgent, isLoading])

  // --- Shared callback builder ---

  const buildCallbacks = useCallback((myRequestId: number): ChatStreamCallbacks => ({
    onToken: (delta) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'TEXT_DELTA', delta })
    },
    onThinking: (delta) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'THINKING_DELTA', delta })
    },
    onToolCall: (name, args) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'TOOL_CALL', id: `tc_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`, name, args })
    },
    onToolResult: (name, result) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'TOOL_RESULT', id: '', name, result, isError: false })
    },
    onProgress: (data) => {
      if (requestIdRef.current !== myRequestId) return
      if (data.current > 0 && data.total > 0) {
        dispatch({ type: 'TURN_START', turn: data.current, total: data.total })
        execDispatch({ type: 'TURN_START', turn: data.current, total: data.total })
      }
    },
    onInterrupt: (data) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'INTERRUPT', reason: data.reason, fields: data.fields })
      setPendingInterrupt({ reason: data.reason, fields: data.fields })
      setIsLoading(false)
    },
    onDone: () => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'DONE' })
      execDispatch({ type: 'DONE' })
      setIsLoading(false)
    },
    onPreempted: () => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'PREEMPTED' })
      execDispatch({ type: 'PREEMPTED' })
      setIsLoading(false)
    },
    onError: (err) => {
      if (requestIdRef.current !== myRequestId) return
      dispatch({ type: 'ERROR', code: 'STREAM_ERROR', message: err.message })
      execDispatch({ type: 'ERROR', code: 'STREAM_ERROR', message: err.message })
      setIsLoading(false)
    },
  }), [])

  return (
    <div className="flex flex-col h-full">
      {/* Header with TurnLoop controls */}
      <div className="shrink-0 px-4 py-3 border-b border-gray-200 bg-white flex items-center gap-3">
        <h1 className="text-lg font-semibold text-gray-900">AgentLoop Demo</h1>

        {/* Agent selector */}
        <select
          value={selectedAgent}
          onChange={(e) => setSelectedAgent(e.target.value)}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm bg-white text-gray-700 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
        >
          {agents.map((a) => (
            <option key={a.name} value={a.name}>{a.name}</option>
          ))}
          {agents.length === 0 && <option value="">{t('common.loading', 'Loading...')}</option>}
        </select>

        {/* TurnLoop action buttons */}
        <div className="flex items-center gap-2 ml-auto">
          {/* Abort button (visible when streaming) */}
          {isLoading && (
            <button
              onClick={handleStop}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 bg-red-50 border border-red-200 rounded-md hover:bg-red-100 transition-colors"
              title="Abort current turn"
            >
              <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
              {t('common.stop', 'Stop')}
            </button>
          )}

          {/* New session button */}
          <button
            onClick={handleNewSession}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 bg-gray-50 border border-gray-200 rounded-md hover:bg-gray-100 transition-colors"
            title="Start a new session"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
            </svg>
            New Session
          </button>
        </div>

        {/* Session ID */}
        <span className="text-xs text-gray-400 font-mono hidden lg:inline">{sessionIdRef.current}</span>
      </div>

      {/* Execution status panel */}
      <div className="shrink-0 px-4 pt-3">
        <ExecutionStatusPanel state={execState} />
      </div>

      {/* Pending interrupt banner (from interrupt_state check) */}
      {pendingInterrupt && !isLoading && (
        <div className="shrink-0 mx-4 mb-3 p-3 bg-amber-50 border border-amber-200 rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-amber-600">⏸</span>
            <span className="text-sm font-medium text-amber-800">Agent is waiting for your input</span>
          </div>
          <p className="text-xs text-amber-700 mb-2">{pendingInterrupt.reason}</p>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              const formData = new FormData(e.currentTarget)
              const values: Record<string, string> = {}
              pendingInterrupt.fields.forEach((f) => {
                values[f.name] = String(formData.get(f.name) || '')
              })
              handleResume(values)
            }}
            className="flex gap-2"
          >
            {pendingInterrupt.fields.map((f) => (
              <input
                key={f.name}
                name={f.name}
                type={f.type === 'confirm' ? 'text' : f.type}
                placeholder={f.label}
                required={f.required}
                className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm"
              />
            ))}
            <button
              type="submit"
              className="px-4 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors"
            >
              Resume
            </button>
          </form>
        </div>
      )}

      {/* Messages */}
      <AgentLoopMessageList messages={messages} onResume={handleResume} />

      {/* Input */}
      <div className="shrink-0 border-t border-gray-200 bg-white p-4">
        <div className="max-w-[840px] mx-auto">
          <ChatInput value={input} onChange={setInput} onSend={() => handleSend(input)} isLoading={isLoading} onStop={handleStop} />
        </div>
      </div>
    </div>
  )
}
