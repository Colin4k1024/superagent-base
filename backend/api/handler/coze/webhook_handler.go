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

package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"

	"github.com/superagent-ai/superagent-base/backend/pkg/webhook"
)

// WebhookHandler provides CRUD + test/log endpoints for webhook subscriptions.
// Authentication is enforced by middleware.APIKeyAdminAuthMW at the route group level.
type WebhookHandler struct {
	dispatcher *webhook.Dispatcher
}

// NewWebhookHandler creates a WebhookHandler backed by the given Dispatcher.
func NewWebhookHandler(d *webhook.Dispatcher) *WebhookHandler {
	return &WebhookHandler{dispatcher: d}
}

// createSubscriptionRequest is the body for POST /api/v1/admin/webhooks.
type createSubscriptionRequest struct {
	URL      string            `json:"url"`
	Events   []webhook.EventType `json:"events"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// updateSubscriptionRequest is the body for PUT /api/v1/admin/webhooks/:id.
type updateSubscriptionRequest struct {
	URL      string            `json:"url,omitempty"`
	Events   []webhook.EventType `json:"events,omitempty"`
	Active   *bool             `json:"active,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HandleCreate registers a new webhook subscription.
// POST /api/v1/admin/webhooks
func (h *WebhookHandler) HandleCreate(_ context.Context, c *app.RequestContext) {
	var req createSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	if req.URL == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "url is required"})
		return
	}
	if len(req.Events) == 0 {
		c.JSON(400, map[string]any{"code": 400, "msg": "events must not be empty"})
		return
	}

	secret := uuid.New().String() // raw secret — only returned once
	sub := &webhook.Subscription{
		ID:       uuid.New().String(),
		URL:      req.URL,
		Events:   req.Events,
		Secret:   secret,
		Active:   true,
		Metadata: req.Metadata,
	}

	if err := h.dispatcher.Store().CreateSubscription(sub); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("create subscription failed: %v", err)})
		return
	}

	// Return the plain-text secret only on creation; subsequent GETs omit it.
	c.JSON(201, sub)
}

// HandleList returns all webhook subscriptions.
// GET /api/v1/admin/webhooks
func (h *WebhookHandler) HandleList(_ context.Context, c *app.RequestContext) {
	subs := h.dispatcher.Store().ListSubscriptions()
	// Strip secrets from list response.
	for _, s := range subs {
		s.Secret = ""
	}
	c.JSON(200, map[string]any{"webhooks": subs, "total": len(subs)})
}

// HandleGet returns a single webhook subscription by ID.
// GET /api/v1/admin/webhooks/:id
func (h *WebhookHandler) HandleGet(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	sub, err := h.dispatcher.Store().GetSubscription(id)
	if err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("subscription %q not found", id)})
		return
	}
	sub.Secret = "" // never expose the secret after creation
	c.JSON(200, sub)
}

// HandleUpdate modifies an existing webhook subscription.
// PUT /api/v1/admin/webhooks/:id
func (h *WebhookHandler) HandleUpdate(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	existing, err := h.dispatcher.Store().GetSubscription(id)
	if err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("subscription %q not found", id)})
		return
	}

	var req updateSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}

	if req.URL != "" {
		existing.URL = req.URL
	}
	if len(req.Events) > 0 {
		existing.Events = req.Events
	}
	if req.Active != nil {
		existing.Active = *req.Active
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}

	if err := h.dispatcher.Store().UpdateSubscription(id, existing); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("update failed: %v", err)})
		return
	}

	existing.Secret = ""
	c.JSON(200, existing)
}

// HandleDelete removes a webhook subscription.
// DELETE /api/v1/admin/webhooks/:id
func (h *WebhookHandler) HandleDelete(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if err := h.dispatcher.Store().DeleteSubscription(id); err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("subscription %q not found", id)})
		return
	}
	c.JSON(200, map[string]any{"id": id, "message": "deleted"})
}

// HandleTest fires a synthetic test event to a subscription immediately.
// POST /api/v1/admin/webhooks/:id/test
func (h *WebhookHandler) HandleTest(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if _, err := h.dispatcher.Store().GetSubscription(id); err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("subscription %q not found", id)})
		return
	}

	testEvent := webhook.Event{
		ID:        uuid.New().String(),
		Type:      "webhook.test",
		Timestamp: time.Now().UnixMilli(),
		Data:      map[string]string{"subscription_id": id, "message": "test delivery"},
	}

	// Temporarily emit directly against the specific subscription via a
	// synthetic one-subscription store to avoid fan-out to other subscribers.
	h.dispatcher.EmitTo(ctx, id, testEvent)

	c.JSON(200, map[string]any{"message": "test event queued", "event_id": testEvent.ID})
}

// HandleLogs returns delivery logs for a subscription.
// GET /api/v1/admin/webhooks/:id/logs
func (h *WebhookHandler) HandleLogs(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if _, err := h.dispatcher.Store().GetSubscription(id); err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("subscription %q not found", id)})
		return
	}

	logs := h.dispatcher.Store().GetDeliveryLogs(id, 100)
	c.JSON(200, map[string]any{"logs": logs, "total": len(logs)})
}
