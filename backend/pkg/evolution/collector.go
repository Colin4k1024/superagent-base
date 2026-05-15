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
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

const (
	maxConcurrentShares = 64
	shareTimeout        = 5 * time.Second
	drainTimeout        = 10 * time.Second
)

// SignalCollector receives execution signals and persists them as Genes
// in the local MySQL store.
type SignalCollector struct {
	store    *LocalGeneStore
	senderID string
	sem      chan struct{}
	wg       sync.WaitGroup
}

func newSignalCollector(store *LocalGeneStore, senderID string) *SignalCollector {
	return &SignalCollector{
		store:    store,
		senderID: senderID,
		sem:      make(chan struct{}, maxConcurrentShares),
	}
}

// Collect serialises sig into a payload and saves it locally.
// Goroutine count is bounded by maxConcurrentShares.
func (c *SignalCollector) Collect(_ context.Context, sig Signal) {
	if c == nil || c.store == nil {
		return
	}
	observe.EvolutionSignalsTotal.WithLabelValues(sig.Type).Inc()

	select {
	case c.sem <- struct{}{}:
		c.wg.Add(1)
		go c.save(sig)
	default:
		observe.EvolutionShareDropped.Inc()
		log.Printf("[evolution] save dropped: semaphore full (type=%s component=%s)", sig.Type, sig.Component)
	}
}

// Drain waits for all in-flight save goroutines to complete, with a timeout.
func (c *SignalCollector) Drain() {
	if c == nil {
		return
	}
	done := make(chan struct{})
	go func() { c.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(drainTimeout):
		log.Printf("[evolution] drain timeout: some saves may be lost")
	}
}

func (c *SignalCollector) save(sig Signal) {
	defer c.wg.Done()
	defer func() { <-c.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), shareTimeout)
	defer cancel()

	payload := buildSharePayload(sig)
	payload["sender_id"] = c.senderID
	if _, err := c.store.SaveGene(ctx, payload); err != nil {
		observe.EvolutionShareFailed.Inc()
		log.Printf("[evolution] local save failed: %v (type=%s)", err, sig.Type)
	} else {
		observe.EvolutionGenesShared.Inc()
	}
}

// buildSharePayload converts a Signal into the map stored as a Gene.
func buildSharePayload(sig Signal) map[string]any {
	outcome := "success"
	if sig.Error != "" {
		outcome = "failure"
	}

	signals := map[string]any{
		"signal_type": sig.Type,
		"agent_name":  sig.AgentName,
		"session_id":  sig.SessionID,
		"component":   sig.Component,
		"input":       sig.Input,
		"outcome":     outcome,
		"timestamp":   sig.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if sig.Duration > 0 {
		signals["latency_ms"] = sig.Duration.Milliseconds()
	}

	strategy := map[string]any{
		"component": sig.Component,
		"type":      sig.Type,
	}
	if sig.Output != "" {
		strategy["output_summary"] = sig.Output
	}
	if len(sig.Metadata) > 0 {
		strategy["metadata"] = sig.Metadata
	}

	validation := map[string]any{
		"outcome": outcome,
	}
	if sig.Error != "" {
		validation["error"] = sig.Error
	}

	label := fmt.Sprintf("%s:%s", sig.Type, sig.Component)

	return map[string]any{
		"message_type": "gene_contribution",
		"signals":      signals,
		"strategy":     strategy,
		"validation":   validation,
		"label":        label,
	}
}
