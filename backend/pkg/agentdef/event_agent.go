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

package agentdef

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/superagent-ai/superagent-base/backend/pkg/a2ui"
)

// EventAgent extends Agent with A2UI event streaming.
type EventAgent interface {
	Agent
	// ChatWithEvents streams response as structured A2UI events instead of
	// raw text tokens. The returned EventStream is closed when the response
	// is complete or an unrecoverable error occurs.
	ChatWithEvents(ctx context.Context, sessionID string, message string) (*a2ui.EventStream, error)
}

// eventAgentWrapper wraps any Agent to produce A2UI events.
type eventAgentWrapper struct {
	Agent
}

// NewEventAgent wraps inner so that ChatWithEvents is available.
func NewEventAgent(inner Agent) EventAgent {
	return &eventAgentWrapper{Agent: inner}
}

// ChatWithEvents translates the plain text token stream from the underlying
// Agent into structured A2UI events. Special interrupt signals embedded in
// the token stream are decoded; all other tokens become text delta events.
func (e *eventAgentWrapper) ChatWithEvents(ctx context.Context, sessionID string, message string) (*a2ui.EventStream, error) {
	stream := a2ui.NewEventStream(200)

	// Attach the EventStream to context so the A2UI callback can emit
	// tool_call/tool_result events during internal ReAct processing.
	ctx = a2ui.WithEventStream(ctx, stream)

	// Register the A2UI callback handler to intercept tool lifecycle events.
	ctx = callbacks.InitCallbacks(ctx, &callbacks.RunInfo{Name: "a2ui-event-agent"}, a2ui.NewA2UICallback())

	ch, err := e.Agent.Chat(ctx, sessionID, message)
	if err != nil {
		stream.SendError("chat_failed", err.Error())
		stream.Close()
		return stream, nil
	}

	go func() {
		for token := range ch {
			if isInterruptSignal(token) {
				stream.Send(a2ui.NewEvent(a2ui.EventInterrupt, parseInterruptData(token)))
				continue
			}
			// Detect AgentLoop turn headers and emit as progress events
			// so the frontend can render structured turn separators.
			if turn, total, ok := parseTurnHeader(token); ok {
				stream.Send(a2ui.NewEvent(a2ui.EventProgress, &a2ui.ProgressData{
					AgentName: e.Agent.Name(),
					Step:      fmt.Sprintf("turn_%d", turn),
					Total:     total,
					Current:   turn,
				}))
				continue
			}
			stream.SendText(token)
		}
		stream.SendDone()
	}()

	return stream, nil
}

// isInterruptSignal returns true when token carries an embedded interrupt
// directive. The convention is a NUL-prefixed marker: "\x00INTERRUPT:...".
func isInterruptSignal(token string) bool {
	return len(token) > 0 && token[0] == '\x00' && strings.HasPrefix(token, "\x00INTERRUPT:")
}

// parseInterruptData extracts interrupt metadata from an interrupt token.
// Tokens that cannot be parsed produce a generic interrupt with the raw
// payload as the reason.
func parseInterruptData(token string) *a2ui.InterruptData {
	payload := strings.TrimPrefix(token, "\x00INTERRUPT:")
	return &a2ui.InterruptData{
		Reason: payload,
		Fields: []a2ui.InterruptField{
			{
				Name:     "response",
				Type:     "text",
				Label:    "Your response",
				Required: true,
			},
		},
	}
}

// parseTurnHeader detects the AgentLoop turn header pattern emitted by
// AgentLoopAgent.Chat() and extracts turn/total numbers.
// Pattern: "\n--- Turn N/M ---\n"
func parseTurnHeader(token string) (turn, total int, ok bool) {
	s := strings.TrimSpace(token)
	if !strings.HasPrefix(s, "--- Turn ") || !strings.HasSuffix(s, "---") {
		return 0, 0, false
	}
	// Extract "N/M" from "--- Turn N/M ---"
	inner := strings.TrimPrefix(s, "--- Turn ")
	inner = strings.TrimSuffix(inner, "---")
	inner = strings.TrimSpace(inner)
	// Reject if inner contains anything beyond "N/M"
	parts := strings.SplitN(inner, "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	n, m := 0, 0
	if _, err := fmt.Sscanf(parts[0], "%d", &n); err != nil || n <= 0 {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil || m <= 0 {
		return 0, 0, false
	}
	// Exact match: reconstructed string must equal inner
	if fmt.Sprintf("%d/%d", n, m) != inner {
		return 0, 0, false
	}
	return n, m, true
}
