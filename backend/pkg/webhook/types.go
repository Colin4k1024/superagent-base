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

// EventType identifies the category of a webhook event.
type EventType string

const (
	EventAgentRunCompleted   EventType = "agent.run.completed"
	EventAgentRunFailed      EventType = "agent.run.failed"
	EventAgentRunInterrupted EventType = "agent.run.interrupted"
	EventToolCallCompleted   EventType = "tool.call.completed"
	EventToolCallFailed      EventType = "tool.call.failed"
	EventAgentCreated        EventType = "agent.created"
	EventAgentUpdated        EventType = "agent.updated"
	EventAgentDeleted        EventType = "agent.deleted"
	EventAgentReloaded       EventType = "agent.reloaded"
)

// Event is the payload delivered to webhook subscribers.
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp int64     `json:"timestamp"`
	Data      any       `json:"data"`
}

// Subscription represents a registered webhook endpoint.
// The Secret field is only populated when the subscription is first created.
type Subscription struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Events   []EventType       `json:"events"`
	Secret   string            `json:"secret,omitempty"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DeliveryLog records a single delivery attempt for an event.
type DeliveryLog struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	EventType      EventType `json:"event_type"`
	StatusCode     int       `json:"status_code"`
	Attempt        int       `json:"attempt"`
	DeliveredAt    string    `json:"delivered_at"`
	Error          string    `json:"error,omitempty"`
}
