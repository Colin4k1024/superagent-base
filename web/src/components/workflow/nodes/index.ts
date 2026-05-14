import LLMNode from './LLMNode'
import AgentNode from './AgentNode'
import ToolNode from './ToolNode'
import CodeNode from './CodeNode'
import ConditionNode from './ConditionNode'

export { LLMNode, AgentNode, ToolNode, CodeNode, ConditionNode }

export const nodeTypes = {
  llm_call: LLMNode,
  agent_call: AgentNode,
  tool_call: ToolNode,
  code: CodeNode,
  condition: ConditionNode,
}
