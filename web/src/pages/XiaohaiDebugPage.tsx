import { useState, useRef, useEffect } from 'react'
import { agentsApi, type Agent } from '../lib/api'
import Header from '../components/Header'

interface LogEntry {
  time: string
  type: 'sent' | 'received' | 'error' | 'info'
  data: string
}

export default function XiaohaiDebugPage() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [selectedAgent, setSelectedAgent] = useState('')
  const [userQuery, setUserQuery] = useState('你好，介绍一下你自己')
  const [sessionId, setSessionId] = useState(() => `debug-${Date.now()}`)
  const [terminalType, setTerminalType] = useState('PC')
  const [apiKey, setApiKey] = useState('')
  const [mode, setMode] = useState<'stream' | 'chat'>('stream')
  const [isLoading, setIsLoading] = useState(false)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [fullResponse, setFullResponse] = useState('')
  const logsEndRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    agentsApi.list().then((list) => {
      setAgents(list)
      if (list.length > 0) setSelectedAgent(list[0].name)
    }).catch(() => {})
  }, [])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const addLog = (type: LogEntry['type'], data: string) => {
    setLogs((prev) => [...prev, { time: new Date().toLocaleTimeString(), type, data }])
  }

  const handleStop = () => {
    abortRef.current?.abort()
    abortRef.current = null
    setIsLoading(false)
    addLog('info', '--- 已停止 ---')
  }

  const handleSend = async () => {
    if (!selectedAgent || !userQuery.trim() || isLoading) return

    setIsLoading(true)
    setFullResponse('')
    const url = `/api/v1/xiaohai/${mode}/${selectedAgent}`
    const body = {
      userQuery: userQuery.trim(),
      userToken: 'debug-user',
      terminalType,
      sessionId,
      hisMsg: [],
      fileList: [],
      ext_param: {},
    }

    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (apiKey) headers['Api-Key'] = apiKey

    addLog('info', `--- ${mode === 'stream' ? '流式' : '非流式'}请求 → ${url} ---`)
    addLog('sent', JSON.stringify(body, null, 2))

    if (mode === 'chat') {
      // Non-streaming
      try {
        const res = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) })
        const data = await res.json()
        addLog('received', JSON.stringify(data, null, 2))
        setFullResponse(JSON.stringify(data, null, 2))
      } catch (err) {
        addLog('error', String(err))
      }
      setIsLoading(false)
      return
    }

    // Streaming
    const controller = new AbortController()
    abortRef.current = controller

    try {
      const res = await fetch(url, {
        method: 'POST',
        headers,
        body: JSON.stringify(body),
        signal: controller.signal,
      })

      if (!res.ok) {
        const errText = await res.text()
        addLog('error', `HTTP ${res.status}: ${errText}`)
        setIsLoading(false)
        return
      }

      const reader = res.body?.getReader()
      const decoder = new TextDecoder()
      if (!reader) { addLog('error', 'No readable stream'); setIsLoading(false); return }

      let buffer = ''
      let accumulated = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const raw = line.slice(6)
            try {
              const evt = JSON.parse(raw)
              addLog('received', JSON.stringify(evt))
              // Accumulate answer content
              if (evt.type === 'answer' && evt.data?.content) {
                accumulated += evt.data.content
                setFullResponse(accumulated)
              }
            } catch {
              addLog('received', raw)
            }
          }
        }
      }
      addLog('info', '--- 流结束 ---')
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        addLog('error', String(err))
      }
    }
    setIsLoading(false)
    abortRef.current = null
  }

  return (
    <div className="flex flex-col h-full">
      <Header title="小海接口调试" />

      <div className="flex-1 overflow-hidden flex">
        {/* Left: Config Panel */}
        <div className="w-80 border-r border-gray-200 p-4 overflow-y-auto space-y-4 bg-gray-50">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Agent</label>
            <select
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-1.5 bg-white"
              value={selectedAgent}
              onChange={(e) => setSelectedAgent(e.target.value)}
            >
              {agents.map((a) => (
                <option key={a.name} value={a.name}>{a.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">模式</label>
            <div className="flex gap-2">
              <button
                className={`flex-1 text-xs px-3 py-1.5 rounded-lg border transition-colors ${mode === 'stream' ? 'bg-blue-600 text-white border-blue-600' : 'bg-white border-gray-300 text-gray-700'}`}
                onClick={() => setMode('stream')}
              >流式 (SSE)</button>
              <button
                className={`flex-1 text-xs px-3 py-1.5 rounded-lg border transition-colors ${mode === 'chat' ? 'bg-blue-600 text-white border-blue-600' : 'bg-white border-gray-300 text-gray-700'}`}
                onClick={() => setMode('chat')}
              >非流式 (JSON)</button>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Api-Key (可选)</label>
            <input
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-1.5"
              placeholder="留空则不发送"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Session ID</label>
            <input
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-1.5 font-mono"
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Terminal Type</label>
            <select
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-1.5 bg-white"
              value={terminalType}
              onChange={(e) => setTerminalType(e.target.value)}
            >
              <option value="PC">PC</option>
              <option value="MOBILE">MOBILE</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">userQuery</label>
            <textarea
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-2 resize-none"
              rows={3}
              value={userQuery}
              onChange={(e) => setUserQuery(e.target.value)}
            />
          </div>

          <div className="flex gap-2">
            <button
              className="flex-1 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              onClick={handleSend}
              disabled={isLoading || !selectedAgent || !userQuery.trim()}
            >
              {isLoading ? '请求中...' : '发送请求'}
            </button>
            {isLoading && (
              <button
                className="px-4 py-2 bg-red-500 text-white text-sm font-medium rounded-lg hover:bg-red-600 transition-colors"
                onClick={handleStop}
              >停止</button>
            )}
          </div>

          <button
            className="w-full px-3 py-1.5 text-xs text-gray-500 border border-gray-300 rounded-lg hover:bg-gray-100 transition-colors"
            onClick={() => { setLogs([]); setFullResponse('') }}
          >清空日志</button>

          <div className="text-[10px] text-gray-400 space-y-0.5">
            <div>接口: POST /api/v1/xiaohai/{'{mode}'}/{'{agent_id}'}</div>
            <div>规范: 集团IT智能体输出规范 v1.0.0</div>
          </div>
        </div>

        {/* Right: Logs + Response */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Response preview */}
          {fullResponse && (
            <div className="border-b border-gray-200 p-4 bg-white max-h-48 overflow-y-auto">
              <div className="text-xs font-medium text-gray-500 mb-1">Response Content</div>
              <pre className="text-sm text-gray-800 whitespace-pre-wrap font-mono">{fullResponse}</pre>
            </div>
          )}

          {/* Event log */}
          <div className="flex-1 overflow-y-auto p-4 bg-gray-900 font-mono text-xs">
            {logs.length === 0 && (
              <div className="text-gray-500 text-center mt-8">发送请求后，SSE 事件将在此实时显示</div>
            )}
            {logs.map((log, i) => (
              <div key={i} className="mb-1">
                <span className="text-gray-500">[{log.time}] </span>
                <span className={
                  log.type === 'sent' ? 'text-blue-400' :
                  log.type === 'received' ? 'text-green-400' :
                  log.type === 'error' ? 'text-red-400' :
                  'text-yellow-400'
                }>
                  {log.type === 'sent' ? '>>> ' : log.type === 'received' ? '<<< ' : log.type === 'error' ? 'ERR ' : '--- '}
                </span>
                <span className={
                  log.type === 'error' ? 'text-red-300' : 'text-gray-300'
                }>{log.data}</span>
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        </div>
      </div>
    </div>
  )
}
