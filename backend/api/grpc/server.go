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

// Package grpc provides the gRPC server and service handler implementations
// for the superagent-base backend. It runs alongside the Hertz HTTP server.
package grpc

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	agentv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/agent/v1"
	conversationv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/conversation/v1"
	modelv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/model/v1"
	toolv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/tool/v1"
	"github.com/superagent-ai/superagent-base/backend/application/conversation"
	"github.com/superagent-ai/superagent-base/backend/application/modelmgr"
	"github.com/superagent-ai/superagent-base/backend/application/singleagent"
	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool"
)

// recoveryInterceptor catches panics in gRPC handlers and converts them to
// Internal errors instead of crashing the process.
func recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logs.Errorf("gRPC panic recovered in %s: %v\n%s", info.FullMethod, r, debug.Stack())
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// NewServer creates a configured gRPC server and registers all service handlers.
// rt may be nil if the agent runtime failed to start; handlers degrade gracefully.
// toolMgr may be nil; the ToolHandler returns Unavailable in that case.
// configDir is the path to the agents YAML directory (e.g. "configs/agents").
// store may be nil; agent DB persistence degrades gracefully without it.
func NewServer(rt *agentdef.AgentRuntime, toolMgr *tool.Manager, configDir string, store *agentdef.AgentDefinitionStore) *grpc.Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor()),
	)

	// Register service implementations.
	agentv1.RegisterAgentServiceServer(s, NewAgentHandler(singleagent.SingleAgentSVC, rt, configDir, store))
	conversationv1.RegisterConversationServiceServer(s, NewConversationHandler(conversation.ConversationSVC, rt))
	modelv1.RegisterModelServiceServer(s, NewModelHandler(modelmgr.ModelmgrApplicationSVC))
	toolv1.RegisterToolServiceServer(s, NewToolHandler(toolMgr))

	// Enable reflection so grpcurl and other tools can introspect the server.
	reflection.Register(s)

	return s
}

// ListenAndServe starts the gRPC server on the given address (e.g. ":50051").
// It blocks until the server is stopped.
// configDir is the path to the agents YAML directory (e.g. "configs/agents").
// store may be nil; agent DB persistence degrades gracefully without it.
func ListenAndServe(addr string, rt *agentdef.AgentRuntime, toolMgr *tool.Manager, configDir string, store *agentdef.AgentDefinitionStore) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc: failed to listen on %s: %w", addr, err)
	}

	s := NewServer(rt, toolMgr, configDir, store)
	logs.Infof("gRPC server listening on %s", addr)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("grpc: server error: %w", err)
	}
	return nil
}
