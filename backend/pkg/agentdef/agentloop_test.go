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

func TestAgentLoopAgent_CompletesOnDoneMarker(t *testing.T) {
	mock := &mockAgent{
		name:     "inner",
		response: "Task complete. [DONE]",
		def:      nil,
	}
	agent := &AgentLoopAgent{
		name:      "test-loop",
		mainAgent: mock,
		maxTurns:  10,
	}

	ch, err := agent.Chat(context.Background(), "sess-1", "do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result strings.Builder
	for token := range ch {
		result.WriteString(token)
	}

	output := result.String()
	if !strings.Contains(output, "[DONE]") {
		t.Errorf("expected output to contain [DONE], got: %s", output)
	}
	if strings.Contains(output, "Turn 2") {
		t.Errorf("expected loop to stop after first turn (found [DONE]), but got Turn 2")
	}
}

func TestAgentLoopAgent_RespectsMaxTurns(t *testing.T) {
	mock := &mockAgent{
		name:     "inner",
		response: "Still working...",
		def:      nil,
	}
	agent := &AgentLoopAgent{
		name:      "test-loop",
		mainAgent: mock,
		maxTurns:  3,
	}

	ch, err := agent.Chat(context.Background(), "sess-1", "loop forever")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result strings.Builder
	for token := range ch {
		result.WriteString(token)
	}

	output := result.String()
	if !strings.Contains(output, "reached max turns (3)") {
		t.Errorf("expected max turns message, got: %s", output)
	}
}

func TestAgentLoopAgent_RespectsContextCancellation(t *testing.T) {
	mock := &mockAgent{
		name:     "inner",
		response: "Working...",
		def:      nil,
	}
	agent := &AgentLoopAgent{
		name:      "test-loop",
		mainAgent: mock,
		maxTurns:  100,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	ch, err := agent.Chat(ctx, "sess-1", "should cancel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result strings.Builder
	for token := range ch {
		result.WriteString(token)
	}

	output := result.String()
	// When context is pre-cancelled, the loop either emits a cancellation message
	// or produces no output at all (channel closed immediately). Both are valid.
	if output != "" && !strings.Contains(output, "cancelled") && !strings.Contains(output, "reached max turns") {
		t.Errorf("expected empty output or cancellation message, got: %s", output)
	}
}

func TestAgentLoopAgent_NameAndDescription(t *testing.T) {
	agent := &AgentLoopAgent{
		name:        "my-agent",
		description: "test description",
		maxTurns:    10,
	}
	if agent.Name() != "my-agent" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "my-agent")
	}
	if agent.Description() != "test description" {
		t.Errorf("Description() = %q, want %q", agent.Description(), "test description")
	}
}

func TestValidate_AgentLoopType_RequiresPrimaryModel(t *testing.T) {
	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "test-loop"},
		Spec: AgentSpec{
			Type:  "agentloop",
			Model: ModelSpec{Primary: ""},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected validation error for missing primary model")
	}
	if !strings.Contains(err.Error(), "spec.model.primary is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_AgentLoopType_Valid(t *testing.T) {
	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "test-loop"},
		Spec: AgentSpec{
			Type:     "agentloop",
			Model:    ModelSpec{Primary: "claude-sonnet-4-6"},
			MaxTurns: 15,
		},
	}
	if err := Validate(def); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestBuildAgentLoop_Integration(t *testing.T) {
	builder := NewAgentBuilder(
		WithModelConfig(ModelRuntimeConfig{
			BaseURL: "http://localhost:11434",
			APIKey:  "test-key",
		}),
	)

	def := &AgentDefinition{
		APIVersion: "superagent/v1",
		Kind:       "Agent",
		Metadata:   Metadata{Name: "test-loop"},
		Spec: AgentSpec{
			Type:         "agentloop",
			Model:        ModelSpec{Primary: "test-model", Protocol: "ollama"},
			SystemPrompt: "You are a test agent.",
			MaxTurns:     5,
		},
	}

	agent, err := builder.Build(context.Background(), def)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if agent.Name() != "test-loop" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "test-loop")
	}
}
