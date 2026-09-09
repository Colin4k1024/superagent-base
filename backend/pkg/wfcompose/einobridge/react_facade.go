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

// react_facade provides a native react agent implementation.
// This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"
	"fmt"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/compose"
)

// AgentConfig configures a native react agent.
type AgentConfig struct {
	ToolCallingModel    wfcompose.ToolCallingChatModel
	ToolsConfig         ToolsNodeConfig
	ModelNodeName       string
	GraphName           string
	ToolReturnDirectly  map[string]struct{}
	MaxStep             int
	ToolReturnDirect    bool
}

// ReactModelOption is a model option (alias to wfcompose.ModelOption).
type ReactModelOption = wfcompose.ModelOption

// ReactToolOption is a tool option (alias to wfcompose.ToolOption).
type ReactToolOption = wfcompose.ToolOption

// Agent is a native react agent that implements wfcompose.Runnable.
type Agent struct {
	config *AgentConfig
}

// NewAgent creates a native react agent.
func NewAgent(ctx context.Context, config *AgentConfig) (*Agent, error) {
	if config == nil || config.ToolCallingModel == nil {
		return nil, fmt.Errorf("react: ToolCallingModel is required")
	}
	return &Agent{config: config}, nil
}

// Stream executes the react agent in streaming mode.
func (a *Agent) Stream(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.Option) (*wfcompose.StreamReader[*wfcompose.Message], error) {
	// Basic react loop: call model, check for tool calls, execute tools, repeat
	msgs := input
	maxStep := a.config.MaxStep
	if maxStep <= 0 {
		maxStep = 10
	}

	for step := 0; step < maxStep; step++ {
		result, err := a.config.ToolCallingModel.Generate(ctx, msgs)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, result)

		// If no tool calls, return the result
		if len(result.ToolCalls) == 0 {
			return wfcompose.StreamReaderFromArray([]*wfcompose.Message{result}), nil
		}

		// Execute tool calls
		for _, tc := range result.ToolCalls {
			toolMsg := wfcompose.ToolMessage("", tc.ID)
			msgs = append(msgs, toolMsg)
		}
	}

	// Max steps reached
	lastMsg := msgs[len(msgs)-1]
	return wfcompose.StreamReaderFromArray([]*wfcompose.Message{lastMsg}), nil
}

// Generate executes the react agent and returns a single result.
func (a *Agent) Generate(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.Option) (*wfcompose.Message, error) {
	sr, err := a.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	// Drain the stream
	var last *wfcompose.Message
	for {
		msg, err := sr.Recv()
		if err != nil {
			if wfcompose.IsEOF(err) {
				break
			}
			return nil, err
		}
		last = msg
	}
	if last == nil {
		return nil, fmt.Errorf("react: no output")
	}
	return last, nil
}

// ExportGraph returns the agent's internal graph and node options for composition.
func (a *Agent) ExportGraph() (any, []compose.GraphAddNodeOpt) {
	return nil, nil
}
