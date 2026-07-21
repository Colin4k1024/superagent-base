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

package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// WebhookSubscriptionModel is the GORM model for persisted webhook subscriptions.
// Events and Metadata are JSON-serialised to text columns.
type WebhookSubscriptionModel struct {
	ID        string `gorm:"primaryKey;size:64"`
	URL       string `gorm:"size:2048;not null"`
	Events    string `gorm:"type:text"`  // JSON-serialised []EventType
	Secret    string `gorm:"size:128;not null"`
	Active    bool   `gorm:"default:true"`
	Metadata  string `gorm:"type:text"`  // JSON-serialised map[string]string
	CreatedAt int64  `gorm:"autoCreateTime:milli"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`
}

// TableName overrides the GORM default table name.
func (WebhookSubscriptionModel) TableName() string { return "webhook_subscriptions" }

// WebhookDeliveryLogModel is the GORM model for webhook delivery log entries.
type WebhookDeliveryLogModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	SubscriptionID string `gorm:"size:64;index;not null"`
	EventID        string `gorm:"size:64"`   // DeliveryLog.ID (UUID assigned by dispatcher)
	EventType      string `gorm:"size:64"`
	StatusCode     int
	Attempt        int
	ResponseBody   string `gorm:"type:text"` // reserved for future use
	Error          string `gorm:"type:text"`
	DeliveredAt    int64  `gorm:"autoCreateTime:milli"`
}

// TableName overrides the GORM default table name.
func (WebhookDeliveryLogModel) TableName() string { return "webhook_delivery_logs" }

// MySQLStore is a MySQL-backed Store implementation using GORM.
// Subscriptions and delivery logs survive server restarts and are visible
// across replicas that share the same database.
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore creates a MySQLStore and runs AutoMigrate to ensure the tables exist.
// Returns an error if migration fails.
func NewMySQLStore(db *gorm.DB) (Store, error) {
	if err := db.AutoMigrate(
		&WebhookSubscriptionModel{},
		&WebhookDeliveryLogModel{},
	); err != nil {
		return nil, fmt.Errorf("webhook: MySQLStore.AutoMigrate: %w", err)
	}
	return &MySQLStore{db: db}, nil
}

