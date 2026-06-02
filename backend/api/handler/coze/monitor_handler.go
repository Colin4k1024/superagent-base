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
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// MonitorHandler provides observability dashboard endpoints.
type MonitorHandler struct {
	langfuse *observe.LangfuseClient
}

// NewMonitorHandler creates a MonitorHandler.
func NewMonitorHandler(lf *observe.LangfuseClient) *MonitorHandler {
	return &MonitorHandler{langfuse: lf}
}

// HandleOverview returns aggregated KPI metrics from Prometheus.
// GET /api/v1/admin/monitor/overview
func (h *MonitorHandler) HandleOverview(_ context.Context, c *app.RequestContext) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		c.JSON(500, map[string]string{"error": "failed to gather metrics"})
		return
	}

	lookup := make(map[string]*dto.MetricFamily, len(mfs))
	for _, mf := range mfs {
		lookup[mf.GetName()] = mf
	}

	result := map[string]any{
		"total_requests":   sumCounter(lookup["superagent_agent_requests_total"]),
		"active_sessions":  gaugeValue(lookup["superagent_active_sessions"]),
		"model_errors":     sumCounter(lookup["superagent_model_errors_total"]),
		"tool_invocations": sumCounter(lookup["superagent_tool_invocations_total"]),
		"total_tokens":     buildTokenSummary(lookup["superagent_model_tokens_total"]),
		"by_agent":         buildAgentBreakdown(lookup["superagent_agent_requests_total"], lookup["superagent_agent_request_duration_seconds"]),
		"by_model":         buildModelBreakdown(lookup["superagent_model_tokens_total"], lookup["superagent_model_errors_total"]),
		"by_tool":          buildToolBreakdown(lookup["superagent_tool_invocations_total"]),
		"route_decisions":  buildRouteDecisions(lookup["superagent_model_route_decisions_total"]),
	}

	c.JSON(200, result)
}

