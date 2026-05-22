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
	"testing"
	"time"
)

// stubAgent is a minimal Agent implementation for testing unwrap logic.
type stubAgent struct {
	name string
}

func (s *stubAgent) Name() string                                                             { return s.name }
func (s *stubAgent) Description() string                                                      { return "" }
func (s *stubAgent) GetDefinition() *AgentDefinition                                          { return nil }
func (s *stubAgent) Chat(_ context.Context, _ string, _ string) (<-chan string, error) { return nil, nil }

func TestUnwrapToADKChatModel_Direct(t *testing.T) {
	inner := &adkChatModelAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: "test"}},
		modelID: "gpt-4",
	}
	got := unwrapToADKChatModel(inner)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if got.modelID != "gpt-4" {
		t.Errorf("expected modelID=gpt-4, got %s", got.modelID)
	}
}

func TestUnwrapToADKChatModel_ThroughMiddleware(t *testing.T) {
	inner := &adkChatModelAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: "test"}},
		modelID: "claude-3",
	}
	wrapped := Agent(&observedAgent{
		inner: &timeoutAgent{
			inner:   &retryAgent{inner: inner, maxAttempts: 3},
			timeout: 10 * time.Second,
		},
		enableMetrics: true,
	})

	got := unwrapToADKChatModel(wrapped)
	if got == nil {
		t.Fatal("expected non-nil after unwrapping 3 layers, got nil")
	}
	if got.modelID != "claude-3" {
		t.Errorf("expected modelID=claude-3, got %s", got.modelID)
	}
}

func TestUnwrapToADKChatModel_NonADKAgent(t *testing.T) {
	agent := &stubAgent{name: "plain"}
	got := unwrapToADKChatModel(agent)
	if got != nil {
		t.Errorf("expected nil for non-ADK agent, got %v", got)
	}
}

func TestUnwrapToADKChatModel_FallbackAgent(t *testing.T) {
	inner := &adkChatModelAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: "primary"}},
		modelID: "deepseek",
	}
	fb := &fallbackAgent{
		primary:  inner,
		fallback: &stubAgent{name: "backup"},
	}
	wrapped := Agent(&rateLimitAgent{inner: fb, rpm: 100})

	got := unwrapToADKChatModel(wrapped)
	if got == nil {
		t.Fatal("expected non-nil through fallbackAgent, got nil")
	}
	if got.modelID != "deepseek" {
		t.Errorf("expected modelID=deepseek, got %s", got.modelID)
	}
}

func TestUnwrapToADKRunner_Direct(t *testing.T) {
	inner := &ADKRunnerAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: "runner"}},
		modelID: "gpt-4o",
	}
	got := unwrapToADKRunner(inner)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if got.modelID != "gpt-4o" {
		t.Errorf("expected modelID=gpt-4o, got %s", got.modelID)
	}
}

func TestUnwrapToADKRunner_ThroughCache(t *testing.T) {
	inner := &ADKRunnerAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: "cached-runner"}},
		modelID: "qwen-72b",
	}
	wrapped := Agent(&cacheAgent{
		inner: inner,
		ttl:   5 * time.Minute,
		cache: make(map[string]cacheEntry),
	})

	got := unwrapToADKRunner(wrapped)
	if got == nil {
		t.Fatal("expected non-nil through cacheAgent, got nil")
	}
	if got.modelID != "qwen-72b" {
		t.Errorf("expected modelID=qwen-72b, got %s", got.modelID)
	}
}

func TestUnwrapToEinoAgent_NilForNonADK(t *testing.T) {
	agent := &stubAgent{name: "basic"}
	got := unwrapToEinoAgent(agent)
	if got != nil {
		t.Errorf("expected nil for non-ADK stub, got %v", got)
	}
}

func TestAgentUnwrapper_InterfaceCompliance(t *testing.T) {
	cases := []struct {
		name  string
		agent AgentUnwrapper
	}{
		{"observed", &observedAgent{inner: &stubAgent{}}},
		{"timeout", &timeoutAgent{inner: &stubAgent{}}},
		{"retry", &retryAgent{inner: &stubAgent{}}},
		{"fallback", &fallbackAgent{primary: &stubAgent{}, fallback: &stubAgent{}}},
		{"rateLimit", &rateLimitAgent{inner: &stubAgent{}}},
		{"cache", &cacheAgent{inner: &stubAgent{}, cache: make(map[string]cacheEntry)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unwrapped := tc.agent.UnwrapAgent()
			if unwrapped == nil {
				t.Error("UnwrapAgent() returned nil")
			}
		})
	}
}
