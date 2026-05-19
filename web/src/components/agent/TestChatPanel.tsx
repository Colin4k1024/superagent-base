import { useState, useRef, useEffect } from 'react'
import { chatApi, type ChatMessage } from '@/lib/api'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/button'

export interface TestChatPanelProps {
  agentName: string
  open: boolean
  onClose: () => void
}

export function TestChatPanel({ agentName, open, onClose }: TestChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [sessionId] = useState(() => `test-${Date.now()}`)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  function handleSend() {
    const text = input.trim()
    if (!text || !agentName || isLoading) return

    setMessages((prev) => [...prev, { role: 'user', content: text }])
    setInput('')
    setIsLoading(true)
    setMessages((prev) => [...prev, { role: 'assistant', content: '' }])

    chatApi.sendMessage(agentName, sessionId, text, {
      onToken: (token: string) => {
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + token }
          }
          return updated
        })
      },
      onDone: () => setIsLoading(false),
      onError: (err: Error) => {
        setMessages((prev) => {
          const updated = [...prev]
          updated[updated.length - 1] = {
            role: 'assistant',
            content: `Error: ${err.message}`,
          }
          return updated
        })
        setIsLoading(false)
      },
    })
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void handleSend()
    }
  }

  function handleClear() {
    setMessages([])
    setInput('')
  }

  return (
    <div
      className={cn(
        'fixed top-0 right-0 h-full w-[400px] bg-white border-l border-gray-200 shadow-xl z-50',
        'flex flex-col transition-transform duration-300 ease-in-out',
        open ? 'translate-x-0' : 'translate-x-full',
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 shrink-0">
        <span className="text-sm font-semibold text-gray-800 truncate">
          Test: {agentName || 'No agent selected'}
        </span>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={handleClear} disabled={messages.length === 0}>
            Clear
          </Button>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-gray-100 text-gray-500 hover:text-gray-800 transition-colors"
            aria-label="Close test panel"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {!agentName ? (
          <div className="flex items-center justify-center h-full text-gray-400 text-sm">
            No agent selected
          </div>
        ) : messages.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-400 text-sm">
            Send a message to test this agent
          </div>
        ) : (
          messages.map((msg, i) => (
            <div key={i} className={cn('flex', msg.role === 'user' ? 'justify-end' : 'justify-start')}>
              <div
                className={cn(
                  'max-w-[85%] px-3 py-2 rounded-xl text-sm leading-relaxed whitespace-pre-wrap',
                  msg.role === 'user'
                    ? 'bg-blue-600 text-white rounded-br-none'
                    : msg.content.startsWith('Error:')
                    ? 'bg-red-50 border border-red-200 text-red-700 rounded-bl-none'
                    : 'bg-gray-100 text-gray-800 rounded-bl-none',
                )}
              >
                {msg.content}
                {msg.role === 'assistant' && isLoading && i === messages.length - 1 && (
                  <span className="inline-block w-1.5 h-3.5 bg-gray-400 animate-pulse ml-0.5 align-middle" />
                )}
              </div>
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="border-t border-gray-200 p-3 shrink-0">
        <div className="flex items-end gap-2">
          <textarea
            className="flex-1 resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            rows={2}
            placeholder={agentName ? 'Send a message… (Enter to send)' : 'Select an agent first'}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isLoading || !agentName}
          />
          <Button
            size="sm"
            onClick={() => void handleSend()}
            disabled={!input.trim() || isLoading || !agentName}
            loading={isLoading}
          >
            Send
          </Button>
        </div>
      </div>
    </div>
  )
}
