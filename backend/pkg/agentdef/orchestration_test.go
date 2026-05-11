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
	"testing"
)

// ─── mock agent ──────────────────────────────────────────────────────────────

// mockAgent is a test double that returns a fixed response.
type mockAgent struct {
	name     string
	response string
	def      *AgentDefinition
}

func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return "mock: " + m.name }
func (m *mockAgent) GetDefinition() *AgentDefinition { return m.def }

func (m *mockAgent) Chat(_ context.Context, _ string, _ string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- m.response
	close(ch)
	return ch, nil
}

func newMockAgent(name, response string) *mockAgent {
	return &mockAgent{
		name:     name,
		response: response,
		def: &AgentDefinition{
			APIVersion: "superagent/v1",
			Kind:       "Agent",
			Metadata:   Metadata{Name: name},
			Spec: AgentSpec{
				Type:         "chat_model_agent",
				Model:        ModelSpec{Primary: "test-model"},
				SystemPrompt: "mock: " + name,
			},
		},
	}
}

// ─── SupervisorAgent tests ────────────────────────────────────────────────────

func TestSupervisorAgent_Name(t *testing.T) {
	main := newMockAgent("main", "supervisor response")
	sv := &SupervisorAgent{
		name:      "supervisor",
		mainAgent: main,
		subAgents: map[string]Agent{},
		def:       main.def,
	}
	if sv.Name() != "supervisor" {
		t.Errorf("Name() = %q, want %q", sv.Name(), "supervisor")
	}
}

func TestSupervisorAgent_Description(t *testing.T) {
	main := newMockAgent("main", "supervisor response")
	sv := &SupervisorAgent{
		name:        "supervisor",
		description: "my description",
		mainAgent:   main,
		subAgents:   map[string]Agent{},
		def:         main.def,
	}
	if sv.Description() != "my description" {
		t.Errorf("Description() = %q, want %q", sv.Description(), "my description")
	}
}

func TestSupervisorAgent_Chat(t *testing.T) {
	main := newMockAgent("main", "delegated response")
	subA := newMockAgent("agent-a", "response-a")
	subB := newMockAgent("agent-b", "response-b")

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "sv"},
		Spec: AgentSpec{
			Type:         "supervisor",
			SystemPrompt: "You are the supervisor.",
			SubAgents: []SubAgentRef{
				{Ref: "agent-a", Role: "handles A"},
				{Ref: "agent-b", Role: "handles B"},
			},
		},
	}

	sv := &SupervisorAgent{
		name:        "sv",
		description: "test supervisor",
		mainAgent:   main,
		subAgents:   map[string]Agent{"agent-a": subA, "agent-b": subB},
		maxRounds:   5,
		def:         def,
	}

	ch, err := sv.Chat(context.Background(), "session-1", "hello")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if buf.String() != "delegated response" {
		t.Errorf("Chat output = %q, want %q", buf.String(), "delegated response")
	}
}

func TestBuildSupervisorPrompt_WithSubAgents(t *testing.T) {
	subAgents := map[string]Agent{
		"researcher": newMockAgent("researcher", ""),
	}
	prompt := buildSupervisorPrompt("Base prompt.", subAgents)
	if !strings.Contains(prompt, "Base prompt.") {
		t.Error("prompt should contain the base prompt")
	}
	if !strings.Contains(prompt, "researcher") {
		t.Error("prompt should mention sub-agent 'researcher'")
	}
}

func TestBuildSupervisorPrompt_NoSubAgents(t *testing.T) {
	base := "Base prompt only."
	prompt := buildSupervisorPrompt(base, map[string]Agent{})
	if prompt != base {
		t.Errorf("prompt with no sub-agents should equal base prompt, got %q", prompt)
	}
}

// ─── SequentialAgent tests ────────────────────────────────────────────────────

func TestSequentialAgent_Name(t *testing.T) {
	a := newMockAgent("a", "out-a")
	seq := &SequentialAgent{name: "pipeline", agents: []Agent{a}, def: a.def}
	if seq.Name() != "pipeline" {
		t.Errorf("Name() = %q, want %q", seq.Name(), "pipeline")
	}
}

