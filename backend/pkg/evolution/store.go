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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Gene is the GORM model for locally stored evolution genes.
type Gene struct {
	ID           string    `gorm:"primaryKey;size:64" json:"id"`
	Label        string    `gorm:"size:512;index" json:"label"`
	SignalType   string    `gorm:"size:64;index" json:"signal_type"`
	AgentName    string    `gorm:"size:128;index" json:"agent_name"`
	Component    string    `gorm:"size:128" json:"component"`
	Signals      string    `gorm:"type:text" json:"signals"`
	Strategy     string    `gorm:"type:text" json:"strategy"`
	Validation   string    `gorm:"type:text" json:"validation"`
	Confidence   float64   `gorm:"default:0.5" json:"confidence"`
	UseCount     int       `gorm:"default:0" json:"use_count"`
	SuccessCount int       `gorm:"default:0" json:"success_count"`
	SenderID     string    `gorm:"size:128" json:"sender_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName specifies the MySQL table name for Gene.
func (Gene) TableName() string {
	return "evolution_genes"
}

// StoreStats holds aggregate statistics about the local gene store.
type StoreStats struct {
	TotalGenes    int64   `json:"total_genes"`
	AvgConfidence float64 `json:"avg_confidence"`
	SuccessRate   float64 `json:"success_rate"`
}

// LocalGeneStore provides local persistence for evolution genes using MySQL via GORM.
type LocalGeneStore struct {
	db *gorm.DB
}

// NewLocalGeneStore opens the local gene store and auto-migrates the schema.
func NewLocalGeneStore(db *gorm.DB) (*LocalGeneStore, error) {
	if db == nil {
		return nil, fmt.Errorf("evolution store: gorm.DB is nil")
	}
	if err := db.AutoMigrate(&Gene{}); err != nil {
		return nil, fmt.Errorf("evolution store: auto-migrate failed: %w", err)
	}
	return &LocalGeneStore{db: db}, nil
}

// SaveGene persists an execution signal payload as a Gene record.
// Returns the generated gene ID.
func (s *LocalGeneStore) SaveGene(_ context.Context, payload map[string]any) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("store is nil")
	}

	gene := Gene{
		ID:        fmt.Sprintf("gene-%s", uuid.New().String()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract label.
	if v, ok := payload["label"].(string); ok {
		gene.Label = v
	}

	// Extract sender_id.
	if v, ok := payload["sender_id"].(string); ok {
		gene.SenderID = v
	}

	// Extract signals.
	if signals, ok := payload["signals"].(map[string]any); ok {
		if v, ok := signals["signal_type"].(string); ok {
			gene.SignalType = v
		}
		if v, ok := signals["agent_name"].(string); ok {
			gene.AgentName = v
		}
		if v, ok := signals["component"].(string); ok {
			gene.Component = v
		}
		// Determine confidence from outcome.
		if outcome, ok := signals["outcome"].(string); ok && outcome == "success" {
			gene.Confidence = 0.6
			gene.SuccessCount = 1
		} else {
			gene.Confidence = 0.3
		}
		gene.UseCount = 1
		b, _ := json.Marshal(signals)
		gene.Signals = string(b)
	}

	// Extract strategy.
	if v, ok := payload["strategy"]; ok {
		b, _ := json.Marshal(v)
		gene.Strategy = string(b)
	}

	// Extract validation.
	if v, ok := payload["validation"]; ok {
		b, _ := json.Marshal(v)
		gene.Validation = string(b)
	}

	if err := s.db.Create(&gene).Error; err != nil {
		return "", fmt.Errorf("evolution store: save gene: %w", err)
	}
	return gene.ID, nil
}

// escapeLike escapes SQL LIKE wildcard characters.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// Search queries genes matching the text query with minimum confidence.
// Uses LIKE matching on label, component, and agent_name columns.
func (s *LocalGeneStore) Search(_ context.Context, query string, minConfidence float64, limit int) ([]Gene, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	tx := s.db.Model(&Gene{})

	if query != "" {
		escaped := escapeLike(query)
		pattern := "%" + escaped + "%"
		tx = tx.Where(
			"label LIKE ? ESCAPE '\\' OR component LIKE ? ESCAPE '\\' OR agent_name LIKE ? ESCAPE '\\'",
			pattern, pattern, pattern,
		)
	}
	if minConfidence > 0 {
		tx = tx.Where("confidence >= ?", minConfidence)
	}

	var genes []Gene
	err := tx.Order("confidence DESC, updated_at DESC").Limit(limit).Find(&genes).Error
	if err != nil {
		return nil, fmt.Errorf("evolution store: search: %w", err)
	}
	return genes, nil
}

// Stats returns aggregate statistics about stored genes.
func (s *LocalGeneStore) Stats(_ context.Context) (StoreStats, error) {
	if s == nil || s.db == nil {
		return StoreStats{}, nil
	}

	var stats StoreStats
	s.db.Model(&Gene{}).Count(&stats.TotalGenes)

	row := s.db.Model(&Gene{}).Select("COALESCE(AVG(confidence), 0)").Row()
	if row != nil {
		_ = row.Scan(&stats.AvgConfidence)
	}

	var totalUse, totalSuccess int64
	row = s.db.Model(&Gene{}).Select("COALESCE(SUM(use_count), 0)").Row()
	if row != nil {
		_ = row.Scan(&totalUse)
	}
	row = s.db.Model(&Gene{}).Select("COALESCE(SUM(success_count), 0)").Row()
	if row != nil {
		_ = row.Scan(&totalSuccess)
	}
	if totalUse > 0 {
		stats.SuccessRate = float64(totalSuccess) / float64(totalUse)
	}

	return stats, nil
}

// IncrementUse bumps use_count and optionally success_count for a gene.
func (s *LocalGeneStore) IncrementUse(_ context.Context, geneID string, success bool) error {
	if s == nil || s.db == nil {
		return nil
	}
	updates := map[string]any{
		"use_count":  gorm.Expr("use_count + 1"),
		"updated_at": time.Now(),
	}
	if success {
		updates["success_count"] = gorm.Expr("success_count + 1")
	}
	return s.db.Model(&Gene{}).Where("id = ?", geneID).Updates(updates).Error
}
