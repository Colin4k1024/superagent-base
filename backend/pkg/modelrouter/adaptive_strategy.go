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

package modelrouter

import (
	"context"
)

// AdaptiveStrategy implements Strategy using real-time feedback from a
// FeedbackCollector to rank candidate models by a weighted composite score.
//
// Degradation rules:
//   - Insufficient data (cold-start) → return fallback model.
//   - All candidates circuit-open → return fallback model.
//   - If fallback is empty and no candidates are viable → return ErrNoMatch so
//     the router's strategy chain can try the next strategy.
type AdaptiveStrategy struct {
	collector  *FeedbackCollector
	candidates []string
	weights    ScoreWeights
	fallback   string
}

// NewAdaptiveStrategy creates an AdaptiveStrategy.
// candidates is the ordered list of model IDs to consider.
// fallback is the model ID to return when real-time data is insufficient.
func NewAdaptiveStrategy(collector *FeedbackCollector, candidates []string, weights ScoreWeights, fallback string) *AdaptiveStrategy {
	return &AdaptiveStrategy{
		collector:  collector,
		candidates: candidates,
		weights:    weights,
		fallback:   fallback,
	}
}

// Name satisfies the Strategy interface.
func (as *AdaptiveStrategy) Name() string { return "adaptive" }

// Select picks the highest-scoring non-open-circuit candidate.
// It degrades gracefully:
//  1. Panic recovery → ErrNoMatch
//  2. Cold-start (insufficient data) → fallback
//  3. All candidates open → fallback
//  4. fallback empty with no viable candidates → ErrNoMatch
func (as *AdaptiveStrategy) Select(_ context.Context, _ *RouteRequest) (modelID string, err error) {
	defer func() {
		if r := recover(); r != nil {
			modelID = ""
			err = ErrNoMatch
		}
	}()

	if !as.collector.HasSufficientData(as.candidates) {
		return as.degraded()
	}

	bestModel := ""
	bestScore := -1.0
	for _, candidate := range as.candidates {
		if as.collector.IsCircuitOpen(candidate) {
			continue
		}
		score := as.collector.Score(candidate, as.weights)
		if score > bestScore {
			bestModel = candidate
			bestScore = score
		}
	}

	if bestModel == "" {
		return as.degraded()
	}
	return bestModel, nil
}

// degraded returns the fallback model if configured, otherwise ErrNoMatch.
func (as *AdaptiveStrategy) degraded() (string, error) {
	if as.fallback != "" {
		return as.fallback, nil
	}
	return "", ErrNoMatch
}
