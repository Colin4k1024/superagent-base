import { Handle, Position, type NodeProps } from '@xyflow/react'

interface AgentNodeData {
  label?: string
  agent?: string
  [key: string]: unknown
}

export default function AgentNode({ data, selected }: NodeProps) {
  const d = data as AgentNodeData
  const agentName = d.agent || 'No agent set'

  return (
    <div
      className={`rounded-lg border-2 bg-blue-50 px-3 py-2 min-w-[160px] shadow-sm transition-shadow ${
        selected ? 'border-blue-500 shadow-md' : 'border-blue-200'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-blue-400" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-base">🔗</span>
        <span className="text-xs font-semibold text-blue-800">Agent Call</span>
      </div>
      <p className="text-xs text-blue-600 font-mono break-words">{agentName}</p>
      <Handle type="source" position={Position.Bottom} className="!bg-blue-400" />
    </div>
  )
}
