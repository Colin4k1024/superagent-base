/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agentdef

import "github.com/cloudwego/eino/adk"

// AgentUnwrapper is implemented by middleware wrappers that decorate an Agent.
// It allows traversal through middleware layers to reach the underlying agent.
type AgentUnwrapper interface {
	UnwrapAgent() Agent
}

// Compile-time interface assertions for all middleware wrappers.
var (
	_ AgentUnwrapper = (*observedAgent)(nil)
	_ AgentUnwrapper = (*timeoutAgent)(nil)
	_ AgentUnwrapper = (*retryAgent)(nil)
	_ AgentUnwrapper = (*fallbackAgent)(nil)
	_ AgentUnwrapper = (*rateLimitAgent)(nil)
	_ AgentUnwrapper = (*cacheAgent)(nil)
)

// unwrapToADKChatModel recursively unwraps middleware layers to find the
// underlying *adkChatModelAgent. Returns nil if not found within 20 layers
// (guards against accidental cycles).
func unwrapToADKChatModel(a Agent) *adkChatModelAgent {
	const maxDepth = 20
	for i := 0; i < maxDepth; i++ {
		switch v := a.(type) {
		case *adkChatModelAgent:
			return v
		case AgentUnwrapper:
			a = v.UnwrapAgent()
		default:
			return nil
		}
	}
	return nil
}

// unwrapToADKRunner recursively unwraps middleware layers to find the
// underlying *ADKRunnerAgent. Returns nil if not found.
func unwrapToADKRunner(a Agent) *ADKRunnerAgent {
	const maxDepth = 20
	for i := 0; i < maxDepth; i++ {
		switch v := a.(type) {
		case *ADKRunnerAgent:
			return v
		case AgentUnwrapper:
			a = v.UnwrapAgent()
		default:
			return nil
		}
	}
	return nil
}

// unwrapToEinoAgent recursively unwraps middleware layers to extract the
// underlying adk.Agent (either from *adkChatModelAgent or *ADKRunnerAgent).
// Used by buildADKSupervisor to wrap sub-agents as AgentTools.
func unwrapToEinoAgent(a Agent) adk.Agent {
	const maxDepth = 20
	for i := 0; i < maxDepth; i++ {
		switch v := a.(type) {
		case *adkChatModelAgent:
			return v.agent
		case *ADKRunnerAgent:
			return v.agent
		case AgentUnwrapper:
			a = v.UnwrapAgent()
		default:
			return nil
		}
	}
	return nil
}
