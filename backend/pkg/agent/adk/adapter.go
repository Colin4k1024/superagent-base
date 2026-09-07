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

package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	adkmodel "google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	aclllm "github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// AgentAdapter wraps a Google ADK Go Runner as an aclagent.AgentRuntime.
type AgentAdapter struct {
	name        string
	description string
	runner      *runner.Runner
	llmModel    adkmodel.LLM
	maxIter     int
}

// NewAgentAdapter creates an aclagent.AgentRuntime from a Google ADK Go model.
func NewAgentAdapter(name, description string, llmModel adkmodel.LLM, tools []aclllm.Tool, maxIter int) (*AgentAdapter, error) {
	funcDecls := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			continue
		}
		funcDecls = append(funcDecls, &genai.FunctionDeclaration{
			Name:        info.Name,
			Description: info.Desc,
		})
	}

	cfg := llmagent.Config{
		Name:        name,
		Description: description,
		Model:       llmModel,
	}
	if len(funcDecls) > 0 {
		cfg.GenerateContentConfig = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{FunctionDeclarations: funcDecls}},
		}
	}

	adkAgent, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("adk agent adapter: create llmagent: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "superagent",
		Agent:             adkAgent,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("adk agent adapter: create runner: %w", err)
	}

	return &AgentAdapter{
		name:        name,
		description: description,
		runner:      r,
		llmModel:    llmModel,
		maxIter:     maxIter,
	}, nil
}

func (a *AgentAdapter) Name() string        { return a.name }
func (a *AgentAdapter) Description() string  { return a.description }

// run executes the runner and returns an EventIterator backed by iter.Pull2.
func (a *AgentAdapter) run(ctx context.Context, userMessage string) (*aclagent.EventIterator, error) {
	userContent := genai.NewContentFromText(userMessage, genai.RoleUser)
	seq := a.runner.Run(ctx, "default", "session-1", userContent, agent.RunConfig{})

	pull, stop := iter.Pull2(seq)

	return aclagent.NewEventIterator(
		func() (*aclagent.AgentEvent, bool) {
			for {
				event, err, ok := pull()
				if !ok {
					return nil, false
				}
				if err != nil {
					return &aclagent.AgentEvent{Err: err}, true
				}
				ae := fromADKEvent(event)
				if ae != nil {
					return ae, true
				}
				// Skip nil events and continue pulling.
			}
		},
		func() error { stop(); return nil },
	), nil
}

// Run executes the agent with the given input.
func (a *AgentAdapter) Run(ctx context.Context, input *aclagent.AgentInput) (*aclagent.EventIterator, error) {
	var lastUserMsg string
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if input.Messages[i].Role == aclllm.RoleUser {
			lastUserMsg = input.Messages[i].Content
			break
		}
	}
	if lastUserMsg == "" {
		return nil, fmt.Errorf("adk agent adapter: no user message in input")
	}
	return a.run(ctx, lastUserMsg)
}

// Resume continues a previously interrupted agent run.
func (a *AgentAdapter) Resume(ctx context.Context, input *aclagent.ResumeInput) (*aclagent.EventIterator, error) {
	return a.run(ctx, input.UserMessage)
}

// fromADKEvent converts a Google ADK Go session.Event to agent.AgentEvent.
func fromADKEvent(event *session.Event) *aclagent.AgentEvent {
	if event == nil {
		return nil
	}
	result := &aclagent.AgentEvent{}

	if event.Content != nil {
		msg := fromGenaiContent(event.Content)
		if msg != nil {
			result.MessageOutput = &aclagent.MessageOutput{
				Message:    msg,
				IsStreaming: event.Partial,
			}
		}

		for _, part := range event.Content.Parts {
			if part != nil && part.FunctionCall != nil {
				if result.Action == nil {
					result.Action = &aclagent.EventAction{}
				}
				argsBytes, _ := json.Marshal(part.FunctionCall.Args)
				result.Action.ToolCalls = append(result.Action.ToolCalls, aclllm.ToolCall{
					ID:       part.FunctionCall.ID,
					Name:     part.FunctionCall.Name,
					ArgsJSON: string(argsBytes),
				})
			}
		}
	}

	return result
}

// fromGenaiContent converts a genai.Content to an ACL Message.
func fromGenaiContent(c *genai.Content) *aclllm.Message {
	if c == nil {
		return nil
	}
	role := aclllm.RoleAssistant
	if c.Role == string(genai.RoleUser) {
		role = aclllm.RoleUser
	}
	msg := &aclllm.Message{Role: role}
	for _, part := range c.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			msg.Content += part.Text
		}
		if part.FunctionCall != nil {
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			msg.ToolCalls = append(msg.ToolCalls, aclllm.ToolCall{
				ID:       part.FunctionCall.ID,
				Name:     part.FunctionCall.Name,
				ArgsJSON: string(argsBytes),
			})
		}
	}
	return msg
}

var _ aclagent.AgentRuntime = (*AgentAdapter)(nil)
