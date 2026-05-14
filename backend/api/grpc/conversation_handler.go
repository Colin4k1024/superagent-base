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
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	conversationv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/conversation/v1"
	"github.com/superagent-ai/superagent-base/backend/api/model/conversation/common"
	"github.com/superagent-ai/superagent-base/backend/application/conversation"
	convEntity "github.com/superagent-ai/superagent-base/backend/domain/conversation/conversation/entity"
	msgEntity "github.com/superagent-ai/superagent-base/backend/domain/conversation/message/entity"
	convModel "github.com/superagent-ai/superagent-base/backend/crossdomain/conversation/model"
	msgModel "github.com/superagent-ai/superagent-base/backend/crossdomain/message/model"
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
func (h *ConversationHandler) CreateConversation(ctx context.Context, req *conversationv1.CreateConversationRequest) (*conversationv1.CreateConversationResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}

	conv, err := h.svc.ConversationDomainSVC.Create(ctx, &convEntity.CreateMeta{
		AgentID:  req.AgentId,
		Name:     req.Title,
		Scene:    common.Scene_SceneOpenApi,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create conversation: %v", err)
	}

	return &conversationv1.CreateConversationResponse{
		Conversation: domainConvToProto(conv),
	}, nil
}

// GetConversation retrieves a conversation with its messages.
func (h *ConversationHandler) GetConversation(ctx context.Context, req *conversationv1.GetConversationRequest) (*conversationv1.GetConversationResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}

	conv, err := h.svc.ConversationDomainSVC.GetByID(ctx, req.ConversationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get conversation: %v", err)
	}
	if conv == nil {
		return nil, status.Errorf(codes.NotFound, "conversation %d not found", req.ConversationId)
	}

	// Fetch the most recent 50 messages for this conversation.
	listResult, err := h.svc.MessageDomainSVC.List(ctx, &msgEntity.ListMeta{
		ConversationID: req.ConversationId,
		Limit:          50,
		Direction:      msgEntity.ScrollPageDirectionPrev,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list messages: %v", err)
	}

	msgs := make([]*conversationv1.Message, 0, len(listResult.Messages))
	for _, m := range listResult.Messages {
		msgs = append(msgs, domainMsgToProto(m))
	}

	return &conversationv1.GetConversationResponse{
		Conversation: domainConvToProto(conv),
		Messages:     msgs,
	}, nil
}

// ListConversations lists conversations for an agent/user.
func (h *ConversationHandler) ListConversations(ctx context.Context, req *conversationv1.ListConversationsRequest) (*conversationv1.ListConversationsResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "conversation service not initialised")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	convs, hasMore, err := h.svc.ConversationDomainSVC.List(ctx, &convEntity.ListMeta{
		AgentID: req.AgentId,
		Page:    page,
		Limit:   pageSize,
		Scene:   common.Scene_SceneOpenApi,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list conversations: %v", err)
	}

	protoConvs := make([]*conversationv1.Conversation, 0, len(convs))
	for _, c := range convs {
		protoConvs = append(protoConvs, domainConvToProto(c))
	}

	return &conversationv1.ListConversationsResponse{
		Conversations: protoConvs,
		Total:         int32(len(protoConvs)),
		HasMore:       hasMore,
	}, nil
}

// domainConvToProto maps a domain Conversation to its proto representation.
func domainConvToProto(c *convModel.Conversation) *conversationv1.Conversation {
	proto := &conversationv1.Conversation{
		Id:      c.ID,
		AgentId: c.AgentID,
		Title:   c.Name,
	}
	if c.UserID != nil {
		// UserID is stored as a string in the domain model; best-effort zero for proto int64.
		_ = c.UserID
	}
	if c.CreatedAt > 0 {
		proto.CreatedAt = timestamppb.New(time.UnixMilli(c.CreatedAt))
	}
	if c.UpdatedAt > 0 {
		proto.UpdatedAt = timestamppb.New(time.UnixMilli(c.UpdatedAt))
	}
	return proto
}

// domainMsgToProto maps a domain Message to its proto representation.
func domainMsgToProto(m *msgModel.Message) *conversationv1.Message {
	proto := &conversationv1.Message{
		Id:             m.ID,
		ConversationId: m.ConversationID,
		Role:           string(m.Role),
		Content:        m.Content,
		ContentType:    string(m.ContentType),
	}
	if m.CreatedAt > 0 {
		proto.CreatedAt = timestamppb.New(time.UnixMilli(m.CreatedAt))
	}
	return proto
}
