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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExposeAgentAsTool creates a ToolDefinition and ToolHandler that wrap a chat
// function so that an agent can be called by any MCP client.
//
// The generated tool accepts a single "message" string parameter and returns
// the agent's reply as a text ContentBlock.
//
//	def, handler := ExposeAgentAsTool("my-agent", "Runs my agent", chatFn)
//	server.RegisterTool(def, handler)
func ExposeAgentAsTool(
	agentName string,
	description string,
	chatFn func(ctx context.Context, msg string) (string, error),
) (ToolDefinition, ToolHandler) {
	def := ToolDefinition{
		Name:        agentName,
		Description: description,
		InputSchema: &JSONSchema{
			Type: "object",
			Properties: map[string]*JSONSchema{
				"message": {
					Type:        "string",
					Description: "The message to send to the agent.",
				},
			},
			Required: []string{"message"},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (*ToolCallResult, error) {
		msg, err := stringArg(args, "message")
		if err != nil {
			return nil, fmt.Errorf("agent tool %q: %w", agentName, err)
		}

		reply, err := chatFn(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("agent tool %q: %w", agentName, err)
		}

		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: reply}},
		}, nil
	}

	return def, handler
}

// ExposeWorkflowAsTool creates a ToolDefinition and ToolHandler that wrap a
// workflow execution function so that an arbitrary workflow can be triggered by
// any MCP client.
//
// The generated tool accepts a single "input" object parameter whose value is
// forwarded as-is to execFn. The output map is JSON-encoded and returned as a
// text ContentBlock.
//
//	def, handler := ExposeWorkflowAsTool("my-workflow", "Runs my workflow", execFn)
//	server.RegisterTool(def, handler)
func ExposeWorkflowAsTool(
	wfName string,
	description string,
	execFn func(ctx context.Context, input map[string]any) (map[string]any, error),
) (ToolDefinition, ToolHandler) {
	def := ToolDefinition{
		Name:        wfName,
		Description: description,
		InputSchema: &JSONSchema{
			Type: "object",
			Properties: map[string]*JSONSchema{
				"input": {
					Type:        "object",
					Description: "Key-value pairs forwarded to the workflow as input.",
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (*ToolCallResult, error) {
		// "input" is optional; default to an empty map.
		var input map[string]any
		if raw, ok := args["input"]; ok {
			switch v := raw.(type) {
			case map[string]any:
				input = v
			default:
				return nil, fmt.Errorf("workflow tool %q: 'input' must be an object", wfName)
			}
		}

		output, err := execFn(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("workflow tool %q: %w", wfName, err)
		}

		text, err := json.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("workflow tool %q: marshal output: %w", wfName, err)
		}

		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: string(text)}},
		}, nil
	}

	return def, handler
}

// stringArg extracts a required string argument from args.
func stringArg(args map[string]any, key string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string, got %T", key, raw)
	}
	return s, nil
}
