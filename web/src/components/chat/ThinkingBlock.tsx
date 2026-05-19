import { useState } from 'react'

interface ThinkingBlockProps {
  content: string
  steps?: string[]
  isStreaming?: boolean
}

export default function ThinkingBlock({ content, steps, isStreaming }: ThinkingBlockProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="my-2">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 text-xs text-gray-500 hover:text-gray-700 transition-colors"
      >
        <span className={`transform transition-transform ${expanded ? 'rotate-90' : ''}`}>
          ▶
        </span>
        <span className="flex items-center gap-1.5">
          {isStreaming && (
            <span className="w-2 h-2 rounded-full bg-purple-400 animate-pulse" />
          )}
          思考过程
          {steps && steps.length > 0 && (
            <span className="text-gray-400">({steps.length} 步)</span>
          )}
        </span>
      </button>

      {expanded && (
        <div className="mt-2 pl-4 border-l-2 border-purple-200 space-y-2">
          {steps && steps.length > 0 ? (
            steps.map((step, i) => (
              <div key={i} className="flex gap-2 text-xs text-gray-600">
                <span className="flex-shrink-0 w-5 h-5 rounded-full bg-purple-100 text-purple-600 flex items-center justify-center text-[10px] font-medium">
                  {i + 1}
                </span>
                <span className="pt-0.5">{step}</span>
              </div>
            ))
          ) : (
            <div className="text-xs text-gray-500 whitespace-pre-wrap leading-relaxed bg-purple-50/50 rounded-lg p-3">
              {content}
              {isStreaming && (
                <span className="inline-block w-1 h-3 bg-purple-400 animate-pulse ml-0.5 align-middle" />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
