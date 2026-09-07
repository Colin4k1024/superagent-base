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
	"time"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type contextInjectionMiddleware struct {
	aclagent.BaseMiddleware
	injectTimestamp       bool
	injectSessionMetadata bool
	staticContext         string
}

func (m *contextInjectionMiddleware) BeforeModel(ctx context.Context, mState *aclagent.MiddlewareState) (context.Context, error) {
	var injections []string
	if m.injectTimestamp {
		injections = append(injections, fmt.Sprintf("[current_time: %s]", time.Now().Format(time.RFC3339)))
	}
	if m.staticContext != "" {
		injections = append(injections, m.staticContext)
	}
	if len(injections) == 0 {
		return ctx, nil
	}
	contextMsg := llm.SystemMessage("[context_injection]\n" + joinStrings(injections, "\n"))
	mState.Messages = append([]*llm.Message{contextMsg}, mState.Messages...)
	return ctx, nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}

func buildContextInjectionHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &contextInjectionMiddleware{}
	if v, ok := cfg["inject_timestamp"]; ok {
		if b, ok := v.(bool); ok {
			m.injectTimestamp = b
		}
	}
	if v, ok := cfg["inject_session_metadata"]; ok {
		if b, ok := v.(bool); ok {
			m.injectSessionMetadata = b
		}
	}
	if v, ok := cfg["static_context"]; ok {
		if s, ok := v.(string); ok {
			m.staticContext = s
		}
	}
	return m, nil
}

func init() { RegisterMiddleware("context_injection", buildContextInjectionHandler) }
