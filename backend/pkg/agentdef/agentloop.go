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
)

const (
	defaultMaxTurns = 25
	doneMarker      = "[DONE]"
)

// AgentLoopAgent implements an autonomous agent loop that iterates until the
// task is complete or maxTurns is exhausted. Each turn calls the underlying
// LLM agent, checks if the output signals completion (via [DONE] marker),
// and feeds accumulated context back for the next turn.
type AgentLoopAgent struct {
	name        string
	description string
	mainAgent   Agent
	def         *AgentDefinition
	maxTurns    int
}

func (a *AgentLoopAgent) Name() string                    { return a.name }
func (a *AgentLoopAgent) Description() string             { return a.description }
func (a *AgentLoopAgent) GetDefinition() *AgentDefinition { return a.def }

// Chat runs the autonomous loop. The agent iterates on the user's message,
// accumulating context across turns until it emits [DONE] or hits maxTurns.
func (a *AgentLoopAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		currentInput := message
		var history strings.Builder
		history.WriteString("Task: ")
		history.WriteString(message)
		history.WriteString("\n")

		for turn := 1; turn <= a.maxTurns; turn++ {
			if ctx.Err() != nil {
				a.emit(ch, ctx, fmt.Sprintf("\n[agentloop] cancelled at turn %d: %v", turn, ctx.Err()))
				return
			}

			// Emit turn header.
			if turn > 1 {
				a.emit(ch, ctx, fmt.Sprintf("\n--- Turn %d/%d ---\n", turn, a.maxTurns))
			}

			// Call the underlying agent.
			turnCh, err := a.mainAgent.Chat(ctx, sessionID, currentInput)
			if err != nil {
				a.emit(ch, ctx, fmt.Sprintf("\n[agentloop] error at turn %d: %v", turn, err))
				return
			}

			// Collect and stream the turn output.
			var turnOutput strings.Builder
			for token := range turnCh {
				select {
				case ch <- token:
				case <-ctx.Done():
					return
				}
				turnOutput.WriteString(token)
			}

			output := turnOutput.String()

			// Check for completion signal.
			if strings.Contains(output, doneMarker) {
				return
			}

			// Accumulate history and prepare next turn input.
			fmt.Fprintf(&history, "\n[Turn %d output]:\n%s\n", turn, output)
			currentInput = history.String() + "\nContinue working on the task. If the task is complete, include [DONE] in your response."
		}

		a.emit(ch, ctx, fmt.Sprintf("\n[agentloop] reached max turns (%d)", a.maxTurns))
	}()
	return ch, nil
}

func (a *AgentLoopAgent) emit(ch chan<- string, ctx context.Context, msg string) {
	select {
	case ch <- msg:
	case <-ctx.Done():
	}
}
