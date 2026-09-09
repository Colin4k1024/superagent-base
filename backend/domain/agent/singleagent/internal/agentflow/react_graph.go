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

// Package agentflow react_graph.go — self-built ReAct agent graph.
//
// This file replaces the dependency on github.com/cloudwego/eino/flow/agent/react
// by constructing the same compose graph topology (model → branch → tools →
// model loop) using eino's compose primitives directly. The callback system,
// streaming, and interrupt/resume all continue to work because the compose
// graph drives execution.
package agentflow

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// reactState is the per-execution state for the self-built ReAct graph.
// It mirrors the internal state struct from the react package.
type reactState struct {
	Messages                 []*schema.Message
	ReturnDirectlyToolCallID string
}

func init() {
	schema.RegisterName[*reactState]("_self_react_state")
}

// reactGraphResult holds the compiled graph and node options, matching
// the return type of react.Agent.ExportGraph().
type reactGraphResult struct {
	graph    einobridge.AnyGraph
	nodeOpts []einobridge.GraphAddNodeOpt
}

// buildReActGraph constructs a ReAct agent graph using eino compose primitives
// directly, without importing the react package. The topology is:
//
//	START → chatModel → branch(hasToolCalls?) → tools → branch(shouldReturnDirectly?) → chatModel (loop)
//	                                            ↘ END                         ↘ directReturn → END
//
// This faithfully replicates react.NewAgent's graph construction.
func buildReActGraph(ctx context.Context, chatModel model.ToolCallingChatModel,
	agentTools []tool.BaseTool, returnDirectlyTools map[string]struct{},
	modelNodeName, toolsNodeName string) (*reactGraphResult, error) {

	// Generate tool infos.
	toolInfos := make([]*schema.ToolInfo, 0, len(agentTools))
	for _, t := range agentTools {
		tl, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		toolInfos = append(toolInfos, tl)
	}

	// Bind tools to the chat model (safe, immutable).
	var chatModelNode model.BaseChatModel
	if len(toolInfos) > 0 {
		var err error
		chatModelNode, err = chatModel.WithTools(toolInfos)
		if err != nil {
			return nil, err
		}
	} else {
		chatModelNode = chatModel
	}

	// Create the tools node.
	toolsNode, err := einobridge.NewToolNode(ctx, &einobridge.ToolsNodeConfig{
		Tools: agentTools,
	})
	if err != nil {
		return nil, err
	}

	const (
		nodeKeyModel        = "chat"
		nodeKeyTools        = "tools"
		nodeKeyDirectReturn = "direct_return"
	)

	graph := einobridge.NewGraph[[]*schema.Message, *schema.Message](
		einobridge.WithGenLocalState(func(ctx context.Context) *reactState {
			return &reactState{Messages: make([]*schema.Message, 0)}
		}))

	// Model node — accumulates messages in state before calling the LLM.
	modelPreHandle := func(ctx context.Context, input []*schema.Message, state *reactState) ([]*schema.Message, error) {
		state.Messages = append(state.Messages, input...)
		return state.Messages, nil
	}

	if err = graph.AddChatModelNode(nodeKeyModel, chatModelNode,
		einobridge.WithStatePreHandler(modelPreHandle),
		einobridge.WithNodeName(modelNodeName)); err != nil {
		return nil, err
	}

	if err = graph.AddEdge(einobridge.START, nodeKeyModel); err != nil {
		return nil, err
	}

	// Tools node — executes tool calls and records return-directly.
	toolsPreHandle := func(ctx context.Context, input *schema.Message, state *reactState) (*schema.Message, error) {
		if input == nil {
			// Used for rerun interrupt resume — use last message.
			return state.Messages[len(state.Messages)-1], nil
		}
		state.Messages = append(state.Messages, input)
		state.ReturnDirectlyToolCallID = getReturnDirectlyToolCallID(input, returnDirectlyTools)
		return input, nil
	}

	if err = graph.AddToolsNode(nodeKeyTools, toolsNode,
		einobridge.WithStatePreHandler(toolsPreHandle),
		einobridge.WithNodeName(toolsNodeName)); err != nil {
		return nil, err
	}

	// Branch after model: if tool calls → tools node, else → END.
	modelBranchCondition := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (string, error) {
		isToolCall, err := firstChunkStreamToolCallChecker(ctx, sr)
		if err != nil {
			return "", err
		}
		if isToolCall {
			return nodeKeyTools, nil
		}
		return einobridge.END, nil
	}

	if err = graph.AddBranch(nodeKeyModel,
		einobridge.NewStreamGraphBranch(modelBranchCondition,
			map[string]bool{nodeKeyTools: true, einobridge.END: true})); err != nil {
		return nil, err
	}

	// Return-directly handling: a lambda node that extracts the directly
	// returned tool result, plus a branch after the tools node.
	directReturn := func(ctx context.Context, msgs *schema.StreamReader[[]*schema.Message]) (*schema.StreamReader[*schema.Message], error) {
		return schema.StreamReaderWithConvert(msgs, func(msgs []*schema.Message) (*schema.Message, error) {
			var msg *schema.Message
			err = einobridge.ProcessState[*reactState](ctx, func(_ context.Context, state *reactState) error {
				for i := range msgs {
					if msgs[i] != nil && msgs[i].ToolCallID == state.ReturnDirectlyToolCallID {
						msg = msgs[i]
						return nil
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			if msg == nil {
				return nil, schema.ErrNoValue
			}
			return msg, nil
		}), nil
	}

	if err = graph.AddLambdaNode(nodeKeyDirectReturn,
		einobridge.TransformableLambda(directReturn)); err != nil {
		return nil, err
	}

	// Branch after tools: if return-directly → directReturn, else → model (loop).
	if err = graph.AddBranch(nodeKeyTools,
		einobridge.NewStreamGraphBranch(func(ctx context.Context, msgsStream *schema.StreamReader[[]*schema.Message]) (string, error) {
			msgsStream.Close()
			var endNode string
			err = einobridge.ProcessState[*reactState](ctx, func(_ context.Context, state *reactState) error {
				if len(state.ReturnDirectlyToolCallID) > 0 {
					endNode = nodeKeyDirectReturn
				} else {
					endNode = nodeKeyModel
				}
				return nil
			})
			if err != nil {
				return "", err
			}
			return endNode, nil
		}, map[string]bool{nodeKeyModel: true, nodeKeyDirectReturn: true})); err != nil {
		return nil, err
	}

	if err = graph.AddEdge(nodeKeyDirectReturn, einobridge.END); err != nil {
		return nil, err
	}

	compileOpts := []einobridge.GraphCompileOption{
		einobridge.WithMaxRunSteps(0), // 0 = use default max steps
		einobridge.WithNodeTriggerMode(einobridge.AnyPredecessor),
		einobridge.WithGraphName("SelfReActAgent"),
	}

	return &reactGraphResult{
		graph:    graph,
		nodeOpts: []einobridge.GraphAddNodeOpt{einobridge.WithGraphCompileOptions(compileOpts...)},
	}, nil
}

// firstChunkStreamToolCallChecker checks whether the model's streaming output
// contains tool calls. It reads chunks until it finds tool calls or non-empty
// content, then returns the result. This mirrors react.firstChunkStreamToolCallChecker.
func firstChunkStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if len(msg.ToolCalls) > 0 {
			return true, nil
		}

		if len(msg.Content) > 0 {
			return false, nil
		}
		// Skip empty chunks at the front.
	}
}

// getReturnDirectlyToolCallID checks if any tool call in the message
// matches a tool in the returnDirectlyTools set. Returns the first
// matching tool call ID, or empty string if none match.
func getReturnDirectlyToolCallID(msg *schema.Message, returnDirectlyTools map[string]struct{}) string {
	if len(returnDirectlyTools) == 0 || msg == nil {
		return ""
	}
	for _, tc := range msg.ToolCalls {
		if _, ok := returnDirectlyTools[tc.Function.Name]; ok {
			return tc.ID
		}
	}
	return ""
}
