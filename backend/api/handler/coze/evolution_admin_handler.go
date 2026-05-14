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

package coze

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/evolution"
)

// EvolutionAdminHandler exposes evolution management endpoints.
// Authentication is enforced at the route group level.
type EvolutionAdminHandler struct {
	engine *evolution.Engine
}

// NewEvolutionAdminHandler creates an EvolutionAdminHandler.
// engine may be nil when EVOLUTION_ENABLED=false; all handlers are safe.
func NewEvolutionAdminHandler(engine *evolution.Engine) *EvolutionAdminHandler {
	return &EvolutionAdminHandler{engine: engine}
}

// geneItem is the wire format for a single Gene in the list response.
type geneItem struct {
	ID          string  `json:"id"`
	Strategy    any     `json:"strategy"`
	Confidence  float64 `json:"confidence"`
	QualityScore float64 `json:"quality_score"`
	UseCount    int     `json:"use_count"`
	SuccessCount int    `json:"success_count"`
	SuccessRate float64 `json:"success_rate"`
	ContributorID string `json:"contributor_id"`
	CreatedAt   string  `json:"created_at"`
}

// HandleListGenes returns genes from the Experience Repo with optional search.
// GET /api/v1/admin/evolution/genes?q=<query>&min_confidence=0.5&limit=20
func (h *EvolutionAdminHandler) HandleListGenes(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{
			"enabled": false,
			"genes":   []geneItem{},
		})
		return
	}

	query := string(c.Query("q"))
	minConf := clampFloat(parseQueryFloat(string(c.Query("min_confidence")), 0.0), 0.0, 1.0)
	limit := clampInt(parseQueryInt(string(c.Query("limit")), 20), 1, 100)

	recs := h.engine.Advisor().RecommendWithOpts(ctx, query, minConf, limit)
	items := make([]geneItem, 0, len(recs))
	for _, r := range recs {
		items = append(items, geneItem{
			ID:          r.GeneID,
			Strategy:    r.Strategy,
			Confidence:  r.Confidence,
			UseCount:    r.UseCount,
			SuccessRate: r.SuccessRate,
		})
	}

	c.JSON(200, map[string]any{
		"enabled": true,
		"genes":   items,
		"total":   len(items),
	})
}

// HandleStats returns evolution engine status and Prometheus-mirrored counters.
// GET /api/v1/admin/evolution/stats
func (h *EvolutionAdminHandler) HandleStats(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{
			"enabled":      false,
			"experience_url": "",
			"hub_url":      "",
			"sender_id":    "",
		})
		return
	}

	cfg := h.engine.Config()

	// Peer node discovery (requires Hub; returns empty when not configured).
	nodes, _ := h.engine.DiscoverNodes(ctx)
	peerCount := len(nodes)

	c.JSON(200, map[string]any{
		"enabled":        true,
		"experience_url": cfg.ExperienceURL,
		"hub_url":        cfg.HubURL,
		"sender_id":      cfg.SenderID,
		"peer_nodes":     peerCount,
		"min_confidence": cfg.MinConfidence,
		"max_suggestions": cfg.MaxSuggestions,
	})
}

// HandleFederatedSearch queries genes across all Hub-connected nodes.
// GET /api/v1/admin/evolution/federated?q=<query>&min_confidence=0.5&limit=10
func (h *EvolutionAdminHandler) HandleFederatedSearch(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{"enabled": false, "results": []any{}})
		return
	}

	query := string(c.Query("q"))
	if query == "" {
		c.JSON(400, map[string]any{"error": "query parameter 'q' is required"})
		return
	}
	minConf := parseQueryFloat(string(c.Query("min_confidence")), 0.5)
	limit := parseQueryInt(string(c.Query("limit")), 10)

	minConf = clampFloat(minConf, 0.0, 1.0)
	limit = clampInt(limit, 1, 100)

	results, err := h.engine.FederatedSearch(ctx, query, minConf, limit)
	if err != nil {
		c.JSON(502, map[string]any{"error": "federated search unavailable"})
		return
	}
	c.JSON(200, map[string]any{
		"results": results,
		"total":   len(results),
	})
}

func parseQueryFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func parseQueryInt(s string, def int) int {
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
