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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/evolution"
)

// WorkflowAgent executes a graph-based workflow defined as a DAG of nodes
// connected by directed edges.  Execution follows a topological order derived
// from the edge declarations; each node's output is stored in a shared state
// map and made available to downstream nodes via {{.varName}} template syntax.
type WorkflowAgent struct {
	name        string
	description string
	nodes       []WorkflowNode
	edges       []WorkflowEdge
	variables   []WorkflowVariable
	registry    func(name string) (Agent, bool) // for agent_call resolution
	modelCfg    ModelRuntimeConfig
	def         *AgentDefinition
	// collector is optional; when non-nil node execution signals are reported.
	collector *evolution.SignalCollector
}

func (w *WorkflowAgent) Name() string                    { return w.name }
func (w *WorkflowAgent) Description() string             { return w.description }
func (w *WorkflowAgent) GetDefinition() *AgentDefinition { return w.def }

// Chat executes the workflow level-by-level, running same-level nodes in
// parallel up to maxParallelism.  The final level's last-written output is
// streamed back to the caller.
func (w *WorkflowAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		// Seed the state with the incoming message.
		state := newSafeState(map[string]string{"message": message})

		levels, err := w.topologicalLevels()
		if err != nil {
			ch <- fmt.Sprintf("[workflow error] %v", err)
			return
		}

		if len(levels) == 0 {
			ch <- "[workflow error] no executable nodes found"
			return
		}

		var lastNodeID string
		for _, level := range levels {
			if execErr := w.executeLevel(ctx, sessionID, level, state); execErr != nil {
				ch <- fmt.Sprintf("[workflow error] %v", execErr)
				return
			}
			// Apply variable aliases for all nodes in this level (single-threaded).
			for _, nodeID := range level {
				result := state.get(nodeID + ".output")
				for _, v := range w.variables {
					if strings.HasPrefix(v.From, nodeID+".") {
						state.set(v.Name, result)
					}
				}
			}
			lastNodeID = level[len(level)-1]
		}

		// Stream the final node's output.
		ch <- state.get(lastNodeID + ".output")
	}()
	return ch, nil
}

// executeNode dispatches to the appropriate node handler based on node.Type.
// It wraps execution with timing and signals to the evolution collector when present.
func (w *WorkflowAgent) executeNode(ctx context.Context, sessionID string, node *WorkflowNode, state map[string]string) (string, error) {
	start := time.Now()
	var result string
	var execErr error

	switch node.Type {
	case "llm_call":
		result, execErr = w.executeLLMNode(ctx, sessionID, node, state)
	case "agent_call":
		result, execErr = w.executeAgentNode(ctx, sessionID, node, state)
	case "tool_call":
		result, execErr = w.executeToolNode(node, state)
	case "code":
		result, execErr = w.executeCodeNode(ctx, node, state)
	case "condition":
		result, execErr = w.executeConditionNode(node, state)
	default:
		return "", fmt.Errorf("unknown node type %q", node.Type)
	}

	// Report node_done signal to evolution engine asynchronously.
	if w.collector != nil {
		sig := evolution.Signal{
			Type:      "node_done",
			AgentName: w.name,
			SessionID: sessionID,
			Component: node.ID,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
			Metadata:  map[string]any{"node_type": node.Type},
		}
		if execErr != nil {
			sig.Type = "node_error"
			sig.Error = execErr.Error()
		} else {
			sig.Output = evolution.Truncate(result, 200)
		}
		w.collector.Collect(ctx, sig)
	}

	return result, execErr
}


// executeLLMNode runs an LLM inference step.  When a real model endpoint is
// configured a lightweight einoChatAgent is created on the fly; otherwise the
// stub chatAgent is used so tests without a live model continue to pass.
func (w *WorkflowAgent) executeLLMNode(ctx context.Context, sessionID string, node *WorkflowNode, state map[string]string) (string, error) {
	prompt := w.resolveTemplate(node.Prompt, state)
	input := w.resolveInput(node.InputMapping, state)

	// Synthesise a minimal AgentDefinition so the embedded agent satisfies the
	// Agent interface contract (Name, Description, GetDefinition).
	// H9 fix: prefer the workflow's YAML-declared model.primary over the global
	// runtime model; fall back to global only if YAML primary is empty.
	nodeModelID := w.def.Spec.Model.Primary
	if nodeModelID == "" {
		nodeModelID = w.modelCfg.ModelID
	}
	synthDef := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: node.ID},
		Spec: AgentSpec{
			Type:         "chat_model_agent",
			Model:        ModelSpec{Primary: nodeModelID},
			SystemPrompt: prompt,
		},
	}

	var nodeAgent Agent
	if w.modelCfg.BaseURL != "" {
		nodeBuilder := NewAgentBuilder(WithModelConfig(w.modelCfg))
		built, err := nodeBuilder.Build(ctx, synthDef)
		if err != nil {
			return "", fmt.Errorf("executeLLMNode: build agent: %w", err)
		}
		nodeAgent = built
	} else {
		nodeAgent = &chatAgent{
			def:     synthDef,
			modelID: synthDef.Spec.Model.Primary,
		}
	}

	ch, err := nodeAgent.Chat(ctx, sessionID+"-"+node.ID, input)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for tok := range ch {
		sb.WriteString(tok)
	}
	return sb.String(), nil
}

