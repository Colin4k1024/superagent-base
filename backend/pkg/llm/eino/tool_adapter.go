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

package eino

import (
	"context"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// ToolAdapter wraps an llm.Tool as an eino InvokableTool, so that
// framework-agnostic tools can be passed into eino-based agents during
// the migration transition period.
type ToolAdapter struct {
	tool llm.Tool
}

// NewToolAdapter creates an eino InvokableTool from an llm.Tool.
func NewToolAdapter(t llm.Tool) *ToolAdapter {
	return &ToolAdapter{tool: t}
}

// Info returns eino ToolInfo derived from the underlying llm.Tool.
func (a *ToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info, err := a.tool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("eino tool adapter: get info: %w", err)
	}
	return &schema.ToolInfo{
		Name:        info.Name,
		Desc:        info.Desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(nil),
	}, nil
}

// InvokableRun executes the underlying llm.Tool.
func (a *ToolAdapter) InvokableRun(ctx context.Context, argsJSON string, _ ...einotool.Option) (string, error) {
	return a.tool.Run(ctx, argsJSON, llm.ToolOptionWithContext(ctx))
}

var _ einotool.InvokableTool = (*ToolAdapter)(nil)

// ReverseToolAdapter wraps an eino InvokableTool as an llm.Tool, so that
// existing eino-based tools (MCP adapter, builtin tools) can be used
// through the framework-agnostic interface.
type ReverseToolAdapter struct {
	tool einotool.InvokableTool
}

// NewReverseToolAdapter creates an llm.Tool from an eino InvokableTool.
func NewReverseToolAdapter(t einotool.InvokableTool) *ReverseToolAdapter {
	return &ReverseToolAdapter{tool: t}
}

// Info returns llm.ToolInfo derived from the eino tool.
func (a *ReverseToolAdapter) Info(ctx context.Context) (*llm.ToolInfo, error) {
	einoInfo, err := a.tool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("eino reverse adapter: get info: %w", err)
	}
	return &llm.ToolInfo{
		Name: einoInfo.Name,
		Desc: einoInfo.Desc,
	}, nil
}

// Run executes the underlying eino InvokableTool.
func (a *ReverseToolAdapter) Run(ctx context.Context, argsJSON string, _ ...llm.ToolOption) (string, error) {
	return a.tool.InvokableRun(ctx, argsJSON)
}

var _ llm.Tool = (*ReverseToolAdapter)(nil)

// UnwrapEinoTool returns the underlying eino InvokableTool from a
// ReverseToolAdapter. Returns nil if the tool is not a ReverseToolAdapter.
// This is used during the migration transition period where eino's adk
// still requires the raw eino tool type.
func UnwrapEinoTool(t llm.Tool) einotool.BaseTool {
	if rta, ok := t.(*ReverseToolAdapter); ok {
		return rta.tool
	}
	// If it's a ToolAdapter (forward direction), the underlying is already llm.Tool;
	// wrap it as an eino InvokableTool.
	return NewToolAdapter(t)
}

// UnwrapEinoToolFromSlice converts a slice of llm.Tool to eino InvokableTool,
// unwrapping ReverseToolAdapters and wrapping plain llm.Tools via ToolAdapter.
func UnwrapEinoToolFromSlice(tools []llm.Tool) []einotool.BaseTool {
	result := make([]einotool.BaseTool, 0, len(tools))
	for _, t := range tools {
		if unwrapped := UnwrapEinoTool(t); unwrapped != nil {
			result = append(result, unwrapped)
		}
	}
	return result
}
