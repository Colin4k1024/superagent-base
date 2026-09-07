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
	"strings"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type toolPermissionMiddleware struct {
	aclagent.BaseMiddleware
	allow []string
	deny  []string
}

func (m *toolPermissionMiddleware) isAllowed(toolName string) bool {
	if len(m.deny) > 0 {
		for _, pattern := range m.deny {
			if matchPattern(pattern, toolName) {
				return false
			}
		}
	}
	if len(m.allow) > 0 {
		for _, pattern := range m.allow {
			if matchPattern(pattern, toolName) {
				return true
			}
		}
		return false
	}
	return true
}

func matchPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	return pattern == name
}

func (m *toolPermissionMiddleware) WrapTool(ctx context.Context, tool llm.Tool) (llm.Tool, error) {
	info, err := tool.Info(ctx)
	if err != nil {
		return nil, err
	}
	if m.isAllowed(info.Name) {
		return tool, nil
	}
	return &deniedTool{name: info.Name}, nil
}

type deniedTool struct {
	name string
}

func (t *deniedTool) Info(_ context.Context) (*llm.ToolInfo, error) {
	return &llm.ToolInfo{Name: t.name}, nil
}

func (t *deniedTool) Run(_ context.Context, _ string, _ ...llm.ToolOption) (string, error) {
	return "", fmt.Errorf("tool %q denied by tool_permission middleware", t.name)
}

func buildToolPermissionHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &toolPermissionMiddleware{}
	if v, ok := cfg["allow"]; ok {
		m.allow = toStringSlice(v)
	}
	if v, ok := cfg["deny"]; ok {
		m.deny = toStringSlice(v)
	}
	return m, nil
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return t
	}
	return nil
}

func init() { RegisterMiddleware("tool_permission", buildToolPermissionHandler) }
