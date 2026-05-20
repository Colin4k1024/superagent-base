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
	"time"
)

// ─── Delegation types ─────────────────────────────────────────────────────────

// DelegateToolInput is the structured input for the delegate_to_agent tool.
type DelegateToolInput struct {
	AgentName string `json:"agent_name"`
	Task      string `json:"task"`
	Context   string `json:"context,omitempty"`
}

// DelegateToolOutput is the structured result returned from a delegation.
type DelegateToolOutput struct {
	AgentName string `json:"agent_name"`
	Result    string `json:"result"`
	Status    string `json:"status"`   // success | error | timeout
	Duration  string `json:"duration"`
}

// delegationResult carries the outcome of a single delegation execution.
type delegationResult struct {
	input    DelegateToolInput
	output   DelegateToolOutput
	err      error
}

// ─── delegateTool ─────────────────────────────────────────────────────────────

// delegateTool is an in-process tool that routes calls to sub-agents.
// It is registered with the supervisor's ReAct agent so the LLM can invoke it
// via standard tool-call mechanics.
type delegateTool struct {
	subAgents   map[string]Agent
	timeout     time.Duration
	fallback    string // skip | abort | ask_supervisor
	parallelMax int
	sessionID   string
}

// Info returns the tool schema consumed by Eino / the LLM.
func (d *delegateTool) Info(_ context.Context) (*toolInfo, error) {
	return &toolInfo{
		Name:        "delegate_to_agent",
		Description: "Delegate a task to a named sub-agent and return the result.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_name": map[string]any{
					"type":        "string",
					"description": "The name of the sub-agent to delegate to.",
				},
				"task": map[string]any{
					"type":        "string",
					"description": "The task description to send to the sub-agent.",
				},
				"context": map[string]any{
					"type":        "string",
					"description": "Optional additional context for the sub-agent.",
				},
			},
			"required": []string{"agent_name", "task"},
		},
	}, nil
}

// Invoke executes the delegation synchronously.
func (d *delegateTool) Invoke(ctx context.Context, inputJSON string) (string, error) {
	var inp DelegateToolInput
	if err := json.Unmarshal([]byte(inputJSON), &inp); err != nil {
		return "", fmt.Errorf("delegate_to_agent: invalid input JSON: %w", err)
	}

	out := d.execute(ctx, inp)
	raw, _ := json.Marshal(out)
	return string(raw), nil
}

// execute runs a single delegation with timeout.
func (d *delegateTool) execute(ctx context.Context, inp DelegateToolInput) DelegateToolOutput {
	sub, ok := d.subAgents[inp.AgentName]
	if !ok {
		return DelegateToolOutput{
			AgentName: inp.AgentName,
			Status:    "error",
			Result:    fmt.Sprintf("sub-agent %q not found", inp.AgentName),
			Duration:  "0s",
		}
	}

	timeout := d.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	delegateCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	msg := inp.Task
	if inp.Context != "" {
		msg = inp.Context + "\n\n" + inp.Task
	}

	start := time.Now()
	ch, err := sub.Chat(delegateCtx, d.sessionID, msg)
	if err != nil {
		return DelegateToolOutput{
			AgentName: inp.AgentName,
			Status:    "error",
			Result:    err.Error(),
			Duration:  time.Since(start).String(),
		}
	}

	var sb strings.Builder
	for token := range ch {
		sb.WriteString(token)
	}

	elapsed := time.Since(start)
	status := "success"
	if delegateCtx.Err() != nil {
		status = "timeout"
	}

	return DelegateToolOutput{
		AgentName: inp.AgentName,
		Result:    sb.String(),
		Status:    status,
		Duration:  elapsed.String(),
	}
}

// ─── Batch execution ─────────────────────────────────────────────────────────

// executeDelegations runs a batch of delegations, honouring parallelMax.
// It returns all results; if fallback is "abort" any error causes early return.
func (d *delegateTool) executeDelegations(ctx context.Context, inputs []DelegateToolInput) []delegationResult {
	if len(inputs) == 0 {
		return nil
	}

	max := d.parallelMax
	if max <= 0 {
		max = 3
	}
	sem := make(chan struct{}, max)

	results := make([]delegationResult, len(inputs))
	var wg sync.WaitGroup
	var abort bool
	var mu sync.Mutex

	for i, inp := range inputs {
		wg.Add(1)
		go func(idx int, in DelegateToolInput) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			if abort {
				mu.Unlock()
				return
			}
			mu.Unlock()

			out := d.execute(ctx, in)
			res := delegationResult{input: in, output: out}
			if out.Status != "success" {
				res.err = fmt.Errorf("delegation to %q: %s", in.AgentName, out.Result)
			}

			mu.Lock()
			results[idx] = res
			if res.err != nil && d.fallback == "abort" {
				abort = true
			}
			mu.Unlock()
		}(i, inp)
	}

	wg.Wait()
	return results
}

// ─── Result aggregation ──────────────────────────────────────────────────────

// aggregateResults combines delegation results into a single string for the
// next round's input.  mode: "concat" (default), "summarize", "structured".
func aggregateResults(results []delegationResult, mode string) string {
	if len(results) == 0 {
		return ""
	}
	switch mode {
	case "structured":
		out := make([]DelegateToolOutput, 0, len(results))
		for _, r := range results {
			out = append(out, r.output)
		}
		raw, _ := json.MarshalIndent(out, "", "  ")
		return string(raw)
	case "summarize":
		// Simple concat with agent name headers — richer summarisation would
		// require another LLM call which we avoid to keep the loop clean.
		fallthrough
	default: // "concat"
		var sb strings.Builder
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", r.output.AgentName, r.output.Result))
		}
		return sb.String()
	}
}

// ─── toolInfo ─────────────────────────────────────────────────────────────────

// toolInfo is a minimal tool descriptor used internally by delegateTool.
type toolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
}
