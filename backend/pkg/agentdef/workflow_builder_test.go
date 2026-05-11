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
	"strings"
	"testing"
)

// ─── topologicalSort tests ────────────────────────────────────────────────────

func TestWorkflowAgent_TopologicalSort_Linear(t *testing.T) {
	w := &WorkflowAgent{
		nodes: []WorkflowNode{
			{ID: "a", Type: "llm_call"},
			{ID: "b", Type: "llm_call"},
			{ID: "c", Type: "llm_call"},
		},
		edges: []WorkflowEdge{
			{From: "START", To: "a"},
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "END"},
		},
	}

	order, err := w.topologicalSort()
	if err != nil {
		t.Fatalf("topologicalSort error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes in order, got %d: %v", len(order), order)
	}
	// Verify ordering: a must precede b, b must precede c.
	pos := func(id string) int {
		for i, n := range order {
			if n == id {
				return i
			}
		}
		return -1
	}
	if pos("a") >= pos("b") {
		t.Errorf("expected a before b, got order %v", order)
	}
	if pos("b") >= pos("c") {
		t.Errorf("expected b before c, got order %v", order)
	}
}

func TestWorkflowAgent_TopologicalSort_Branching(t *testing.T) {
	// DAG: a → b, a → c, b → d, c → d
	w := &WorkflowAgent{
		nodes: []WorkflowNode{
			{ID: "a", Type: "llm_call"},
			{ID: "b", Type: "llm_call"},
			{ID: "c", Type: "llm_call"},
			{ID: "d", Type: "llm_call"},
		},
		edges: []WorkflowEdge{
			{From: "START", To: "a"},
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
			{From: "d", To: "END"},
		},
	}

	order, err := w.topologicalSort()
	if err != nil {
		t.Fatalf("topologicalSort error: %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d: %v", len(order), order)
	}

	pos := func(id string) int {
		for i, n := range order {
			if n == id {
				return i
			}
		}
		return -1
	}
	if pos("a") >= pos("b") || pos("a") >= pos("c") {
		t.Errorf("a should appear before b and c; order: %v", order)
	}
	if pos("b") >= pos("d") || pos("c") >= pos("d") {
		t.Errorf("b and c should appear before d; order: %v", order)
	}
}

func TestWorkflowAgent_TopologicalSort_CycleDetected(t *testing.T) {
	w := &WorkflowAgent{
		nodes: []WorkflowNode{
			{ID: "a", Type: "llm_call"},
			{ID: "b", Type: "llm_call"},
		},
		edges: []WorkflowEdge{
			{From: "a", To: "b"},
			{From: "b", To: "a"}, // cycle
		},
	}

	_, err := w.topologicalSort()
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

// ─── YAML parsing tests ───────────────────────────────────────────────────────

const workflowYAML = `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-workflow
  version: "1.0.0"
spec:
  type: workflow
  model:
    primary: gpt-4o
  workflow:
    nodes:
      - id: step1
        type: llm_call
        prompt: "Summarise: {{.message}}"
      - id: step2
        type: llm_call
        prompt: "Expand on: {{.step1_output}}"
    edges:
      - from: START
        to: step1
      - from: step1
        to: step2
      - from: step2
        to: END
    variables:
      - name: step1_output
        from: step1.output
`

func TestParse_WorkflowYAML(t *testing.T) {
	def, err := Parse([]byte(workflowYAML))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if def.Spec.Type != "workflow" {
		t.Errorf("Type = %q, want workflow", def.Spec.Type)
	}
	if def.Spec.Workflow == nil {
		t.Fatal("expected non-nil workflow spec")
	}
	if len(def.Spec.Workflow.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(def.Spec.Workflow.Nodes))
	}
	if len(def.Spec.Workflow.Edges) != 3 {
		t.Errorf("expected 3 edges (START→step1, step1→step2, step2→END), got %d", len(def.Spec.Workflow.Edges))
	}
	if len(def.Spec.Workflow.Variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(def.Spec.Workflow.Variables))
	}
	if def.Spec.Workflow.Variables[0].Name != "step1_output" {
		t.Errorf("variable name = %q, want step1_output", def.Spec.Workflow.Variables[0].Name)
	}
}

func TestValidate_WorkflowType_MissingSpec(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: bad-workflow
spec:
  type: workflow
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing workflow spec, got nil")
	}
	if !strings.Contains(err.Error(), "spec.workflow") {
		t.Errorf("error should mention spec.workflow, got: %v", err)
	}
}

func TestValidate_WorkflowType_NoNodes(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: bad-workflow
spec:
  type: workflow
  model:
    primary: gpt-4o
  workflow:
    nodes: []
    edges:
      - from: START
        to: END
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty nodes, got nil")
	}
}

func TestValidate_WorkflowType_NoEdges(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: bad-workflow
spec:
  type: workflow
  model:
    primary: gpt-4o
  workflow:
    nodes:
      - id: step1
        type: llm_call
    edges: []
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty edges, got nil")
	}
}

// ─── WorkflowAgent.Chat tests (mock LLM) ─────────────────────────────────────

