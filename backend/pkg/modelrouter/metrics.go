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

package modelrouter

import (
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// RecordModelLatency records the actual response latency (time-to-first-token)
// for a model. Call this from the chat model after receiving the first token.
func RecordModelLatency(modelID, provider string, latency time.Duration) {
	observe.ModelResponseLatency.WithLabelValues(modelID, provider).Observe(latency.Seconds())
}
