import { describe, it, expect } from 'vitest'
import {
  workflowToGraph,
  graphToWorkflow,
  autoLayout,
  type RFNode,
  type RFEdge,
} from '../workflow-serializer'
import type { WorkflowSpec } from '../workflow-types'

describe('workflowToGraph', () => {
  it('returns empty arrays for empty spec (no nodes)', () => {
    const spec: WorkflowSpec = { nodes: [], edges: [] }
    const { nodes, edges } = workflowToGraph(spec)
    // Still inserts __start__ and __end__ sentinels
    expect(nodes.find((n) => n.id === '__start__')).toBeDefined()
    expect(nodes.find((n) => n.id === '__end__')).toBeDefined()
    expect(edges).toHaveLength(0)
  })

  it('returns empty arrays for null/undefined spec', () => {
    // @ts-expect-error testing null input
    const { nodes, edges } = workflowToGraph(null)
    expect(nodes).toHaveLength(0)
    expect(edges).toHaveLength(0)
  })

  it('creates start and end sentinel nodes', () => {
    const spec: WorkflowSpec = {
      nodes: [{ id: 'n1', type: 'agent' }],
      edges: [],
    }
    const { nodes } = workflowToGraph(spec)
    const ids = nodes.map((n) => n.id)
    expect(ids).toContain('__start__')
    expect(ids).toContain('__end__')
  })

  it('maps START/END in edges to sentinel node ids', () => {
    const spec: WorkflowSpec = {
      nodes: [{ id: 'n1', type: 'agent' }],
      edges: [
        { from: 'START', to: 'n1' },
        { from: 'n1', to: 'END' },
      ],
    }
    const { edges } = workflowToGraph(spec)
    expect(edges.some((e) => e.source === '__start__' && e.target === 'n1')).toBe(true)
    expect(edges.some((e) => e.source === 'n1' && e.target === '__end__')).toBe(true)
  })
})

describe('graphToWorkflow', () => {
  it('filters out __start__ and __end__ nodes', () => {
    const nodes: RFNode[] = [
      { id: '__start__', type: 'start', position: { x: 0, y: 0 }, data: {} },
      { id: 'n1', type: 'agent', position: { x: 0, y: 0 }, data: {} },
      { id: '__end__', type: 'end', position: { x: 0, y: 0 }, data: {} },
    ]
    const edges: RFEdge[] = []
    const spec = graphToWorkflow(nodes, edges)
    expect(spec.nodes).toHaveLength(1)
    expect(spec.nodes[0].id).toBe('n1')
  })

  it('converts sentinel edge sources back to START/END', () => {
    const nodes: RFNode[] = [
      { id: 'n1', type: 'agent', position: { x: 0, y: 0 }, data: {} },
    ]
    const edges: RFEdge[] = [
      { id: '__start__-n1', source: '__start__', target: 'n1' },
      { id: 'n1-__end__', source: 'n1', target: '__end__' },
    ]
    const spec = graphToWorkflow(nodes, edges)
    expect(spec.edges[0].from).toBe('START')
    expect(spec.edges[1].to).toBe('END')
  })
})

describe('autoLayout', () => {
  it('returns empty array for empty nodes', () => {
    expect(autoLayout([], [])).toHaveLength(0)
  })

  it('assigns positions in layers', () => {
    const nodes: RFNode[] = [
      { id: 'a', type: 'start', position: { x: 0, y: 0 }, data: {} },
      { id: 'b', type: 'agent', position: { x: 0, y: 0 }, data: {} },
      { id: 'c', type: 'end', position: { x: 0, y: 0 }, data: {} },
    ]
    const edges: RFEdge[] = [
      { id: 'a-b', source: 'a', target: 'b' },
      { id: 'b-c', source: 'b', target: 'c' },
    ]
    const laid = autoLayout(nodes, edges)
    const posA = laid.find((n) => n.id === 'a')!.position
    const posB = laid.find((n) => n.id === 'b')!.position
    const posC = laid.find((n) => n.id === 'c')!.position
    // Layers increase left to right
    expect(posA.x).toBeLessThan(posB.x)
    expect(posB.x).toBeLessThan(posC.x)
  })

  it('positions parallel nodes at different y values', () => {
    const nodes: RFNode[] = [
      { id: 'start', type: 'start', position: { x: 0, y: 0 }, data: {} },
      { id: 'p1', type: 'agent', position: { x: 0, y: 0 }, data: {} },
      { id: 'p2', type: 'agent', position: { x: 0, y: 0 }, data: {} },
    ]
    const edges: RFEdge[] = [
      { id: 'start-p1', source: 'start', target: 'p1' },
      { id: 'start-p2', source: 'start', target: 'p2' },
    ]
    const laid = autoLayout(nodes, edges)
    const posP1 = laid.find((n) => n.id === 'p1')!.position
    const posP2 = laid.find((n) => n.id === 'p2')!.position
    // Same layer (same x), different y
    expect(posP1.x).toBe(posP2.x)
    expect(posP1.y).not.toBe(posP2.y)
  })
})
