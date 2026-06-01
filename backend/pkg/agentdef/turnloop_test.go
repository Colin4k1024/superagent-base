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
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestTurnLoopManager_NonADKAgentFallsBack(t *testing.T) {
	mgr := NewTurnLoopManager()

	ch, handled, err := mgr.Chat(context.Background(), "plain", &stubAgent{name: "plain"}, "s1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Fatal("expected non-ADK agent to fall back to Agent.Chat")
	}
	if ch != nil {
		t.Fatalf("expected nil channel, got %v", ch)
	}
}

func TestTurnLoopManager_AbortMissingSession(t *testing.T) {
	mgr := NewTurnLoopManager()
	if mgr.Abort("agent", "missing") {
		t.Fatal("expected abort on missing session to return false")
	}
}

func TestTurnLoopManager_StreamsAndCleansIdleSession(t *testing.T) {
	agent := newTurnLoopTestAgent(t, "test-agent", newTurnLoopScriptedModel("hello"))
	mgr := newTurnLoopManager(10 * time.Millisecond)

	ch, handled, err := mgr.Chat(context.Background(), "test-agent", agent, "s1", "hello?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected TurnLoop to handle ADK chat model agent")
	}

	if got := readTurnLoopTokens(t, ch); got != "hello" {
		t.Fatalf("unexpected stream output: %q", got)
	}

	waitUntil(t, time.Second, func() bool {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()
		return len(mgr.sessions) == 0
	})
}

func TestTurnLoopManager_AbortActiveLoopAndRecreate(t *testing.T) {
	model := newTurnLoopScriptedModel("first", "second")
	model.blockFirstStream = true
	agent := newTurnLoopTestAgent(t, "test-agent", model)
	mgr := newTurnLoopManager(10 * time.Millisecond)

	ch, handled, err := mgr.Chat(context.Background(), "test-agent", agent, "s1", "first?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected TurnLoop to handle ADK chat model agent")
	}
	waitForSignal(t, model.firstStreamEntered, "model stream")

	if !mgr.Abort("test-agent", "s1") {
		t.Fatal("expected abort to stop active session")
	}
	if got := readTurnLoopTokens(t, ch); got != "" {
		t.Fatalf("expected aborted stream to close without tokens, got %q", got)
	}

	close(model.releaseFirstStream)
	ch, handled, err = mgr.Chat(context.Background(), "test-agent", agent, "s1", "second?")
	if err != nil {
		t.Fatalf("unexpected recreate error: %v", err)
	}
	if !handled {
		t.Fatal("expected recreated TurnLoop to handle ADK chat model agent")
	}
	if got := readTurnLoopTokens(t, ch); got != "second" {
		t.Fatalf("unexpected recreated stream output: %q", got)
	}
}

type turnLoopScriptedModel struct {
	mu                 sync.Mutex
	responses          []string
	blockFirstStream   bool
	firstStreamEntered chan struct{}
	releaseFirstStream chan struct{}
	streamCalls        int
}

func newTurnLoopScriptedModel(responses ...string) *turnLoopScriptedModel {
	return &turnLoopScriptedModel{
		responses:          responses,
		firstStreamEntered: make(chan struct{}),
		releaseFirstStream: make(chan struct{}),
	}
}

func (m *turnLoopScriptedModel) Generate(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.Message, error) {
	return schema.AssistantMessage(m.nextResponse(), nil), nil
}

func (m *turnLoopScriptedModel) Stream(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	call := m.nextStreamCall()
	if call == 1 && m.blockFirstStream {
		close(m.firstStreamEntered)
		select {
		case <-m.releaseFirstStream:
		case <-ctx.Done():
			return schema.StreamReaderFromArray([]*schema.Message{}), nil
		}
	}

	return schema.StreamReaderFromArray([]*schema.Message{
		schema.AssistantMessage(m.nextResponse(), nil),
	}), nil
}

func (m *turnLoopScriptedModel) nextStreamCall() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamCalls++
	return m.streamCalls
}

func (m *turnLoopScriptedModel) nextResponse() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.responses) == 0 {
		return ""
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]
	return resp
}

func newTurnLoopTestAgent(t *testing.T, name string, model *turnLoopScriptedModel) *adkChatModelAgent {
	t.Helper()

	adkAgent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        name,
		Description: "test agent",
		Model:       model,
	})
	if err != nil {
		t.Fatalf("NewChatModelAgent() error: %v", err)
	}

	return &adkChatModelAgent{
		def:     &AgentDefinition{Metadata: Metadata{Name: name}},
		modelID: "test-model",
		agent:   adkAgent,
	}
}

func readTurnLoopTokens(t *testing.T, ch <-chan string) string {
	t.Helper()

	var out strings.Builder
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case token, ok := <-ch:
			if !ok {
				return out.String()
			}
			if token == "[error] internal error occurred" {
				t.Fatalf("unexpected internal error token")
			}
			out.WriteString(token)
		case <-timer.C:
			t.Fatal("timed out waiting for stream to close")
		}
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, pred func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

var _ einoModel.BaseChatModel = (*turnLoopScriptedModel)(nil)
