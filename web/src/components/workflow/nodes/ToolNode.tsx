import { Handle, Position, type NodeProps } from '@xyflow/react'

interface ToolNodeData {
  label?: string
  tool?: string
  [key: string]: unknown
}

export default function ToolNode({ data, selected }: NodeProps) {
  const d = data as ToolNodeData
  const toolRef = d.tool || 'No tool set'

  return (
    <div
      className={`rounded-lg border-2 bg-green-50 px-3 py-2 min-w-[160px] shadow-sm transition-shadow ${
        selected ? 'border-green-500 shadow-md' : 'border-green-200'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-green-400" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-base">🔧</span>
        <span className="text-xs font-semibold text-green-800">Tool Call</span>
      </div>
      <p className="text-xs text-green-600 font-mono break-words">{toolRef}</p>
      <Handle type="source" position={Position.Bottom} className="!bg-green-400" />
    </div>
  )
}
