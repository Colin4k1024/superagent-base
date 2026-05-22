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
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

// Compile-time interface assertions.
var (
	_ Agent         = (*ADKRunnerAgent)(nil)
	_ Interruptable = (*ADKRunnerAgent)(nil)
)

// ADKRunnerAgent wraps an ADK ChatModelAgent with adk.Runner for native
// interrupt/resume and checkpoint support. It implements both the Agent
// and Interruptable interfaces.
type ADKRunnerAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string
	memBackend   memory.Backend
	agent        *adk.ChatModelAgent
	runner       *adk.Runner
	store        adk.CheckPointStore
	systemPrompt string
}

// NewADKRunnerAgent creates an agent with ADK Runner-based interrupt/resume.
func NewADKRunnerAgent(
	ctx context.Context,
	agent *adk.ChatModelAgent,
	store adk.CheckPointStore,
	def *AgentDefinition,
	modelID, provider, systemPrompt string,
	memBackend memory.Backend,
) *ADKRunnerAgent {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming:  true,
		CheckPointStore: store,
	})
	return &ADKRunnerAgent{
		def:          def,
		modelID:      modelID,
		provider:     provider,
		memBackend:   memBackend,
		agent:        agent,
		runner:       runner,
		store:        store,
		systemPrompt: systemPrompt,
	}
}

func (a *ADKRunnerAgent) Name() string                    { return a.def.Metadata.Name }
func (a *ADKRunnerAgent) Description() string             { return a.systemPrompt }
func (a *ADKRunnerAgent) GetDefinition() *AgentDefinition { return a.def }

// Chat executes the agent with Runner.Run(). If an interrupt occurs, it emits
// a JSON-encoded interrupt event on the channel using the interruptPrefix sentinel.
func (a *ADKRunnerAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	// Build history BEFORE persisting so the current message isn't loaded twice.
	msgs := buildMessageHistory(ctx, a.systemPrompt, sessionID, a.memBackend)
	msgs = append(msgs, schema.UserMessage(message))
	persistUserMessage(ctx, sessionID, message, a.memBackend)

	opts := []adk.AgentRunOption{adk.WithCheckPointID(sessionID)}
	iter := a.runner.Run(ctx, msgs, opts...)

	ch := make(chan string, 64)
	params := streamConsumerParams{
		sessionID:  sessionID,
		modelID:    a.modelID,
		provider:   a.provider,
		memBackend: a.memBackend,
	}
	go consumeADKIterator(ctx, params, iter, ch, a.handleInterrupt(sessionID))
	return ch, nil
}

// Resume continues an interrupted execution using ADK's native checkpoint resume.
func (a *ADKRunnerAgent) Resume(ctx context.Context, sessionID string, input map[string]any) (<-chan string, error) {
	var iter *adk.AsyncIterator[*adk.AgentEvent]
	var err error

	if len(input) > 0 {
		params := &adk.ResumeParams{Targets: input}
		iter, err = a.runner.ResumeWithParams(ctx, sessionID, params)
	} else {
		iter, err = a.runner.Resume(ctx, sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("agentdef: ADKRunnerAgent.Resume: %w", err)
	}

	ch := make(chan string, 64)
	p := streamConsumerParams{
		sessionID:  sessionID,
		modelID:    a.modelID,
		provider:   a.provider,
		memBackend: a.memBackend,
	}
	go consumeADKIterator(ctx, p, iter, ch, a.handleInterrupt(sessionID))
	return ch, nil
}

// GetInterruptState checks if there is a pending interrupt for this session
// by querying the checkpoint store.
func (a *ADKRunnerAgent) GetInterruptState(ctx context.Context, sessionID string) (*InterruptState, bool) {
	if a.store == nil || sessionID == "" {
		return nil, false
	}
	_, exists, err := a.store.Get(ctx, sessionID)
	if err != nil || !exists {
		return nil, false
	}
	return &InterruptState{
		SessionID: sessionID,
		AgentName: a.def.Metadata.Name,
		Reason:    "Agent has a pending checkpoint.",
		Fields:    []InputField{{Name: "confirm", Type: "confirm", Label: "Confirm action", Required: true}},
		Timestamp: time.Now().Unix(),
	}, true
}

// handleInterrupt returns an interruptHandler closure for the given session.
func (a *ADKRunnerAgent) handleInterrupt(sessionID string) interruptHandler {
	return func(ctx context.Context, event *adk.AgentEvent, ch chan<- string) bool {
		interruptData := &InterruptState{
			SessionID: sessionID,
			AgentName: a.def.Metadata.Name,
			Reason:    "Agent requested confirmation before proceeding.",
			Fields:    []InputField{{Name: "confirm", Type: "confirm", Label: "Confirm action", Required: true}},
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(interruptData)
		select {
		case ch <- interruptPrefix + string(data):
		case <-ctx.Done():
		}
		return true
	}
}
