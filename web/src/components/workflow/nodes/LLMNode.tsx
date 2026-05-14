import { Handle, Position, type NodeProps } from '@xyflow/react'

interface LLMNodeData {
  label?: string
  prompt?: string
  [key: string]: unknown
}

export default function LLMNode({ data, selected }: NodeProps) {
  const d = data as LLMNodeData
  const prompt = d.prompt ?? ''
  const preview = prompt.length > 50 ? prompt.slice(0, 50) + '…' : prompt || 'No prompt set'

  return (
    <div
      className={`rounded-lg border-2 bg-purple-50 px-3 py-2 min-w-[160px] shadow-sm transition-shadow ${
        selected ? 'border-purple-500 shadow-md' : 'border-purple-200'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-purple-400" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-base">🤖</span>
        <span className="text-xs font-semibold text-purple-800">LLM Call</span>
      </div>
      <p className="text-xs text-purple-600 leading-relaxed break-words">{preview}</p>
      <Handle type="source" position={Position.Bottom} className="!bg-purple-400" />
    </div>
  )
}
