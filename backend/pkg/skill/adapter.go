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

package skill

import (
	"context"
	"encoding/json"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// Compile-time assertion: SkillTool must implement llm.Tool.
var _ llm.Tool = (*SkillTool)(nil)

// SkillInvoker calls a skill's runtime and returns the result.
type SkillInvoker interface {
	Invoke(ctx context.Context, skillName string, input map[string]any) (map[string]any, error)
}

// SkillTool wraps a SkillMeta and a SkillInvoker as a framework-agnostic llm.Tool.
type SkillTool struct {
	meta    SkillMeta
	invoker SkillInvoker
}

// NewSkillTool creates a SkillTool from the provided metadata and invoker.
func NewSkillTool(meta SkillMeta, invoker SkillInvoker) *SkillTool {
	return &SkillTool{meta: meta, invoker: invoker}
}

// Info returns the ACL ToolInfo derived from the skill's metadata and input schema.
func (s *SkillTool) Info(_ context.Context) (*llm.ToolInfo, error) {
	params := jsonSchemaToACLParams(s.meta.Input)
	info := &llm.ToolInfo{
		Name: s.meta.Name,
		Desc: s.meta.Description,
	}
	if len(params) > 0 {
		info.ParamsOneOf = llm.NewParamsOneOfByParams(params)
	}
	return info, nil
}

// Run parses the JSON arguments string, calls the skill's runtime via the invoker,
// and returns the result serialised as JSON.
func (s *SkillTool) Run(ctx context.Context, argumentsInJSON string, _ ...llm.ToolOption) (string, error) {
	var input map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}

	output, err := s.invoker.Invoke(ctx, s.meta.Name, input)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// jsonSchemaToACLParams converts a JSONSchema into ACL ParameterInfo entries.
// Only the top-level object properties are mapped; nested objects become sub-parameters.
func jsonSchemaToACLParams(js *JSONSchema) map[string]*llm.ParameterInfo {
	if js == nil || js.Type != "object" || len(js.Properties) == 0 {
		return nil
	}

	requiredSet := make(map[string]bool, len(js.Required))
	for _, r := range js.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*llm.ParameterInfo, len(js.Properties))
	for name, prop := range js.Properties {
		if prop == nil {
			continue
		}
		p := &llm.ParameterInfo{
			Desc:     prop.Description,
			Type:     mapSchemaType(prop.Type),
			Required: requiredSet[name],
		}
		if prop.Type == "object" && len(prop.Properties) > 0 {
			p.SubParams = jsonSchemaToACLParams(prop)
		}
		params[name] = p
	}
	return params
}

func mapSchemaType(t string) llm.DataType {
	switch t {
	case "string":
		return llm.DTString
	case "integer":
		return llm.DTInteger
	case "number":
		return llm.DTNumber
	case "boolean":
		return llm.DTBoolean
	case "array":
		return llm.DTArray
	case "object":
		return llm.DTObject
	default:
		return llm.DTString
	}
}
