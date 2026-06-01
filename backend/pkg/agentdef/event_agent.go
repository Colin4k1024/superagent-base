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
	// Attach the EventStream to context so the A2UI callback can emit
	// tool_call/tool_result events during internal ReAct processing.
	stream := a2ui.NewEventStream(200)
	ctx = a2ui.WithEventStream(ctx, stream)

	// Register the A2UI callback handler to intercept tool lifecycle events.
	ctx = callbacks.InitCallbacks(ctx, &callbacks.RunInfo{Name: "a2ui-event-agent"}, a2ui.NewA2UICallback())

	ch, err := e.Agent.Chat(ctx, sessionID, message)
	return TokenStreamToEventStream(ctx, ch, err, stream), nil
}

// ContextWithA2UIStream attaches stream and the A2UI callback to ctx.
func ContextWithA2UIStream(ctx context.Context, stream *a2ui.EventStream) context.Context {
	ctx = a2ui.WithEventStream(ctx, stream)
	return callbacks.InitCallbacks(ctx, &callbacks.RunInfo{Name: "a2ui-event-agent"}, a2ui.NewA2UICallback())
}

// TokenStreamToEventStream translates a plain token channel into A2UI events.
func TokenStreamToEventStream(ctx context.Context, ch <-chan string, err error, stream *a2ui.EventStream) *a2ui.EventStream {
	if err != nil {
		stream.SendError("chat_failed", err.Error())
		stream.Close()
		return stream
	}

	go func() {
		for token := range ch {
			if isInterruptSignal(token) {
				stream.Send(a2ui.NewEvent(a2ui.EventInterrupt, parseInterruptData(token)))
				continue
			}
			stream.SendText(token)
		}
		stream.SendDone()
	}()

	return stream
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
