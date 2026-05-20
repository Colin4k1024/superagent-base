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

import "sync"

var (
	globalDispatcher *Dispatcher
	globalOnce       sync.Once
)

// InitGlobalDispatcher initialises the package-level singleton Dispatcher.
// Subsequent calls are no-ops. Call this once during application startup.
func InitGlobalDispatcher(d *Dispatcher) {
	globalOnce.Do(func() {
		globalDispatcher = d
	})
}

// GetWebhookDispatcher returns the global Dispatcher.
// Returns nil when InitGlobalDispatcher has not been called yet.
func GetWebhookDispatcher() *Dispatcher {
	return globalDispatcher
}
