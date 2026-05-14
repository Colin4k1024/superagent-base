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

	experienceclient "github.com/Colin4k1024/Oris/sdks/go/experience"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// SignalCollector receives execution signals and forwards them to the
// Oris Experience Repo via experience.Client.Share().
type SignalCollector struct {
	client *experienceclient.Client
}

func newSignalCollector(client *experienceclient.Client) *SignalCollector {
	return &SignalCollector{client: client}
}

// Collect serialises sig into an OEN payload and calls Share asynchronously.
// This method is goroutine-safe and must not block the caller.
func (c *SignalCollector) Collect(ctx context.Context, sig Signal) {
	if c == nil || c.client == nil {
		return
	}
	observe.EvolutionSignalsTotal.WithLabelValues(sig.Type).Inc()
	go c.share(ctx, sig)
}

func (c *SignalCollector) share(ctx context.Context, sig Signal) {
	payload := buildSharePayload(sig)
	if _, err := c.client.Share(ctx, payload); err == nil {
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
