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

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ─── SupervisorAgent ─────────────────────────────────────────────────────────

// SupervisorAgent coordinates sub-agents via a main LLM.
// In v1 the supervisor augments its system prompt with the list of available
// sub-agents and their descriptions, then delegates the conversation to its
// own LLM.  Future versions will parse structured delegation directives from
// the LLM output and fan-out to the appropriate sub-agent.
type SupervisorAgent struct {
	name        string
	description string
	mainAgent   Agent
	subAgents   map[string]Agent
	maxRounds   int
	def         *AgentDefinition
}

func (s *SupervisorAgent) Name() string                    { return s.name }
func (s *SupervisorAgent) Description() string             { return s.description }
func (s *SupervisorAgent) GetDefinition() *AgentDefinition { return s.def }

// Chat runs the supervisor: it enriches the system prompt with sub-agent
// metadata and then streams the main agent's response.
func (s *SupervisorAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	enrichedPrompt := buildSupervisorPrompt(s.def.Spec.SystemPrompt, s.subAgents)

	// Temporarily override the definition's system prompt so the underlying
	// mainAgent uses the enriched version.  We create a synthetic message that
	// prepends the enriched system context to the user message.
	fullMessage := enrichedPrompt + "\n\nUser: " + message

	subCh, err := s.mainAgent.Chat(ctx, sessionID, fullMessage)
	if err != nil {
		return nil, fmt.Errorf("supervisor %q: main agent chat: %w", s.name, err)
	}

	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		for token := range subCh {
			ch <- token
		}
	}()
	return ch, nil
}

// buildSupervisorPrompt appends a list of available sub-agents to the base
// system prompt so the supervisor LLM is aware of its delegation options.
func buildSupervisorPrompt(basePrompt string, subAgents map[string]Agent) string {
	if len(subAgents) == 0 {
		return basePrompt
	}
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\nAvailable sub-agents:\n")
	for name, agent := range subAgents {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, agent.Description()))
	}
	return sb.String()
}

// ─── SequentialAgent ─────────────────────────────────────────────────────────

// SequentialAgent runs sub-agents in declaration order, feeding each agent's
// full output as the input to the next.  Only the final agent's tokens are
// streamed to the caller.
type SequentialAgent struct {
	name        string
	description string
	agents      []Agent
	def         *AgentDefinition
}

func (s *SequentialAgent) Name() string                    { return s.name }
func (s *SequentialAgent) Description() string             { return s.description }
func (s *SequentialAgent) GetDefinition() *AgentDefinition { return s.def }

// Chat runs agents sequentially: agent[0] gets the original message, each
// subsequent agent receives the accumulated output of the previous one.
func (s *SequentialAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		currentInput := message
		for i, agent := range s.agents {
			var result strings.Builder
			subCh, err := agent.Chat(ctx, sessionID, currentInput)
			if err != nil {
				ch <- fmt.Sprintf("[error in step %d (%s)]: %v", i+1, agent.Name(), err)
				return
			}
			isLast := i == len(s.agents)-1
			for token := range subCh {
				result.WriteString(token)
				// Only stream tokens from the last agent to the caller.
				if isLast {
					ch <- token
				}
			}
			currentInput = result.String()
		}
	}()
	return ch, nil
}

// ─── ParallelAgent ───────────────────────────────────────────────────────────

// ParallelAgent runs all sub-agents concurrently with the same input and
// combines their results into a single ordered stream.
type ParallelAgent struct {
	name        string
	description string
	agents      []Agent
	def         *AgentDefinition
}

func (p *ParallelAgent) Name() string                    { return p.name }
func (p *ParallelAgent) Description() string             { return p.description }
func (p *ParallelAgent) GetDefinition() *AgentDefinition { return p.def }

// Chat runs all agents concurrently and streams their combined results.
// Results are written in order of completion; each agent's output is prefixed
// with a separator header.
func (p *ParallelAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		type agentResult struct {
			index  int
			name   string
			output string
		}

		results := make(chan agentResult, len(p.agents))
		var wg sync.WaitGroup

		for i, agent := range p.agents {
			wg.Add(1)
			go func(idx int, a Agent) {
				defer wg.Done()
				var out strings.Builder
				subCh, err := a.Chat(ctx, sessionID, message)
				if err != nil {
					results <- agentResult{index: idx, name: a.Name(), output: fmt.Sprintf("[error]: %v", err)}
					return
				}
				for token := range subCh {
					out.WriteString(token)
				}
				results <- agentResult{index: idx, name: a.Name(), output: out.String()}
			}(i, agent)
		}

		// Close results channel once all goroutines finish.
		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect all results then emit in arrival order.
		collected := make([]agentResult, 0, len(p.agents))
		for r := range results {
			collected = append(collected, r)
		}

		for _, r := range collected {
			ch <- fmt.Sprintf("\n--- %s ---\n%s\n", r.name, r.output)
		}
	}()
	return ch, nil
}
