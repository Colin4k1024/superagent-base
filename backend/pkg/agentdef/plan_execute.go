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
	"regexp"
	"strings"
)

// planStepPattern matches numbered plan steps like "1. Do something" or "1) Do something".
var planStepPattern = regexp.MustCompile(`(?m)^\s*\d+[\.\)]\s+(.+)$`)

// PlanExecuteAgent implements a plan-then-execute workflow:
// 1. Calls the main LLM to generate a step-by-step plan.
// 2. Parses the plan into discrete steps.
// 3. Delegates each step to the first executor sub-agent.
// 4. Collects and streams all results back.
type PlanExecuteAgent struct {
	name        string
	description string
	mainAgent   Agent   // LLM agent for planning
	executors   []Agent // Sub-agents for step execution
	def         *AgentDefinition
	maxSteps    int
}

func (a *PlanExecuteAgent) Name() string                    { return a.name }
func (a *PlanExecuteAgent) Description() string             { return a.description }
func (a *PlanExecuteAgent) GetDefinition() *AgentDefinition { return a.def }

// Chat generates a plan via the main agent, parses it into steps, and executes
// each step through the first available executor sub-agent.
func (a *PlanExecuteAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		// Phase 1: Generate plan.
		planPrompt := "Create a step-by-step plan to accomplish: " + message
		planSessionID := SubSessionID(sessionID, "plan", "planner")
		planCh, err := a.mainAgent.Chat(ctx, planSessionID, planPrompt)
		if err != nil {
			select {
			case ch <- fmt.Sprintf("[plan_execute] planning error: %v", err):
			case <-ctx.Done():
			}
			return
		}

		var planBuf strings.Builder
		for token := range planCh {
			planBuf.WriteString(token)
		}
		planText := planBuf.String()

		// Emit the plan to the caller.
		select {
		case ch <- fmt.Sprintf("## Plan\n%s\n\n## Execution\n", planText):
		case <-ctx.Done():
			return
		}

		// Phase 2: Parse steps from the plan.
		steps := parsePlanSteps(planText)
		if len(steps) == 0 {
			// Fallback: treat the entire plan as a single step.
			steps = []string{planText}
		}

		// Enforce max steps limit.
		if a.maxSteps > 0 && len(steps) > a.maxSteps {
			steps = steps[:a.maxSteps]
			select {
			case ch <- fmt.Sprintf("[plan_execute] truncated to %d steps\n", a.maxSteps):
			case <-ctx.Done():
				return
			}
		}

		// Phase 3: Execute each step via the first executor.
		if len(a.executors) == 0 {
			select {
			case ch <- "[plan_execute] no executor sub-agents configured":
			case <-ctx.Done():
			}
			return
		}
		executor := a.executors[0]

		for i, step := range steps {
			if ctx.Err() != nil {
				select {
				case ch <- fmt.Sprintf("[plan_execute] cancelled: %v", ctx.Err()):
				case <-ctx.Done():
				}
				return
			}

			select {
			case ch <- fmt.Sprintf("\n### Step %d\n", i+1):
			case <-ctx.Done():
				return
			}

			stepSessionID := SubSessionID(sessionID, "plan", fmt.Sprintf("step%d", i))
			stepCh, stepErr := executor.Chat(ctx, stepSessionID, step)
			if stepErr != nil {
				select {
				case ch <- fmt.Sprintf("[error in step %d]: %v\n", i+1, stepErr):
				case <-ctx.Done():
					return
				}
				continue
			}
			for token := range stepCh {
				select {
				case ch <- token:
				case <-ctx.Done():
					return
				}
			}
			select {
			case ch <- "\n":
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// parsePlanSteps extracts numbered steps from the plan text.
func parsePlanSteps(plan string) []string {
	matches := planStepPattern.FindAllStringSubmatch(plan, -1)
	steps := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			step := strings.TrimSpace(m[1])
			if step != "" {
				steps = append(steps, step)
			}
		}
	}
	return steps
}
