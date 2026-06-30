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
      // Fallback: show hardcoded agents
      setAgents(AGENTLOOP_AGENTS_FALLBACK.map((name) => ({
        name, type: 'agentloop', description: '', status: 'active', file: '',
      })))
      setSelectedAgent(AGENTLOOP_AGENTS_FALLBACK[0])
    })
  }, [])

  const handleSend = useCallback(async (text: string) => {
    if (!selectedAgent || !text.trim()) return

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

    // Dispatch user message + initial assistant placeholder
    dispatch({ type: 'USER_MESSAGE', content: text })
    execDispatch({ type: 'USER_MESSAGE', content: text })

    const callbacks: ChatStreamCallbacks = {
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
        // Match by name (tool_call and tool_result share the same name)
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
    }

    setIsLoading(true)
    const controller = chatApi.sendMessage(selectedAgent, sessionIdRef.current, text, callbacks)
    abortRef.current = controller
  }, [selectedAgent, isLoading])

  const handleResume = useCallback(async (values: Record<string, string>) => {
    if (!selectedAgent || isLoading) return
    const input = values.response || JSON.stringify(values)
    const myRequestId = ++requestIdRef.current

    dispatch({ type: 'TEXT_DELTA', delta: '\n' })
    setIsLoading(true)

    try {
      const callbacks: ChatStreamCallbacks = {
        onToken: (delta) => { if (requestIdRef.current === myRequestId) dispatch({ type: 'TEXT_DELTA', delta }) },
        onThinking: (delta) => { if (requestIdRef.current === myRequestId) dispatch({ type: 'THINKING_DELTA', delta }) },
        onToolCall: (name, args) => { if (requestIdRef.current === myRequestId) dispatch({ type: 'TOOL_CALL', id: `tc_${Date.now()}`, name, args }) },
        onToolResult: (name, result) => { if (requestIdRef.current === myRequestId) dispatch({ type: 'TOOL_RESULT', id: '', name, result, isError: false }) },
        onProgress: (data) => { if (requestIdRef.current === myRequestId && data.current > 0) { dispatch({ type: 'TURN_START', turn: data.current, total: data.total }); execDispatch({ type: 'TURN_START', turn: data.current, total: data.total }) } },
        onInterrupt: (data) => { if (requestIdRef.current === myRequestId) dispatch({ type: 'INTERRUPT', reason: data.reason, fields: data.fields }) },
        onDone: () => { if (requestIdRef.current === myRequestId) { dispatch({ type: 'DONE' }); execDispatch({ type: 'DONE' }); setIsLoading(false) } },
        onError: (err) => { if (requestIdRef.current === myRequestId) { dispatch({ type: 'ERROR', code: 'RESUME_ERROR', message: err.message }); execDispatch({ type: 'ERROR', code: '', message: err.message }); setIsLoading(false) } },
      }

      const controller = await chatApi.resume(selectedAgent, sessionIdRef.current, input, callbacks)
      abortRef.current = controller
    } catch (err) {
      dispatch({ type: 'ERROR', code: 'RESUME_ERROR', message: err instanceof Error ? err.message : String(err) })
      execDispatch({ type: 'ERROR', code: '', message: String(err) })
      setIsLoading(false)
    }
  }, [selectedAgent, isLoading])

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

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="shrink-0 px-4 py-3 border-b border-gray-200 bg-white flex items-center gap-4">
        <h1 className="text-lg font-semibold text-gray-900">AgentLoop Demo</h1>
        <select
          value={selectedAgent}
          onChange={(e) => setSelectedAgent(e.target.value)}
          className="ml-auto rounded-md border border-gray-300 px-3 py-1.5 text-sm bg-white text-gray-700 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
        >
          {agents.map((a) => (
            <option key={a.name} value={a.name}>{a.name}</option>
          ))}
          {agents.length === 0 && <option value="">{t('common.loading', 'Loading...')}</option>}
        </select>
        <span className="text-xs text-gray-400 font-mono">{sessionIdRef.current}</span>
      </div>

      {/* Execution status panel */}
      <div className="shrink-0 px-4 pt-3">
        <ExecutionStatusPanel state={execState} />
      </div>

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
