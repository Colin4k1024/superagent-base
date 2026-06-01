import { useState, useRef, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { agentsApi, chatApi, type Agent, type ChatMessage, type ChatStreamCallbacks } from '../lib/api'
import Header from '../components/Header'
import MessageBubble from '../components/chat/MessageBubble'
import ChatInput from '../components/chat/ChatInput'

export default function ChatPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const [agents, setAgents] = useState<Agent[]>([])
  const [selectedAgent, setSelectedAgent] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [sessionId] = useState(() => `session-${Date.now()}`)
  const [showBackToBottom, setShowBackToBottom] = useState(false)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const userScrollingRef = useRef(false)
  // Incremented on each new send; guards stale callbacks from updating the wrong message.
  const requestIdRef = useRef(0)

  // Load agents
  useEffect(() => {
    agentsApi
      .list()
      .then((list) => {
        setAgents(list)
        const agentParam = searchParams.get('agent')
        if (agentParam && list.some((a) => a.name === agentParam)) {
          setSelectedAgent(agentParam)
        } else if (list.length > 0) {
          setSelectedAgent(list[0].name)
        }
      })
      .catch(() => {})
  }, [searchParams])

  // Auto-scroll (only when user is not scrolling up)
  useEffect(() => {
    if (!userScrollingRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages])

  // Scroll detection
  const handleScroll = useCallback(() => {
    const el = scrollContainerRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    userScrollingRef.current = distanceFromBottom > 100
    setShowBackToBottom(distanceFromBottom > 300)
  }, [])

  const scrollToBottom = useCallback(() => {
    userScrollingRef.current = false
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    setShowBackToBottom(false)
  }, [])

  // Stop generation
  const handleStop = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setIsLoading(false)
  }, [])

  // Send message (also supports preempting an in-flight request).
  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || !selectedAgent) return

    // Preempt: if a turn is active, cancel it before starting the new one.
    if (isLoading) {
      // Mark the current assistant message as preempted.
      setMessages((prev) => {
        const updated = [...prev]
        const last = updated[updated.length - 1]
        if (last.role === 'assistant') {
          updated[updated.length - 1] = { ...last, preempted: true }
        }
        return updated
      })
      // Tell the backend to abort, then abort the SSE connection locally.
      chatApi.abort(selectedAgent, sessionId).catch(() => {})
      const oldCtrl = abortRef.current
      abortRef.current = null
      oldCtrl?.abort()
    }

    requestIdRef.current++
    const myRequestId = requestIdRef.current

    const userMsg: ChatMessage = { role: 'user', content: text }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)
    userScrollingRef.current = false

    // Create assistant placeholder
    const assistantMsg: ChatMessage = { role: 'assistant', content: '' }
    setMessages((prev) => [...prev, assistantMsg])

    const callbacks: ChatStreamCallbacks = {
      onToken: (token) => {
        if (requestIdRef.current !== myRequestId) return
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + token }
          }
          return updated
        })
      },
      onThinking: (text) => {
        if (requestIdRef.current !== myRequestId) return
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
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
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            const toolCalls = [...(last.toolCalls || [])]
            toolCalls.push({ name, args, status: 'calling' })
            updated[updated.length - 1] = { ...last, toolCalls }
          }
          return updated
        })
      },
      onToolResult: (name: string, result: string) => {
        if (requestIdRef.current !== myRequestId) return
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant' && last.toolCalls) {
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
        // Backend confirmed preemption — the stream for this turn ended early.
        // The UI already marked the message preempted; nothing extra needed.
        if (requestIdRef.current !== myRequestId) return
        setIsLoading(false)
        abortRef.current = null
      },
      onError: (err) => {
        if (requestIdRef.current !== myRequestId) return
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: `Error: ${err.message}` }
          }
          return updated
        })
        setIsLoading(false)
        abortRef.current = null
      },
    }

    abortRef.current = chatApi.sendMessage(selectedAgent, sessionId, text, callbacks)
  }, [input, selectedAgent, isLoading, sessionId])

  return (
    <div className="flex flex-col h-full bg-gradient-to-b from-gray-50 to-white">
      <Header
        title="Chat"
        actions={
          agents.length > 0 ? (
            <select
              className="text-sm border border-gray-300 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white shadow-sm"
              value={selectedAgent}
              onChange={(e) => setSelectedAgent(e.target.value)}
            >
              {agents.map((a) => (
                <option key={a.name} value={a.name}>
                  {a.name}
                </option>
              ))}
            </select>
          ) : (
            <span className="text-sm text-gray-400">{t('chat.noAgent')}</span>
          )
        }
      />

      {/* Message area */}
      <div
        ref={scrollContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto"
      >
        <div className="max-w-[840px] mx-auto px-4 py-6 space-y-4">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center min-h-[50vh] text-gray-400 gap-3">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
                <svg className="w-8 h-8 opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                  />
                </svg>
              </div>
              <p className="text-sm">{t('chat.placeholder')}</p>
              {selectedAgent && (
                <p className="text-xs text-gray-300">
                  当前 Agent: <span className="font-mono">{selectedAgent}</span>
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
                  <span className="text-xs text-gray-400 italic">已中断</span>
                </div>
              )}
            </div>
          ))}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Back to bottom button */}
      {showBackToBottom && (
        <div className="absolute bottom-24 left-1/2 -translate-x-1/2 z-10">
          <button
            onClick={scrollToBottom}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white border border-gray-200 shadow-lg text-xs text-gray-600 hover:bg-gray-50 transition-all"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
            </svg>
            回到底部
          </button>
        </div>
      )}

      {/* Input area */}
      <ChatInput
        value={input}
        onChange={setInput}
        onSend={handleSend}
        onStop={handleStop}
        isLoading={isLoading}
        disabled={!selectedAgent}
      />
    </div>
  )
}
