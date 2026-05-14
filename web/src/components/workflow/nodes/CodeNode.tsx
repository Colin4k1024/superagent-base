import { Handle, Position, type NodeProps } from '@xyflow/react'

interface CodeNodeData {
  label?: string
  language?: string
  code?: string
  [key: string]: unknown
}

export default function CodeNode({ data, selected }: NodeProps) {
  const d = data as CodeNodeData
  const language = d.language || 'python'
  const code = d.code ?? ''
  const firstLine = code.split('\n')[0] || 'No code yet'

  return (
    <div
      className={`rounded-lg border-2 bg-orange-50 px-3 py-2 min-w-[160px] shadow-sm transition-shadow ${
        selected ? 'border-orange-500 shadow-md' : 'border-orange-200'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-orange-400" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-base">💻</span>
        <span className="text-xs font-semibold text-orange-800">Code</span>
        <span className="ml-auto text-xs bg-orange-200 text-orange-700 rounded px-1">{language}</span>
      </div>
      <p className="text-xs text-orange-600 font-mono truncate">{firstLine}</p>
      <Handle type="source" position={Position.Bottom} className="!bg-orange-400" />
    </div>
  )
}
