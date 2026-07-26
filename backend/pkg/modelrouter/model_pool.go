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

package modelrouter

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ModelPoolRouter selects a model from a complexity-tiered pool per request.
type ModelPoolRouter struct {
	analyzer   ComplexityAnalyzer
	models     map[string]model.ToolCallingChatModel // level -> model
	primary    model.ToolCallingChatModel
	fallback   model.ToolCallingChatModel // nil if not configured
	mu         sync.RWMutex
}

// ModelPoolConfig holds the configuration for building a ModelPoolRouter.
type ModelPoolConfig struct {
	Analyzer ComplexityAnalyzer
	Models   map[string]model.ToolCallingChatModel // level -> model
	Primary  model.ToolCallingChatModel
	Fallback model.ToolCallingChatModel // optional
}

// NewModelPoolRouter creates a new ModelPoolRouter.
func NewModelPoolRouter(cfg ModelPoolConfig) *ModelPoolRouter {
	return &ModelPoolRouter{
		analyzer: cfg.Analyzer,
		models:   cfg.Models,
		primary:  cfg.Primary,
		fallback: cfg.Fallback,
	}
}

// SelectModel picks the best model for the given messages based on complexity analysis.
// Returns the model and the detected complexity level.
func (r *ModelPoolRouter) SelectModel(ctx context.Context, messages []*schema.Message) (model.ToolCallingChatModel, string) {
	if r.analyzer == nil || len(r.models) == 0 {
		return r.primary, ""
	}

	complexity, err := r.analyzer.Analyze(ctx, messages)
	if err != nil {
		return r.primary, ""
	}

	r.mu.RLock()
	m, ok := r.models[complexity]
	r.mu.RUnlock()

	if ok {
		return m, complexity
	}

	// No exact match: try adjacent levels.
	adjacent := findAdjacentLevel(complexity, r.models)
	if adjacent != nil {
		return adjacent, complexity
	}

	return r.primary, complexity
}

// SelectModelWithFallback picks a model and returns a fallback model if available.
func (r *ModelPoolRouter) SelectModelWithFallback(ctx context.Context, messages []*schema.Message) (primary model.ToolCallingChatModel, fb model.ToolCallingChatModel, complexity string) {
	m, c := r.SelectModel(ctx, messages)
	fb = r.fallback
	if fb == nil {
		fb = r.primary
	}
	return m, fb, c
}

// UpdateModels hot-reloads the model pool.
func (r *ModelPoolRouter) UpdateModels(models map[string]model.ToolCallingChatModel) {
	r.mu.Lock()
	r.models = models
	r.mu.Unlock()
}

// findAdjacentLevel tries to find a model for an adjacent complexity level.
func findAdjacentLevel(level string, models map[string]model.ToolCallingChatModel) model.ToolCallingChatModel {
	adjacency := map[string][]string{
		"high":   {"medium", "low"},
		"medium": {"high", "low"},
		"low":    {"medium", "high"},
	}

	for _, adj := range adjacency[level] {
		if m, ok := models[adj]; ok {
			return m
		}
	}
	return nil
}

// ValidateModelPool checks that all required levels have models configured.
func ValidateModelPool(models map[string]model.ToolCallingChatModel) error {
	validLevels := map[string]bool{"low": true, "medium": true, "high": true}
	for level := range models {
		if !validLevels[level] {
			return fmt.Errorf("modelrouter: invalid complexity level %q (valid: low, medium, high)", level)
		}
	}
	return nil
}
