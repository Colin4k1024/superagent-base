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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCollector returns a FeedbackCollector wired with test-friendly defaults.
func newTestCollector() *FeedbackCollector {
	return NewFeedbackCollector(
		0.5, // high alpha for fast convergence in tests
		2,   // minSamples
		5*time.Minute,
		map[string]PricingInfo{
			"model-a": {InputPer1K: 0.001, OutputPer1K: 0.002},
			"model-b": {InputPer1K: 0.005, OutputPer1K: 0.010},
		},
		DefaultCircuitBreakerConfig(),
	)
}

// ---------- EMA correctness ----------

func TestEMALatencyConverges(t *testing.T) {
	fc := newTestCollector()

	// After several identical samples, EMA should converge toward the sample value.
	for i := 0; i < 10; i++ {
		fc.RecordLatency("model-a", 100*time.Millisecond, 500*time.Millisecond, 50)
	}

	fc.stats["model-a"].mu.RLock()
	ema := fc.stats["model-a"].latencyEMA
	fc.stats["model-a"].mu.RUnlock()

	// With alpha=0.5 and 10 identical 0.1s samples the EMA should be close to 0.1.
	assert.InDelta(t, 0.1, ema, 0.01, "latency EMA should converge to 0.1s")
}

func TestEMASuccessRateConverges(t *testing.T) {
	fc := newTestCollector()

	// 8 successes → success rate EMA should be high.
	for i := 0; i < 8; i++ {
		fc.RecordOutcome("model-a", OutcomeSuccess)
	}

	fc.stats["model-a"].mu.RLock()
	rate := fc.stats["model-a"].successRate
	fc.stats["model-a"].mu.RUnlock()

	assert.Greater(t, rate, 0.9, "success rate EMA should be > 0.9 after 8 successes")
}

func TestEMASuccessRateDropsOnFailures(t *testing.T) {
	fc := newTestCollector()

	// Start with successes then hammer failures.
	for i := 0; i < 5; i++ {
		fc.RecordOutcome("model-a", OutcomeSuccess)
	}
	for i := 0; i < 10; i++ {
		fc.RecordOutcome("model-a", OutcomeFailure)
	}

	fc.stats["model-a"].mu.RLock()
	rate := fc.stats["model-a"].successRate
	fc.stats["model-a"].mu.RUnlock()

	assert.Less(t, rate, 0.3, "success rate should be low after many failures")
}

// ---------- CircuitBreaker state transitions ----------

func TestCircuitBreakerClosedByDefault(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	assert.Equal(t, CBStateClosed, cb.State())
	assert.False(t, cb.IsOpen())
}

func TestCircuitBreakerOpensOnHighFailureRate(t *testing.T) {
	cfg := CircuitBreakerConfig{
		ErrorThreshold: 0.5,
		Cooldown:       1 * time.Hour, // keep open
		ProbeInterval:  1 * time.Hour,
	}
	cb := NewCircuitBreaker(cfg)

	// Record failures until EMA exceeds threshold.
	for i := 0; i < 20; i++ {
		cb.RecordFailure()
	}

	assert.True(t, cb.IsOpen(), "circuit should be open after sustained failures")
	assert.Equal(t, CBStateOpen, cb.State())
}

