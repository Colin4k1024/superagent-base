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

package observe

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Agent metrics.
var (
	// AgentRequestsTotal counts agent requests by agent_id and final status (success|error).
	AgentRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "superagent_agent_requests_total",
		Help: "Total number of agent requests",
	}, []string{"agent_id", "status"})

	// AgentRequestDuration tracks end-to-end agent request latency in seconds.
	AgentRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "superagent_agent_request_duration_seconds",
		Help:    "Agent request duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	}, []string{"agent_id"})
)

// Model metrics.
var (
	// ModelTokensTotal counts tokens consumed, labeled by model, provider, and type (input|output).
	ModelTokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "superagent_model_tokens_total",
		Help: "Total tokens consumed",
	}, []string{"model_id", "provider", "type"})

	// ModelRequestDuration tracks model API call latency in seconds.
	ModelRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "superagent_model_request_duration_seconds",
		Help:    "Model API request duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.05, 2, 12),
	}, []string{"model_id", "provider"})

	// ModelErrorsTotal counts model API errors by model, provider, and error_type.
	ModelErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "superagent_model_errors_total",
		Help: "Total model API errors",
	}, []string{"model_id", "provider", "error_type"})
)

// Tool metrics.
var (
	// ToolInvocationsTotal counts tool invocations by tool_name and status (success|error).
	ToolInvocationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "superagent_tool_invocations_total",
		Help: "Total tool invocations",
	}, []string{"tool_name", "status"})

	// ToolInvocationDuration tracks tool invocation latency in seconds.
	ToolInvocationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "superagent_tool_invocation_duration_seconds",
		Help:    "Tool invocation duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
	}, []string{"tool_name"})
)

// Session metrics.
var (
	// ActiveSessions tracks the number of in-flight conversation sessions.
	ActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "superagent_active_sessions",
		Help: "Number of active conversation sessions",
	})
)

// AgentRequestsByMode counts requests by agent and protocol mode (legacy/a2ui).
var AgentRequestsByMode = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "superagent_agent_requests_by_mode_total",
	Help: "Total agent requests by protocol mode",
}, []string{"agent_id", "mode"})

// Runtime metrics.
var (
	// AgentReloadFailures counts agent hot-reload failures by agent name.
	AgentReloadFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "superagent_agent_reload_failures_total",
		Help: "Total agent hot-reload build failures",
	}, []string{"agent_id"})
)
