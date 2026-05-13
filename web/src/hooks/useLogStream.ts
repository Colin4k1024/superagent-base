import { useEffect, useRef, useState } from 'react'

export interface LogEntry {
  id: number
  level: string
  msg: string
  ts: string
}

export interface UseLogStreamResult {
  logs: LogEntry[]
  connected: boolean
  paused: boolean
  setPaused: (p: boolean) => void
  clear: () => void
}

const MAX_LOGS = 200

let _idCounter = 0

export function useLogStream(): UseLogStreamResult {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [connected, setConnected] = useState(false)
  const [paused, setPaused] = useState(false)
  const pausedRef = useRef(false)
  const esRef = useRef<EventSource | null>(null)

  // Keep pausedRef in sync so the EventSource callback can read current value
  useEffect(() => {
    pausedRef.current = paused
  }, [paused])

  useEffect(() => {
    const es = new EventSource('/api/v1/admin/logs')
    esRef.current = es

    es.addEventListener('open', () => {
      setConnected(true)
    })

    es.addEventListener('log', (event: MessageEvent) => {
      if (pausedRef.current) return
      try {
        const entry = JSON.parse(event.data) as { level: string; msg: string; ts: string }
        const logEntry: LogEntry = {
          id: ++_idCounter,
          level: entry.level ?? 'info',
          msg: entry.msg ?? '',
          ts: entry.ts ?? new Date().toISOString(),
        }
        setLogs((prev) => {
          const next = [...prev, logEntry]
          return next.length > MAX_LOGS ? next.slice(next.length - MAX_LOGS) : next
        })
      } catch {
        // Malformed JSON — skip
      }
    })

    es.addEventListener('error', () => {
      setConnected(false)
    })

    return () => {
      es.close()
      esRef.current = null
      setConnected(false)
    }
  }, [])

  function clear() {
    setLogs([])
  }

  return { logs, connected, paused, setPaused, clear }
}
