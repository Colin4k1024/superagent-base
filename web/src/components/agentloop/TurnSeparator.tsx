import { useEffect, useRef, useState } from 'react'

interface TurnSeparatorProps {
  turn: number
  total: number
  startTime: number
  endTime?: number
  isStreaming: boolean
}

export default function TurnSeparator({ turn, total, startTime, endTime, isStreaming }: TurnSeparatorProps) {
  const [elapsed, setElapsed] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval>>()

  useEffect(() => {
    if (isStreaming) {
      setElapsed(Date.now() - startTime)
      intervalRef.current = setInterval(() => setElapsed(Date.now() - startTime), 100)
    } else {
      if (intervalRef.current) clearInterval(intervalRef.current)
      if (endTime) setElapsed(endTime - startTime)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [isStreaming, startTime, endTime])

  const secs = (elapsed / 1000).toFixed(1)

  return (
    <div className="flex items-center gap-3 my-4 animate-[fadeInSlideDown_200ms_ease-out]">
      <div className="flex-1 border-t border-dashed border-gray-300" />
      <span className="shrink-0 inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-200">
        <span className="font-semibold">Turn {turn}/{total}</span>
        <span className="text-blue-400">|</span>
        <span>{secs}s</span>
        {isStreaming && (
          <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
        )}
      </span>
      <div className="flex-1 border-t border-dashed border-gray-300" />
    </div>
  )
}
