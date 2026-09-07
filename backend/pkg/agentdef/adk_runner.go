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

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	adkagent "github.com/superagent-ai/superagent-base/backend/pkg/agent/adk"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

// ADKRunnerAgent wraps a Google ADK Go AgentAdapter for native
// interrupt/resume and checkpoint support. It implements both the Agent
// and Interruptable interfaces.
//
// This replaces the previous eino-based ADKRunnerAgent. The Google ADK Go
// runner provides session-based checkpointing via session.Service.
type ADKRunnerAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string
	memBackend   memory.Backend
	agent        *adkagent.AgentAdapter
	systemPrompt string
}

// Compile-time interface assertions.
var (
	_ Agent         = (*ADKRunnerAgent)(nil)
	_ Interruptable = (*ADKRunnerAgent)(nil)
)

// NewADKRunnerAgent creates an agent with Google ADK Go runner for
// interrupt/resume support.
func NewADKRunnerAgent(
	_ context.Context,
	agentAdapter *adkagent.AgentAdapter,
	_ CheckpointStore,
	def *AgentDefinition,
	modelID, provider, systemPrompt string,
	memBackend memory.Backend,
) *ADKRunnerAgent {
	return &ADKRunnerAgent{
		def:          def,
		modelID:      modelID,
		provider:     provider,
		memBackend:   memBackend,
		agent:        agentAdapter,
		systemPrompt: systemPrompt,
	}
}

func (a *ADKRunnerAgent) Name() string                    { return a.def.Metadata.Name }
func (a *ADKRunnerAgent) Description() string             { return a.systemPrompt }
func (a *ADKRunnerAgent) GetDefinition() *AgentDefinition { return a.def }

// Chat executes the agent via Google ADK Go runner.
func (a *ADKRunnerAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	msgs := buildMessageHistory(ctx, a.systemPrompt, sessionID, a.memBackend)
	msgs = append(msgs, llm.UserMessage(message))
	persistUserMessage(ctx, sessionID, message, a.memBackend)

	iter, err := a.agent.Run(ctx, &aclagent.AgentInput{
		Messages:       msgs,
		EnableStreaming: true,
		MaxIterations:  10,
	})
	if err != nil {
		return nil, fmt.Errorf("agentdef: ADKRunnerAgent.Chat: %w", err)
	}

	ch := make(chan string, 64)
	go consumeGoogleADKIterator(ctx, streamConsumerParams{
		sessionID:  sessionID,
		modelID:    a.modelID,
		provider:   a.provider,
		memBackend: a.memBackend,
	}, iter, ch)
	return ch, nil
}

// Resume continues an interrupted execution.
func (a *ADKRunnerAgent) Resume(ctx context.Context, sessionID string, input map[string]any) (<-chan string, error) {
	userMsg := ""
	if v, ok := input["message"]; ok {
		if s, ok := v.(string); ok {
			userMsg = s
		}
	}

	iter, err := a.agent.Resume(ctx, &aclagent.ResumeInput{
		SessionID:   sessionID,
		UserMessage: userMsg,
	})
	if err != nil {
		return nil, fmt.Errorf("agentdef: ADKRunnerAgent.Resume: %w", err)
	}

	ch := make(chan string, 64)
	go consumeGoogleADKIterator(ctx, streamConsumerParams{
		sessionID:  sessionID,
		modelID:    a.modelID,
		provider:   a.provider,
		memBackend: a.memBackend,
	}, iter, ch)
	return ch, nil
}

// GetInterruptState checks if there is a pending interrupt for this session.
func (a *ADKRunnerAgent) GetInterruptState(_ context.Context, _ string) (*InterruptState, bool) {
	// Google ADK Go session-based interrupt detection will be wired
	// in a follow-up. For now, no pending interrupts are reported.
	return nil, false
}
