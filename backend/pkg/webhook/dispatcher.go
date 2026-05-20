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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultWorkers   = 4
	defaultQueueSize = 256
	maxAttempts      = 3
	deliveryTimeout  = 10 * time.Second
)

// retryDelays defines the back-off pause before each retry attempt.
// Index 0 = before attempt 2, index 1 = before attempt 3.
var retryDelays = []time.Duration{1 * time.Second, 5 * time.Second}

// deliveryTask is the unit of work sent to worker goroutines.
type deliveryTask struct {
	sub   *Subscription
	event Event
	body  []byte // pre-marshalled JSON
}

// Dispatcher routes Events to matching Subscriptions via HTTP POST.
// It uses a bounded channel + goroutine pool to decouple emission from delivery.
type Dispatcher struct {
	store      Store
	httpClient *http.Client
	queue      chan *deliveryTask
	wg         sync.WaitGroup
	workers    int
}

// NewDispatcher creates a Dispatcher with the specified worker count.
// If workers <= 0, defaultWorkers is used.
func NewDispatcher(store Store, workers int) *Dispatcher {
	if workers <= 0 {
		workers = defaultWorkers
	}
	return &Dispatcher{
		store:   store,
		workers: workers,
		httpClient: &http.Client{
			Timeout: deliveryTimeout,
		},
		queue: make(chan *deliveryTask, defaultQueueSize),
	}
}

// Start launches worker goroutines and returns immediately.
// The workers run until ctx is cancelled or Stop is called.
func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case task, ok := <-d.queue:
					if !ok {
						return
					}
					d.deliver(ctx, task)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// Stop drains the queue and waits for all in-flight deliveries to finish.
func (d *Dispatcher) Stop() {
	close(d.queue)
	d.wg.Wait()
}

// Emit fans out an event to all active subscriptions that include its EventType.
// It marshals the event once and enqueues one task per matching subscription.
// Tasks dropped when the queue is full are logged but do not block the caller.
func (d *Dispatcher) Emit(_ context.Context, event Event) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	body, err := json.Marshal(event)
	if err != nil {
		// Serialisation failure is a programming error; log and discard.
		return
	}

	subs := d.store.ListSubscriptions()
	for _, sub := range subs {
		if !sub.Active {
			continue
		}
		if !matchesAny(event.Type, sub.Events) {
			continue
		}
		task := &deliveryTask{sub: sub, event: event, body: body}
		select {
		case d.queue <- task:
		default:
			// Queue full — record a failed delivery log without blocking.
			_ = d.store.AddDeliveryLog(&DeliveryLog{
				ID:             uuid.New().String(),
				SubscriptionID: sub.ID,
				EventType:      event.Type,
				StatusCode:     0,
				Attempt:        1,
				DeliveredAt:    time.Now().UTC().Format(time.RFC3339),
				Error:          "delivery queue full; event dropped",
			})
		}
	}
}

// deliver executes up to maxAttempts HTTP POSTs with exponential back-off.
func (d *Dispatcher) deliver(ctx context.Context, task *deliveryTask) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		statusCode, deliveryErr := d.post(ctx, task.sub, task.body, task.event.Timestamp)

		logEntry := &DeliveryLog{
			ID:             uuid.New().String(),
			SubscriptionID: task.sub.ID,
			EventType:      task.event.Type,
			StatusCode:     statusCode,
			Attempt:        attempt,
			DeliveredAt:    time.Now().UTC().Format(time.RFC3339),
		}
		if deliveryErr != nil {
			logEntry.Error = deliveryErr.Error()
		}
		_ = d.store.AddDeliveryLog(logEntry)

		if deliveryErr == nil && statusCode >= 200 && statusCode < 300 {
			return // success
		}

		// Back off before retrying (but not after the last attempt).
		if attempt < maxAttempts {
			delay := retryDelays[attempt-1]
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}
	}
}

// post sends a single HTTP POST and returns (statusCode, error).
func (d *Dispatcher) post(ctx context.Context, sub *Subscription, body []byte, timestamp int64) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Superagent-Event", "webhook")
	req.Header.Set("X-Superagent-Timestamp", fmt.Sprintf("%d", timestamp))
	if sub.Secret != "" {
		req.Header.Set("X-Superagent-Signature", Sign(sub.Secret, timestamp, body))
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// Store returns the underlying Store for direct access by handlers.
func (d *Dispatcher) Store() Store {
	return d.store
}

// EmitTo delivers an event exclusively to the subscription identified by id.
// This is used by the test endpoint to trigger a targeted delivery without
// fanning out to all other subscribers.
func (d *Dispatcher) EmitTo(ctx context.Context, subscriptionID string, event Event) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	sub, err := d.store.GetSubscription(subscriptionID)
	if err != nil {
		return
	}

	body, err := json.Marshal(event)
	if err != nil {
		return
	}

	task := &deliveryTask{sub: sub, event: event, body: body}
	select {
	case d.queue <- task:
	default:
		_ = d.store.AddDeliveryLog(&DeliveryLog{
			ID:             uuid.New().String(),
			SubscriptionID: sub.ID,
			EventType:      event.Type,
			StatusCode:     0,
			Attempt:        1,
			DeliveredAt:    time.Now().UTC().Format(time.RFC3339),
			Error:          "delivery queue full; event dropped",
		})
	}
	_ = ctx // ctx already threaded into worker via Start
}

// matchesAny returns true when target is present in the allowed slice.
// An empty slice is treated as "subscribe to nothing".
func matchesAny(target EventType, allowed []EventType) bool {
	for _, e := range allowed {
		if e == target {
			return true
		}
	}
	return false
}
