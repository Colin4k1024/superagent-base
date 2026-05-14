import '@xyflow/react/dist/style.css'

import { useState, useCallback, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  useReactFlow,
  type Node,
  type Edge,
  type Connection,
} from '@xyflow/react'
import { toast } from 'sonner'
import { agentAdminApi } from '../lib/api'
import { nodeTypes } from '../components/workflow/nodes'
import NodePalette from '../components/workflow/NodePalette'
import PropertyPanel from '../components/workflow/PropertyPanel'

// ---- Types ----
interface WorkflowNode {
  id: string
  type: string
  agent?: string
  tool?: string
  prompt?: string
  code?: string
  language?: string
  condition?: string
  input_mapping?: Record<string, string>
}

interface WorkflowEdge {
  from: string
  to: string
  condition?: string
}

interface WorkflowSpec {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  variables?: Array<{ name: string; from: string }>
}

// ---- Convert backend spec to React Flow nodes/edges ----
function specToFlow(spec: WorkflowSpec): { nodes: Node[]; edges: Edge[] } {
  const rfNodes: Node[] = (spec.nodes ?? []).map((n, idx) => {
    const col = idx % 3
    const row = Math.floor(idx / 3)
    return {
      id: n.id,
      type: n.type,
      position: { x: col * 250, y: row * 150 },
      data: {
        label: n.id,
        prompt: n.prompt,
        agent: n.agent,
        tool: n.tool,
        code: n.code,
        language: n.language,
        condition: n.condition,
        input_mapping: n.input_mapping ?? {},
      },
    }
  })

  const rfEdges: Edge[] = (spec.edges ?? []).map((e, idx) => ({
    id: `e-${idx}-${e.from}-${e.to}`,
    source: e.from === 'START' ? '__start__' : e.from,
    target: e.to === 'END' ? '__end__' : e.to,
    label: e.condition,
  }))

  return { nodes: rfNodes, edges: rfEdges }
}

// ---- Inner canvas component (uses useReactFlow so must be inside Provider) ----
function WorkflowCanvas({ agentName }: { agentName: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { screenToFlowPosition } = useReactFlow()

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [saving, setSaving] = useState(false)
  const [rawYaml, setRawYaml] = useState('')

  // Load agent YAML on mount
  useEffect(() => {
    if (!agentName) return
    agentAdminApi
      .get(agentName)
      .then(({ yaml, agent }) => {
        setRawYaml(yaml)
        const spec = extractWorkflowSpec(agent)
        if (spec) {
          const { nodes: rfNodes, edges: rfEdges } = specToFlow(spec)
          setNodes(rfNodes)
          setEdges(rfEdges)
        }
      })
      .catch((err: unknown) => {
        toast.error(`Failed to load agent: ${err instanceof Error ? err.message : String(err)}`)
      })
  }, [agentName, setNodes, setEdges])

  const onConnect = useCallback(
    (connection: Connection) => setEdges((eds) => addEdge(connection, eds)),
    [setEdges],
  )

  const onDragOver = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const onDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault()
      const type = event.dataTransfer.getData('application/reactflow')
      if (!type) return
      const position = screenToFlowPosition({ x: event.clientX, y: event.clientY })
      const newNode: Node = {
        id: `${type}_${Date.now()}`,
        type,
        position,
        data: { label: type, input_mapping: {} },
      }
      setNodes((nds) => [...nds, newNode])
    },
    [screenToFlowPosition, setNodes],
  )

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node)
  }, [])

  const onPaneClick = useCallback(() => {
    setSelectedNode(null)
  }, [])

  function handleNodeUpdate(nodeId: string, data: Record<string, unknown>) {
    setNodes((nds) =>
      nds.map((n) => (n.id === nodeId ? { ...n, data: { ...n.data, ...data } } : n)),
    )
    // Keep selectedNode in sync
    setSelectedNode((prev) =>
      prev?.id === nodeId ? { ...prev, data: { ...prev.data, ...data } } : prev,
    )
  }

  function handleNodeDelete(nodeId: string) {
    setNodes((nds) => nds.filter((n) => n.id !== nodeId))
    setEdges((eds) => eds.filter((e) => e.source !== nodeId && e.target !== nodeId))
    setSelectedNode(null)
  }

  async function handleSave() {
    setSaving(true)
    try {
      const updatedYaml = rebuildYamlWithFlow(rawYaml, nodes, edges)
      await agentAdminApi.update(agentName, updatedYaml)
      setRawYaml(updatedYaml)
      toast.success(t('workflow.save'))
    } catch (err) {
      toast.error(`Save failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(false)
    }
  }

  function handleExportYaml() {
    const updatedYaml = rebuildYamlWithFlow(rawYaml, nodes, edges)
    const blob = new Blob([updatedYaml], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${agentName}.yaml`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Top bar */}
      <div className="flex items-center gap-3 px-4 py-2.5 bg-white border-b border-gray-200 flex-shrink-0">
        <button
          onClick={() => navigate('/agents')}
          className="text-sm text-gray-500 hover:text-gray-700 flex items-center gap-1 transition-colors"
        >
          {t('workflow.backToAgent')}
        </button>
        <span className="text-gray-300">|</span>
        <span className="text-sm font-semibold text-gray-800 truncate">{agentName}</span>
        <span className="text-xs bg-orange-100 text-orange-700 rounded px-2 py-0.5 font-medium">workflow</span>
        <div className="ml-auto flex items-center gap-2">
          <button
            onClick={handleExportYaml}
            className="text-xs border border-gray-200 rounded px-3 py-1.5 text-gray-600 hover:bg-gray-50 transition-colors"
          >
            {t('workflow.export')}
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="text-xs bg-blue-600 text-white rounded px-3 py-1.5 font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? t('editor.saving') : t('workflow.save')}
          </button>
        </div>
      </div>

      {/* Three-panel layout */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Node palette */}
        <div className="w-[200px] flex-shrink-0">
          <NodePalette />
        </div>

        {/* Center: canvas */}
        <div className="flex-1 relative">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            nodeTypes={nodeTypes}
            fitView
          >
            <Background />
            <Controls />
            <MiniMap />
          </ReactFlow>
        </div>

        {/* Right: Property panel (conditional) */}
        {selectedNode && (
          <div className="w-[280px] flex-shrink-0">
            <PropertyPanel
              node={selectedNode}
              onUpdate={handleNodeUpdate}
              onDelete={handleNodeDelete}
            />
          </div>
        )}
      </div>
    </div>
  )
}

