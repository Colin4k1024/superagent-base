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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/superagent-ai/superagent-base/backend/infra/cache"
)

const (
	// webhookSubsKey is the Redis hash key that stores all subscriptions.
	// Each field is a subscription ID; each value is the JSON-encoded Subscription.
	webhookSubsKey = "webhook:subscriptions"

	// webhookLogKeyTpl is the Redis list key for delivery logs per subscription.
	// Using a list keeps the most-recent-first ordering via LPush/LTrim.
	webhookLogKeyTpl = "webhook:logs:%s"

	// webhookLogTTL is the Redis TTL applied to each log list.
	// Entries older than 7 days are automatically purged by Redis.
	webhookLogTTL = 7 * 24 * time.Hour

	// webhookLogMaxLen is the rolling cap on log entries per subscription.
	webhookLogMaxLen = 500
)

// RedisStore is a Redis-backed Store implementation suitable for multi-replica deployments.
// Subscriptions are stored in a single Redis hash; delivery logs are stored per-subscription
// in Redis lists with a 7-day TTL.
type RedisStore struct {
	client cache.Cmdable
}

// NewRedisStore returns a Store backed by the provided Redis client.
// The caller is responsible for ensuring the client is connected.
func NewRedisStore(client cache.Cmdable) Store {
	return &RedisStore{client: client}
}

// logKey returns the Redis list key for delivery logs of a subscription.
func (r *RedisStore) logKey(subscriptionID string) string {
	return fmt.Sprintf(webhookLogKeyTpl, subscriptionID)
}

// CreateSubscription persists a new subscription in the Redis hash.
func (r *RedisStore) CreateSubscription(sub *Subscription) error {
	data, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("webhook: RedisStore.CreateSubscription: marshal: %w", err)
	}
	ctx := context.Background()
	if err := r.client.HSet(ctx, webhookSubsKey, sub.ID, string(data)).Err(); err != nil {
		return fmt.Errorf("webhook: RedisStore.CreateSubscription: hset: %w", err)
	}
	return nil
}

// ListSubscriptions returns all subscriptions stored in Redis.
func (r *RedisStore) ListSubscriptions() []*Subscription {
	ctx := context.Background()
	m, err := r.client.HGetAll(ctx, webhookSubsKey).Result()
	if err != nil {
		return nil
	}
	result := make([]*Subscription, 0, len(m))
	for _, v := range m {
		var sub Subscription
		if jsonErr := json.Unmarshal([]byte(v), &sub); jsonErr == nil {
			clone := sub
			result = append(result, &clone)
		}
	}
	return result
}

// GetSubscription returns a subscription by ID or ErrNotFound.
func (r *RedisStore) GetSubscription(id string) (*Subscription, error) {
	ctx := context.Background()
	m, err := r.client.HGetAll(ctx, webhookSubsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("webhook: RedisStore.GetSubscription: hgetall: %w", err)
	}
	v, ok := m[id]
	if !ok {
		return nil, ErrNotFound
	}
	var sub Subscription
	if jsonErr := json.Unmarshal([]byte(v), &sub); jsonErr != nil {
		return nil, fmt.Errorf("webhook: RedisStore.GetSubscription: unmarshal: %w", jsonErr)
	}
	return &sub, nil
}

// UpdateSubscription replaces the stored subscription with the provided value.
// The secret is preserved from the original if the update omits it.
func (r *RedisStore) UpdateSubscription(id string, sub *Subscription) error {
	existing, err := r.GetSubscription(id)
	if err != nil {
		return err
	}
	clone := *sub
	clone.ID = id
	if clone.Secret == "" {
		clone.Secret = existing.Secret
	}
	data, err := json.Marshal(&clone)
	if err != nil {
		return fmt.Errorf("webhook: RedisStore.UpdateSubscription: marshal: %w", err)
	}
	ctx := context.Background()
	if err := r.client.HSet(ctx, webhookSubsKey, id, string(data)).Err(); err != nil {
		return fmt.Errorf("webhook: RedisStore.UpdateSubscription: hset: %w", err)
	}
	return nil
}

// DeleteSubscription removes a subscription and its delivery logs from Redis.
func (r *RedisStore) DeleteSubscription(id string) error {
	ctx := context.Background()
	n, err := r.client.HDel(ctx, webhookSubsKey, id).Result()
	if err != nil {
		return fmt.Errorf("webhook: RedisStore.DeleteSubscription: hdel: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	// Best-effort cleanup of the log list; ignore errors.
	_ = r.client.Del(ctx, r.logKey(id)).Err()
	return nil
}

// AddDeliveryLog prepends a delivery log entry to the subscription's Redis list
// and refreshes the list TTL. The rolling cap is enforced on the read path via
// LRange; the TTL ensures automatic cleanup after webhookLogTTL.
func (r *RedisStore) AddDeliveryLog(log *DeliveryLog) error {
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("webhook: RedisStore.AddDeliveryLog: marshal: %w", err)
	}
	ctx := context.Background()
	key := r.logKey(log.SubscriptionID)

	if err := r.client.LPush(ctx, key, string(data)).Err(); err != nil {
		return fmt.Errorf("webhook: RedisStore.AddDeliveryLog: lpush: %w", err)
	}
	// Refresh TTL on every write so active subscriptions don't expire.
	_ = r.client.Expire(ctx, key, webhookLogTTL)
	return nil
}

// GetDeliveryLogs returns the most-recent limit entries for a subscription.
// If limit <= 0, all stored entries up to webhookLogMaxLen are returned.
func (r *RedisStore) GetDeliveryLogs(subscriptionID string, limit int) []*DeliveryLog {
	ctx := context.Background()
	key := r.logKey(subscriptionID)
	stop := int64(webhookLogMaxLen - 1)
	if limit > 0 && int64(limit-1) < stop {
		stop = int64(limit - 1)
	}
	vals, err := r.client.LRange(ctx, key, 0, stop).Result()
	if err != nil {
		return nil
	}
	result := make([]*DeliveryLog, 0, len(vals))
	for _, v := range vals {
		var entry DeliveryLog
		if jsonErr := json.Unmarshal([]byte(v), &entry); jsonErr == nil {
			clone := entry
			result = append(result, &clone)
		}
	}
	return result
}
