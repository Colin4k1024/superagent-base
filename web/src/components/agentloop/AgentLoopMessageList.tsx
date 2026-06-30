import { useEffect, useRef } from 'react'
import type { AgentLoopMessage } from '@/lib/agentloop-types'
import AgentLoopMessageItem from './AgentLoopMessageItem'

interface Props {
  messages: AgentLoopMessage[]
  onResume?: (values: Record<string, string>) => void
}

export default function AgentLoopMessageList({ messages, onResume }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const userScrollingRef = useRef(false)

  useEffect(() => {
    if (!userScrollingRef.current) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages])

  function handleScroll() {
    const el = containerRef.current
    if (!el) return
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    userScrollingRef.current = distFromBottom > 300
  }

  return (
    <div ref={containerRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-4 py-6">
      <div className="max-w-[840px] mx-auto space-y-4">
        {messages.map((msg, i) => (
          <AgentLoopMessageItem
            key={msg.id}
            message={msg}
            isLast={i === messages.length - 1}
            onResume={i === messages.length - 1 ? onResume : undefined}
          />
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
