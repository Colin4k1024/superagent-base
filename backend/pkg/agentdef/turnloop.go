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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
)

const (
	turnLoopPreemptTimeout = 10 * time.Second
	turnLoopIdleTimeout    = 5 * time.Minute
)

// TurnLoopManager owns per-agent-session Eino ADK TurnLoops. It is intentionally
// scoped above Agent implementations so existing non-ADK agents can keep using
// Agent.Chat while ADK-backed chat agents gain push/preempt/abort semantics.
type TurnLoopManager struct {
	mu          sync.Mutex
	sessions    map[string]*turnLoopSession
	idleTimeout time.Duration
}

// NewTurnLoopManager creates a manager for HTTP chat sessions.
func NewTurnLoopManager() *TurnLoopManager {
	return newTurnLoopManager(turnLoopIdleTimeout)
}

func newTurnLoopManager(idleTimeout time.Duration) *TurnLoopManager {
	return &TurnLoopManager{
		sessions:    make(map[string]*turnLoopSession),
		idleTimeout: idleTimeout,
	}
}

// Chat pushes a message into the session TurnLoop when agent is ADK-backed.
// The boolean return value tells callers whether TurnLoop handled the request.
func (m *TurnLoopManager) Chat(ctx context.Context, agentID string, agent Agent, sessionID, message string) (<-chan string, bool, error) {
	adkAgent := unwrapToADKChatModel(agent)
	if adkAgent == nil {
		return nil, false, nil
	}
	if sessionID == "" {
		sessionID = "default"
	}
	if agentID == "" {
		agentID = adkAgent.Name()
	}

	item := turnLoopItem{
		sessionID: sessionID,
		message:   message,
		ctx:       context.WithoutCancel(ctx),
		out:       make(chan string, 64),
	}
	key := turnLoopKey(agentID, sessionID)

	for attempt := 0; attempt < 2; attempt++ {
		session := m.getOrCreateSession(key, adkAgent)
		ok, _ := session.loop.Push(
			item,
			adk.WithPreemptTimeout[turnLoopItem, *schema.Message](adk.AfterToolCalls, turnLoopPreemptTimeout),
		)
		if ok {
			session.loop.Stop(adk.UntilIdleFor(m.idleTimeout))
			return item.out, true, nil
		}

		m.removeSessionIf(key, session)
	}

	close(item.out)
	return nil, true, fmt.Errorf("agentdef: turnloop session %s is stopped", key)
}

// Abort stops an active TurnLoop immediately.
func (m *TurnLoopManager) Abort(agentID, sessionID string) bool {
	key := turnLoopKey(agentID, sessionID)

	m.mu.Lock()
	session, ok := m.sessions[key]
	if ok {
		delete(m.sessions, key)
	}
	m.mu.Unlock()

	if !ok {
		return false
	}

	session.stopImmediate("abort")
	return true
}

func (m *TurnLoopManager) getOrCreateSession(key string, agent *adkChatModelAgent) *turnLoopSession {
	var stale *turnLoopSession

	m.mu.Lock()
	session := m.sessions[key]
	if session != nil && session.agent != agent {
		stale = session
		delete(m.sessions, key)
		session = nil
	}
	if session == nil {
		session = newTurnLoopSession(key, agent, m)
		m.sessions[key] = session
	}
	m.mu.Unlock()

	if stale != nil {
		stale.stopImmediate("agent_reloaded")
	}

	return session
}

