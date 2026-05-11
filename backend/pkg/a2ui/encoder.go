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

package a2ui

import (
	"encoding/json"
	"fmt"
)

// EncodeSSE formats an A2UI event as a named SSE frame.
// Format: event: <type>\ndata: <json>\n\n
// Browsers' EventSource can listen by event name via addEventListener.
func EncodeSSE(evt *Event) string {
	data, _ := json.Marshal(evt)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", evt.Type, string(data))
}

// EncodeCompatible formats an event as a plain data-only SSE frame for
// backward compatibility with clients that expect the legacy protocol.
//
// Text events emit the raw delta; done events emit "[DONE]"; all other
// event types fall back to JSON so no information is lost.
func EncodeCompatible(evt *Event) string {
	switch evt.Type {
	case EventText:
		if td, ok := evt.Data.(*TextData); ok {
			return fmt.Sprintf("data: %s\n\n", td.Delta)
		}
	case EventDone:
		return "data: [DONE]\n\n"
	}
	data, _ := json.Marshal(evt)
	return fmt.Sprintf("data: %s\n\n", string(data))
}