// executeAgentNode delegates execution to a named agent resolved from the registry.
func (w *WorkflowAgent) executeAgentNode(ctx context.Context, sessionID string, node *WorkflowNode, state map[string]string) (string, error) {
	if w.registry == nil {
		return "", fmt.Errorf("executeAgentNode: no agent registry configured")
	}
	agent, ok := w.registry(node.Agent)
	if !ok {
		return "", fmt.Errorf("executeAgentNode: agent %q not found", node.Agent)
	}

	input := w.resolveInput(node.InputMapping, state)
	ch, err := agent.Chat(ctx, sessionID+"-"+node.ID, input)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for tok := range ch {
		sb.WriteString(tok)
	}
	return sb.String(), nil
}

// executeToolNode is a placeholder for tool invocation.
// Full implementation will delegate to the tool.Manager in a future iteration.
func (w *WorkflowAgent) executeToolNode(node *WorkflowNode, state map[string]string) (string, error) {
	input := w.resolveInput(node.InputMapping, state)
	return fmt.Sprintf("[tool:%s] input=%s", node.Tool, input), nil
}

// codeSandboxMode determines the execution mode for code nodes.
// Controlled by env WORKFLOW_CODE_SANDBOX: "docker" (default) or "none" (reject).
func codeSandboxMode() string {
	mode := os.Getenv("WORKFLOW_CODE_SANDBOX")
	if mode == "" {
		mode = "docker"
	}
	return mode
}

// dockerAvailable checks if Docker daemon is reachable.
// The result is cached for 30 seconds so that a temporarily unavailable Docker
// daemon is detected on the next attempt instead of being permanently cached.
var (
	dockerMu        sync.Mutex
	dockerOK        bool
	dockerCheckedAt time.Time
)

func checkDockerAvailable() bool {
	dockerMu.Lock()
	defer dockerMu.Unlock()
	if time.Since(dockerCheckedAt) < 30*time.Second {
		return dockerOK
	}
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	dockerOK = cmd.Run() == nil
	dockerCheckedAt = time.Now()
	return dockerOK
}

// dockerImageForLang returns the Docker image to use for a given language.
func dockerImageForLang(lang string) string {
	switch lang {
	case "python":
		return "python:3.11-slim"
	case "javascript", "js":
		return "node:20-slim"
	case "bash", "shell", "sh":
		return "alpine:3.19"
	default:
		return ""
	}
}

// executeCodeNode executes user-defined code in a sandboxed container.
// Supported languages: python, javascript, bash.
// Input data is passed via stdin to eliminate injection vectors.
// Execution is bounded by a 10-second timeout.
//
// Security: Code runs inside a Docker container with:
//   - --network=none (no network access)
//   - --memory=128m (memory limit)
//   - --cpus=0.5 (CPU limit)
//   - --read-only (read-only filesystem)
//   - --no-new-privileges
//
// When Docker is unavailable or WORKFLOW_CODE_SANDBOX=none, execution is rejected.
func (w *WorkflowAgent) executeCodeNode(ctx context.Context, node *WorkflowNode, state map[string]string) (string, error) {
	code := w.resolveTemplate(node.Code, state)
	input := w.resolveInput(node.InputMapping, state)
	lang := strings.ToLower(node.Language)

	if lang == "" {
		return "", fmt.Errorf("executeCodeNode: language is required")
	}

	// Check sandbox mode.
	mode := codeSandboxMode()
	if mode == "none" {
		return "", fmt.Errorf("executeCodeNode: code execution is disabled (WORKFLOW_CODE_SANDBOX=none)")
	}

	if !checkDockerAvailable() {
		return "", fmt.Errorf("executeCodeNode: Docker is not available; code execution requires a container runtime")
	}

	image := dockerImageForLang(lang)
	if image == "" {
		return "", fmt.Errorf("executeCodeNode: unsupported language %q (supported: python, javascript, bash)", lang)
	}

	// Build the command to run inside the container.
	// Input is passed via stdin; code is passed as an inline argument.
	var containerCmd []string
	switch lang {
	case "python":
		// Read stdin into input_data variable, then exec user code.
		wrappedCode := "import sys; input_data = sys.stdin.read()\n" + code
		containerCmd = []string{"python3", "-c", wrappedCode}
	case "javascript", "js":
		wrappedCode := "const input_data = require('fs').readFileSync(0, 'utf8');\n" + code
		containerCmd = []string{"node", "-e", wrappedCode}
	case "bash", "shell", "sh":
		wrappedCode := "INPUT_DATA=$(cat)\n" + code
		containerCmd = []string{"sh", "-c", wrappedCode}
	}

	// Execute with a 10-second timeout derived from the caller's context.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Build docker run command with security constraints.
	args := []string{
		"run", "--rm",
		"--network=none",
		"--memory=128m",
		"--cpus=0.5",
		"--read-only",
		"--no-new-privileges",
		"--tmpfs", "/tmp:rw,size=16m",
		"-i", // stdin enabled
		image,
	}
	args = append(args, containerCmd...)

	execCmd := exec.CommandContext(ctx, "docker", args...)

	// Pass input via stdin to eliminate injection risk.
	execCmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("executeCodeNode: execution timed out after 10s")
	}
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Sprintf("[code error] %s", strings.TrimSpace(errMsg)), nil
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}

