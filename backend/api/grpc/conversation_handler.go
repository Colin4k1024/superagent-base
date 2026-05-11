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

package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	conversationv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/conversation/v1"
	"github.com/superagent-ai/superagent-base/backend/application/conversation"
	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// ConversationHandler implements conversationv1.ConversationServiceServer by
// delegating to the conversation application service and the AgentRuntime.
type ConversationHandler struct {
	conversationv1.UnimplementedConversationServiceServer
	svc     *conversation.ConversationApplicationService
	runtime *agentdef.AgentRuntime
}

// NewConversationHandler creates a ConversationHandler.
// rt may be nil; Chat falls back to a placeholder response in that case.
func NewConversationHandler(svc *conversation.ConversationApplicationService, rt *agentdef.AgentRuntime) *ConversationHandler {
	return &ConversationHandler{svc: svc, runtime: rt}
}

// agentNameFromRequest extracts the agent name from the request.
// It checks Extra.Fields["agent_name"] first, then falls back to "research-agent".
func agentNameFromRequest(req *conversationv1.ChatRequest) string {
	if req.Extra != nil {
		if v, ok := req.Extra.Fields["agent_name"]; ok {
			if s := v.GetStringValue(); s != "" {
				return s
			}
		}
	}
	// Default to the primary demo agent.
	return "research-agent"
}

// Chat streams agent response chunks back to the caller.
// When the AgentRuntime is available it routes to the named agent and streams
// real LLM tokens.  Otherwise it falls back to a placeholder response.
func (h *ConversationHandler) Chat(req *conversationv1.ChatRequest, stream conversationv1.ConversationService_ChatServer) error {
	if req.Content == "" {
		return status.Error(codes.InvalidArgument, "content is required")
	}

	// Route to real agent when the runtime is available.
	if h.runtime != nil {
		return h.chatWithRuntime(req, stream)
	}

	// Fallback placeholder when no runtime is configured.
	if err := stream.Send(&conversationv1.ChatResponse{
		EventType: "done",
		Delta:     "",
		Message: &conversationv1.Message{
			ConversationId: req.ConversationId,
			Role:           "assistant",
			Content:        "[placeholder] response for: " + req.Content,
			ContentType:    "text",
		},
	}); err != nil {
		return status.Errorf(codes.Internal, "stream send failed: %v", err)
	}
	return nil
}

// chatWithRuntime performs a real streaming chat via the AgentRuntime.
func (h *ConversationHandler) chatWithRuntime(req *conversationv1.ChatRequest, stream conversationv1.ConversationService_ChatServer) error {
	agentName := agentNameFromRequest(req)

	agent, ok := h.runtime.GetAgent(agentName)
	if !ok {
		return status.Errorf(codes.NotFound, "agent %q not found", agentName)
	}

	sessionID := fmt.Sprintf("conv-%d", req.ConversationId)
	ch, err := agent.Chat(stream.Context(), sessionID, req.Content)
	if err != nil {
		return status.Errorf(codes.Internal, "chat failed: %v", err)
	}

	// Stream tokens as delta events.
	var fullContent string
	for token := range ch {
		fullContent += token
		if sendErr := stream.Send(&conversationv1.ChatResponse{
			EventType: "delta",
			Delta:     token,
		}); sendErr != nil {
			return sendErr
		}
	}

	// Send final done event with the complete assembled message.
	return stream.Send(&conversationv1.ChatResponse{
		EventType: "done",
		Delta:     "",
		Message: &conversationv1.Message{
			ConversationId: req.ConversationId,
			Role:           "assistant",
			Content:        fullContent,
			ContentType:    "text",
		},
	})
}

// CreateConversation creates a new conversation.
func (h *ConversationHandler) CreateConversation(_ context.Context, req *conversationv1.CreateConversationRequest) (*conversationv1.CreateConversationResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}
	// TODO: delegate to h.svc.
	return &conversationv1.CreateConversationResponse{
		Conversation: &conversationv1.Conversation{
			AgentId: req.AgentId,
			UserId:  req.UserId,
			Title:   req.Title,
		},
	}, nil
}

// GetConversation retrieves a conversation with its messages.
func (h *ConversationHandler) GetConversation(_ context.Context, req *conversationv1.GetConversationRequest) (*conversationv1.GetConversationResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}
	// TODO: delegate to h.svc.
	return &conversationv1.GetConversationResponse{
		Conversation: &conversationv1.Conversation{Id: req.ConversationId},
		Messages:     []*conversationv1.Message{},
	}, nil
}

// ListConversations lists conversations for an agent/user.
func (h *ConversationHandler) ListConversations(_ context.Context, _ *conversationv1.ListConversationsRequest) (*conversationv1.ListConversationsResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}
	// TODO: delegate to h.svc.
	return &conversationv1.ListConversationsResponse{
		Conversations: []*conversationv1.Conversation{},
	}, nil
}
