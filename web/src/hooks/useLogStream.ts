import { useEffect, useRef, useState, useCallback } from 'react'

export interface LogEntry {
  id: string
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
const RECONNECT_BASE_MS = 1000
const RECONNECT_MAX_MS = 30000

export function useLogStream(): UseLogStreamResult {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [connected, setConnected] = useState(false)
  const [paused, setPaused] = useState(false)
  const pausedRef = useRef(false)
  const esRef = useRef<EventSource | null>(null)
  const reconnectDelayRef = useRef(RECONNECT_BASE_MS)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const idCounterRef = useRef(0)
  const mountedRef = useRef(true)

  // Keep pausedRef in sync so the EventSource callback can read current value.
  useEffect(() => {
    pausedRef.current = paused
  }, [paused])

  const connect = useCallback(() => {
    if (!mountedRef.current) return

    const es = new EventSource('/api/v1/admin/logs')
    esRef.current = es

    es.addEventListener('open', () => {
      if (!mountedRef.current) return
      setConnected(true)
      reconnectDelayRef.current = RECONNECT_BASE_MS // reset backoff on success
    })

    es.addEventListener('log', (event: MessageEvent) => {
      if (!mountedRef.current || pausedRef.current) return
      try {
        const entry = JSON.parse(event.data) as { level: string; msg: string; ts: string }
        const logEntry: LogEntry = {
          id: `log-${++idCounterRef.current}`,
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
      if (!mountedRef.current) return
      setConnected(false)
      es.close()
      esRef.current = null

      // Reconnect with exponential backoff.
      const delay = reconnectDelayRef.current
      reconnectDelayRef.current = Math.min(delay * 2, RECONNECT_MAX_MS)
      reconnectTimerRef.current = setTimeout(() => {
        if (mountedRef.current) connect()
      }, delay)
    })
  }, [])

  useEffect(() => {
    mountedRef.current = true
    connect()

    return () => {
      mountedRef.current = false
      if (esRef.current) {
        esRef.current.close()
        esRef.current = null
      }
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      setConnected(false)
    }
  }, [connect])

  function clear() {
    setLogs([])
  }

  return { logs, connected, paused, setPaused, clear }
}
