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
	"testing"
	"time"
)

func TestInitDisabled(t *testing.T) {
	e, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if e != nil {
		t.Fatal("expected nil engine when disabled")
	}
}

func TestInitMissingURL(t *testing.T) {
	_, err := Init(context.Background(), Config{Enabled: true, ExperienceURL: ""})
	if err == nil {
		t.Fatal("expected error when ExperienceURL is empty")
	}
}

func TestNilEngineSafe(t *testing.T) {
	var e *Engine
	// All accessors must be safe on nil receiver.
	if e.Collector() != nil {
		t.Fatal("expected nil collector")
	}
	if e.Advisor() != nil {
		t.Fatal("expected nil advisor")
	}
	_ = e.Config()
}

func TestNilCollectorSafe(t *testing.T) {
	var c *SignalCollector
	// Must not panic.
	c.Collect(context.Background(), Signal{
		Type:      "tool_success",
		Component: "web_search",
		Timestamp: time.Now(),
	})
}

func TestNilAdvisorSafe(t *testing.T) {
	var a *EvolutionAdvisor
	recs := a.Recommend(context.Background(), "search query")
	if recs != nil {
		t.Fatal("expected nil recommendations from nil advisor")
	}
}

func TestBuildSharePayload(t *testing.T) {
	sig := Signal{
		Type:      "tool_success",
		Component: "web_search",
		Output:    "result summary",
		Timestamp: time.Now(),
		Duration:  150 * time.Millisecond,
	}
	p := buildSharePayload(sig)
	if p["message_type"] != "gene_contribution" {
		t.Errorf("unexpected message_type: %v", p["message_type"])
	}
	signals := p["signals"].(map[string]any)
	if signals["signal_type"] != "tool_success" {
		t.Errorf("unexpected signal_type: %v", signals["signal_type"])
	}
	if signals["outcome"] != "success" {
		t.Errorf("unexpected outcome: %v", signals["outcome"])
	}
}

func TestBuildSharePayloadError(t *testing.T) {
	sig := Signal{
		Type:      "tool_error",
		Component: "web_search",
		Error:     "connection refused",
		Timestamp: time.Now(),
	}
	p := buildSharePayload(sig)
	signals := p["signals"].(map[string]any)
	if signals["outcome"] != "failure" {
		t.Errorf("expected failure outcome, got: %v", signals["outcome"])
	}
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	cfg := LoadConfigFromEnv()
	if cfg.Enabled {
		t.Fatal("expected Enabled=false by default")
	}
	if cfg.SenderID != "superagent-node-1" {
		t.Errorf("unexpected default SenderID: %s", cfg.SenderID)
	}
	if cfg.MinConfidence != 0.5 {
		t.Errorf("unexpected default MinConfidence: %f", cfg.MinConfidence)
	}
	if cfg.MaxSuggestions != 3 {
		t.Errorf("unexpected default MaxSuggestions: %d", cfg.MaxSuggestions)
	}
}
