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

// CBState represents the current state of a CircuitBreaker.
type CBState int

const (
	// CBStateClosed means the circuit is healthy; requests are allowed.
	CBStateClosed CBState = iota
	// CBStateOpen means the circuit is tripped; requests are blocked.
	CBStateOpen
	// CBStateHalfOpen means a probe request is allowed to test recovery.
	CBStateHalfOpen
)

// CircuitBreakerConfig holds the tuneable parameters for a CircuitBreaker.
type CircuitBreakerConfig struct {
	// ErrorThreshold is the failure rate (0.0–1.0) that trips the circuit.
	// Default: 0.5.
	ErrorThreshold float64
	// Cooldown is how long the circuit stays open before moving to half-open.
	// Default: 30s.
	Cooldown time.Duration
	// ProbeInterval is the minimum time between probe attempts in half-open state.
	// Default: 10s.
	ProbeInterval time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		ErrorThreshold: 0.5,
		Cooldown:       30 * time.Second,
		ProbeInterval:  10 * time.Second,
	}
}

// CircuitBreaker implements a three-state circuit breaker (closed → open → half-open → closed).
// It is safe for concurrent use.
type CircuitBreaker struct {
	mu                 sync.RWMutex
	state              CBState
	errorThreshold     float64
	cooldown           time.Duration
	probeInterval      time.Duration
	lastStateChange    time.Time
	consecutiveSuccess int
	// EMA of failure rate.
	failureRateEMA float64
}

const cbAlpha = 0.2 // EMA coefficient for the circuit breaker's failure rate.

// NewCircuitBreaker creates a CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.ErrorThreshold <= 0 || cfg.ErrorThreshold > 1 {
		cfg.ErrorThreshold = 0.5
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 30 * time.Second
	}
	if cfg.ProbeInterval <= 0 {
		cfg.ProbeInterval = 10 * time.Second
	}
	return &CircuitBreaker{
		state:           CBStateClosed,
		errorThreshold:  cfg.ErrorThreshold,
		cooldown:        cfg.Cooldown,
		probeInterval:   cfg.ProbeInterval,
		lastStateChange: time.Now(),
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureRateEMA = (1-cbAlpha)*cb.failureRateEMA // 0 failure sample

	switch cb.state {
	case CBStateHalfOpen:
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= 2 {
			cb.state = CBStateClosed
			cb.lastStateChange = time.Now()
			cb.consecutiveSuccess = 0
		}
	case CBStateOpen:
		// Shouldn't happen normally, but reset if we get a success.
		cb.state = CBStateClosed
		cb.lastStateChange = time.Now()
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureRateEMA = cbAlpha*1.0 + (1-cbAlpha)*cb.failureRateEMA

	if cb.state == CBStateClosed && cb.failureRateEMA >= cb.errorThreshold {
		cb.state = CBStateOpen
		cb.lastStateChange = time.Now()
		cb.consecutiveSuccess = 0
	}
	if cb.state == CBStateHalfOpen {
		// Probe failed; go back to open.
		cb.state = CBStateOpen
		cb.lastStateChange = time.Now()
		cb.consecutiveSuccess = 0
	}
}

// IsOpen returns true when the circuit is open (requests should be blocked).
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	switch cb.state {
	case CBStateClosed:
		return false
	case CBStateOpen:
		if now.Sub(cb.lastStateChange) >= cb.cooldown {
			cb.state = CBStateHalfOpen
			cb.lastStateChange = now
			cb.consecutiveSuccess = 0
			return false // allow a probe
		}
		return true
	case CBStateHalfOpen:
		// Only allow one probe per probeInterval.
		if now.Sub(cb.lastStateChange) >= cb.probeInterval {
			cb.lastStateChange = now
			return false
		}
		return true
	}
	return false
}

// AllowRequest returns true when the circuit is not open and a request may proceed.
func (cb *CircuitBreaker) AllowRequest() bool {
	return !cb.IsOpen()
}

// State returns the current CBState. Useful for testing.
func (cb *CircuitBreaker) State() CBState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