// ---- Page wrapper with ReactFlowProvider ----
export default function WorkflowEditorPage() {
  const { name } = useParams<{ name: string }>()

  if (!name) {
    return <div className="p-8 text-sm text-gray-500">No agent name in URL.</div>
  }

  return (
    <ReactFlowProvider>
      <WorkflowCanvas agentName={name} />
    </ReactFlowProvider>
  )
}

// ---- Helpers ----

function extractWorkflowSpec(agent: unknown): WorkflowSpec | null {
  if (!agent || typeof agent !== 'object') return null
  const a = agent as Record<string, unknown>
  const spec = a['spec'] as Record<string, unknown> | undefined
  if (!spec) return null
  const wf = spec['workflow'] as WorkflowSpec | undefined
  return wf ?? null
}

/**
 * Naive YAML rebuilder: replaces the workflow block in the raw YAML string.
 * This avoids a full YAML parse/serialize roundtrip that could mangle comments.
 * For simple cases it injects a JSON-compatible YAML snippet.
 */
function rebuildYamlWithFlow(yaml: string, nodes: Node[], edges: Edge[]): string {
  const wfNodes = nodes.map((n) => {
    const d = n.data as Record<string, unknown>
    const node: Record<string, unknown> = { id: n.id, type: n.type }
    if (d.prompt) node.prompt = d.prompt
    if (d.agent) node.agent = d.agent
    if (d.tool) node.tool = d.tool
    if (d.code) node.code = d.code
    if (d.language) node.language = d.language
    if (d.condition) node.condition = d.condition
    if (d.input_mapping && Object.keys(d.input_mapping as object).length > 0) {
      node.input_mapping = d.input_mapping
    }
    return node
  })

  const wfEdges = edges.map((e) => {
    const from = e.source === '__start__' ? 'START' : e.source
    const to = e.target === '__end__' ? 'END' : e.target
    const edge: Record<string, unknown> = { from, to }
    if (e.label) edge.condition = e.label
    return edge
  })

  const workflowYaml = toYamlWorkflow({ nodes: wfNodes, edges: wfEdges })

  // Replace existing workflow block or append after spec:
  if (/^\s*workflow:/m.test(yaml)) {
    // Remove old workflow block (until next top-level key or EOF)
    const cleaned = yaml.replace(/^(\s*)workflow:[\s\S]*?(?=^\1\w|\Z)/m, '')
    return cleaned + workflowYaml
  }
  return yaml + '\n' + workflowYaml
}

function toYamlWorkflow(spec: { nodes: unknown[]; edges: unknown[] }): string {
  const indent = (s: string, n: number) =>
    s
      .split('\n')
      .map((l) => ' '.repeat(n) + l)
      .join('\n')

  const nodeLines = spec.nodes
    .map((n) => indent('- ' + JSON.stringify(n), 4))
    .join('\n')
  const edgeLines = spec.edges
    .map((e) => indent('- ' + JSON.stringify(e), 4))
    .join('\n')

  return `  workflow:\n    nodes:\n${nodeLines || '      []'}\n    edges:\n${edgeLines || '      []'}\n`
}
