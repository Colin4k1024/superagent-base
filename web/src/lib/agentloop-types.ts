/**
 * AgentLoop-specific types for the AgentLoop Demo page.
 * Independent from ChatMessage — does not pollute existing ChatPage.
 */

export interface ToolCallInfo {
  id: string
  name: string
  args: string
  result?: string
  status: 'calling' | 'done' | 'error'
}

export interface AgentLoopTurn {
  turn: number
  total: number
  text: string
  thinking: string
  toolCalls: ToolCallInfo[]
  startTime: number
  endTime?: number
  isStreaming: boolean
}

export interface InterruptField {
  name: string
  type: string
  label: string
  required: boolean
  options?: string[]
}

export interface AgentLoopMessage {
  id: string
  role: 'user' | 'assistant'
  content?: string
  turns: AgentLoopTurn[]
  status: 'streaming' | 'done' | 'preempted' | 'error'
  errorMessage?: string
  /** interrupt data from A2UI interrupt event */
  interrupt?: { reason: string; fields: InterruptField[] }
}

export type ExecutionStatus = 'idle' | 'streaming' | 'done' | 'error' | 'preempted'

export interface ExecutionState {
  currentTurn: number
  maxTurns: number
  startTime?: number
  endTime?: number
  status: ExecutionStatus
}

// --- Reducer actions ---

export type AgentLoopAction =
  | { type: 'USER_MESSAGE'; content: string }
  | { type: 'ASSISTANT_START' }
  | { type: 'TURN_START'; turn: number; total: number }
  | { type: 'TEXT_DELTA'; delta: string }
  | { type: 'THINKING_DELTA'; delta: string }
  | { type: 'TOOL_CALL'; id: string; name: string; args: string }
  | { type: 'TOOL_RESULT'; id: string; name: string; result: string; isError: boolean }
  | { type: 'INTERRUPT'; reason: string; fields: InterruptField[] }
  | { type: 'ERROR'; code: string; message: string }
  | { type: 'DONE' }
  | { type: 'PREEMPTED' }
