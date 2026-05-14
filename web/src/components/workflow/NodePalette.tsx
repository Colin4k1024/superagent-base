interface PaletteItem {
  type: string
  label: string
  icon: string
  colorClass: string
}

const PALETTE_ITEMS: PaletteItem[] = [
  { type: 'llm_call',   label: 'LLM Call',   icon: '🤖', colorClass: 'border-purple-200 bg-purple-50 hover:bg-purple-100' },
  { type: 'agent_call', label: 'Agent Call', icon: '🔗', colorClass: 'border-blue-200 bg-blue-50 hover:bg-blue-100' },
  { type: 'tool_call',  label: 'Tool Call',  icon: '🔧', colorClass: 'border-green-200 bg-green-50 hover:bg-green-100' },
  { type: 'code',       label: 'Code',       icon: '💻', colorClass: 'border-orange-200 bg-orange-50 hover:bg-orange-100' },
  { type: 'condition',  label: 'Condition',  icon: '❓', colorClass: 'border-yellow-200 bg-yellow-50 hover:bg-yellow-100' },
]

export default function NodePalette() {
  function onDragStart(event: React.DragEvent<HTMLDivElement>, nodeType: string) {
    event.dataTransfer.setData('application/reactflow', nodeType)
    event.dataTransfer.effectAllowed = 'move'
  }

  return (
    <div className="flex flex-col h-full bg-white border-r border-gray-200">
      <div className="px-3 py-3 border-b border-gray-100">
        <h2 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Nodes</h2>
      </div>
      <div className="flex-1 overflow-auto p-3 flex flex-col gap-2">
        {PALETTE_ITEMS.map((item) => (
          <div
            key={item.type}
            draggable
            onDragStart={(e) => onDragStart(e, item.type)}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-grab active:cursor-grabbing select-none transition-colors ${item.colorClass}`}
          >
            <span className="text-base">{item.icon}</span>
            <span className="text-xs font-medium text-gray-700">{item.label}</span>
          </div>
        ))}
      </div>
      <div className="px-3 py-2 border-t border-gray-100">
        <p className="text-xs text-gray-400">Drag to canvas</p>
      </div>
    </div>
  )
}
