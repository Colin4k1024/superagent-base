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
	"encoding/json"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// EvolutionAdvisor queries the local gene store for relevant Gene
// recommendations that can be injected into agent system prompts.
type EvolutionAdvisor struct {
	store          *LocalGeneStore
	minConfidence  float64
	maxSuggestions int
}

func newEvolutionAdvisor(store *LocalGeneStore, minConfidence float64, maxSuggestions int) *EvolutionAdvisor {
	if maxSuggestions <= 0 {
		maxSuggestions = 3
	}
	return &EvolutionAdvisor{
		store:          store,
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
	if a == nil || a.store == nil || query == "" {
		return nil
	}
	if minConfidence <= 0 {
		minConfidence = a.minConfidence
	}
	if limit <= 0 {
		limit = a.maxSuggestions
	}

	genes, err := a.store.Search(ctx, query, minConfidence, limit)
	if err != nil || len(genes) == 0 {
		return nil
	}

	recs := make([]Recommendation, 0, len(genes))
	for _, g := range genes {
		var rate float64
		if g.UseCount > 0 {
			rate = float64(g.SuccessCount) / float64(g.UseCount)
		}

		// Parse strategy back to any type for the recommendation.
		var strategy any
		if g.Strategy != "" {
			_ = json.Unmarshal([]byte(g.Strategy), &strategy)
		}

		recs = append(recs, Recommendation{
			GeneID:      g.ID,
			Strategy:    strategy,
			Confidence:  g.Confidence,
			UseCount:    g.UseCount,
			SuccessRate: rate,
		})
	}
	if len(recs) > 0 {
		observe.EvolutionRecommendationsServed.Add(float64(len(recs)))
	}
	return recs
}
