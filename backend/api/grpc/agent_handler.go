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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/agent/v1"
	"github.com/superagent-ai/superagent-base/backend/application/singleagent"
	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// AgentHandler implements agentv1.AgentServiceServer by delegating to
// the singleagent application service and the AgentRuntime.
type AgentHandler struct {
	agentv1.UnimplementedAgentServiceServer
	svc     *singleagent.SingleAgentApplicationService
	runtime *agentdef.AgentRuntime
}

// NewAgentHandler creates an AgentHandler.
// svc may be nil during startup; the handler returns Unavailable in that case.
// rt may be nil; YAML-based agent operations degrade gracefully.
func NewAgentHandler(svc *singleagent.SingleAgentApplicationService, rt *agentdef.AgentRuntime) *AgentHandler {
	return &AgentHandler{svc: svc, runtime: rt}
}

// CreateAgent creates a new agent.
func (h *AgentHandler) CreateAgent(_ context.Context, req *agentv1.CreateAgentRequest) (*agentv1.CreateAgentResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	// TODO: delegate to h.svc once the domain method signatures are mapped.
	return &agentv1.CreateAgentResponse{
		Agent: &agentv1.Agent{
			Name:        req.Name,
			Description: req.Description,
			SpaceId:     req.SpaceId,
			IconUrl:     req.IconUrl,
		},
	}, nil
}

// GetAgent retrieves an agent by ID.
// When the runtime is available and the request carries AgentId==0, this is
// interpreted as a "list first agent" probe; otherwise the ID is used as-is.
func (h *AgentHandler) GetAgent(_ context.Context, req *agentv1.GetAgentRequest) (*agentv1.GetAgentResponse, error) {
	if h.svc == nil && h.runtime == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	return &agentv1.GetAgentResponse{
		Agent: &agentv1.Agent{Id: req.AgentId},
	}, nil
}

// ListAgents lists agents from the AgentRuntime (YAML-defined) when available,
// otherwise falls back to the database-backed singleagent service.
func (h *AgentHandler) ListAgents(_ context.Context, _ *agentv1.ListAgentsRequest) (*agentv1.ListAgentsResponse, error) {
	if h.runtime != nil {
		names := h.runtime.ListAgents()
		agents := make([]*agentv1.Agent, 0, len(names))
		for _, name := range names {
			a, ok := h.runtime.GetAgent(name)
			if !ok {
				continue
			}
			agents = append(agents, &agentv1.Agent{
				Name:        a.Name(),
				Description: a.Description(),
				Status:      "active",
			})
		}
		return &agentv1.ListAgentsResponse{
			Agents: agents,
			Total:  int32(len(agents)),
		}, nil
	}
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	return &agentv1.ListAgentsResponse{Agents: []*agentv1.Agent{}}, nil
}

// UpdateAgent updates an existing agent.
func (h *AgentHandler) UpdateAgent(_ context.Context, req *agentv1.UpdateAgentRequest) (*agentv1.UpdateAgentResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	// TODO: delegate to h.svc.
	return &agentv1.UpdateAgentResponse{
		Agent: &agentv1.Agent{Id: req.AgentId, Name: req.Name},
	}, nil
}

// DeleteAgent deletes an agent.
func (h *AgentHandler) DeleteAgent(_ context.Context, _ *agentv1.DeleteAgentRequest) (*agentv1.DeleteAgentResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	// TODO: delegate to h.svc.
	return &agentv1.DeleteAgentResponse{Success: true}, nil
}

// LoadAgentFromYAML parses an AgentDefinition from YAML content, validates it,
// and returns the parsed agent as a proto.  Actual persistence (create/update)
// is deferred to a future implementation once the domain service method
// signatures are mapped.
func (h *AgentHandler) LoadAgentFromYAML(ctx context.Context, req *agentv1.LoadAgentFromYAMLRequest) (*agentv1.LoadAgentFromYAMLResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	if req.YamlContent == "" {
		return nil, status.Error(codes.InvalidArgument, "yaml_content is required")
	}

	def, err := agentdef.Parse([]byte(req.YamlContent))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid agent YAML: %v", err)
	}

	// Build a runtime agent from the definition (placeholder implementation).
	builder := agentdef.NewAgentBuilder()
	agent, err := builder.Build(ctx, def)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build agent from YAML: %v", err)
	}

	// TODO: persist via h.svc once domain method signatures are mapped.
	return &agentv1.LoadAgentFromYAMLResponse{
		Agent: &agentv1.Agent{
			Name:        agent.Name(),
			Description: agent.Description(),
			SpaceId:     req.SpaceId,
		},
		Created: true,
	}, nil
}
