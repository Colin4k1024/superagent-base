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

	"github.com/cloudwego/eino/components/model"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	einollm "github.com/superagent-ai/superagent-base/backend/pkg/llm/eino"
)

// createChatModel creates a ChatModel via the anti-corruption layer's
// ModelProviderRegistry. The returned llm.ChatModel is unwrapped to
// the eino ToolCallingChatModel for compatibility with eino's adk
// during the migration transition period.
//
// Once the migration to the target framework is complete, this unwrap
// step will be removed and callers will use llm.ChatModel directly.
func (b *AgentBuilder) createChatModel(ctx context.Context, protocol, baseURL, apiKey, modelID string) (model.ToolCallingChatModel, error) {
	if b.modelProviderRegistry == nil {
		return nil, fmt.Errorf("agentdef: model provider registry is not configured")
	}

	// Empty protocol defaults to "openai" (OpenAI-compatible API).
	effectiveProtocol := protocol
	if effectiveProtocol == "" {
		effectiveProtocol = "openai"
	}

	chatModel, err := b.modelProviderRegistry.Create(ctx, llm.ModelConfig{
		Protocol: effectiveProtocol,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ModelID:  modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("agentdef: create model (protocol=%s): %w", effectiveProtocol, err)
	}

	// Unwrap the ACL adapter to get the eino model for adk compatibility.
	adapter, ok := chatModel.(*einollm.ChatModelAdapter)
	if !ok {
		return nil, fmt.Errorf("agentdef: unexpected ChatModel type %T (expected *eino.ChatModelAdapter)", chatModel)
	}
	return adapter.UnwrapEinoModel(), nil
}
