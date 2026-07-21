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

package mcp

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// MCPServerConfig is the GORM model that persists MCP server connection
// configurations across restarts.
type MCPServerConfig struct {
	ID        int64          `gorm:"primaryKey;autoIncrement"`
	Name      string         `gorm:"size:128;uniqueIndex;not null"` // server name, unique
	Transport string         `gorm:"size:32;not null"`              // "stdio" or "sse"
	Command   string         `gorm:"size:1024"`                     // stdio command path
	Args      string         `gorm:"type:text"`                     // JSON-encoded []string
	URL       string         `gorm:"size:2048"`                     // SSE endpoint URL
	Env       string         `gorm:"type:text"`                     // JSON-encoded map[string]string
	Enabled   bool           `gorm:"default:true"`                  // auto-reconnect on startup
	CreatedAt int64          `gorm:"autoCreateTime:milli"`
	UpdatedAt int64          `gorm:"autoUpdateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the MySQL table name.
func (MCPServerConfig) TableName() string {
	return "mcp_server_config"
}

// ConfigStore provides GORM-backed persistence for MCP server configurations.
type ConfigStore struct {
	db *gorm.DB
}

// NewConfigStore opens the config store and auto-migrates the schema.
// Returns an error if db is nil or migration fails.
func NewConfigStore(db *gorm.DB) (*ConfigStore, error) {
	if db == nil {
		return nil, fmt.Errorf("mcp config store: gorm.DB is nil")
	}
	if err := db.AutoMigrate(&MCPServerConfig{}); err != nil {
		return nil, fmt.Errorf("mcp config store: auto-migrate failed: %w", err)
	}
	return &ConfigStore{db: db}, nil
}

// Save upserts a ServerConfig into the database.
// If a record with the same name already exists it is updated; otherwise a new
// row is inserted.
func (s *ConfigStore) Save(cfg ServerConfig) error {
	argsJSON, err := json.Marshal(cfg.Args)
	if err != nil {
		return fmt.Errorf("mcp config store: marshal args: %w", err)
	}
	envJSON, err := json.Marshal(cfg.Env)
	if err != nil {
		return fmt.Errorf("mcp config store: marshal env: %w", err)
	}

	row := MCPServerConfig{
		Name:      cfg.Name,
		Transport: cfg.Transport,
		Command:   cfg.Command,
		Args:      string(argsJSON),
		URL:       cfg.URL,
		Env:       string(envJSON),
		Enabled:   true,
	}

	// Use Save with a WHERE clause so that soft-deleted rows are also restored.
	result := s.db.Where(MCPServerConfig{Name: cfg.Name}).
		Assign(row).
		FirstOrCreate(&row)
	if result.Error != nil {
		return fmt.Errorf("mcp config store: save %q: %w", cfg.Name, result.Error)
	}

	// If the row already existed, update mutable fields.
	if result.RowsAffected == 0 {
		updates := map[string]any{
			"transport":  cfg.Transport,
			"command":    cfg.Command,
			"args":       string(argsJSON),
			"url":        cfg.URL,
			"env":        string(envJSON),
			"enabled":    true,
			"deleted_at": nil,
		}
		if err := s.db.Model(&MCPServerConfig{}).
			Where("name = ?", cfg.Name).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("mcp config store: update %q: %w", cfg.Name, err)
		}
	}
	return nil
}

// Delete soft-deletes the config for the named server.
func (s *ConfigStore) Delete(name string) error {
	if err := s.db.Where("name = ?", name).Delete(&MCPServerConfig{}).Error; err != nil {
		return fmt.Errorf("mcp config store: delete %q: %w", name, err)
	}
	return nil
}

// ListEnabled returns all ServerConfig entries that are not deleted and have
// Enabled = true. Used at startup to restore connections.
func (s *ConfigStore) ListEnabled() ([]ServerConfig, error) {
	var rows []MCPServerConfig
	if err := s.db.Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("mcp config store: list enabled: %w", err)
	}

	cfgs := make([]ServerConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := rowToServerConfig(row)
		if err != nil {
			// Skip malformed rows but continue loading others.
			continue
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

// rowToServerConfig converts a DB row back to a ServerConfig, deserialising
// the JSON-encoded Args and Env fields.
func rowToServerConfig(row MCPServerConfig) (ServerConfig, error) {
	cfg := ServerConfig{
		Name:      row.Name,
		Transport: row.Transport,
		Command:   row.Command,
		URL:       row.URL,
	}

	if row.Args != "" {
		if err := json.Unmarshal([]byte(row.Args), &cfg.Args); err != nil {
			return cfg, fmt.Errorf("unmarshal args for %q: %w", row.Name, err)
		}
	}
	if row.Env != "" {
		if err := json.Unmarshal([]byte(row.Env), &cfg.Env); err != nil {
			return cfg, fmt.Errorf("unmarshal env for %q: %w", row.Name, err)
		}
	}
	return cfg, nil
}
