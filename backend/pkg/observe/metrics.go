/*
 * Copyright 2025 coze-dev Authors
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

// Evolution metrics.
var (
	// Q-4: all evolution metrics use the superagent_ namespace for consistency.
	EvolutionSignalsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "evolution_signals_total",
		Help:      "Total evolution signals collected by type",
	}, []string{"signal_type"})

	EvolutionGenesShared = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "evolution_genes_shared_total",
		Help:      "Total genes successfully shared to Experience Repo",
	})

	EvolutionShareFailed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "evolution_share_failed_total",
		Help:      "Total gene share failures",
	})

	EvolutionShareDropped = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "evolution_share_dropped_total",
		Help:      "Total signals dropped due to semaphore backpressure",
	})

	EvolutionRecommendationsServed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "evolution_recommendations_served_total",
		Help:      "Total gene recommendations served to agents",
	})
)


// Model router metrics.
var (
	// ModelRouteDecisions tracks which model was selected by the router.
	ModelRouteDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "model_route_decisions_total",
		Help:      "Number of routing decisions by strategy and selected model",
	}, []string{"strategy", "model_id"})

	// ModelRouteLatency tracks how long the routing decision itself takes.
	ModelRouteLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "superagent",
		Name:      "model_route_latency_seconds",
		Help:      "Time to make a routing decision",
		Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01},
	}, []string{"strategy"})

	// ModelResponseLatency tracks actual LLM response latency per model (time-to-first-token).
	ModelResponseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "superagent",
		Name:      "model_response_latency_seconds",
		Help:      "Actual LLM response latency per model (first token)",
		Buckets:   prometheus.DefBuckets,
	}, []string{"model_id", "provider"})

	// DynamicRouteTotal tracks per-request dynamic model routing decisions.
	DynamicRouteTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "superagent",
		Name:      "dynamic_route_total",
		Help:      "Dynamic routing decisions by agent, complexity, and selected model",
	}, []string{"agent", "complexity", "model_id"})

	// ComplexityAnalyzeDuration tracks the latency of the complexity analyzer.
	ComplexityAnalyzeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "superagent",
		Name:      "complexity_analyze_duration_seconds",
		Help:      "Time to analyze task complexity",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 3.0},
	}, []string{"result"})
)
