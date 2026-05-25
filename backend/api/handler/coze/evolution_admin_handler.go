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
	UseCount    int     `json:"use_count"`
	SuccessRate float64 `json:"success_rate"`
	Label       string  `json:"label"`
	SignalType  string  `json:"signal_type"`
	Component   string  `json:"component"`
	CreatedAt   string  `json:"created_at"`
}

// HandleListGenes returns genes from the local store with optional search.
// GET /api/v1/admin/evolution/genes?q=<query>&min_confidence=0.5&limit=20
func (h *EvolutionAdminHandler) HandleListGenes(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{
			"enabled": false,
			"genes":   []geneItem{},
			"total":   0,
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

// HandleStats returns evolution engine status and local store statistics.
// GET /api/v1/admin/evolution/stats
func (h *EvolutionAdminHandler) HandleStats(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{
			"enabled":   false,
			"sender_id": "",
		})
		return
	}

	cfg := h.engine.Config()
	stats, _ := h.engine.Store().Stats(ctx)

	c.JSON(200, map[string]any{
		"enabled":         true,
		"mode":            "local",
		"sender_id":       cfg.SenderID,
		"min_confidence":  cfg.MinConfidence,
		"max_suggestions": cfg.MaxSuggestions,
		"total_genes":     stats.TotalGenes,
		"avg_confidence":  stats.AvgConfidence,
		"success_rate":    stats.SuccessRate,
	})
}

// HandleRecommend returns strategy recommendations for a given query context.
//
// POST /api/v2/admin/evolution/recommend
// Body: {"query": "...", "min_confidence": 0.5, "limit": 5}
func (h *EvolutionAdminHandler) HandleRecommend(ctx context.Context, c *app.RequestContext) {
	if h.engine == nil {
		c.JSON(200, map[string]any{"enabled": false, "recommendations": []any{}})
		return
	}

	var req struct {
		Query         string  `json:"query"`
		MinConfidence float64 `json:"min_confidence"`
		Limit         int     `json:"limit"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.Query == "" {
		c.JSON(400, map[string]any{"error": "query is required"})
		return
	}
	if req.MinConfidence == 0 {
		req.MinConfidence = h.engine.Config().MinConfidence
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = h.engine.Config().MaxSuggestions
	}

	recs := h.engine.Advisor().RecommendWithOpts(ctx, req.Query, req.MinConfidence, req.Limit)
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
	c.JSON(200, map[string]any{"enabled": true, "recommendations": items, "count": len(items)})
}

// HandleFederatedSearch is a no-op in local-only mode.
// GET /api/v1/admin/evolution/federated?q=<query>
func (h *EvolutionAdminHandler) HandleFederatedSearch(_ context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]any{
		"results": []any{},
		"total":   0,
		"note":    "federated search is not available in local-only mode",
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
