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
	"time"
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

// ─── SupervisorAgent V2 delegation tests ─────────────────────────────────────

// mockMainAgentDelegating returns a JSON delegation directive on the first call,
// then a plain-text final answer on the second call.
type mockMainAgentDelegating struct {
	name      string
	callCount int
	responses []string
	def       *AgentDefinition
}

func (m *mockMainAgentDelegating) Name() string                    { return m.name }
func (m *mockMainAgentDelegating) Description() string             { return "delegating mock" }
func (m *mockMainAgentDelegating) GetDefinition() *AgentDefinition { return m.def }

func (m *mockMainAgentDelegating) Chat(_ context.Context, _ string, _ string) (<-chan string, error) {
	ch := make(chan string, 1)
	resp := ""
	if m.callCount < len(m.responses) {
		resp = m.responses[m.callCount]
	}
	m.callCount++
	ch <- resp
	close(ch)
	return ch, nil
}

func TestSupervisorV2_SingleDelegation(t *testing.T) {
	subA := newMockAgent("worker-a", "worker-a result")

	main := &mockMainAgentDelegating{
		name: "main",
		// Round 1: delegate to worker-a; Round 2: final answer
		responses: []string{
			`{"delegations":[{"agent_name":"worker-a","task":"do the thing"}]}`,
			"Final answer based on worker-a result.",
		},
		def: newMockAgent("main", "").def,
	}

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "sv"},
		Spec: AgentSpec{
			Type:         "supervisor",
			SystemPrompt: "You coordinate.",
			SubAgents:    []SubAgentRef{{Ref: "worker-a"}},
			Orchestration: &OrchestrationSpec{
				Mode:      "supervisor",
				MaxRounds: 5,
			},
		},
	}

	dt := &delegateTool{
		subAgents:   map[string]Agent{"worker-a": subA},
		timeout:     5 * time.Second,
		fallback:    "ask_supervisor",
		parallelMax: 3,
	}

	sv := &SupervisorAgent{
		name:      "sv",
		mainAgent: main,
		subAgents: map[string]Agent{"worker-a": subA},
		maxRounds: 5,
		delegate:  dt,
		def:       def,
	}

	ch, err := sv.Chat(context.Background(), "session-1", "start task")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	output := buf.String()
	if !strings.Contains(output, "Final answer") {
		t.Errorf("expected final answer in output, got: %q", output)
	}
}

func TestSupervisorV2_ParallelDelegation(t *testing.T) {
	subA := newMockAgent("agent-a", "result-a")
	subB := newMockAgent("agent-b", "result-b")

	main := &mockMainAgentDelegating{
		name: "main",
		responses: []string{
			`{"delegations":[{"agent_name":"agent-a","task":"task A"},{"agent_name":"agent-b","task":"task B"}]}`,
			"Both done.",
		},
		def: newMockAgent("main", "").def,
	}

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "sv-par"},
		Spec: AgentSpec{
			Type:         "supervisor",
			SystemPrompt: "Coordinate.",
			SubAgents:    []SubAgentRef{{Ref: "agent-a"}, {Ref: "agent-b"}},
			Orchestration: &OrchestrationSpec{
				Mode:      "supervisor",
				MaxRounds: 5,
				Delegation: &DelegationConfig{
					ParallelMax: 2,
					Timeout:     "5s",
				},
			},
		},
	}

	dt := &delegateTool{
		subAgents:   map[string]Agent{"agent-a": subA, "agent-b": subB},
		timeout:     5 * time.Second,
		fallback:    "ask_supervisor",
		parallelMax: 2,
	}

	sv := &SupervisorAgent{
		name:      "sv-par",
		mainAgent: main,
		subAgents: map[string]Agent{"agent-a": subA, "agent-b": subB},
		maxRounds: 5,
		delegate:  dt,
		def:       def,
	}

	ch, err := sv.Chat(context.Background(), "session-2", "start")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	output := buf.String()
	if !strings.Contains(output, "Both done") {
		t.Errorf("expected final 'Both done' in output, got: %q", output)
	}
}

