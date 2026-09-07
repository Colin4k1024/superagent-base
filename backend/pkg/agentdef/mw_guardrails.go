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

type guardrailsMiddleware struct {
	aclagent.BaseMiddleware
	maxInputLength  int
	maxOutputLength int
	blockedPatterns []string
}

func (m *guardrailsMiddleware) BeforeAgent(ctx context.Context, mCtx *aclagent.MiddlewareContext) (context.Context, error) {
	if m.maxInputLength > 0 && len(mCtx.Instruction) > m.maxInputLength {
		return ctx, fmt.Errorf("guardrails: input exceeds max length (%d > %d)", len(mCtx.Instruction), m.maxInputLength)
	}
	return ctx, nil
}

func (m *guardrailsMiddleware) AfterModel(ctx context.Context, mState *aclagent.MiddlewareState) error {
	if len(mState.Messages) == 0 {
		return nil
	}
	last := mState.Messages[len(mState.Messages)-1]
	if last.Role != llm.RoleAssistant {
		return nil
	}
	content := last.Content
	if m.maxOutputLength > 0 && len(content) > m.maxOutputLength {
		return fmt.Errorf("guardrails: output exceeds max length (%d > %d)", len(content), m.maxOutputLength)
	}
	for _, pattern := range m.blockedPatterns {
		if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
			return fmt.Errorf("guardrails: output contains blocked pattern %q", pattern)
		}
	}
	return nil
}

func buildGuardrailsHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &guardrailsMiddleware{}
	if v, ok := cfg["max_input_length"]; ok {
		m.maxInputLength = toInt(v)
	}
	if v, ok := cfg["max_output_length"]; ok {
		m.maxOutputLength = toInt(v)
	}
	if v, ok := cfg["blocked_patterns"]; ok {
		m.blockedPatterns = toStringSlice(v)
	}
	return m, nil
}

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case int64:
		return int(t)
	}
	return 0
}

func init() { RegisterMiddleware("guardrails", buildGuardrailsHandler) }
