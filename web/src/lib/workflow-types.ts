// WorkflowSpec types matching backend pkg/agentdef workflow schema.

export interface WorkflowSpec {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  variables?: WorkflowVariable[]
}

export interface WorkflowNode {
  id: string
  type: string // "llm_call" | "agent_call" | "tool_call" | "code" | "condition"
  agent?: string
  tool?: string
  prompt?: string
  code?: string
  language?: string
  condition?: string
  input_mapping?: Record<string, string>
}

export interface WorkflowEdge {
  from: string // node id or "START"
  to: string   // node id or "END"
  condition?: string
}

export interface WorkflowVariable {
  name: string
  from: string // "node_id.output"
}
