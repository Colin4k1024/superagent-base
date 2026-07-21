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

package agentdef

import (
	"fmt"

	"gorm.io/gorm"
)

// AgentDefinitionRecord 持久化 Agent 定义的元数据和 YAML 原文。
// 真相源仍是 YAML 文件；此表用于多实例共享查询和历史追溯。
type AgentDefinitionRecord struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	Name        string         `gorm:"size:128;uniqueIndex;not null"` // metadata.name
	AgentType   string         `gorm:"size:64"`                       // spec.type
	Description string         `gorm:"size:512"`
	YAMLContent string         `gorm:"type:longtext;not null"` // 完整 YAML 原文
	Status      string         `gorm:"size:32;default:active"` // active / disabled / error
	Version     int            `gorm:"default:1"`              // 每次更新 +1
	CreatorID   int64          `gorm:"default:0"`
	CreatedAt   int64          `gorm:"autoCreateTime:milli"`
	UpdatedAt   int64          `gorm:"autoUpdateTime:milli"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName returns the MySQL table name for AgentDefinitionRecord.
func (AgentDefinitionRecord) TableName() string {
	return "agent_definition"
}

// AgentDefinitionStore wraps GORM to provide persistence for agent definitions.
// A nil *AgentDefinitionStore is safe — all methods degrade gracefully when db is nil.
type AgentDefinitionStore struct {
	db *gorm.DB
}

// NewAgentDefinitionStore opens the store and auto-migrates the schema.
// Returns (nil, nil) when db is nil so callers can skip the store without special-casing.
func NewAgentDefinitionStore(db *gorm.DB) (*AgentDefinitionStore, error) {
	if db == nil {
		return nil, nil
	}
	if err := db.AutoMigrate(&AgentDefinitionRecord{}); err != nil {
		return nil, fmt.Errorf("agentdef store: auto-migrate failed: %w", err)
	}
	return &AgentDefinitionStore{db: db}, nil
}

// Save upserts an agent definition record.
// When the name already exists (including soft-deleted rows), the record is
// undeleted, its version is incremented, and all fields are overwritten.
// When the name is new, a fresh record is inserted at version 1.
func (s *AgentDefinitionStore) Save(name, agentType, description, yamlContent string) error {
	if s == nil || s.db == nil {
		return nil
	}

	// Try to find an existing record (including soft-deleted).
	var existing AgentDefinitionRecord
	err := s.db.Unscoped().Where("name = ?", name).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// Insert new record.
		rec := AgentDefinitionRecord{
			Name:        name,
			AgentType:   agentType,
			Description: description,
			YAMLContent: yamlContent,
			Status:      "active",
			Version:     1,
		}
		return s.db.Create(&rec).Error
	}
	if err != nil {
		return fmt.Errorf("agentdef store: lookup %q: %w", name, err)
	}

	// Update existing record (restore soft-deleted if necessary).
	updates := map[string]interface{}{
		"agent_type":   agentType,
		"description":  description,
		"yaml_content": yamlContent,
		"status":       "active",
		"version":      existing.Version + 1,
		"deleted_at":   nil, // undelete
	}
	return s.db.Unscoped().Model(&existing).Updates(updates).Error
}

// Delete soft-deletes an agent definition record by name.
func (s *AgentDefinitionStore) Delete(name string) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Where("name = ?", name).Delete(&AgentDefinitionRecord{}).Error
}

// List returns all non-deleted agent definition records.
func (s *AgentDefinitionStore) List() ([]AgentDefinitionRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var records []AgentDefinitionRecord
	if err := s.db.Order("name ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("agentdef store: list: %w", err)
	}
	return records, nil
}

// GetByName returns a single non-deleted record by agent name.
// Returns (nil, nil) when not found so callers can treat missing as a no-op.
func (s *AgentDefinitionStore) GetByName(name string) (*AgentDefinitionRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rec AgentDefinitionRecord
	err := s.db.Where("name = ?", name).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agentdef store: get %q: %w", name, err)
	}
	return &rec, nil
}