func TestCircuitBreakerTransitionsToHalfOpenAfterCooldown(t *testing.T) {
	cfg := CircuitBreakerConfig{
		ErrorThreshold: 0.5,
		Cooldown:       50 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 20; i++ {
		cb.RecordFailure()
	}
	require.True(t, cb.IsOpen())

	time.Sleep(60 * time.Millisecond) // wait past cooldown

	// The next IsOpen call should transition to half-open and return false (probe allowed).
	assert.False(t, cb.IsOpen(), "after cooldown a probe should be allowed")
	assert.Equal(t, CBStateHalfOpen, cb.State())
}

func TestCircuitBreakerClosesAfterConsecutiveSuccesses(t *testing.T) {
	cfg := CircuitBreakerConfig{
		ErrorThreshold: 0.5,
		Cooldown:       50 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 20; i++ {
		cb.RecordFailure()
	}
	require.True(t, cb.IsOpen())

	time.Sleep(60 * time.Millisecond)
	cb.IsOpen() // transition to half-open

	cb.RecordSuccess()
	cb.RecordSuccess()
	assert.Equal(t, CBStateClosed, cb.State())
}

// ---------- AdaptiveStrategy.Select ----------

func TestAdaptiveStrategySelectsBestModel(t *testing.T) {
	fc := newTestCollector()

	// model-a gets good metrics, model-b gets poor metrics.
	for i := 0; i < 5; i++ {
		fc.RecordLatency("model-a", 50*time.Millisecond, 200*time.Millisecond, 100)
		fc.RecordOutcome("model-a", OutcomeSuccess)

		fc.RecordLatency("model-b", 500*time.Millisecond, 2*time.Second, 20)
		fc.RecordOutcome("model-b", OutcomeFailure)
	}

	as := NewAdaptiveStrategy(fc, []string{"model-a", "model-b"}, DefaultScoreWeights(), "model-b")
	modelID, err := as.Select(context.Background(), &RouteRequest{})
	require.NoError(t, err)
	assert.Equal(t, "model-a", modelID, "model-a should win due to lower latency and higher success rate")
}

// ---------- Cold-start degradation ----------

func TestAdaptiveStrategyColdStartReturnsFallback(t *testing.T) {
	fc := newTestCollector()
	// No data recorded.
	as := NewAdaptiveStrategy(fc, []string{"model-a", "model-b"}, DefaultScoreWeights(), "model-a")

	modelID, err := as.Select(context.Background(), &RouteRequest{})
	require.NoError(t, err)
	assert.Equal(t, "model-a", modelID, "should return fallback on cold start")
}

func TestAdaptiveStrategyNoFallbackReturnsErrNoMatch(t *testing.T) {
	fc := newTestCollector()
	as := NewAdaptiveStrategy(fc, []string{"model-a"}, DefaultScoreWeights(), "" /* no fallback */)

	_, err := as.Select(context.Background(), &RouteRequest{})
	assert.ErrorIs(t, err, ErrNoMatch)
}

// ---------- Circuit-open candidate exclusion ----------

func TestAdaptiveStrategySkipsOpenCircuitCandidates(t *testing.T) {
	fc := NewFeedbackCollector(
		0.5, 2, 5*time.Minute,
		nil,
		CircuitBreakerConfig{
			ErrorThreshold: 0.3,
			Cooldown:       1 * time.Hour,
			ProbeInterval:  1 * time.Hour,
		},
	)

	// Give both models enough data.
	for i := 0; i < 10; i++ {
		fc.RecordOutcome("model-a", OutcomeSuccess)
		fc.RecordLatency("model-a", 100*time.Millisecond, 400*time.Millisecond, 50)
	}
	for i := 0; i < 10; i++ {
		fc.RecordOutcome("model-b", OutcomeFailure) // will trip breaker
		fc.RecordLatency("model-b", 100*time.Millisecond, 400*time.Millisecond, 50)
	}

	as := NewAdaptiveStrategy(fc, []string{"model-a", "model-b"}, DefaultScoreWeights(), "fallback")
	modelID, err := as.Select(context.Background(), &RouteRequest{})
	require.NoError(t, err)
	assert.Equal(t, "model-a", modelID, "open-circuit model-b should be excluded")
}

// ---------- Router.RecordOutcome integration ----------

func TestRouterRecordOutcomeIsNoopWhenFeedbackDisabled(t *testing.T) {
	router, err := NewDefaultRouter(testConfig)
	require.NoError(t, err)

	// Should not panic.
	router.RecordOutcome("gpt-4o", 100*time.Millisecond, 500*time.Millisecond, 100, 200, nil)
}

func TestRouterRecordOutcomeUpdatesCollectorWhenEnabled(t *testing.T) {
	cfg := &RouterConfig{
		Strategies: testConfig.Strategies,
		Providers:  testConfig.Providers,
		Feedback: &FeedbackConfig{
			Enabled:    true,
			EMAAlpha:   0.5,
			MinSamples: 2,
		},
	}

	router, err := NewDefaultRouter(cfg)
	require.NoError(t, err)

	router.RecordOutcome("gpt-4o", 100*time.Millisecond, 500*time.Millisecond, 100, 200, nil)

	router.mu.RLock()
	fc := router.feedback
	router.mu.RUnlock()
	require.NotNil(t, fc)

	fc.mu.RLock()
	s, ok := fc.stats["gpt-4o"]
	fc.mu.RUnlock()

	require.True(t, ok)
	s.mu.RLock()
	count := s.sampleCount
	s.mu.RUnlock()
	assert.Greater(t, count, int64(0))
}

// ---------- Concurrent safety ----------

func TestFeedbackCollectorConcurrentSafety(t *testing.T) {
	fc := newTestCollector()
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			model := "model-a"
			if id%2 == 0 {
				model = "model-b"
			}
			for i := 0; i < iterations; i++ {
				fc.RecordLatency(model, 100*time.Millisecond, 400*time.Millisecond, 50)
				fc.RecordOutcome(model, OutcomeSuccess)
				fc.RecordTokens(model, 100, 200)
				_ = fc.Score(model, DefaultScoreWeights())
				_ = fc.HasSufficientData([]string{model})
				_ = fc.IsCircuitOpen(model)
			}
		}(g)
	}
	wg.Wait()
}

func TestCircuitBreakerConcurrentSafety(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if id%3 == 0 {
					cb.RecordFailure()
				} else {
					cb.RecordSuccess()
				}
				_ = cb.IsOpen()
				_ = cb.AllowRequest()
			}
		}(g)
	}
	wg.Wait()
}
