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

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// MCPToolAdapter bridges an MCP ToolDefinition to Eino's InvokableTool
// interface so that MCP-backed tools can be used directly in Eino agent graphs.
type MCPToolAdapter struct {
	client  *Client
	toolDef ToolDefinition
}

// NewMCPToolAdapter wraps a single MCP ToolDefinition.
func NewMCPToolAdapter(client *Client, toolDef ToolDefinition) *MCPToolAdapter {
	return &MCPToolAdapter{client: client, toolDef: toolDef}
}

// compile-time interface check.
var _ einotool.InvokableTool = (*MCPToolAdapter)(nil)

// Info returns the Eino ToolInfo derived from the MCP ToolDefinition.
func (a *MCPToolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	js := mcpSchemaToJSONSchema(a.toolDef.InputSchema)
	return &schema.ToolInfo{
		Name: a.toolDef.Name,
		Desc: a.toolDef.Description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(js),
	}, nil
}

// InvokableRun calls the MCP tool with the JSON-encoded arguments and returns
// the concatenated text from the result content blocks.
func (a *MCPToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
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
// InvokableTool slice ready for use in an Eino ToolsNode.
func AdaptAllTools(ctx context.Context, client *Client) ([]einotool.InvokableTool, error) {
	defs, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcp adapt tools: %w", err)
	}

	tools := make([]einotool.InvokableTool, len(defs))
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

// mcpSchemaToJSONSchema converts a minimal MCP JSONSchema to the eino-contrib
// jsonschema.Schema used by schema.NewParamsOneOfByJSONSchema.
func mcpSchemaToJSONSchema(s *JSONSchema) *jsonschema.Schema {
	if s == nil {
		return &jsonschema.Schema{Type: "object"}
	}

	js := &jsonschema.Schema{
		Type:        s.Type,
		Description: s.Description,
		Required:    s.Required,
	}

	if len(s.Properties) > 0 {
		js.Properties = orderedmap.New[string, *jsonschema.Schema]()
		for k, v := range s.Properties {
			js.Properties.Set(k, mcpSchemaToJSONSchema(v))
		}
	}

	if s.Items != nil {
		js.Items = mcpSchemaToJSONSchema(s.Items)
	}

	if len(s.Enum) > 0 {
		js.Enum = s.Enum
	}

	return js
}
