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
	"strings"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type postProcessingMiddleware struct {
	aclagent.BaseMiddleware
	maxOutputLength int
	stripPII        bool
	piiPatterns     []string
}

func (m *postProcessingMiddleware) AfterModel(ctx context.Context, mState *aclagent.MiddlewareState) error {
	if len(mState.Messages) == 0 {
		return nil
	}
	last := mState.Messages[len(mState.Messages)-1]
	if last.Role != llm.RoleAssistant {
		return nil
	}
	content := last.Content
	if m.maxOutputLength > 0 && len(content) > m.maxOutputLength {
		content = content[:m.maxOutputLength] + "\n[truncated by post_processing middleware]"
	}
	if m.stripPII && len(m.piiPatterns) > 0 {
		for _, pattern := range m.piiPatterns {
			content = strings.ReplaceAll(content, pattern, "[REDACTED]")
		}
	}
	if content != last.Content {
		last.Content = content
	}
	return nil
}

func buildPostProcessingHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &postProcessingMiddleware{}
	if v, ok := cfg["max_output_length"]; ok {
		m.maxOutputLength = toInt(v)
	}
	if v, ok := cfg["strip_pii"]; ok {
		if b, ok := v.(bool); ok {
			m.stripPII = b
		}
	}
	if v, ok := cfg["pii_patterns"]; ok {
		m.piiPatterns = toStringSlice(v)
	}
	return m, nil
}

func init() { RegisterMiddleware("post_processing", buildPostProcessingHandler) }