func TestSequentialAgent_Chat_SingleAgent(t *testing.T) {
	a := newMockAgent("a", "only output")
	def := &AgentDefinition{
		Metadata: Metadata{Name: "seq"},
		Spec:     AgentSpec{Type: "sequential"},
	}
	seq := &SequentialAgent{name: "seq", agents: []Agent{a}, def: def}

	ch, err := seq.Chat(context.Background(), "s", "hello")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if buf.String() != "only output" {
		t.Errorf("output = %q, want %q", buf.String(), "only output")
	}
}

func TestSequentialAgent_Chat_Chain(t *testing.T) {
	// Each agent echoes "step-N: <input>" so we can verify chaining.
	makeStep := func(n int) Agent {
		return &mockAgent{
			name:     fmt.Sprintf("step-%d", n),
			response: fmt.Sprintf("step-%d output", n),
			def:      newMockAgent(fmt.Sprintf("step-%d", n), "").def,
		}
	}

	def := &AgentDefinition{
		Metadata: Metadata{Name: "chain"},
		Spec:     AgentSpec{Type: "sequential"},
	}
	seq := &SequentialAgent{
		name:   "chain",
		agents: []Agent{makeStep(1), makeStep(2), makeStep(3)},
		def:    def,
	}

	ch, err := seq.Chat(context.Background(), "s", "start")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	// Only the last agent's output is streamed.
	if !strings.Contains(buf.String(), "step-3 output") {
		t.Errorf("expected last step output, got %q", buf.String())
	}
}

// ─── ParallelAgent tests ──────────────────────────────────────────────────────

func TestParallelAgent_Name(t *testing.T) {
	a := newMockAgent("a", "")
	par := &ParallelAgent{name: "parallel", agents: []Agent{a}, def: a.def}
	if par.Name() != "parallel" {
		t.Errorf("Name() = %q, want %q", par.Name(), "parallel")
	}
}

func TestParallelAgent_Chat_CombinesResults(t *testing.T) {
	a := newMockAgent("alpha", "alpha-result")
	b := newMockAgent("beta", "beta-result")

	def := &AgentDefinition{
		Metadata: Metadata{Name: "par"},
		Spec:     AgentSpec{Type: "parallel"},
	}
	par := &ParallelAgent{
		name:   "par",
		agents: []Agent{a, b},
		def:    def,
	}

	ch, err := par.Chat(context.Background(), "s", "hello")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	combined := buf.String()
	if !strings.Contains(combined, "alpha-result") {
		t.Errorf("expected alpha-result in output, got %q", combined)
	}
	if !strings.Contains(combined, "beta-result") {
		t.Errorf("expected beta-result in output, got %q", combined)
	}
}

func TestParallelAgent_Chat_SingleAgent(t *testing.T) {
	a := newMockAgent("solo", "solo-result")
	def := &AgentDefinition{
		Metadata: Metadata{Name: "par-solo"},
		Spec:     AgentSpec{Type: "parallel"},
	}
	par := &ParallelAgent{name: "par-solo", agents: []Agent{a}, def: def}

	ch, err := par.Chat(context.Background(), "s", "query")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if !strings.Contains(buf.String(), "solo-result") {
		t.Errorf("expected solo-result in output, got %q", buf.String())
	}
}

// ─── Builder orchestration tests ─────────────────────────────────────────────

func TestBuildSequential_ResolvesSubAgents(t *testing.T) {
	agentA := newMockAgent("agent-a", "a-out")
	agentB := newMockAgent("agent-b", "b-out")

	registry := map[string]Agent{
		"agent-a": agentA,
		"agent-b": agentB,
	}

	b := NewAgentBuilder(WithAgentRegistry(func(name string) (Agent, bool) {
		a, ok := registry[name]
		return a, ok
	}))

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "seq-test"},
		Spec: AgentSpec{
			Type: "sequential",
			SubAgents: []SubAgentRef{
				{Ref: "agent-a"},
				{Ref: "agent-b"},
			},
		},
	}

	agent, err := b.Build(context.Background(), def)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if agent.Name() != "seq-test" {
		t.Errorf("Name() = %q, want seq-test", agent.Name())
	}
	if _, ok := agent.(*SequentialAgent); !ok {
		t.Errorf("expected *SequentialAgent, got %T", agent)
	}
}

