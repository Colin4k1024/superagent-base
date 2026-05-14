// Pure utility: converts between backend WorkflowSpec and React Flow graph state.
// No React dependency — safe to import in any context.

import type { WorkflowSpec, WorkflowNode, WorkflowEdge } from './workflow-types'

// ─── React Flow shape stubs (keep in sync with @xyflow/react without importing it) ───

export interface RFNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: Record<string, unknown>
}

export interface RFEdge {
  id: string
  source: string
  target: string
  label?: string
  sourceHandle?: string
}

// Virtual node ids for START / END sentinels
const START_NODE_ID = '__start__'
const END_NODE_ID = '__end__'

// ─── workflowToGraph ────────────────────────────────────────────────────────

/**
 * Convert a backend WorkflowSpec into React Flow nodes + edges.
 * Inserts __start__ and __end__ sentinel nodes and runs a BFS auto-layout.
 */
export function workflowToGraph(spec: WorkflowSpec): { nodes: RFNode[]; edges: RFEdge[] } {
  if (!spec || !spec.nodes) {
    return { nodes: [], edges: [] }
  }

  // Build RF nodes from spec nodes
  const rfNodes: RFNode[] = spec.nodes.map((n) => {
    const { id, type, ...rest } = n
    return {
      id,
      type,
      position: { x: 0, y: 0 }, // layout fills this in below
      data: rest as Record<string, unknown>,
    }
  })

  // Add sentinel nodes
  rfNodes.unshift({
    id: START_NODE_ID,
    type: 'start',
    position: { x: 0, y: 0 },
    data: { label: 'START' },
  })
  rfNodes.push({
    id: END_NODE_ID,
    type: 'end',
    position: { x: 0, y: 0 },
    data: { label: 'END' },
  })

  // Build RF edges
  const rfEdges: RFEdge[] = (spec.edges ?? []).map((e) => {
    const source = e.from === 'START' ? START_NODE_ID : e.from
    const target = e.to === 'END' ? END_NODE_ID : e.to

    const rfEdge: RFEdge = {
      id: `${source}-${target}`,
      source,
      target,
    }

    if (e.condition) {
      rfEdge.label = e.condition
      // Condition nodes expose "true" / "false" handles
      if (e.condition === 'true' || e.condition === 'false') {
        rfEdge.sourceHandle = e.condition
      }
    }

    return rfEdge
  })

  // Apply layout
  const laid = autoLayout(rfNodes, rfEdges)
  return { nodes: laid, edges: rfEdges }
}

// ─── graphToWorkflow ─────────────────────────────────────────────────────────

/**
 * Convert React Flow nodes + edges back to a backend WorkflowSpec.
 * Filters out __start__ / __end__ meta-nodes.
 */
export function graphToWorkflow(nodes: RFNode[], edges: RFEdge[]): WorkflowSpec {
  const specNodes: WorkflowNode[] = nodes
    .filter((n) => n.id !== START_NODE_ID && n.id !== END_NODE_ID)
    .map((n) => {
      const node: WorkflowNode = { id: n.id, type: n.type }
      const d = n.data as Record<string, unknown>

      if (typeof d.agent === 'string') node.agent = d.agent
      if (typeof d.tool === 'string') node.tool = d.tool
      if (typeof d.prompt === 'string') node.prompt = d.prompt
      if (typeof d.code === 'string') node.code = d.code
      if (typeof d.language === 'string') node.language = d.language
      if (typeof d.condition === 'string') node.condition = d.condition
      if (d.input_mapping && typeof d.input_mapping === 'object') {
        node.input_mapping = d.input_mapping as Record<string, string>
      }

      return node
    })

  const specEdges: WorkflowEdge[] = edges.map((e) => {
    const from = e.source === START_NODE_ID ? 'START' : e.source
    const to = e.target === END_NODE_ID ? 'END' : e.target
    const edge: WorkflowEdge = { from, to }
    if (e.label) edge.condition = String(e.label)
    return edge
  })

  return { nodes: specNodes, edges: specEdges }
}

// ─── workflowToYaml ──────────────────────────────────────────────────────────

/**
 * Serialize a WorkflowSpec to a YAML string using simple string templates
 * (no external YAML library required).
 */