// HandleListTraces proxies trace listing from Langfuse.
// GET /api/v1/admin/monitor/traces
func (h *MonitorHandler) HandleListTraces(ctx context.Context, c *app.RequestContext) {
	if h.langfuse == nil {
		c.JSON(503, map[string]string{"error": "langfuse not configured"})
		return
	}

	params := observe.TraceListParams{
		Page:    queryInt(c, "page", 1),
		Limit:   queryInt(c, "limit", 20),
		OrderBy: queryStr(c, "orderBy"),
		Name:    queryStr(c, "name"),
		FromTS:  queryStr(c, "fromTimestamp"),
		ToTS:    queryStr(c, "toTimestamp"),
	}

	resp, err := h.langfuse.ListTraces(ctx, params)
	if err != nil {
		c.JSON(502, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, resp)
}

// HandleGetTrace retrieves a single trace by ID from Langfuse.
// GET /api/v1/admin/monitor/traces/:id
func (h *MonitorHandler) HandleGetTrace(ctx context.Context, c *app.RequestContext) {
	if h.langfuse == nil {
		c.JSON(503, map[string]string{"error": "langfuse not configured"})
		return
	}

	traceID := c.Param("id")
	if traceID == "" {
		c.JSON(400, map[string]string{"error": "missing trace id"})
		return
	}

	resp, err := h.langfuse.GetTrace(ctx, traceID)
	if err != nil {
		c.JSON(502, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, resp)
}

// HandleDailyMetrics retrieves daily usage metrics from Langfuse.
// GET /api/v1/admin/monitor/metrics/daily
func (h *MonitorHandler) HandleDailyMetrics(ctx context.Context, c *app.RequestContext) {
	if h.langfuse == nil {
		c.JSON(503, map[string]string{"error": "langfuse not configured"})
		return
	}

	fromDate := queryStr(c, "fromTimestamp")
	toDate := queryStr(c, "toTimestamp")

	resp, err := h.langfuse.GetDailyMetrics(ctx, fromDate, toDate)
	if err != nil {
		c.JSON(502, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, resp)
}

// HandleSessions retrieves session list from Langfuse.
// GET /api/v1/admin/monitor/sessions
func (h *MonitorHandler) HandleSessions(ctx context.Context, c *app.RequestContext) {
	if h.langfuse == nil {
		c.JSON(503, map[string]string{"error": "langfuse not configured"})
		return
	}

	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)

	resp, err := h.langfuse.ListSessions(ctx, page, limit)
	if err != nil {
		c.JSON(502, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, resp)
}

// --- helpers ---

func queryStr(c *app.RequestContext, key string) string {
	v, _ := c.GetQuery(key)
	return v
}

func queryInt(c *app.RequestContext, key string, defaultVal int) int {
	v := queryStr(c, key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func sumCounter(mf *dto.MetricFamily) float64 {
	if mf == nil {
		return 0
	}
	var total float64
	for _, m := range mf.GetMetric() {
		if c := m.GetCounter(); c != nil {
			total += c.GetValue()
		}
	}
	return total
}

func gaugeValue(mf *dto.MetricFamily) float64 {
	if mf == nil {
		return 0
	}
	for _, m := range mf.GetMetric() {
		if g := m.GetGauge(); g != nil {
			return g.GetValue()
		}
	}
	return 0
}

func buildTokenSummary(mf *dto.MetricFamily) map[string]float64 {
	result := map[string]float64{"input": 0, "output": 0}
	if mf == nil {
		return result
	}
	for _, m := range mf.GetMetric() {
		tokenType := labelValue(m, "type")
		if c := m.GetCounter(); c != nil {
			result[tokenType] += c.GetValue()
		}
	}
	return result
}

func buildAgentBreakdown(reqMF, durMF *dto.MetricFamily) []map[string]any {
	if reqMF == nil {
		return nil
	}
	agentReqs := make(map[string]float64)
	for _, m := range reqMF.GetMetric() {
		agentID := labelValue(m, "agent_id")
		if c := m.GetCounter(); c != nil {
			agentReqs[agentID] += c.GetValue()
		}
	}

	agentLatency := make(map[string]float64)
	if durMF != nil {
		for _, m := range durMF.GetMetric() {
			agentID := labelValue(m, "agent_id")
			if h := m.GetHistogram(); h != nil && h.GetSampleCount() > 0 {
				agentLatency[agentID] = (h.GetSampleSum() / float64(h.GetSampleCount())) * 1000
			}
		}
	}

	var out []map[string]any
	for id, reqs := range agentReqs {
		out = append(out, map[string]any{
			"agent_id":       id,
			"requests":       reqs,
			"avg_latency_ms": agentLatency[id],
		})
	}
	return out
}

func buildModelBreakdown(tokenMF, errorMF *dto.MetricFamily) []map[string]any {
	if tokenMF == nil {
		return nil
	}
	type modelStats struct {
		tokens float64
		errors float64
	}
	models := make(map[string]*modelStats)

	for _, m := range tokenMF.GetMetric() {
		modelID := labelValue(m, "model_id")
		if models[modelID] == nil {
			models[modelID] = &modelStats{}
		}
		if c := m.GetCounter(); c != nil {
			models[modelID].tokens += c.GetValue()
		}
	}

	if errorMF != nil {
		for _, m := range errorMF.GetMetric() {
			modelID := labelValue(m, "model_id")
			if models[modelID] == nil {
				models[modelID] = &modelStats{}
			}
			if c := m.GetCounter(); c != nil {
				models[modelID].errors += c.GetValue()
			}
		}
	}

	var out []map[string]any
	for id, s := range models {
		out = append(out, map[string]any{
			"model_id": id,
			"tokens":   s.tokens,
			"errors":   s.errors,
		})
	}
	return out
}

func buildToolBreakdown(mf *dto.MetricFamily) []map[string]any {
	if mf == nil {
		return nil
	}
	type toolStats struct {
		success float64
		error_  float64
	}
	tools := make(map[string]*toolStats)

	for _, m := range mf.GetMetric() {
		name := labelValue(m, "tool_name")
		status := labelValue(m, "status")
		if tools[name] == nil {
			tools[name] = &toolStats{}
		}
		if c := m.GetCounter(); c != nil {
			switch status {
			case "success":
				tools[name].success += c.GetValue()
			case "error":
				tools[name].error_ += c.GetValue()
			}
		}
	}

	var out []map[string]any
	for name, s := range tools {
		total := s.success + s.error_
		rate := float64(0)
		if total > 0 {
			rate = s.success / total
		}
		out = append(out, map[string]any{
			"tool_name":    name,
			"invocations":  total,
			"success_rate": rate,
		})
	}
	return out
}

func buildRouteDecisions(mf *dto.MetricFamily) []map[string]any {
	if mf == nil {
		return nil
	}
	var out []map[string]any
	for _, m := range mf.GetMetric() {
		strategy := labelValue(m, "strategy")
		modelID := labelValue(m, "model_id")
		count := float64(0)
		if c := m.GetCounter(); c != nil {
			count = c.GetValue()
		}
		out = append(out, map[string]any{
			"strategy": strategy,
			"model_id": modelID,
			"count":    count,
		})
	}
	return out
}

func labelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}