func TestBuildParallel_ResolvesSubAgents(t *testing.T) {
	agentA := newMockAgent("par-a", "pa-out")
	agentB := newMockAgent("par-b", "pb-out")

	registry := map[string]Agent{
		"par-a": agentA,
		"par-b": agentB,
	}

	b := NewAgentBuilder(WithAgentRegistry(func(name string) (Agent, bool) {
		a, ok := registry[name]
		return a, ok
	}))

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "par-test"},
		Spec: AgentSpec{
			Type: "parallel",
			SubAgents: []SubAgentRef{
				{Ref: "par-a"},
				{Ref: "par-b"},
			},
		},
	}

	agent, err := b.Build(context.Background(), def)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if _, ok := agent.(*ParallelAgent); !ok {
		t.Errorf("expected *ParallelAgent, got %T", agent)
	}
}

func TestBuild_MissingSubAgent_ReturnsError(t *testing.T) {
	registry := map[string]Agent{} // empty — no sub-agents registered

	b := NewAgentBuilder(WithAgentRegistry(func(name string) (Agent, bool) {
		a, ok := registry[name]
		return a, ok
	}))

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "seq-missing"},
		Spec: AgentSpec{
			Type:      "sequential",
			SubAgents: []SubAgentRef{{Ref: "nonexistent"}},
		},
	}

	_, err := b.Build(context.Background(), def)
	if err == nil {
		t.Fatal("expected error for missing sub-agent, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention missing agent name, got: %v", err)
	}
}

func TestBuild_NoRegistry_ReturnsError(t *testing.T) {
	b := NewAgentBuilder() // no registry

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "seq-noreg"},
		Spec: AgentSpec{
			Type:      "sequential",
			SubAgents: []SubAgentRef{{Ref: "some-agent"}},
		},
	}

	_, err := b.Build(context.Background(), def)
	if err == nil {
		t.Fatal("expected error when no registry is set, got nil")
	}
}

// ─── Validate orchestration types ────────────────────────────────────────────

func TestValidate_SupervisorType_Valid(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-supervisor
spec:
  type: supervisor
  model:
    primary: gpt-4o
  sub_agents:
    - ref: research-agent
      role: researcher
  orchestration:
    mode: supervisor
    max_rounds: 3
`
	def, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if def.Spec.Type != "supervisor" {
		t.Errorf("Type = %q, want supervisor", def.Spec.Type)
	}
	if len(def.Spec.SubAgents) != 1 || def.Spec.SubAgents[0].Ref != "research-agent" {
		t.Errorf("unexpected sub_agents: %+v", def.Spec.SubAgents)
	}
	if def.Spec.Orchestration == nil || def.Spec.Orchestration.MaxRounds != 3 {
		t.Errorf("unexpected orchestration: %+v", def.Spec.Orchestration)
	}
}

func TestValidate_SequentialType_Valid(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-pipeline
spec:
  type: sequential
  sub_agents:
    - ref: step-one
    - ref: step-two
`
	def, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if def.Spec.Type != "sequential" {
		t.Errorf("Type = %q, want sequential", def.Spec.Type)
	}
	if len(def.Spec.SubAgents) != 2 {
		t.Errorf("expected 2 sub_agents, got %d", len(def.Spec.SubAgents))
	}
}

func TestValidate_ParallelType_Valid(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-fan-out
spec:
  type: parallel
  sub_agents:
    - ref: worker-a
    - ref: worker-b
`
	def, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if def.Spec.Type != "parallel" {
		t.Errorf("Type = %q, want parallel", def.Spec.Type)
	}
}

func TestValidate_OrchestrationNoSubAgents_ReturnsError(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: bad-supervisor
spec:
  type: supervisor
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error when supervisor has no sub_agents, got nil")
	}
}

func TestValidate_OrchestrationEmptyRef_ReturnsError(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: bad-seq
spec:
  type: sequential
  sub_agents:
    - ref: ""
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error when sub_agents ref is empty, got nil")
	}
}
