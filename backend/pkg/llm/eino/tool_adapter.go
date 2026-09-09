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
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"

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
func (a *ToolAdapter) Info(ctx context.Context) (*einobridge.ToolInfo, error) {
	info, err := a.tool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("eino tool adapter: get info: %w", err)
	}
	ei := &einobridge.ToolInfo{
		Name:  info.Name,
		Desc:  info.Desc,
		Extra: info.Extra,
	}
	if pp, ok := info.ParamsOneOf.(*llm.ParamsOneOfByParams); ok && pp != nil {
		ei.ParamsOneOf = einobridge.NewParamsOneOfByParams(aclParamsToEino(pp.Params()))
	}
	return ei, nil
}

// aclParamsToEino converts ACL ParameterInfo map to eino ParameterInfo map.
func aclParamsToEino(params map[string]*llm.ParameterInfo) map[string]*einobridge.ParameterInfo {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]*einobridge.ParameterInfo, len(params))
	for name, p := range params {
		if p == nil {
			continue
		}
		ep := &einobridge.ParameterInfo{
			Type:      einobridge.DataType(p.Type),
			Desc:      p.Desc,
			Enum:      p.Enum,
			Required:  p.Required,
			SubParams: aclParamsToEino(p.SubParams),
		}
		if p.ElemInfo != nil {
			ep.ElemInfo = &einobridge.ParameterInfo{
				Type:     einobridge.DataType(p.ElemInfo.Type),
				Desc:     p.ElemInfo.Desc,
				Enum:     p.ElemInfo.Enum,
				Required: p.ElemInfo.Required,
			}
		}
		out[name] = ep
	}
	return out
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
	ti := &llm.ToolInfo{
		Name:  einoInfo.Name,
		Desc:  einoInfo.Desc,
		Extra: einoInfo.Extra,
	}
	if einoInfo.ParamsOneOf != nil {
		ti.ParamsOneOf = llm.NewParamsOneOfByParams(einoParamsToACL(einoInfo.ParamsOneOf))
	}
	return ti, nil
}

// Run executes the underlying eino InvokableTool.
func (a *ReverseToolAdapter) Run(ctx context.Context, argsJSON string, _ ...llm.ToolOption) (string, error) {
	return a.tool.InvokableRun(ctx, argsJSON)
}

var _ llm.Tool = (*ReverseToolAdapter)(nil)

// einoParamsToACL converts eino ParamsOneOf to ACL ParameterInfo map.
// Uses the ParamsOneOf.ToJSONSchema method to extract params when available.
func einoParamsToACL(p *einobridge.ParamsOneOf) map[string]*llm.ParameterInfo {
	if p == nil {
		return nil
	}
	js, err := p.ToJSONSchema()
	if err != nil || js == nil || js.Properties == nil {
		return nil
	}
	requiredSet := make(map[string]bool, len(js.Required))
	for _, r := range js.Required {
		requiredSet[r] = true
	}
	out := make(map[string]*llm.ParameterInfo)
	for pair := js.Properties.Oldest(); pair != nil; pair = pair.Next() {
		v := pair.Value
		if v == nil {
			continue
		}
		pi := &llm.ParameterInfo{
			Desc:     v.Description,
			Required: requiredSet[pair.Key],
		}
		if len(v.Type) > 0 {
			pi.Type = llm.DataType(v.Type)
		}
		out[pair.Key] = pi
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

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
