import { useState, useEffect } from 'react'
import type { Node } from '@xyflow/react'

interface NodeData {
  label?: string
  prompt?: string
  agent?: string
  tool?: string
  code?: string
  language?: string
  condition?: string
  input_mapping?: Record<string, string>
  [key: string]: unknown
}

interface Props {
  node: Node
  onUpdate: (nodeId: string, data: NodeData) => void
  onDelete: (nodeId: string) => void
}

export default function PropertyPanel({ node, onUpdate, onDelete }: Props) {
  const [form, setForm] = useState<NodeData>({ ...(node.data as NodeData) })
  const [mappingKeys, setMappingKeys] = useState<string[]>(
    Object.keys((node.data as NodeData).input_mapping ?? {}),
  )

  // Sync when a different node is selected
  useEffect(() => {
    const d = node.data as NodeData
    setForm({ ...d })
    setMappingKeys(Object.keys(d.input_mapping ?? {}))
  }, [node.id])

  function set<K extends keyof NodeData>(key: K, value: NodeData[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  function setMappingValue(k: string, v: string) {
    setForm((prev) => ({
      ...prev,
      input_mapping: { ...(prev.input_mapping ?? {}), [k]: v },
    }))
  }

  function renameMappingKey(oldKey: string, newKey: string) {
    setMappingKeys((prev) => prev.map((k) => (k === oldKey ? newKey : k)))
    setForm((prev) => {
      const m = { ...(prev.input_mapping ?? {}) }
      const val = m[oldKey] ?? ''
      delete m[oldKey]
      m[newKey] = val
      return { ...prev, input_mapping: m }
    })
  }

  function addMapping() {
    const k = `var_${Date.now()}`
    setMappingKeys((prev) => [...prev, k])
    setForm((prev) => ({ ...prev, input_mapping: { ...(prev.input_mapping ?? {}), [k]: '' } }))
  }

  function removeMapping(k: string) {
    setMappingKeys((prev) => prev.filter((x) => x !== k))
    setForm((prev) => {
      const m = { ...(prev.input_mapping ?? {}) }
      delete m[k]
      return { ...prev, input_mapping: m }
    })
  }

  function handleApply() {
    onUpdate(node.id, form)
  }

  const type = node.type ?? 'unknown'
  const inputCls =
    'w-full rounded border border-gray-200 px-2 py-1.5 text-xs focus:outline-none focus:border-blue-400'
  const textareaCls = `${inputCls} font-mono resize-none`

  return (
    <div className="flex flex-col h-full bg-white border-l border-gray-200 overflow-hidden">
      <div className="px-4 py-3 border-b border-gray-100 flex-shrink-0">
        <div className="text-xs font-semibold text-gray-700 uppercase tracking-wide">{type.replace('_', ' ')}</div>
        <div className="text-xs text-gray-400 font-mono mt-0.5">{node.id}</div>
      </div>

      <div className="flex-1 overflow-auto p-4 flex flex-col gap-4">
        {/* Type-specific fields */}
        {type === 'llm_call' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500 font-medium">Prompt</span>
            <textarea
              rows={6}
              className={textareaCls}
              value={form.prompt ?? ''}
              onChange={(e) => set('prompt', e.target.value)}
            />
          </label>
        )}

        {type === 'agent_call' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500 font-medium">Agent Name</span>
            <input
              type="text"
              className={inputCls}
              value={form.agent ?? ''}
              onChange={(e) => set('agent', e.target.value)}
              placeholder="e.g. research-agent"
            />
          </label>
        )}

        {type === 'tool_call' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500 font-medium">Tool Reference</span>
            <input
              type="text"
              className={inputCls}
              value={form.tool ?? ''}
              onChange={(e) => set('tool', e.target.value)}
              placeholder="e.g. builtin/web_search"
            />
          </label>
        )}

        {type === 'code' && (
          <>
            <label className="flex flex-col gap-1">
              <span className="text-xs text-gray-500 font-medium">Language</span>
              <select
                className={inputCls}
                value={form.language ?? 'python'}
                onChange={(e) => set('language', e.target.value)}
              >
                <option value="python">Python</option>
                <option value="javascript">JavaScript</option>
                <option value="bash">Bash</option>
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs text-gray-500 font-medium">Code</span>
              <textarea
                rows={8}
                className={textareaCls}
                value={form.code ?? ''}
                onChange={(e) => set('code', e.target.value)}
                placeholder="# your code here"
              />
            </label>
          </>
        )}

        {type === 'condition' && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500 font-medium">Expression</span>
            <input
              type="text"
              className={inputCls}
              value={form.condition ?? ''}
              onChange={(e) => set('condition', e.target.value)}
              placeholder="e.g. result.score > 0.8"
            />
          </label>
        )}

        {/* Input mappings */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-500 font-medium">Input Mapping</span>
            <button
              onClick={addMapping}
              className="text-xs text-blue-600 hover:text-blue-700 font-medium"
            >
              + Add
            </button>
          </div>
          {mappingKeys.map((k) => (
            <div key={k} className="flex gap-1 items-center">
              <input
                type="text"
                className="flex-1 rounded border border-gray-200 px-2 py-1 text-xs focus:outline-none focus:border-blue-400"
                value={k}
                onChange={(e) => renameMappingKey(k, e.target.value)}
                placeholder="key"
              />
              <span className="text-gray-300 text-xs">→</span>
              <input
                type="text"
                className="flex-1 rounded border border-gray-200 px-2 py-1 text-xs focus:outline-none focus:border-blue-400"
                value={(form.input_mapping ?? {})[k] ?? ''}
                onChange={(e) => setMappingValue(k, e.target.value)}
                placeholder="node_id.output"
              />
              <button
                onClick={() => removeMapping(k)}
                className="text-gray-300 hover:text-red-400 text-xs px-1"
                aria-label="Remove mapping"
              >
                ✕
              </button>
            </div>
          ))}
          {mappingKeys.length === 0 && (
            <p className="text-xs text-gray-400 italic">No mappings</p>
          )}
        </div>
      </div>

      {/* Footer buttons */}
      <div className="flex-shrink-0 px-4 py-3 border-t border-gray-100 flex flex-col gap-2">
        <button
          onClick={handleApply}
          className="w-full rounded bg-blue-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-blue-700 transition-colors"
        >
          Apply
        </button>
        <button
          onClick={() => onDelete(node.id)}
          className="w-full rounded border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
        >
          Delete Node
        </button>
      </div>
    </div>
  )
}