func TestSupervisorV2_MaxRoundsExceeded(t *testing.T) {
	subA := newMockAgent("worker", "result")

	// main always delegates — never produces a final answer.
	main := &mockMainAgentDelegating{
		name: "main",
		responses: []string{
			`{"delegations":[{"agent_name":"worker","task":"loop"}]}`,
			`{"delegations":[{"agent_name":"worker","task":"loop"}]}`,
			`{"delegations":[{"agent_name":"worker","task":"loop"}]}`,
		},
		def: newMockAgent("main", "").def,
	}

	def := &AgentDefinition{
		Metadata: Metadata{Name: "sv-loop"},
		Spec: AgentSpec{
			Type:         "supervisor",
			SystemPrompt: "Loop forever.",
			Orchestration: &OrchestrationSpec{
				Mode:      "supervisor",
				MaxRounds: 2,
			},
		},
	}

	dt := &delegateTool{
		subAgents:   map[string]Agent{"worker": subA},
		timeout:     5 * time.Second,
		fallback:    "ask_supervisor",
		parallelMax: 3,
	}

	sv := &SupervisorAgent{
		name:      "sv-loop",
		mainAgent: main,
		subAgents: map[string]Agent{"worker": subA},
		maxRounds: 2,
		delegate:  dt,
		def:       def,
	}

	ch, err := sv.Chat(context.Background(), "session-3", "start")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	output := buf.String()
	if !strings.Contains(output, "max rounds") {
		t.Errorf("expected max rounds message, got: %q", output)
	}
}

func TestSupervisorV2_TimeoutFallback(t *testing.T) {
	// slowAgent blocks until context is cancelled.
	slowAgent := &mockAgentSlow{name: "slow", delay: 500 * time.Millisecond}

	main := &mockMainAgentDelegating{
		name: "main",
		responses: []string{
			`{"delegations":[{"agent_name":"slow","task":"be slow"}]}`,
			"Done after timeout.",
		},
		def: newMockAgent("main", "").def,
	}

	def := &AgentDefinition{
		Metadata: Metadata{Name: "sv-timeout"},
		Spec: AgentSpec{
			Type:         "supervisor",
			SystemPrompt: "Test timeout.",
			Orchestration: &OrchestrationSpec{
				Mode:      "supervisor",
				MaxRounds: 5,
				Delegation: &DelegationConfig{
					Timeout:          "10ms",
					FallbackStrategy: "ask_supervisor",
				},
			},
		},
	}

	dt := &delegateTool{
		subAgents:   map[string]Agent{"slow": slowAgent},
		timeout:     10 * time.Millisecond,
		fallback:    "ask_supervisor",
		parallelMax: 3,
	}

	sv := &SupervisorAgent{
		name:      "sv-timeout",
		mainAgent: main,
		subAgents: map[string]Agent{"slow": slowAgent},
		maxRounds: 5,
		delegate:  dt,
		def:       def,
	}

	ch, err := sv.Chat(context.Background(), "session-4", "start")
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	output := buf.String()
	// The slow agent times out; supervisor gets a "timeout" result and then
	// the main agent produces a final answer on the next round.
	if !strings.Contains(output, "Done after timeout") {
		t.Errorf("expected final answer after timeout, got: %q", output)
	}
}

// mockAgentSlow sleeps for `delay` before returning, simulating a slow sub-agent.
type mockAgentSlow struct {
	name  string
	delay time.Duration
	def   *AgentDefinition
}

func (m *mockAgentSlow) Name() string                    { return m.name }
func (m *mockAgentSlow) Description() string             { return "slow mock" }
func (m *mockAgentSlow) GetDefinition() *AgentDefinition { return m.def }