// executeConditionNode evaluates a simple condition expression and returns
// "true" or "false" as a string that downstream edges can test.
func (w *WorkflowAgent) executeConditionNode(node *WorkflowNode, state map[string]string) (string, error) {
	expr := w.resolveTemplate(node.Condition, state)
	// Simple placeholder: non-empty expression evaluates to "true".
	if expr != "" && expr != "false" && expr != "0" {
		return "true", nil
	}
	return "false", nil
}

// maxParallelism returns the effective concurrency limit for node execution.
// When the workflow spec declares Execution.MaxParallelism > 0 that value is
// used; otherwise runtime.NumCPU() is used as a sensible default.
func (w *WorkflowAgent) maxParallelism() int {
	if w.def != nil && w.def.Spec.Workflow != nil &&
		w.def.Spec.Workflow.Execution != nil &&
		w.def.Spec.Workflow.Execution.MaxParallelism > 0 {
		return w.def.Spec.Workflow.Execution.MaxParallelism
	}
	return runtime.NumCPU()
}

// errorStrategy returns the configured error strategy, defaulting to "fail_fast".
func (w *WorkflowAgent) errorStrategy() string {
	if w.def != nil && w.def.Spec.Workflow != nil &&
		w.def.Spec.Workflow.Execution != nil &&
		w.def.Spec.Workflow.Execution.ErrorStrategy != "" {
		return w.def.Spec.Workflow.Execution.ErrorStrategy
	}
	return "fail_fast"
}

// executeLevel runs all nodeIDs in a single topological level concurrently.
// Nodes read a snapshot of state taken before the level starts; each node
// writes only its own nodeID.output key so there are no write conflicts.
//
// Error strategies:
//   - fail_fast (default): cancel all sibling goroutines on first error.
//   - best_effort: wait for all goroutines, collect all errors.
func (w *WorkflowAgent) executeLevel(ctx context.Context, sessionID string, nodeIDs []string, state *safeState) error {
	if len(nodeIDs) == 1 {
		// Fast path: single node — no goroutine overhead, preserves serial semantics.
		node := w.getNode(nodeIDs[0])
		if node == nil {
			return fmt.Errorf("node %q not found", nodeIDs[0])
		}
		snap := state.snapshot()
		result, err := w.executeNode(ctx, sessionID, node, snap)
		if err != nil {
			return fmt.Errorf("node %q: %w", node.ID, err)
		}
		state.set(node.ID+".output", result)
		return nil
	}

	strategy := w.errorStrategy()
	parallelism := w.maxParallelism()

	// Derive a cancellable child context for fail_fast support.
	levelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, parallelism)
	errCh := make(chan error, len(nodeIDs))
	var wg sync.WaitGroup

	// Take a single snapshot before launching goroutines so all nodes see the
	// same upstream state (core invariant: same-level nodes only read outputs
	// from previous levels).
	snap := state.snapshot()

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
					err := fmt.Errorf("node %q panicked: %v", n.ID, r)
					errCh <- err
					if strategy == "fail_fast" {
						cancel()
					}
				}
			}()

			// Acquire semaphore slot.
			select {
			case sem <- struct{}{}:
			case <-levelCtx.Done():
				return
			}
			defer func() { <-sem }()

			// Bail early if context was already cancelled (fail_fast).
			if levelCtx.Err() != nil {
				return
			}

			result, err := w.executeNode(levelCtx, sessionID, n, snap)
			if err != nil {
				errCh <- fmt.Errorf("node %q: %w", n.ID, err)
				if strategy == "fail_fast" {
					cancel()
				}
				return
			}
			state.set(n.ID+".output", result)
		}(node)
	}

	wg.Wait()
	close(errCh)

	// Collect errors.
	var errs []string
	for e := range errCh {
		errs = append(errs, e.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// topologicalLevels returns nodes grouped by topological level using Kahn's
// algorithm.  Nodes at the same level have no dependency on each other and
// may be executed in parallel.  "START" and "END" sentinels are ignored.
// Returns an error if the graph contains a cycle.
func (w *WorkflowAgent) topologicalLevels() ([][]string, error) {
	// Build a set of valid node IDs.
	nodeSet := make(map[string]struct{}, len(w.nodes))
	for _, n := range w.nodes {
		nodeSet[n.ID] = struct{}{}
	}

	inDegree := make(map[string]int, len(w.nodes))
	adj := make(map[string][]string, len(w.nodes))
	for _, n := range w.nodes {
		if _, exists := inDegree[n.ID]; !exists {
			inDegree[n.ID] = 0
		}
	}

	for _, e := range w.edges {
		from, to := e.From, e.To
		if from == "START" || to == "END" {
			continue
		}
		if _, ok := nodeSet[from]; !ok {
			continue
		}
		if _, ok := nodeSet[to]; !ok {
			continue
		}
		adj[from] = append(adj[from], to)
		inDegree[to]++
	}

	// Initialise queue with all zero-in-degree nodes.
	queue := make([]string, 0, len(w.nodes))
	for _, n := range w.nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var levels [][]string
	visited := 0
	for len(queue) > 0 {
		level := queue
		queue = nil
		levels = append(levels, level)
		visited += len(level)
		for _, curr := range level {
			for _, neighbor := range adj[curr] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					queue = append(queue, neighbor)
				}
			}
		}
	}

	if visited != len(w.nodes) {
		return nil, fmt.Errorf("workflow graph contains a cycle")
	}
	return levels, nil
}

