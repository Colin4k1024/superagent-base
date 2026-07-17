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
	"math"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// Aggregator aggregates raw signals into higher-level Gene patterns.
// It runs periodically (or on-demand) to consolidate duplicate signals
// and decay stale ones.
type Aggregator struct {
	store            *LocalGeneStore
	decayHalfLife    time.Duration // Confidence halves every this period
	minConfidence    float64       // Below this threshold, genes are pruned
	pruneAfterDays   int           // Remove genes older than this with low confidence
}

// AggregatorConfig configures the signal aggregator.
type AggregatorConfig struct {
	DecayHalfLife  time.Duration `yaml:"decay_half_life"`
	MinConfidence  float64       `yaml:"min_confidence"`
	PruneAfterDays int           `yaml:"prune_after_days"`
}

// DefaultAggregatorConfig returns sensible defaults.
func DefaultAggregatorConfig() AggregatorConfig {
	return AggregatorConfig{
		DecayHalfLife:  7 * 24 * time.Hour, // 1 week half-life
		MinConfidence:  0.1,
		PruneAfterDays: 30,
	}
}

// NewAggregator creates a new signal aggregator.
func NewAggregator(store *LocalGeneStore, cfg AggregatorConfig) *Aggregator {
	if cfg.DecayHalfLife <= 0 {
		cfg.DecayHalfLife = 7 * 24 * time.Hour
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.PruneAfterDays <= 0 {
		cfg.PruneAfterDays = 30
	}
	return &Aggregator{
		store:          store,
		decayHalfLife:  cfg.DecayHalfLife,
		minConfidence:  cfg.MinConfidence,
		pruneAfterDays: cfg.PruneAfterDays,
	}
}

// Run executes a full aggregation cycle:
// 1. Decay confidence of all genes based on age
// 2. Prune genes that have fallen below minimum confidence
// 3. Consolidate duplicate genes (same agent + component + signal type)
func (a *Aggregator) Run(ctx context.Context) error {
	if a == nil || a.store == nil {
		return nil
	}

	logs.CtxInfof(ctx, "[Evolution] Starting aggregation cycle")

	// Step 1: Decay confidence
	decayed, err := a.decayConfidence(ctx)
	if err != nil {
		logs.CtxWarnf(ctx, "[Evolution] Confidence decay failed: %v", err)
	} else {
		logs.CtxInfof(ctx, "[Evolution] Decayed confidence for %d genes", decayed)
	}

	// Step 2: Prune stale genes
	pruned, err := a.pruneStaleGenes(ctx)
	if err != nil {
		logs.CtxWarnf(ctx, "[Evolution] Pruning failed: %v", err)
	} else {
		logs.CtxInfof(ctx, "[Evolution] Pruned %d stale genes", pruned)
	}

	// Step 3: Consolidate duplicates
	consolidated, err := a.consolidateDuplicates(ctx)
	if err != nil {
		logs.CtxWarnf(ctx, "[Evolution] Consolidation failed: %v", err)
	} else {
		logs.CtxInfof(ctx, "[Evolution] Consolidated %d duplicate gene groups", consolidated)
	}

	logs.CtxInfof(ctx, "[Evolution] Aggregation cycle complete")
	return nil
}

// decayConfidence applies time-based exponential decay to all genes.
// The formula is: new_confidence = old_confidence * 0.5^(age / halfLife)
func (a *Aggregator) decayConfidence(ctx context.Context) (int, error) {
	genes, err := a.store.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	decayed := 0

	for _, gene := range genes {
		age := now.Sub(gene.UpdatedAt)
		if age < time.Hour {
			continue // Skip very recent genes
		}

		// Exponential decay: confidence * 2^(-age/halfLife)
		decayFactor := math.Pow(0.5, float64(age)/float64(a.decayHalfLife))
		newConfidence := gene.Confidence * decayFactor

		// Only update if change is significant
		if newConfidence < gene.Confidence-0.01 {
			if err := a.store.UpdateConfidence(ctx, gene.ID, newConfidence); err != nil {
				logs.CtxWarnf(ctx, "[Evolution] Failed to decay gene %s: %v", gene.ID, err)
				continue
			}
			decayed++
		}
	}

	return decayed, nil
}

// pruneStaleGenes removes genes that have fallen below minimum confidence
// and are older than the prune threshold.
func (a *Aggregator) pruneStaleGenes(ctx context.Context) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -a.pruneAfterDays)
	pruned, err := a.store.DeleteLowConfidence(ctx, a.minConfidence, cutoff)
	if err != nil {
		return 0, err
	}
	return int(pruned), nil
}

// consolidateDuplicates merges genes that share the same (agent, component, signal_type)
// into a single gene with aggregated statistics.
func (a *Aggregator) consolidateDuplicates(ctx context.Context) (int, error) {
	// This is a read-modify-write operation. In a multi-instance deployment,
	// this should be protected by a distributed lock.
	genes, err := a.store.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	// Group by (agent, component, signal_type)
	type geneKey struct {
		agent     string
		component string
		sigType   string
	}
	groups := make(map[geneKey][]*Gene)
	for _, g := range genes {
		key := geneKey{g.AgentName, g.Component, g.SignalType}
		groups[key] = append(groups[key], g)
	}

	consolidated := 0
	for key, group := range groups {
		if len(group) <= 1 {
			continue
		}

		// Keep the gene with highest confidence, merge stats from others
		best := group[0]
		for _, g := range group[1:] {
			if g.Confidence > best.Confidence {
				best = g
			}
		}

		// Aggregate stats into the best gene
		totalUse := 0
		totalSuccess := 0
		for _, g := range group {
			totalUse += g.UseCount
			totalSuccess += g.SuccessCount
		}

		best.UseCount = totalUse
		best.SuccessCount = totalSuccess
		if totalUse > 0 {
			// Recalculate confidence based on success rate and signal count
			successRate := float64(totalSuccess) / float64(totalUse)
			signalBoost := math.Min(float64(totalUse)/10.0, 1.0) // More signals = more confidence
			best.Confidence = successRate * signalBoost
		}

		// Update the best gene and delete the rest
		if err := a.store.UpdateGene(ctx, best); err != nil {
			logs.CtxWarnf(ctx, "[Evolution] Failed to update consolidated gene: %v", err)
			continue
		}

		for _, g := range group {
			if g.ID != best.ID {
				if err := a.store.DeleteGene(ctx, g.ID); err != nil {
					logs.CtxWarnf(ctx, "[Evolution] Failed to delete duplicate gene %s: %v", g.ID, err)
				}
			}
		}

		consolidated++
		_ = key // suppress unused warning
	}

	return consolidated, nil
}