func (m *mockAgentSlow) Chat(ctx context.Context, _ string, _ string) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		select {
		case <-time.After(m.delay):
			ch <- "slow result"
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

// ─── delegateTool unit tests ──────────────────────────────────────────────────

func TestDelegateTool_Execute_Success(t *testing.T) {
	worker := newMockAgent("w", "worker output")
	dt := &delegateTool{
		subAgents:   map[string]Agent{"w": worker},
		timeout:     5 * time.Second,
		fallback:    "skip",
		parallelMax: 3,
	}

	ctx := withDelegationScope(context.Background(), "test-session", 1)
	out := dt.execute(ctx, DelegateToolInput{
		AgentName: "w",
		Task:      "do work",
	}, "test-session", 1)
	if out.Status != "success" {
		t.Errorf("status = %q, want success", out.Status)
	}
	if out.Result != "worker output" {
		t.Errorf("result = %q, want %q", out.Result, "worker output")
	}
}

func TestDelegateTool_Execute_MissingAgent(t *testing.T) {
	dt := &delegateTool{
		subAgents:   map[string]Agent{},
		timeout:     5 * time.Second,
		fallback:    "skip",
		parallelMax: 3,
	}
	out := dt.execute(context.Background(), DelegateToolInput{AgentName: "ghost", Task: "x"}, "test-session", 1)
	if out.Status != "error" {
		t.Errorf("status = %q, want error", out.Status)
	}
}

func TestDelegateTool_ExecuteDelegations_Parallel(t *testing.T) {
	a := newMockAgent("a", "result-a")
	b := newMockAgent("b", "result-b")
	dt := &delegateTool{
		subAgents:   map[string]Agent{"a": a, "b": b},
		timeout:     5 * time.Second,
		fallback:    "skip",
		parallelMax: 2,
	}

	delegCtx := withDelegationScope(context.Background(), "test-session", 1)
	results := dt.executeDelegations(delegCtx, []DelegateToolInput{
		{AgentName: "a", Task: "task a"},
		{AgentName: "b", Task: "task b"},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.output.Status != "success" {
			t.Errorf("agent %q status = %q, want success", r.output.AgentName, r.output.Status)
		}
	}
}

func TestAggregateResults_Concat(t *testing.T) {
	results := []delegationResult{
		{output: DelegateToolOutput{AgentName: "a", Result: "res-a", Status: "success"}},
		{output: DelegateToolOutput{AgentName: "b", Result: "res-b", Status: "success"}},
	}
	out := aggregateResults(context.Background(), results, "concat", nil)
	if !strings.Contains(out, "res-a") || !strings.Contains(out, "res-b") {
		t.Errorf("concat output missing results: %q", out)
	}
}

func TestAggregateResults_Structured(t *testing.T) {
	results := []delegationResult{
		{output: DelegateToolOutput{AgentName: "a", Result: "r", Status: "success"}},
	}
	out := aggregateResults(context.Background(), results, "structured", nil)
	if !strings.Contains(out, `"agent_name"`) {
		t.Errorf("structured output should contain JSON: %q", out)
	}
}

func TestAggregateResults_Summarize_WithFunc(t *testing.T) {
	results := []delegationResult{
		{output: DelegateToolOutput{AgentName: "a", Result: "detailed result from agent a that is quite long"}},
		{output: DelegateToolOutput{AgentName: "b", Result: "detailed result from agent b that is also long"}},
	}
	mockSummarize := func(_ context.Context, inputs []string) (string, error) {
		return fmt.Sprintf("summary of %d results", len(inputs)), nil
	}
	out := aggregateResults(context.Background(), results, "summarize", mockSummarize)
	if out != "summary of 2 results" {
		t.Errorf("summarize output = %q, want %q", out, "summary of 2 results")
	}
}

func TestAggregateResults_Summarize_FallbackOnError(t *testing.T) {
	results := []delegationResult{
		{output: DelegateToolOutput{AgentName: "a", Result: "res-a"}},
	}
	failingSummarize := func(_ context.Context, _ []string) (string, error) {
		return "", fmt.Errorf("model error")
	}
	out := aggregateResults(context.Background(), results, "summarize", failingSummarize)
	if !strings.Contains(out, "res-a") {
		t.Errorf("should fall back to concat on error, got: %q", out)
	}
}

func TestAggregateResults_Summarize_NilFunc(t *testing.T) {
	results := []delegationResult{
		{output: DelegateToolOutput{AgentName: "a", Result: "res-a"}},
	}
	out := aggregateResults(context.Background(), results, "summarize", nil)
	if !strings.Contains(out, "res-a") {
		t.Errorf("should fall back to concat with nil func, got: %q", out)
	}
}

func TestParseDelegationDecision_ValidJSON(t *testing.T) {
	input := `{"delegations":[{"agent_name":"x","task":"do X"}]}`
	delegations, isFinal := parseDelegationDecision(input)
	if isFinal {
		t.Error("expected isFinal=false for valid delegation JSON")
	}
	if len(delegations) != 1 || delegations[0].AgentName != "x" {
		t.Errorf("unexpected delegations: %+v", delegations)
	}
}

func TestParseDelegationDecision_PlainText(t *testing.T) {
	_, isFinal := parseDelegationDecision("Here is my final answer.")
	if !isFinal {
		t.Error("expected isFinal=true for plain text")
	}
}

func TestParseDelegationDecision_EmptyDelegations(t *testing.T) {
	_, isFinal := parseDelegationDecision(`{"delegations":[]}`)
	if !isFinal {
		t.Error("expected isFinal=true for empty delegations list")
	}
}
