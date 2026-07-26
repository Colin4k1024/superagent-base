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

package agentdef

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// DynamicModelSelector handles per-request model selection based on complexity analysis.
// It is used by chat agents that have a tiered model pool configured.
type DynamicModelSelector struct {
	poolRouter *modelrouter.ModelPoolRouter
	def        *AgentDefinition
}

// NewDynamicModelSelector creates a selector for the given agent definition.
func NewDynamicModelSelector(
	def *AgentDefinition,
	analyzer modelrouter.ComplexityAnalyzer,
	tierModels map[string]model.ToolCallingChatModel,
	primary model.ToolCallingChatModel,
	fallback model.ToolCallingChatModel,
) *DynamicModelSelector {
	poolRouter := modelrouter.NewModelPoolRouter(modelrouter.ModelPoolConfig{
		Analyzer: analyzer,
		Models:   tierModels,
		Primary:  primary,
		Fallback: fallback,
	})
	return &DynamicModelSelector{
		poolRouter: poolRouter,
		def:        def,
	}
}

// SelectModel picks the best model for the given messages.
// Returns the selected model and the detected complexity level.
func (s *DynamicModelSelector) SelectModel(ctx context.Context, messages []*schema.Message) (model.ToolCallingChatModel, string) {
	m, complexity := s.poolRouter.SelectModel(ctx, messages)

	if complexity != "" {
		log.Printf("[model-router] agent=%s complexity=%s model_selected", s.def.Metadata.Name, complexity)
		observe.DynamicRouteTotal.WithLabelValues(s.def.Metadata.Name, complexity, "").Inc()
	}

	return m, complexity
}

// SelectModelWithFallback returns both primary and fallback models.
func (s *DynamicModelSelector) SelectModelWithFallback(ctx context.Context, messages []*schema.Message) (model.ToolCallingChatModel, model.ToolCallingChatModel, string) {
	primary, fb, complexity := s.poolRouter.SelectModelWithFallback(ctx, messages)

	if complexity != "" {
		log.Printf("[model-router] agent=%s complexity=%s model_with_fallback", s.def.Metadata.Name, complexity)
		observe.DynamicRouteTotal.WithLabelValues(s.def.Metadata.Name, complexity, "").Inc()
	}

	return primary, fb, complexity
}

// BuildTierModels creates ChatModel instances for each tier in the agent's model spec.
// providers maps model ID to (endpoint, apiKey) from the routing config.
// Returns a map of level -> ChatModel.
func BuildTierModels(
	ctx context.Context,
	def *AgentDefinition,
	createModel func(ctx context.Context, protocol, baseURL, apiKey, modelID string) (model.ToolCallingChatModel, error),
	defaultBaseURL string,
	defaultAPIKey string,
	defaultProtocol string,
	providers map[string]ProviderEndpoint,
) (map[string]model.ToolCallingChatModel, error) {
	tiers := def.Spec.Model.Models
	if len(tiers) == 0 {
		return nil, nil
	}

	result := make(map[string]model.ToolCallingChatModel, len(tiers))
	for _, tier := range tiers {
		protocol := defaultProtocol
		if tier.Protocol != "" {
			protocol = tier.Protocol
		} else if def.Spec.Model.Protocol != "" {
			protocol = def.Spec.Model.Protocol
		}

		// Look up provider-specific endpoint from routing config.
		baseURL := defaultBaseURL
		apiKey := defaultAPIKey
		if pe, ok := providers[tier.ModelID]; ok {
			if pe.Endpoint != "" {
				baseURL = pe.Endpoint
			}
			if pe.APIKey != "" {
				apiKey = pe.APIKey
			}
		}
		// Per-tier overrides take highest priority.
		if tier.BaseURL != "" {
			baseURL = tier.BaseURL
		} else if def.Spec.Model.BaseURL != "" {
			baseURL = def.Spec.Model.BaseURL
		}

		m, err := createModel(ctx, protocol, baseURL, apiKey, tier.ModelID)
		if err != nil {
			return nil, err
		}
		result[tier.Level] = m
	}

	return result, nil
}

// ProviderEndpoint holds the connection info for a model provider.
type ProviderEndpoint struct {
	Endpoint string
	APIKey   string
}