// topologicalSort returns node IDs in a valid execution order using Kahn's
// algorithm.  "START" and "END" sentinel values in edges are ignored.
// Returns an error if the graph contains a cycle.
func (w *WorkflowAgent) topologicalSort() ([]string, error) {
	// Build a set of valid node IDs.
	nodeSet := make(map[string]struct{}, len(w.nodes))
	for _, n := range w.nodes {
		nodeSet[n.ID] = struct{}{}
	}

	// Build in-degree count and adjacency list considering only real nodes.
	inDegree := make(map[string]int, len(w.nodes))
	adj := make(map[string][]string, len(w.nodes))
	for _, n := range w.nodes {
		if _, exists := inDegree[n.ID]; !exists {
			inDegree[n.ID] = 0
		}
	}

	for _, e := range w.edges {
		from, to := e.From, e.To
		// Skip sentinel edges.
		if from == "START" || to == "END" {
			// If from is START, the target node starts with in-degree 0
			// (already initialised above). Nothing to change.
			continue
		}
		if _, ok := nodeSet[from]; !ok {
			continue
		}
		if _, ok := nodeSet[to]; !ok {
			continue
		}
		adj[from] = append(adj[from], to)
		inDegree[to]++
	}

	// Initialise queue with all zero-in-degree nodes.
	queue := make([]string, 0, len(w.nodes))
	for _, n := range w.nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	order := make([]string, 0, len(w.nodes))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		for _, neighbor := range adj[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(order) != len(w.nodes) {
		return nil, fmt.Errorf("workflow graph contains a cycle")
	}
	return order, nil
}

// resolveTemplate replaces all {{.varName}} occurrences in tmpl with the
// corresponding values from state.
func (w *WorkflowAgent) resolveTemplate(tmpl string, state map[string]string) string {
	result := tmpl
	for k, v := range state {
		result = strings.ReplaceAll(result, "{{."+k+"}}", v)
	}
	return result
}

// resolveInput builds the input string from an input_mapping declaration.
// Each mapped value is resolved from state; results are joined with newlines.
// When the mapping is empty, the raw "message" from state is used.
func (w *WorkflowAgent) resolveInput(mapping map[string]string, state map[string]string) string {
	if len(mapping) == 0 {
		return state["message"]
	}
	parts := make([]string, 0, len(mapping))
	for key, expr := range mapping {
		// Resolve "$." prefix as a state lookup, otherwise use the expression
		// as a literal template.
		var val string
		if strings.HasPrefix(expr, "$.") {
			stateKey := strings.TrimPrefix(expr, "$.")
			val = state[stateKey]
		} else {
			val = w.resolveTemplate(expr, state)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, val))
	}
	return strings.Join(parts, "\n")
}

// getNode returns the WorkflowNode with the given ID, or nil if not found.
func (w *WorkflowAgent) getNode(id string) *WorkflowNode {
	for i := range w.nodes {
		if w.nodes[i].ID == id {
			return &w.nodes[i]
		}
	}
	return nil
}
