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
	"strings"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// MCPToolAdapter bridges an MCP ToolDefinition to the framework-agnostic
// llm.Tool interface so that MCP-backed tools can be used directly in
// agent tool lists without importing eino.
type MCPToolAdapter struct {
	client  *Client
	toolDef ToolDefinition
}

// NewMCPToolAdapter wraps a single MCP ToolDefinition as an llm.Tool.
func NewMCPToolAdapter(client *Client, toolDef ToolDefinition) *MCPToolAdapter {
	return &MCPToolAdapter{client: client, toolDef: toolDef}
}

// compile-time interface check.
var _ llm.Tool = (*MCPToolAdapter)(nil)

// Info returns the tool metadata derived from the MCP ToolDefinition.
func (a *MCPToolAdapter) Info(_ context.Context) (*llm.ToolInfo, error) {
	return &llm.ToolInfo{
		Name: a.toolDef.Name,
		Desc: a.toolDef.Description,
	}, nil
}

// Run calls the MCP tool with the JSON-encoded arguments and returns
// the concatenated text from the result content blocks.
func (a *MCPToolAdapter) Run(ctx context.Context, argumentsInJSON string, _ ...llm.ToolOption) (string, error) {
	var args map[string]any
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return "", fmt.Errorf("mcp adapter: unmarshal arguments: %w", err)
		}
	}

	result, err := a.client.CallTool(ctx, a.toolDef.Name, args)
	if err != nil {
		return "", err
	}

	if result.IsError {
		return "", fmt.Errorf("mcp tool %q returned an error: %s", a.toolDef.Name, contentText(result.Content))
	}

	return contentText(result.Content), nil
}

// AdaptAllTools fetches the tool list from the client and returns an
// llm.Tool slice ready for use in agent tool lists.
func AdaptAllTools(ctx context.Context, client *Client) ([]llm.Tool, error) {
	defs, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcp adapt tools: %w", err)
	}

	tools := make([]llm.Tool, len(defs))
	for i, def := range defs {
		tools[i] = NewMCPToolAdapter(client, def)
	}
	return tools, nil
}

// contentText concatenates the Text fields of all "text" content blocks.
func contentText(blocks []ContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}
