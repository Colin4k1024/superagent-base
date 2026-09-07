/*
 * Copyright 2025 coze-dev Authors
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
	"sync"
	"time"
)

// NodeTrace captures the execution trace of a single workflow node.
// It is populated during DryRun or trace-enabled Chat executions.
type NodeTrace struct {
	NodeID   string            `json:"node_id"`
	Type     string            `json:"type"`
	Level    int               `json:"level"`
	Input    map[string]string `json:"input"`       // state snapshot before execution
	Output   string            `json:"output"`      // node output
	Error    string            `json:"error,omitempty"`
	Duration time.Duration     `json:"duration_ms"` // in milliseconds
}

// DryRunResult holds the structured output of a dry-run execution.
type DryRunResult struct {
	AgentName string      `json:"agent_name"`
	Message   string      `json:"message"`
	Levels    [][]string  `json:"levels"`
	Traces    []NodeTrace `json:"traces"`
	FinalOutput string     `json:"final_output"`
	Error    string       `json:"error,omitempty"`
}

// DryRun executes the workflow DAG against a user-supplied input message
// and returns per-node traces without persisting side effects (no memory
// writes, no session state). This replaces Eino Dev's step debugger.
//
// The execution path is identical to Chat — only side-effect persistence
// is gated. The same executeNode dispatch, topological ordering, and
// parallel execution semantics apply.
func (w *WorkflowAgent) DryRun(ctx context.Context, message string) (*DryRunResult, error) {
	result := &DryRunResult{
		AgentName: w.name,
		Message:   message,
	}

	state := newSafeState(map[string]string{"message": message})

	levels, err := w.topologicalLevels()
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	if len(levels) == 0 {
		result.Error = "no executable nodes found"
		return result, nil
	}

	result.Levels = levels

	var traces []NodeTrace
	var tracesMu sync.Mutex
	var lastNodeID string

	for levelIdx, level := range levels {
		if execErr := w.executeLevelWithTrace(ctx, "", level, levelIdx, state, &traces, &tracesMu); execErr != nil {
			result.Error = execErr.Error()
			result.Traces = traces
			return result, nil
		}
		for _, nodeID := range level {
			result := state.get(nodeID + ".output")
			for _, v := range w.variables {
				if hasPrefix(v.From, nodeID+".") {
					state.set(v.Name, result)
				}
			}
		}
		lastNodeID = level[len(level)-1]
	}

	result.FinalOutput = state.get(lastNodeID + ".output")
	result.Traces = traces
	return result, nil
}

// executeLevelWithTrace runs all nodeIDs in a single topological level
// concurrently, capturing per-node traces. It mirrors executeLevel but
// appends NodeTrace entries for each node.
func (w *WorkflowAgent) executeLevelWithTrace(
	ctx context.Context,
	sessionID string,
	nodeIDs []string,
	levelIdx int,
	state *safeState,
	traces *[]NodeTrace,
	tracesMu *sync.Mutex,
) error {
	snap := state.snapshot()

	if len(nodeIDs) == 1 {
		node := w.getNode(nodeIDs[0])
		if node == nil {
			return fmt.Errorf("node %q not found", nodeIDs[0])
		}
		start := time.Now()
		result, err := w.executeNode(ctx, sessionID, node, snap)
		dur := time.Since(start)
		state.set(node.ID+".output", result)

		tracesMu.Lock()
		*traces = append(*traces, NodeTrace{
			NodeID:   node.ID,
			Type:     node.Type,
			Level:    levelIdx,
			Input:    snap,
			Output:   result,
			Duration: dur,
			Error:    errStr(err),
		})
		tracesMu.Unlock()
		return nil
	}

	strategy := w.errorStrategy()
	parallelism := w.maxParallelism()
	levelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, parallelism)
	errCh := make(chan error, len(nodeIDs))
	var wg sync.WaitGroup

	for _, id := range nodeIDs {
		node := w.getNode(id)
		if node == nil {
			cancel()
			return fmt.Errorf("node %q not found", id)
		}
		wg.Add(1)
		go func(n *WorkflowNode) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errCh <- fmt.Errorf("node %q panicked: %v", n.ID, r)
					if strategy == "fail_fast" {
						cancel()
					}
				}
			}()

			select {
			case sem <- struct{}{}:
			case <-levelCtx.Done():
				return
			}
			defer func() { <-sem }()

			if levelCtx.Err() != nil {
				return
			}

			start := time.Now()
			result, err := w.executeNode(levelCtx, sessionID, n, snap)
			dur := time.Since(start)
			state.set(n.ID+".output", result)

			tracesMu.Lock()
			*traces = append(*traces, NodeTrace{
				NodeID:   n.ID,
				Type:     n.Type,
				Level:    levelIdx,
				Input:    snap,
				Output:   result,
				Duration: dur,
				Error:    errStr(err),
			})
			tracesMu.Unlock()

			if err != nil {
				errCh <- fmt.Errorf("node %q: %w", n.ID, err)
				if strategy == "fail_fast" {
					cancel()
				}
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	var errs []string
	for e := range errCh {
		errs = append(errs, e.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", joinStrings(errs, "; "))
	}
	return nil
}

// errStr returns the error string or empty string for nil errors.
func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// hasPrefix is a local helper to avoid importing strings in this file.
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

