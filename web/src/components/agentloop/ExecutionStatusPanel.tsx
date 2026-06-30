import { useEffect, useRef, useState } from 'react'
import type { ExecutionState } from '@/lib/agentloop-types'

const statusConfig = {
  idle: { label: 'Idle', color: 'bg-gray-400', textColor: 'text-gray-600' },
  streaming: { label: 'Running', color: 'bg-blue-500', textColor: 'text-blue-700' },
  done: { label: 'Done', color: 'bg-green-500', textColor: 'text-green-700' },
  error: { label: 'Error', color: 'bg-red-500', textColor: 'text-red-700' },
  preempted: { label: 'Interrupted', color: 'bg-amber-500', textColor: 'text-amber-700' },
} as const

export default function ExecutionStatusPanel({ state }: { state: ExecutionState }) {
  const [elapsed, setElapsed] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval>>()

  useEffect(() => {
    if (state.status === 'streaming' && state.startTime) {
      setElapsed(Date.now() - state.startTime)
      intervalRef.current = setInterval(() => setElapsed(Date.now() - state.startTime!), 200)
    } else {
      if (intervalRef.current) clearInterval(intervalRef.current)
      if (state.startTime && state.endTime) setElapsed(state.endTime - state.startTime)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [state.status, state.startTime, state.endTime])

  if (state.status === 'idle') return null

  const cfg = statusConfig[state.status]
  const secs = (elapsed / 1000).toFixed(1)

  return (
    <div className="flex items-center gap-4 px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-lg mb-4">
      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${cfg.textColor}`}>
        <span className={`w-2 h-2 rounded-full ${cfg.color} ${state.status === 'streaming' ? 'animate-pulse' : ''}`} />
        {cfg.label}
      </span>
      <span className="text-xs text-gray-500">
        Turn <span className="font-semibold text-gray-700">{state.currentTurn}</span>/{state.maxTurns}
      </span>
      <span className="text-xs text-gray-500">
        {secs}s
      </span>
    </div>
  )
}