// newWorkflowWithMockRegistry returns a WorkflowAgent whose agent_call nodes
// are backed by the provided registry of mock agents.
func newTestWorkflowAgent(nodes []WorkflowNode, edges []WorkflowEdge, variables []WorkflowVariable, registry map[string]Agent) *WorkflowAgent {
	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "test-wf"},
		Spec:       AgentSpec{Type: "workflow"},
	}
	return &WorkflowAgent{
		name:  "test-wf",
		nodes: nodes,
		edges: edges,
		variables: variables,
		registry: func(name string) (Agent, bool) {
			a, ok := registry[name]
			return a, ok
		},
		def: def,
	}
}

func TestWorkflowAgent_Chat_LinearLLM(t *testing.T) {
	// Two llm_call nodes backed by stub chatAgents (no modelCfg.BaseURL).
	// The stub returns "[<modelID>] placeholder response for: <input>".
	nodes := []WorkflowNode{
		{ID: "n1", Type: "llm_call", Prompt: "step1 prompt"},
		{ID: "n2", Type: "llm_call", Prompt: "step2 prompt"},
	}
	edges := []WorkflowEdge{
		{From: "START", To: "n1"},
		{From: "n1", To: "n2"},
		{From: "n2", To: "END"},
	}

	w := newTestWorkflowAgent(nodes, edges, nil, nil)

	ch, err := w.Chat(context.Background(), "sess", "hello")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	output := buf.String()
	// The stub returns the model-ID-prefixed placeholder, which will contain
	// the input derived from the state map.
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestWorkflowAgent_Chat_AgentCall(t *testing.T) {
	mockA := newMockAgent("worker", "worker-result")
	registry := map[string]Agent{"worker": mockA}

	nodes := []WorkflowNode{
		{ID: "call", Type: "agent_call", Agent: "worker"},
	}
	edges := []WorkflowEdge{
		{From: "START", To: "call"},
		{From: "call", To: "END"},
	}

	w := newTestWorkflowAgent(nodes, edges, nil, registry)
	ch, err := w.Chat(context.Background(), "s", "input")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if buf.String() != "worker-result" {
		t.Errorf("expected worker-result, got %q", buf.String())
	}
}

func TestWorkflowAgent_Chat_AgentNotFound(t *testing.T) {
	nodes := []WorkflowNode{
		{ID: "call", Type: "agent_call", Agent: "missing"},
	}
	edges := []WorkflowEdge{
		{From: "START", To: "call"},
	}

	w := newTestWorkflowAgent(nodes, edges, nil, map[string]Agent{})
	ch, err := w.Chat(context.Background(), "s", "input")
	if err != nil {
		t.Fatalf("Chat should not return an error synchronously: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if !strings.Contains(buf.String(), "workflow error") {
		t.Errorf("expected workflow error message, got %q", buf.String())
	}
}

func TestWorkflowAgent_Chat_VariableResolution(t *testing.T) {
	// n1 produces "result-from-n1"; n2 uses {{.n1_out}} which should resolve to that.
	mockRegistry := map[string]Agent{
		"worker": &mockAgent{name: "worker", response: "result-from-n1"},
	}
	nodes := []WorkflowNode{
		{ID: "n1", Type: "agent_call", Agent: "worker"},
		{ID: "n2", Type: "tool_call", Tool: "echo"},
	}
	edges := []WorkflowEdge{
		{From: "START", To: "n1"},
		{From: "n1", To: "n2"},
		{From: "n2", To: "END"},
	}
	variables := []WorkflowVariable{
		{Name: "n1_out", From: "n1.output"},
	}

	w := newTestWorkflowAgent(nodes, edges, variables, mockRegistry)
	ch, err := w.Chat(context.Background(), "s", "query")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	// n2 is the last node — its output comes from the tool placeholder.
	if !strings.Contains(buf.String(), "echo") {
		t.Errorf("expected tool placeholder in output, got %q", buf.String())
	}
}

// ─── Builder.Build workflow dispatch test ────────────────────────────────────

func TestBuild_WorkflowType(t *testing.T) {
	b := NewAgentBuilder()
	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "wf-build-test"},
		Spec: AgentSpec{
			Type:  "workflow",
			Model: ModelSpec{Primary: "gpt-4o"},
			Workflow: &WorkflowSpec{
				Nodes: []WorkflowNode{
					{ID: "step1", Type: "llm_call", Prompt: "hello"},
				},
				Edges: []WorkflowEdge{
					{From: "START", To: "step1"},
					{From: "step1", To: "END"},
				},
			},
		},
	}

	agent, err := b.Build(context.Background(), def)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if _, ok := agent.(*WorkflowAgent); !ok {
		t.Errorf("expected *WorkflowAgent, got %T", agent)
	}
	if agent.Name() != "wf-build-test" {
		t.Errorf("Name() = %q, want wf-build-test", agent.Name())
	}
}

func TestBuild_WorkflowType_NilSpec_ReturnsError(t *testing.T) {
	b := NewAgentBuilder()
	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "wf-nil"},
		Spec: AgentSpec{
			Type:     "workflow",
			Workflow: nil,
		},
	}

	_, err := b.Build(context.Background(), def)
	if err == nil {
		t.Fatal("expected error for nil workflow spec, got nil")
	}
}
