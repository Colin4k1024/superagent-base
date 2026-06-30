import type { AgentLoopMessage, AgentLoopTurn, AgentLoopAction, ExecutionState } from './agentloop-types'

// --- Helpers ---

function newTurn(turn: number, total: number): AgentLoopTurn {
  return { turn, total, text: '', thinking: '', toolCalls: [], startTime: Date.now(), isStreaming: true }
}

function lastAssistantMsg(messages: AgentLoopMessage[]): AgentLoopMessage | undefined {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === 'assistant') return messages[i]
  }
  return undefined
}

function currentTurn(msg: AgentLoopMessage): AgentLoopTurn | undefined {
  return msg.turns[msg.turns.length - 1]
}

/** Shallow-copy a turn for immutable update. */
function copyTurn(t: AgentLoopTurn): AgentLoopTurn {
  return { ...t, toolCalls: t.toolCalls.map(tc => ({ ...tc })) }
}

/** Shallow-copy a message (including its turns) for immutable update. */
function copyMsg(m: AgentLoopMessage): AgentLoopMessage {
  return { ...m, turns: m.turns.map(copyTurn), interrupt: m.interrupt ? { ...m.interrupt } : undefined }
}

// --- Message Reducer ---

export function agentLoopMessageReducer(
  state: AgentLoopMessage[],
  action: AgentLoopAction,
): AgentLoopMessage[] {
  const last = lastAssistantMsg(state)

  switch (action.type) {
    case 'USER_MESSAGE':
      return [
        ...state,
        { id: crypto.randomUUID(), role: 'user', content: action.content, turns: [], status: 'done' },
        { id: crypto.randomUUID(), role: 'assistant', turns: [newTurn(1, 1)], status: 'streaming' },
      ]

    case 'ASSISTANT_START': {
      if (last && last.status === 'streaming') return state
      return [
        ...state,
        { id: crypto.randomUUID(), role: 'assistant', turns: [newTurn(1, 1)], status: 'streaming' },
      ]
    }

    case 'TURN_START': {
      if (!last) return state
      const msg = copyMsg(last)
      // Finish previous turn
      const prev = msg.turns[msg.turns.length - 1]
      if (prev) {
        prev.isStreaming = false
        prev.endTime = Date.now()
      }
      // Update maxTurns on all existing turns
      msg.turns = msg.turns.map(t => ({ ...t, total: action.total }))
      // Add new turn
      msg.turns.push(newTurn(action.turn, action.total))
      return [...state.slice(0, -1), msg]
    }

    case 'TEXT_DELTA': {
      if (!last) return state
      const turn = currentTurn(last)
      if (!turn) return state
      const msg = copyMsg(last)
      const t = msg.turns[msg.turns.length - 1]
      t.text += action.delta
      return [...state.slice(0, -1), msg]
    }

    case 'THINKING_DELTA': {
      if (!last) return state
      const turn = currentTurn(last)
      if (!turn) return state
      const msg = copyMsg(last)
      const t = msg.turns[msg.turns.length - 1]
      t.thinking += action.delta
      return [...state.slice(0, -1), msg]
    }

    case 'TOOL_CALL': {
      if (!last) return state
      const turn = currentTurn(last)
      if (!turn) return state
      const msg = copyMsg(last)
      const t = msg.turns[msg.turns.length - 1]
      t.toolCalls = [
        ...t.toolCalls,
        { id: action.id, name: action.name, args: action.args, status: 'calling' },
      ]
      return [...state.slice(0, -1), msg]
    }

    case 'TOOL_RESULT': {
      if (!last) return state
      const turn = currentTurn(last)
      if (!turn) return state
      // Match by name + 'calling' status (not by ID, since callback doesn't pass matching ID)
      const idx = [...turn.toolCalls].reverse().findIndex(tc => tc.name === action.name && tc.status === 'calling')
      if (idx < 0) return state
      const realIdx = turn.toolCalls.length - 1 - idx
      const msg = copyMsg(last)
      const t = msg.turns[msg.turns.length - 1]
      t.toolCalls = t.toolCalls.map((tc, i) =>
        i === realIdx
          ? { ...tc, result: action.result, status: action.isError ? 'error' as const : 'done' as const }
          : tc,
      )
      return [...state.slice(0, -1), msg]
    }

    case 'INTERRUPT': {
      if (!last) return state
      const msg = copyMsg(last)
      msg.status = 'streaming'
      msg.interrupt = { reason: action.reason, fields: action.fields }
      return [...state.slice(0, -1), msg]
    }

    case 'ERROR': {
      if (!last) return state
      const msg = copyMsg(last)
      msg.status = 'error'
      msg.errorMessage = action.message
      const t = msg.turns[msg.turns.length - 1]
      if (t) { t.isStreaming = false; t.endTime = Date.now() }
      return [...state.slice(0, -1), msg]
    }

    case 'DONE': {
      if (!last) return state
      const msg = copyMsg(last)
      msg.status = 'done'
      const t = msg.turns[msg.turns.length - 1]
      if (t) { t.isStreaming = false; t.endTime = Date.now() }
      return [...state.slice(0, -1), msg]
    }

    case 'PREEMPTED': {
      if (!last) return state
      const msg = copyMsg(last)
      msg.status = 'preempted'
      const t = msg.turns[msg.turns.length - 1]
      if (t) { t.isStreaming = false; t.endTime = Date.now() }
      return [...state.slice(0, -1), msg]
    }

    default:
      return state
  }
}

// --- Execution State Reducer ---

export function executionStateReducer(
  state: ExecutionState,
  action: AgentLoopAction,
): ExecutionState {
  switch (action.type) {
    case 'USER_MESSAGE':
      return { currentTurn: 1, maxTurns: 1, startTime: Date.now(), status: 'streaming' }
    case 'TURN_START':
      return { ...state, currentTurn: action.turn, maxTurns: action.total, status: 'streaming' }
    case 'DONE':
      return { ...state, endTime: Date.now(), status: 'done' }
    case 'ERROR':
      return { ...state, endTime: Date.now(), status: 'error' }
    case 'PREEMPTED':
      return { ...state, endTime: Date.now(), status: 'preempted' }
    default:
      return state
  }
}

export const initialExecutionState: ExecutionState = {
  currentTurn: 0,
  maxTurns: 0,
  status: 'idle',
}
