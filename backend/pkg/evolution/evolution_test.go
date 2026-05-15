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

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB creates an in-memory SQLite GORM DB for testing.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func TestInitDisabled(t *testing.T) {
	e, err := Init(context.Background(), Config{Enabled: false}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if e != nil {
		t.Fatal("expected nil engine when disabled")
	}
}

func TestInitNilDB(t *testing.T) {
	_, err := Init(context.Background(), Config{Enabled: true}, nil)
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}

func TestInitSuccess(t *testing.T) {
	db := testDB(t)
	e, err := Init(context.Background(), Config{
		Enabled:        true,
		SenderID:       "test-node",
		MinConfidence:  0.3,
		MaxSuggestions: 5,
	}, db)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.Store() == nil {
		t.Fatal("expected non-nil store")
	}
	if e.Collector() == nil {
		t.Fatal("expected non-nil collector")
	}
	if e.Advisor() == nil {
		t.Fatal("expected non-nil advisor")
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
	if e.Store() != nil {
		t.Fatal("expected nil store")
	}
	_ = e.Config()
	e.Shutdown() // no panic
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

func TestLocalGeneStore_SaveAndSearch(t *testing.T) {
	db := testDB(t)
	store, err := NewLocalGeneStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()

	// Save a gene.
	payload := buildSharePayload(Signal{
		Type:      "tool_success",
		AgentName: "research-agent",
		Component: "web_search",
		Output:    "found 5 results",
		Timestamp: time.Now(),
		Duration:  200 * time.Millisecond,
	})
	id, err := store.SaveGene(ctx, payload)
	if err != nil {
		t.Fatalf("save gene: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty gene ID")
	}

	// Search by component name.
	genes, err := store.Search(ctx, "web_search", 0.0, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(genes) != 1 {
		t.Fatalf("expected 1 gene, got %d", len(genes))
	}
	if genes[0].ID != id {
		t.Errorf("expected id %s, got %s", id, genes[0].ID)
	}
	if genes[0].Component != "web_search" {
		t.Errorf("expected component web_search, got %s", genes[0].Component)
	}

	// Search with high confidence threshold should return nothing (0.6 gene vs 0.7 threshold).
	genes, err = store.Search(ctx, "web_search", 0.7, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(genes) != 0 {
		t.Fatalf("expected 0 genes with high threshold, got %d", len(genes))
	}
}

func TestLocalGeneStore_Stats(t *testing.T) {
	db := testDB(t)
	store, err := NewLocalGeneStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()

	// Empty store stats.
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalGenes != 0 {
		t.Errorf("expected 0 genes, got %d", stats.TotalGenes)
	}

	// Add a gene and check stats.
	payload := buildSharePayload(Signal{
		Type:      "tool_success",
		Component: "calc",
		Timestamp: time.Now(),
	})
	_, _ = store.SaveGene(ctx, payload)

	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalGenes != 1 {
		t.Errorf("expected 1 gene, got %d", stats.TotalGenes)
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
