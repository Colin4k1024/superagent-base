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
	"sync"
	"time"
)

// Outcome represents the result of a model call.
type Outcome int

const (
	// OutcomeSuccess indicates the model call succeeded.
	OutcomeSuccess Outcome = iota
	// OutcomeFailure indicates the model call failed.
	OutcomeFailure
)

// ScoreWeights holds the relative weights for the composite routing score.
type ScoreWeights struct {
	Latency float64
	Success float64
	Cost    float64
}

// DefaultScoreWeights returns the default scoring weights (latency=0.3, success=0.5, cost=0.2).
func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{Latency: 0.3, Success: 0.5, Cost: 0.2}
}

// ProviderStats holds the EMA-aggregated runtime metrics for a single provider/model.
type ProviderStats struct {
	mu           sync.RWMutex
	latencyEMA   float64
	tpsEMA       float64
	successRate  float64
	costPerToken float64
	lastUpdated  time.Time
	sampleCount  int64
}

// FeedbackCollector aggregates real-time provider metrics in memory using EMA.
// It is safe for concurrent use.
type FeedbackCollector struct {
	mu         sync.RWMutex
	stats      map[string]*ProviderStats
	alpha      float64
	minSamples int
	staleDur   time.Duration
	pricing    map[string]PricingInfo
	breakers   map[string]*CircuitBreaker
	cbConfig   CircuitBreakerConfig
}

// NewFeedbackCollector creates a FeedbackCollector with the given configuration.
// alpha is the EMA decay coefficient (0 < alpha <= 1, default 0.1).
// minSamples is the cold-start threshold below which HasSufficientData returns false.
// staleDur is the duration after which a provider's data is considered stale.
func NewFeedbackCollector(alpha float64, minSamples int, staleDur time.Duration, pricing map[string]PricingInfo, cbCfg CircuitBreakerConfig) *FeedbackCollector {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.1
	}
	if minSamples <= 0 {
		minSamples = 5
	}
	if staleDur <= 0 {
		staleDur = 5 * time.Minute
	}
	if pricing == nil {
		pricing = make(map[string]PricingInfo)
	}
	return &FeedbackCollector{
		stats:    make(map[string]*ProviderStats),
		alpha:    alpha,
		minSamples: minSamples,
		staleDur: staleDur,
		pricing:  pricing,
		breakers: make(map[string]*CircuitBreaker),
		cbConfig: cbCfg,
	}
}

// getOrCreate returns the ProviderStats for modelID, creating it if absent.
func (fc *FeedbackCollector) getOrCreate(modelID string) *ProviderStats {
	fc.mu.RLock()
	s, ok := fc.stats[modelID]
	fc.mu.RUnlock()
	if ok {
		return s
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()
	// Double-check after acquiring write lock.
	if s, ok = fc.stats[modelID]; ok {
		return s
	}
	s = &ProviderStats{}
	fc.stats[modelID] = s
	return s
}

// getOrCreateBreaker returns the CircuitBreaker for modelID, creating it if absent.
func (fc *FeedbackCollector) getOrCreateBreaker(modelID string) *CircuitBreaker {
	fc.mu.RLock()
	b, ok := fc.breakers[modelID]
	fc.mu.RUnlock()
	if ok {
		return b
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()
	if b, ok = fc.breakers[modelID]; ok {
		return b
	}
	b = NewCircuitBreaker(fc.cbConfig)
	fc.breakers[modelID] = b
	return b
}

// RecordLatency records TTFT and throughput metrics for modelID.
// outputTokens may be 0 when tokens are unknown; tpsEMA is not updated in that case.
func (fc *FeedbackCollector) RecordLatency(modelID string, ttft, totalDur time.Duration, outputTokens int) {
	stats := fc.getOrCreate(modelID)
	stats.mu.Lock()
	defer stats.mu.Unlock()

	alpha := fc.alpha
	stats.latencyEMA = alpha*ttft.Seconds() + (1-alpha)*stats.latencyEMA
	if outputTokens > 0 && totalDur.Seconds() > 0 {
		tps := float64(outputTokens) / totalDur.Seconds()
		stats.tpsEMA = alpha*tps + (1-alpha)*stats.tpsEMA
	}
	stats.lastUpdated = time.Now()
	stats.sampleCount++
}

// RecordOutcome records the success/failure outcome for modelID.
func (fc *FeedbackCollector) RecordOutcome(modelID string, outcome Outcome) {
	stats := fc.getOrCreate(modelID)
	stats.mu.Lock()
	defer stats.mu.Unlock()

	successSample := 0.0
	if outcome == OutcomeSuccess {
		successSample = 1.0
	}
	alpha := fc.alpha
	stats.successRate = alpha*successSample + (1-alpha)*stats.successRate
	stats.lastUpdated = time.Now()
	stats.sampleCount++

	// Update circuit breaker.
	b := fc.getOrCreateBreaker(modelID)
	if outcome == OutcomeSuccess {
		b.RecordSuccess()
	} else {
		b.RecordFailure()
	}
}

// RecordTokens records input/output token counts and updates the cost EMA for modelID.
func (fc *FeedbackCollector) RecordTokens(modelID string, inputTokens, outputTokens int) {
	stats := fc.getOrCreate(modelID)

	fc.mu.RLock()
	pricing, hasPricing := fc.pricing[modelID]
	fc.mu.RUnlock()

	if !hasPricing || (inputTokens+outputTokens) == 0 {
		return
	}

	cost := float64(inputTokens)/1000*pricing.InputPer1K +
		float64(outputTokens)/1000*pricing.OutputPer1K
	costPerToken := cost / float64(inputTokens+outputTokens)

	stats.mu.Lock()
	defer stats.mu.Unlock()

	alpha := fc.alpha
	stats.costPerToken = alpha*costPerToken + (1-alpha)*stats.costPerToken
	stats.lastUpdated = time.Now()
}

// Score computes a composite routing score for modelID using the given weights.
// Higher scores indicate a more desirable provider.
// Returns 0.0 if modelID has no recorded data.
func (fc *FeedbackCollector) Score(modelID string, w ScoreWeights) float64 {
	fc.mu.RLock()
	s, ok := fc.stats[modelID]
	fc.mu.RUnlock()
	if !ok {
		return 0.0
	}

	s.mu.RLock()
	latencyEMA := s.latencyEMA
	successRate := s.successRate
	costPerToken := s.costPerToken
	s.mu.RUnlock()

	latencyScore := 1.0 / (1.0 + latencyEMA*10)
	successScore := successRate
	costScore := 1.0 / (1.0 + costPerToken*1000)

	return w.Latency*latencyScore + w.Success*successScore + w.Cost*costScore
}

// HasSufficientData returns true when at least one candidate has >= minSamples recorded.
func (fc *FeedbackCollector) HasSufficientData(candidates []string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	for _, id := range candidates {
		s, ok := fc.stats[id]
		if !ok {
			continue
		}
		s.mu.RLock()
		count := s.sampleCount
		updated := s.lastUpdated
		s.mu.RUnlock()

		if count >= int64(fc.minSamples) && time.Since(updated) < fc.staleDur {
			return true
		}
	}
	return false
}

// IsCircuitOpen returns true when modelID's circuit breaker is in the open state
// and requests should not be forwarded.
func (fc *FeedbackCollector) IsCircuitOpen(modelID string) bool {
	fc.mu.RLock()
	b, ok := fc.breakers[modelID]
	fc.mu.RUnlock()
	if !ok {
		return false
	}
	return b.IsOpen()
}
