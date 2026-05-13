import { useEffect, useRef, useState } from 'react'
import { useLogStream, LogEntry } from '../../hooks/useLogStream'

type LogLevel = 'all' | 'info' | 'warn' | 'error' | 'debug'

function LevelBadge({ level }: { level: string }) {
  const l = level?.toLowerCase() ?? 'info'
  const base = 'inline-block w-12 text-center text-xs font-bold rounded px-1 py-0.5'
  if (l === 'error' || l === 'fatal' || l === 'err') {
    return <span className={`${base} bg-red-100 text-red-700`}>ERR</span>
  }
  if (l === 'warn' || l === 'warning') {
    return <span className={`${base} bg-yellow-100 text-yellow-700`}>WARN</span>
  }
  if (l === 'debug' || l === 'trace') {
    return <span className={`${base} bg-gray-100 text-gray-500`}>DBG</span>
  }
  return <span className={`${base} bg-green-100 text-green-700`}>INFO</span>
}

function matchesLevel(entry: LogEntry, filter: LogLevel): boolean {
  if (filter === 'all') return true
  const l = entry.level?.toLowerCase() ?? 'info'
  if (filter === 'error') return l === 'error' || l === 'fatal' || l === 'err'
  if (filter === 'warn') return l === 'warn' || l === 'warning'
  if (filter === 'debug') return l === 'debug' || l === 'trace'
  return l === 'info'
}

function formatTs(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString('en-US', { hour12: false })
  } catch {
    return ts
  }
}

export default function LogsPanel() {
  const { logs, connected, paused, setPaused, clear } = useLogStream()
  const [filter, setFilter] = useState<LogLevel>('all')
  const [autoScroll, setAutoScroll] = useState(true)
  const bottomRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const filtered = logs.filter((e) => matchesLevel(e, filter))

  // Auto-scroll to bottom when new logs arrive and not paused
  useEffect(() => {
    if (autoScroll && !paused && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [filtered.length, autoScroll, paused])

  // Detect manual scroll up → disable auto-scroll
  function handleScroll() {
    const el = containerRef.current
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 32
    setAutoScroll(atBottom)
  }

  const levels: LogLevel[] = ['all', 'error', 'warn', 'info', 'debug']

  return (
    <div className="flex flex-col h-full space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        {/* Connection status */}
        <div className="flex items-center gap-2">
          <span
            className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-400'}`}
          />
          <span className="text-xs text-gray-500">{connected ? 'Connected' : 'Disconnected'}</span>
          <span className="text-xs text-gray-400">({logs.length} total)</span>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {/* Level filter */}
          <div className="flex items-center gap-1 bg-gray-100 rounded-md p-0.5">
            {levels.map((l) => (
              <button
                key={l}
                onClick={() => setFilter(l)}
                className={`px-2 py-1 rounded text-xs font-medium transition-colors capitalize ${
                  filter === l
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {l}
              </button>
            ))}
          </div>

          {/* Pause / Resume */}
          <button
            onClick={() => setPaused(!paused)}
            className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
              paused
                ? 'bg-green-600 text-white hover:bg-green-700'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            {paused ? '▶ Resume' : '⏸ Pause'}
          </button>

          {/* Clear */}
          <button
            onClick={clear}
            className="px-3 py-1 rounded text-xs font-medium bg-gray-200 text-gray-700 hover:bg-gray-300 transition-colors"
          >
            Clear
          </button>

          {/* Auto-scroll toggle */}
          <button
            onClick={() => setAutoScroll((v) => !v)}
            className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
              autoScroll
                ? 'bg-blue-100 text-blue-700'
                : 'bg-gray-200 text-gray-500'
            }`}
          >
            ↓ Auto-scroll
          </button>
        </div>
      </div>

      {/* Log viewer */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 bg-gray-950 rounded-lg border border-gray-800 overflow-y-auto font-mono text-xs"
        style={{ minHeight: '400px', maxHeight: '600px' }}
      >
        {filtered.length === 0 ? (
          <div className="flex items-center justify-center h-40 text-gray-500">
            {connected ? 'Waiting for logs…' : 'Not connected to log stream'}
          </div>
        ) : (
          <table className="w-full border-collapse">
            <tbody>
              {filtered.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-gray-800/50 hover:bg-gray-900/50 transition-colors"
                >
                  <td className="pl-3 pr-2 py-1 text-gray-500 whitespace-nowrap w-20">
                    {formatTs(entry.ts)}
                  </td>
                  <td className="px-2 py-1 w-14">
                    <LevelBadge level={entry.level} />
                  </td>
                  <td className="px-2 py-1 text-gray-200 break-all">{entry.msg}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <div ref={bottomRef} />
      </div>

      {paused && (
        <p className="text-xs text-yellow-600 bg-yellow-50 border border-yellow-200 rounded px-3 py-1.5">
          Log stream paused — new entries are being dropped. Click Resume to continue.
        </p>
      )}
    </div>
  )
}
