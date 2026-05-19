import { useState } from 'react'

interface ToolCallBlockProps {
  toolName: string
  args?: string
  result?: string
  status?: 'calling' | 'done' | 'error'
}

export default function ToolCallBlock({ toolName, args, result, status = 'done' }: ToolCallBlockProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="my-2 rounded-lg border border-gray-200 overflow-hidden text-xs">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 bg-gray-50 hover:bg-gray-100 transition-colors text-left"
      >
        {status === 'calling' ? (
          <span className="w-3.5 h-3.5 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
        ) : status === 'error' ? (
          <span className="text-red-500">✕</span>
        ) : (
          <span className="text-green-500">✓</span>
        )}
        <span className="font-mono text-gray-700">{toolName}</span>
        <span className={`ml-auto transform transition-transform ${expanded ? 'rotate-180' : ''}`}>
          ▾
        </span>
      </button>

      {expanded && (
        <div className="border-t border-gray-200">
          {args && (
            <div className="px-3 py-2 border-b border-gray-100">
              <div className="text-[10px] text-gray-400 uppercase mb-1">参数</div>
              <pre className="text-gray-600 whitespace-pre-wrap font-mono bg-gray-50 rounded p-2 overflow-x-auto">
                {args}
              </pre>
            </div>
          )}
          {result && (
            <div className="px-3 py-2">
              <div className="text-[10px] text-gray-400 uppercase mb-1">结果</div>
              <pre className="text-gray-600 whitespace-pre-wrap font-mono bg-gray-50 rounded p-2 overflow-x-auto max-h-40 overflow-y-auto">
                {result}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
