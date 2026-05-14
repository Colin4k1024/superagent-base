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

import "testing"

// TestMetricsRegistered verifies that all package-level metric vars are non-nil,
// i.e. Prometheus registration succeeded at init time.
func TestMetricsRegistered(t *testing.T) {
	if AgentRequestsTotal == nil {
		t.Error("AgentRequestsTotal is nil")
	}
	if AgentRequestDuration == nil {
		t.Error("AgentRequestDuration is nil")
	}
	if ModelTokensTotal == nil {
		t.Error("ModelTokensTotal is nil")
	}
	if ModelRequestDuration == nil {
		t.Error("ModelRequestDuration is nil")
	}
	if ModelErrorsTotal == nil {
		t.Error("ModelErrorsTotal is nil")
	}
	if ToolInvocationsTotal == nil {
		t.Error("ToolInvocationsTotal is nil")
	}
	if ToolInvocationDuration == nil {
		t.Error("ToolInvocationDuration is nil")
	}
	if ActiveSessions == nil {
		t.Error("ActiveSessions is nil")
	}
	if AgentRequestsByMode == nil {
		t.Error("AgentRequestsByMode is nil")
	}
	if AgentReloadFailures == nil {
		t.Error("AgentReloadFailures is nil")
	}
	if ModelRouteDecisions == nil {
		t.Error("ModelRouteDecisions is nil")
	}
	if ModelRouteLatency == nil {
		t.Error("ModelRouteLatency is nil")
	}
	if ModelResponseLatency == nil {
		t.Error("ModelResponseLatency is nil")
	}
}

// TestAgentRequestsTotal_Labels verifies the label signature (agent_id, status).
// Prometheus panics at runtime if the wrong number of labels is supplied.
func TestAgentRequestsTotal_Labels(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AgentRequestsTotal.WithLabelValues panicked: %v", r)
		}
	}()
	AgentRequestsTotal.WithLabelValues("test-agent", "success").Inc()
}

// TestModelRouteDecisions_Labels verifies the label signature (strategy, model_id).
func TestModelRouteDecisions_Labels(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ModelRouteDecisions.WithLabelValues panicked: %v", r)
		}
	}()
	ModelRouteDecisions.WithLabelValues("capability", "gpt-4o").Inc()
}

// TestModelRouteLatency_Labels verifies the label signature (strategy).
func TestModelRouteLatency_Labels(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ModelRouteLatency.WithLabelValues panicked: %v", r)
		}
	}()
	ModelRouteLatency.WithLabelValues("cost-optimized").Observe(0.001)
}

// TestModelResponseLatency_Labels verifies the label signature (model_id, provider).
func TestModelResponseLatency_Labels(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ModelResponseLatency.WithLabelValues panicked: %v", r)
		}
	}()
	ModelResponseLatency.WithLabelValues("gpt-4o", "openai").Observe(0.5)
}
