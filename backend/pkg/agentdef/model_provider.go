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

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// createACLChatModel creates an llm.ChatModel via the ACL provider registry.
// Google ADK Go is the sole runtime; no eino unwrapping is needed.
func (b *AgentBuilder) createACLChatModel(ctx context.Context, protocol, baseURL, apiKey, modelID string) (llm.ChatModel, error) {
	if b.modelProviderRegistry == nil {
		return nil, fmt.Errorf("agentdef: model provider registry is not configured")
	}

	effectiveProtocol := protocol
	if effectiveProtocol == "" {
		effectiveProtocol = "openai"
	}

	return b.modelProviderRegistry.Create(ctx, llm.ModelConfig{
		Protocol: effectiveProtocol,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ModelID:  modelID,
	})
}