export function workflowToYaml(spec: WorkflowSpec): string {
  const lines: string[] = ['workflow:']

  if (!spec || spec.nodes.length === 0) {
    lines.push('  nodes: []')
    lines.push('  edges: []')
    return lines.join('\n')
  }

  lines.push('  nodes:')
  for (const node of spec.nodes) {
    lines.push(`    - id: "${node.id}"`)
    lines.push(`      type: ${node.type}`)
    if (node.agent) lines.push(`      agent: "${node.agent}"`)
    if (node.tool) lines.push(`      tool: "${node.tool}"`)
    if (node.prompt) lines.push(`      prompt: "${escapeYamlString(node.prompt)}"`)
    if (node.code) {
      lines.push('      code: |')
      for (const codeLine of node.code.split('\n')) {
        lines.push(`        ${codeLine}`)
      }
    }
    if (node.language) lines.push(`      language: ${node.language}`)
    if (node.condition) lines.push(`      condition: "${escapeYamlString(node.condition)}"`)
    if (node.input_mapping && Object.keys(node.input_mapping).length > 0) {
      lines.push('      input_mapping:')
      for (const [k, v] of Object.entries(node.input_mapping)) {
        lines.push(`        ${k}: "${v}"`)
      }
    }
  }

  lines.push('  edges:')
  for (const edge of spec.edges) {
    lines.push(`    - from: ${edge.from}`)
    lines.push(`      to: ${edge.to}`)
    if (edge.condition) lines.push(`      condition: "${edge.condition}"`)
  }

  if (spec.variables && spec.variables.length > 0) {
    lines.push('  variables:')
    for (const v of spec.variables) {
      lines.push(`    - name: "${v.name}"`)
      lines.push(`      from: "${v.from}"`)
    }
  }

  return lines.join('\n')
}

// ─── autoLayout ──────────────────────────────────────────────────────────────

const LAYER_X_GAP = 250
const LAYER_Y_GAP = 150

/**
 * BFS topological layout. Assigns x = layerIndex * LAYER_X_GAP,
 * y = nodePositionInLayer * LAYER_Y_GAP. Returns new nodes with updated positions.
 */
export function autoLayout(nodes: RFNode[], edges: RFEdge[]): RFNode[] {
  if (nodes.length === 0) return nodes

  // Build adjacency: inDegree + outgoing edges
  const inDegree = new Map<string, number>()
  const outgoing = new Map<string, string[]>()

  for (const n of nodes) {
    inDegree.set(n.id, 0)
    outgoing.set(n.id, [])
  }

  for (const e of edges) {
    inDegree.set(e.target, (inDegree.get(e.target) ?? 0) + 1)
    const outs = outgoing.get(e.source) ?? []
    outs.push(e.target)
    outgoing.set(e.source, outs)
  }

  // BFS layer assignment
  const layerOf = new Map<string, number>()
  const queue: string[] = []

  for (const [id, deg] of inDegree) {
    if (deg === 0) {
      queue.push(id)
      layerOf.set(id, 0)
    }
  }

  // Handle disconnected nodes that never got a layer
  const allIds = new Set(nodes.map((n) => n.id))

  while (queue.length > 0) {
    const id = queue.shift()!
    const currentLayer = layerOf.get(id) ?? 0
    for (const neighbor of outgoing.get(id) ?? []) {
      const nextLayer = currentLayer + 1
      if (!layerOf.has(neighbor) || layerOf.get(neighbor)! < nextLayer) {
        layerOf.set(neighbor, nextLayer)
      }
      queue.push(neighbor)
    }
  }

  // Assign default layer 0 to any node not reachable from roots (e.g. orphans)
  for (const id of allIds) {
    if (!layerOf.has(id)) layerOf.set(id, 0)
  }

  // Group by layer
  const layers = new Map<number, string[]>()
  for (const [id, layer] of layerOf) {
    const arr = layers.get(layer) ?? []
    arr.push(id)
    layers.set(layer, arr)
  }

  // Build updated node list
  return nodes.map((n) => {
    const layer = layerOf.get(n.id) ?? 0
    const layerNodes = layers.get(layer) ?? []
    const posInLayer = layerNodes.indexOf(n.id)
    return {
      ...n,
      position: {
        x: layer * LAYER_X_GAP,
        y: posInLayer * LAYER_Y_GAP,
      },
    }
  })
}

// ─── helpers ─────────────────────────────────────────────────────────────────

function escapeYamlString(s: string): string {
  return s.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '\\n')
}