// toModel converts a domain Subscription to its GORM model representation.
func toModel(sub *Subscription) (*WebhookSubscriptionModel, error) {
	eventsJSON, err := json.Marshal(sub.Events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}
	metaJSON, err := json.Marshal(sub.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return &WebhookSubscriptionModel{
		ID:       sub.ID,
		URL:      sub.URL,
		Events:   string(eventsJSON),
		Secret:   sub.Secret,
		Active:   sub.Active,
		Metadata: string(metaJSON),
	}, nil
}

// fromModel converts a GORM model to a domain Subscription.
func fromModel(m *WebhookSubscriptionModel) (*Subscription, error) {
	var events []EventType
	if err := json.Unmarshal([]byte(m.Events), &events); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}
	var meta map[string]string
	if m.Metadata != "" && m.Metadata != "null" {
		if err := json.Unmarshal([]byte(m.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return &Subscription{
		ID:       m.ID,
		URL:      m.URL,
		Events:   events,
		Secret:   m.Secret,
		Active:   m.Active,
		Metadata: meta,
	}, nil
}

// CreateSubscription persists a new subscription row. The caller must set a unique ID.
func (s *MySQLStore) CreateSubscription(sub *Subscription) error {
	m, err := toModel(sub)
	if err != nil {
		return fmt.Errorf("webhook: MySQLStore.CreateSubscription: %w", err)
	}
	if err := s.db.Create(m).Error; err != nil {
		return fmt.Errorf("webhook: MySQLStore.CreateSubscription: db create: %w", err)
	}
	return nil
}

// ListSubscriptions returns all subscriptions in an unspecified order.
func (s *MySQLStore) ListSubscriptions() []*Subscription {
	var rows []WebhookSubscriptionModel
	if err := s.db.Find(&rows).Error; err != nil {
		return nil
	}
	result := make([]*Subscription, 0, len(rows))
	for i := range rows {
		sub, err := fromModel(&rows[i])
		if err == nil {
			result = append(result, sub)
		}
	}
	return result
}

// GetSubscription returns a subscription by ID or ErrNotFound.
func (s *MySQLStore) GetSubscription(id string) (*Subscription, error) {
	var m WebhookSubscriptionModel
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("webhook: MySQLStore.GetSubscription: %w", err)
	}
	sub, err := fromModel(&m)
	if err != nil {
		return nil, fmt.Errorf("webhook: MySQLStore.GetSubscription: %w", err)
	}
	return sub, nil
}

// UpdateSubscription replaces the stored subscription. The secret is preserved
// from the original if the update omits it (callers never pass the secret back).
func (s *MySQLStore) UpdateSubscription(id string, sub *Subscription) error {
	existing, err := s.GetSubscription(id)
	if err != nil {
		return err
	}
	clone := *sub
	clone.ID = id
	if clone.Secret == "" {
		clone.Secret = existing.Secret
	}
	m, err := toModel(&clone)
	if err != nil {
		return fmt.Errorf("webhook: MySQLStore.UpdateSubscription: %w", err)
	}
	if err := s.db.Save(m).Error; err != nil {
		return fmt.Errorf("webhook: MySQLStore.UpdateSubscription: db save: %w", err)
	}
	return nil
}

// DeleteSubscription removes a subscription and its delivery logs.
func (s *MySQLStore) DeleteSubscription(id string) error {
	result := s.db.Delete(&WebhookSubscriptionModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("webhook: MySQLStore.DeleteSubscription: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	// Best-effort cascade delete of delivery logs; ignore errors.
	_ = s.db.Delete(&WebhookDeliveryLogModel{}, "subscription_id = ?", id).Error
	return nil
}

// AddDeliveryLog appends a delivery log entry. A rolling cap of 500 rows per
// subscription is enforced by pruning the oldest entries after each insert.
func (s *MySQLStore) AddDeliveryLog(log *DeliveryLog) error {
	m := &WebhookDeliveryLogModel{
		SubscriptionID: log.SubscriptionID,
		EventID:        log.ID,
		EventType:      string(log.EventType),
		StatusCode:     log.StatusCode,
		Attempt:        log.Attempt,
		Error:          log.Error,
	}
	if err := s.db.Create(m).Error; err != nil {
		return fmt.Errorf("webhook: MySQLStore.AddDeliveryLog: %w", err)
	}

	// Prune oldest rows beyond the 500-entry rolling cap for this subscription.
	const maxLogs = 500
	var count int64
	s.db.Model(&WebhookDeliveryLogModel{}).
		Where("subscription_id = ?", log.SubscriptionID).
		Count(&count)
	if count > maxLogs {
		// Delete rows with the lowest IDs (oldest) that exceed the cap.
		excess := count - maxLogs
		s.db.Exec(
			"DELETE FROM webhook_delivery_logs WHERE subscription_id = ? ORDER BY id ASC LIMIT ?",
			log.SubscriptionID, excess,
		)
	}
	return nil
}

// GetDeliveryLogs returns the most-recent limit entries for a subscription.
// If limit <= 0, up to 500 entries are returned.
func (s *MySQLStore) GetDeliveryLogs(subscriptionID string, limit int) []*DeliveryLog {
	if limit <= 0 {
		limit = 500
	}
	var rows []WebhookDeliveryLogModel
	s.db.Where("subscription_id = ?", subscriptionID).
		Order("id DESC").
		Limit(limit).
		Find(&rows)

	result := make([]*DeliveryLog, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		entry := &DeliveryLog{
			ID:             r.EventID,
			SubscriptionID: r.SubscriptionID,
			EventType:      EventType(r.EventType),
			StatusCode:     r.StatusCode,
			Attempt:        r.Attempt,
			Error:          r.Error,
			DeliveredAt:    time.UnixMilli(r.DeliveredAt).UTC().Format(time.RFC3339),
		}
		result = append(result, entry)
	}
	return result
}
