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
	"errors"
	"sync"
)

// Store defines the persistence interface for subscriptions and delivery logs.
// The MVP uses an in-memory implementation; replace with a MySQL-backed store
// later without changing callers.
type Store interface {
	CreateSubscription(sub *Subscription) error
	ListSubscriptions() []*Subscription
	GetSubscription(id string) (*Subscription, error)
	UpdateSubscription(id string, sub *Subscription) error
	DeleteSubscription(id string) error
	AddDeliveryLog(log *DeliveryLog) error
	GetDeliveryLogs(subscriptionID string, limit int) []*DeliveryLog
}

// ErrNotFound is returned when a subscription cannot be located.
var ErrNotFound = errors.New("webhook: subscription not found")

// MemoryStore is a thread-safe in-memory Store implementation.
type MemoryStore struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	logs          map[string][]*DeliveryLog // keyed by subscription ID
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		subscriptions: make(map[string]*Subscription),
		logs:          make(map[string][]*DeliveryLog),
	}
}

// CreateSubscription persists a new subscription. The caller is responsible for
// setting a unique ID before calling this method.
func (s *MemoryStore) CreateSubscription(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Shallow-copy to avoid external mutation after store.
	clone := *sub
	s.subscriptions[sub.ID] = &clone
	return nil
}

// ListSubscriptions returns all subscriptions in an unspecified order.
func (s *MemoryStore) ListSubscriptions() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		clone := *sub
		result = append(result, &clone)
	}
	return result
}

// GetSubscription returns a subscription by ID or ErrNotFound.
func (s *MemoryStore) GetSubscription(id string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subscriptions[id]
	if !ok {
		return nil, ErrNotFound
	}
	clone := *sub
	return &clone, nil
}

// UpdateSubscription replaces the stored subscription with the provided value.
// The secret is preserved from the original if the update omits it.
func (s *MemoryStore) UpdateSubscription(id string, sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.subscriptions[id]
	if !ok {
		return ErrNotFound
	}
	clone := *sub
	clone.ID = id
	// Preserve the hashed secret — callers never pass it back after creation.
	if clone.Secret == "" {
		clone.Secret = existing.Secret
	}
	s.subscriptions[id] = &clone
	return nil
}

// DeleteSubscription removes a subscription by ID.
func (s *MemoryStore) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subscriptions[id]; !ok {
		return ErrNotFound
	}
	delete(s.subscriptions, id)
	delete(s.logs, id)
	return nil
}

// AddDeliveryLog appends a delivery log entry for the given subscription.
// A rolling cap of 500 entries per subscription is applied.
func (s *MemoryStore) AddDeliveryLog(log *DeliveryLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	const maxLogs = 500
	entries := s.logs[log.SubscriptionID]
	entries = append(entries, log)
	if len(entries) > maxLogs {
		entries = entries[len(entries)-maxLogs:]
	}
	s.logs[log.SubscriptionID] = entries
	return nil
}

// GetDeliveryLogs returns the most-recent limit entries for a subscription.
// If limit <= 0 or exceeds the stored count, all stored entries are returned.
func (s *MemoryStore) GetDeliveryLogs(subscriptionID string, limit int) []*DeliveryLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.logs[subscriptionID]
	if limit <= 0 || limit >= len(entries) {
		result := make([]*DeliveryLog, len(entries))
		copy(result, entries)
		return result
	}
	result := make([]*DeliveryLog, limit)
	copy(result, entries[len(entries)-limit:])
	return result
}