func (m *TurnLoopManager) removeSessionIf(key string, session *turnLoopSession) {
	m.mu.Lock()
	if m.sessions[key] == session {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
}

func turnLoopKey(agentID, sessionID string) string {
	return agentID + "\x00" + sessionID
}

type turnLoopItem struct {
	sessionID string
	message   string
	ctx       context.Context
	out       chan string
}

type turnLoopSession struct {
	key    string
	agent  *adkChatModelAgent
	loop   *adk.TurnLoop[turnLoopItem, *schema.Message]
	cancel context.CancelFunc
	once   sync.Once
}

func newTurnLoopSession(key string, agent *adkChatModelAgent, manager *TurnLoopManager) *turnLoopSession {
	ctx, cancel := context.WithCancel(context.Background())
	session := &turnLoopSession{
		key:    key,
		agent:  agent,
		cancel: cancel,
	}
	session.loop = adk.NewTurnLoop[turnLoopItem, *schema.Message](adk.TurnLoopConfig[turnLoopItem, *schema.Message]{
		GenInput:      session.genInput,
		PrepareAgent:  session.prepareAgent,
		OnAgentEvents: session.onAgentEvents,
	})
	session.loop.Run(ctx)

	go func() {
		state := session.loop.Wait()
		cancel()
		closeTurnLoopItems(state.UnhandledItems)
		if state.TakeLateItems != nil {
			closeTurnLoopItems(state.TakeLateItems())
		}
		manager.removeSessionIf(key, session)
	}()

	return session
}

func (s *turnLoopSession) genInput(ctx context.Context, _ *adk.TurnLoop[turnLoopItem, *schema.Message], items []turnLoopItem) (*adk.GenInputResult[turnLoopItem, *schema.Message], error) {
	if len(items) == 0 {
		return &adk.GenInputResult[turnLoopItem, *schema.Message]{RunCtx: ctx}, nil
	}

	current := items[len(items)-1]
	closeTurnLoopItems(items[:len(items)-1])

	msgs := buildMessageHistory(current.ctx, s.agent.systemPrompt, current.sessionID, s.agent.memBackend)
	msgs = append(msgs, schema.UserMessage(current.message))
	persistUserMessage(current.ctx, current.sessionID, current.message, s.agent.memBackend)

	return &adk.GenInputResult[turnLoopItem, *schema.Message]{
		RunCtx: current.ctx,
		Input: &adk.AgentInput{
			Messages:        msgs,
			EnableStreaming: true,
		},
		Consumed: []turnLoopItem{current},
	}, nil
}

func (s *turnLoopSession) prepareAgent(context.Context, *adk.TurnLoop[turnLoopItem, *schema.Message], []turnLoopItem) (adk.Agent, error) {
	return s.agent.agent, nil
}

func (s *turnLoopSession) onAgentEvents(ctx context.Context, tc *adk.TurnContext[turnLoopItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
	if len(tc.Consumed) == 0 {
		return drainTurnLoopEvents(events)
	}

	item := tc.Consumed[len(tc.Consumed)-1]
	defer close(item.out)

	params := streamConsumerParams{
		sessionID:  item.sessionID,
		modelID:    s.agent.modelID,
		provider:   s.agent.provider,
		memBackend: s.agent.memBackend,
	}
	return consumeTurnLoopEvents(ctx, tc, params, events, item.out, s.handleInterrupt(item.sessionID))
}

func (s *turnLoopSession) stopImmediate(cause string) {
	s.once.Do(func() {
		s.loop.Stop(adk.WithImmediate(), adk.WithStopCause(cause))
		s.cancel()
	})
}

func (s *turnLoopSession) handleInterrupt(sessionID string) interruptHandler {
	return func(ctx context.Context, event *adk.AgentEvent, ch chan<- string) bool {
		interruptData := &InterruptState{
			SessionID: sessionID,
			AgentName: s.agent.def.Metadata.Name,
			Reason:    "Agent requested confirmation before proceeding.",
			Fields:    []InputField{{Name: "confirm", Type: "confirm", Label: "Confirm action", Required: true}},
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(interruptData)
		select {
		case ch <- interruptPrefix + string(data):
		case <-ctx.Done():
		}
		return true
	}
}

func consumeTurnLoopEvents(
	ctx context.Context,
	tc *adk.TurnContext[turnLoopItem, *schema.Message],
	params streamConsumerParams,
	iter *adk.AsyncIterator[*adk.AgentEvent],
	ch chan<- string,
	onInterrupt interruptHandler,
) error {
	earlyExit := true
	defer func() {
		if earlyExit {
			drainIterator(iter)
		}
	}()

	streamStart := time.Now()
	firstToken := true
	var fullResponse strings.Builder

	for {
		event, ok := iter.Next()
		if !ok {
			earlyExit = false
			break
		}

		if event.Err != nil {
			if isTurnLoopCancelError(ctx, tc, event.Err) {
				return nil
			}
			log.Printf("[agentdef] turnloop stream error session=%s: %v", params.sessionID, event.Err)
			select {
			case ch <- "[error] internal error occurred":
			case <-ctx.Done():
			}
			return event.Err
		}

		if onInterrupt != nil && event.Action != nil && event.Action.Interrupted != nil {
			if onInterrupt(ctx, event, ch) {
				return nil
			}
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput

		if mv.IsStreaming && mv.MessageStream != nil {
			consumeMessageStream(ctx, mv.MessageStream, params, &fullResponse, &firstToken, streamStart, ch)
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		if mv.Message != nil && mv.Message.Content != "" {
			if firstToken {
				modelrouter.RecordModelLatency(params.modelID, params.provider, time.Since(streamStart))
				firstToken = false
			}
			fullResponse.WriteString(mv.Message.Content)
			select {
			case ch <- mv.Message.Content:
			case <-ctx.Done():
				return nil
			}
		}

		select {
		case <-tc.Preempted:
			return nil
		case <-tc.Stopped:
			return nil
		default:
		}
	}

	if params.memBackend != nil && params.sessionID != "" && fullResponse.Len() > 0 {
		_ = params.memBackend.AddMessage(ctx, params.sessionID, memory.Message{
			Role:      "assistant",
			Content:   fullResponse.String(),
			Timestamp: time.Now().Unix(),
		})
	}

	return nil
}

func isTurnLoopCancelError(ctx context.Context, tc *adk.TurnContext[turnLoopItem, *schema.Message], err error) bool {
	var cancelErr *adk.CancelError
	if errors.As(err, &cancelErr) {
		return true
	}
	if errors.Is(err, context.Canceled) && (ctx.Err() != nil || isTurnLoopPreemptedOrStopped(tc)) {
		return true
	}
	return false
}

func isTurnLoopPreemptedOrStopped(tc *adk.TurnContext[turnLoopItem, *schema.Message]) bool {
	select {
	case <-tc.Preempted:
		return true
	case <-tc.Stopped:
		return true
	default:
		return false
	}
}

func drainTurnLoopEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) error {
	for {
		if _, ok := iter.Next(); !ok {
			return nil
		}
	}
}

func closeTurnLoopItems(items []turnLoopItem) {
	for _, item := range items {
		close(item.out)
	}
}
