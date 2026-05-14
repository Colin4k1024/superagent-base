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
	"time"

	experienceclient "github.com/Colin4k1024/Oris/sdks/go/experience"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

const (
	maxConcurrentShares = 64
	shareTimeout        = 5 * time.Second
)

// SignalCollector receives execution signals and forwards them to the
// Oris Experience Repo via experience.Client.Share().
type SignalCollector struct {
	client *experienceclient.Client
	sem    chan struct{}
}

func newSignalCollector(client *experienceclient.Client) *SignalCollector {
	return &SignalCollector{
		client: client,
		sem:    make(chan struct{}, maxConcurrentShares),
	}
}

// Collect serialises sig into an OEN payload and calls Share asynchronously.
// Goroutine count is bounded by maxConcurrentShares; the caller's context is
// detached to avoid cancellation when the HTTP request completes.
func (c *SignalCollector) Collect(_ context.Context, sig Signal) {
	if c == nil || c.client == nil {
		return
	}
	observe.EvolutionSignalsTotal.WithLabelValues(sig.Type).Inc()

	select {
	case c.sem <- struct{}{}:
		go c.share(sig)
	default:
		observe.EvolutionShareDropped.Inc()
		log.Printf("[evolution] share dropped: semaphore full (type=%s component=%s)", sig.Type, sig.Component)
	}
}

func (c *SignalCollector) share(sig Signal) {
	defer func() { <-c.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), shareTimeout)
	defer cancel()

	payload := buildSharePayload(sig)
	if _, err := c.client.Share(ctx, payload); err != nil {
		observe.EvolutionShareFailed.Inc()
		log.Printf("[evolution] share failed: %v (type=%s)", err, sig.Type)
	} else {
		observe.EvolutionGenesShared.Inc()
	}
}

// buildSharePayload converts a Signal into the map accepted by experience.Client.Share().
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
