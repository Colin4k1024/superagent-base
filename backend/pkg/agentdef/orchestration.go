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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ─── SupervisorAgent ─────────────────────────────────────────────────────────

// SupervisorAgent coordinates sub-agents via a multi-round ReAct loop.
//
// V2 behaviour:
//  1. The supervisor LLM decides which sub-agents to invoke (via the
//     built-in delegate_to_agent tool).
//  2. Delegations are executed (potentially in parallel, bounded by parallelMax).
//  3. Results are aggregated and fed back to the LLM as the next input.
//  4. The loop terminates when the LLM produces a direct answer (no tool
//     calls) or when maxRounds is exhausted.
type SupervisorAgent struct {
	name        string
	description string
	mainAgent   Agent
	subAgents   map[string]Agent
	maxRounds   int
	delegate    *delegateTool
	def         *AgentDefinition
}

func (s *SupervisorAgent) Name() string                    { return s.name }
func (s *SupervisorAgent) Description() string             { return s.description }
func (s *SupervisorAgent) GetDefinition() *AgentDefinition { return s.def }

// Chat runs the multi-round supervisor loop.
//
// Each round the LLM can either:
//   - Emit a JSON object with "delegations": [...] to fan-out to sub-agents, or
//   - Emit any other text, which is treated as the final answer.
//
// The loop exits as soon as a final answer is produced or maxRounds is reached.
func (s *SupervisorAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	if s.delegate == nil {
		// Fallback for tests that construct SupervisorAgent without a delegate.
		return s.mainAgent.Chat(ctx, sessionID, message)
	}
	enrichedPrompt := buildSupervisorPrompt(s.def.Spec.SystemPrompt, s.subAgents)
	ch := make(chan string, 100)

	go func() {
		defer close(ch)

		aggregationMode := ""
		if s.def.Spec.Orchestration != nil {
			aggregationMode = s.def.Spec.Orchestration.ResultAggregation
		}

		input := message
		for round := 0; round < s.maxRounds; round++ {
			if ctx.Err() != nil {
				return
			}

			// Build the current round's prompt.
			roundPrompt := enrichedPrompt + "\n\nUser: " + input

			// Ask the LLM for its decision.
			subCh, err := s.mainAgent.Chat(ctx, sessionID, roundPrompt)
			if err != nil {
				select {
				case ch <- fmt.Sprintf("[supervisor error round %d]: %v", round+1, err):
				case <-ctx.Done():
				}
				return
			}

			var decisionBuf strings.Builder
			for token := range subCh {
				decisionBuf.WriteString(token)
			}
			decision := strings.TrimSpace(decisionBuf.String())

			// Try to parse structured delegations from the LLM output.
			delegations, isFinal := parseDelegationDecision(decision)
			if isFinal || len(delegations) == 0 {
				// Final answer — stream tokens and exit.
				for _, tok := range strings.SplitAfter(decision, "") {
					select {
					case ch <- tok:
					case <-ctx.Done():
						return
					}
				}
				return
			}

			// Emit a progress event so callers can observe the round.
			progressEvent := formatProgressEvent(round+1, delegations)
			select {
			case ch <- progressEvent:
			case <-ctx.Done():
				return
			}

			// Execute delegations (parallel, bounded by parallelMax).
			// Pass session scope via context to avoid race conditions.
			delegCtx := withDelegationScope(ctx, sessionID, round+1)
			results := s.delegate.executeDelegations(delegCtx, delegations)

			// Check abort on any failure.
			if s.delegate.fallback == "abort" {
				for _, r := range results {
					if r.err != nil {
						select {
						case ch <- fmt.Sprintf("[supervisor abort]: %v", r.err):
						case <-ctx.Done():
						}
						return
					}
				}
			}

			// Aggregate results as the next round's input.
			input = aggregateResults(ctx, results, aggregationMode, s.delegate.summarizeFn)
		}

		// maxRounds exhausted — emit a final error token.
		select {
		case ch <- fmt.Sprintf("[supervisor: max rounds (%d) exceeded]", s.maxRounds):
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// parseDelegationDecision attempts to decode the LLM's response as a
// delegation directive.  It returns the list of inputs and a "isFinal" flag.
//
// Expected format when the LLM wants to delegate:
//
//	{"delegations":[{"agent_name":"x","task":"..."},...]}
//
// Any other text is treated as a final answer (isFinal=true).
func parseDelegationDecision(text string) ([]DelegateToolInput, bool) {
	// Quick check before attempting JSON parse.
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") {
		return nil, true
	}

	var envelope struct {
		Delegations []DelegateToolInput `json:"delegations"`
	}
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return nil, true
	}
	if len(envelope.Delegations) == 0 {
		return nil, true
	}
	return envelope.Delegations, false
}

// formatProgressEvent formats a lightweight progress notification.
func formatProgressEvent(round int, delegations []DelegateToolInput) string {
	names := make([]string, 0, len(delegations))
	for _, d := range delegations {
		names = append(names, d.AgentName)
	}
	return fmt.Sprintf("\n[supervisor round %d: delegating to %s]\n",
		round, strings.Join(names, ", "))
}

// buildSupervisorPrompt appends a list of available sub-agents and delegation
// instructions to the base system prompt.
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
	sb.WriteString(`
To delegate tasks, respond ONLY with valid JSON in this format:
{"delegations":[{"agent_name":"<name>","task":"<task>","context":"<optional>"}]}

To provide a final answer, respond with plain text (not JSON).
`)
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
			if ctx.Err() != nil {
				return
			}
			var result strings.Builder
			stepSessionID := SubSessionID(sessionID, fmt.Sprintf("seq.step%d", i), agent.Name())
			subCh, err := agent.Chat(ctx, stepSessionID, currentInput)
			if err != nil {
				select {
				case ch <- fmt.Sprintf("[error in step %d (%s)]: %v", i+1, agent.Name(), err):
				case <-ctx.Done():
				}
				return
			}
			isLast := i == len(s.agents)-1
			for token := range subCh {
				result.WriteString(token)
				// Only stream tokens from the last agent to the caller.
				if isLast {
					select {
					case ch <- token:
					case <-ctx.Done():
						return
					}
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

		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()

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
				branchSessionID := SubSessionID(sessionID, fmt.Sprintf("par.branch%d", idx), a.Name())
				subCh, err := a.Chat(childCtx, branchSessionID, message)
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
			select {
			case <-ctx.Done():
				return
			default:
			}
			collected = append(collected, r)
		}

		for _, r := range collected {
			select {
			case ch <- fmt.Sprintf("\n--- %s ---\n%s\n", r.name, r.output):
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
