import { Handle, Position, type NodeProps } from '@xyflow/react'

interface ConditionNodeData {
  label?: string
  condition?: string
  [key: string]: unknown
}

export default function ConditionNode({ data, selected }: NodeProps) {
  const d = data as ConditionNodeData
  const condition = d.condition || 'No condition set'

  return (
    <div
      className={`rounded-lg border-2 bg-yellow-50 px-3 py-2 min-w-[180px] shadow-sm transition-shadow ${
        selected ? 'border-yellow-500 shadow-md' : 'border-yellow-200'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-yellow-400" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-base">❓</span>
        <span className="text-xs font-semibold text-yellow-800">Condition</span>
      </div>
      <p className="text-xs text-yellow-700 font-mono break-words">{condition}</p>
      {/* true branch — bottom left */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="true"
        style={{ left: '30%' }}
        className="!bg-green-400"
      />
      {/* false branch — bottom right */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="false"
        style={{ left: '70%' }}
        className="!bg-red-400"
      />
      <div className="flex justify-between mt-2 px-1">
        <span className="text-xs text-green-600 font-medium">true</span>
        <span className="text-xs text-red-500 font-medium">false</span>
      </div>
    </div>
  )
}
