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

package evolution

import (
	"context"

	experienceclient "github.com/Colin4k1024/Oris/sdks/go/experience"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// EvolutionAdvisor queries the Oris Experience Repo for relevant Gene
// recommendations that can be injected into agent system prompts.
type EvolutionAdvisor struct {
	client        *experienceclient.Client
	minConfidence float64
	maxSuggestions int
}

func newEvolutionAdvisor(client *experienceclient.Client, minConfidence float64, maxSuggestions int) *EvolutionAdvisor {
	if maxSuggestions <= 0 {
		maxSuggestions = 3
	}
	return &EvolutionAdvisor{
		client:         client,
		minConfidence:  minConfidence,
		maxSuggestions: maxSuggestions,
	}
}

// Recommend fetches Gene recommendations for a free-text query using default confidence/limit.
func (a *EvolutionAdvisor) Recommend(ctx context.Context, query string) []Recommendation {
	return a.RecommendWithOpts(ctx, query, 0, 0)
}

// RecommendWithOpts fetches Gene recommendations with explicit confidence and limit overrides.
// Zero values fall back to the configured defaults.
func (a *EvolutionAdvisor) RecommendWithOpts(ctx context.Context, query string, minConfidence float64, limit int) []Recommendation {
	if a == nil || a.client == nil || query == "" {
		return nil
	}
	if minConfidence <= 0 {
		minConfidence = a.minConfidence
	}
	if limit <= 0 {
		limit = a.maxSuggestions
	}

	results, err := a.client.Fetch(ctx, &experienceclient.FetchQuery{
		Q:             query,
		MinConfidence: minConfidence,
		Limit:         limit,
	})
	if err != nil || results == nil {
		return nil
	}

	recs := make([]Recommendation, 0, len(results.Assets))
	for _, asset := range results.Assets {
		var rate float64
		if asset.UseCount > 0 {
			rate = float64(asset.SuccessCount) / float64(asset.UseCount)
		}
		recs = append(recs, Recommendation{
			GeneID:      asset.ID,
			Strategy:    asset.Strategy,
			Confidence:  asset.Confidence,
			UseCount:    asset.UseCount,
			SuccessRate: rate,
		})
	}
	if len(recs) > 0 {
		observe.EvolutionRecommendationsServed.Add(float64(len(recs)))
	}
	return recs
}
