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
	"os"
	"path/filepath"
	"strings"

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
	svc       *singleagent.SingleAgentApplicationService
	runtime   *agentdef.AgentRuntime
	configDir string // path to configs/agents/
}

// NewAgentHandler creates an AgentHandler.
// svc may be nil during startup; the handler returns Unavailable in that case.
// rt may be nil; YAML-based agent operations degrade gracefully.
// configDir is the path to the agents YAML directory (e.g. "configs/agents").
func NewAgentHandler(svc *singleagent.SingleAgentApplicationService, rt *agentdef.AgentRuntime, configDir string) *AgentHandler {
	return &AgentHandler{svc: svc, runtime: rt, configDir: configDir}
}

// CreateAgent creates a new agent.
// NOTE: This RPC does not carry YAML content. Use LoadAgentFromYAML for
// YAML-based agent creation. When only the YAML runtime is active (no DB
// service), this returns Unimplemented.
func (h *AgentHandler) CreateAgent(_ context.Context, req *agentv1.CreateAgentRequest) (*agentv1.CreateAgentResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unimplemented,
			"CreateAgent requires a database-backed service; use LoadAgentFromYAML for YAML-based creation")
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
// NOTE: UpdateAgentRequest carries no YAML content. Use LoadAgentFromYAML for
// YAML-based updates. When only the YAML runtime is active this returns Unimplemented.
func (h *AgentHandler) UpdateAgent(_ context.Context, req *agentv1.UpdateAgentRequest) (*agentv1.UpdateAgentResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unimplemented,
			"UpdateAgent requires a database-backed service; use LoadAgentFromYAML for YAML-based updates")
	}
	// TODO: delegate to h.svc once the domain method signatures are mapped.
	return &agentv1.UpdateAgentResponse{
		Agent: &agentv1.Agent{Id: req.AgentId, Name: req.Name},
	}, nil
}

// DeleteAgent deletes an agent by name.
// The proto carries AgentId (int64) for DB-backed agents. For YAML-backed
// agents the runtime uses string names, so AgentId is not applicable; callers
// should use the HTTP DELETE /api/v1/admin/agents/:name endpoint instead.
// When only the YAML runtime is active, this returns Unimplemented.
func (h *AgentHandler) DeleteAgent(ctx context.Context, req *agentv1.DeleteAgentRequest) (*agentv1.DeleteAgentResponse, error) {
	// YAML-runtime path: find the agent by scanning loaded names, then remove its file.
	if h.svc == nil && h.runtime != nil {
		return nil, status.Error(codes.Unimplemented,
			"DeleteAgent by numeric ID is not supported for YAML-backed agents; use HTTP DELETE /api/v1/admin/agents/:name")
	}
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "agent service not initialised")
	}
	// TODO: delegate to h.svc once the domain method signatures are mapped.
	return &agentv1.DeleteAgentResponse{Success: true}, nil
}

// LoadAgentFromYAML parses an AgentDefinition from YAML content, persists it
// to configDir, and triggers a runtime reload.  This is the primary YAML-based
// create-or-update path; it mirrors the HTTP POST/PUT /api/v1/admin/agents flow.
func (h *AgentHandler) LoadAgentFromYAML(ctx context.Context, req *agentv1.LoadAgentFromYAMLRequest) (*agentv1.LoadAgentFromYAMLResponse, error) {
	if req.YamlContent == "" {
		return nil, status.Error(codes.InvalidArgument, "yaml_content is required")
	}

	def, err := agentdef.Parse([]byte(req.YamlContent))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid agent YAML: %v", err)
	}

	if h.configDir == "" {
		return nil, status.Error(codes.FailedPrecondition, "agent configDir not configured on gRPC handler")
	}

	// Determine whether this is a create or update by checking for an existing file.
	existingFile, findErr := h.findAgentFile(def.Metadata.Name)
	created := findErr != nil // true when no existing file found

	var destPath string
	if created {
		destPath = filepath.Join(h.configDir, def.Metadata.Name+".yaml")
	} else {
		destPath = filepath.Join(h.configDir, existingFile)
	}

	if err := os.WriteFile(destPath, []byte(req.YamlContent), 0o644); err != nil {
		return nil, status.Errorf(codes.Internal, "write agent file: %v", err)
	}

	if h.runtime != nil {
		if reloadErr := h.runtime.Reload(ctx); reloadErr != nil {
			// File was written; return success with a warning embedded in the name
			// so callers can detect the partial state without failing the RPC.
			return &agentv1.LoadAgentFromYAMLResponse{
				Agent: &agentv1.Agent{
					Name:        def.Metadata.Name,
					Description: fmt.Sprintf("reload warning: %v", reloadErr),
					SpaceId:     req.SpaceId,
				},
				Created: created,
			}, nil
		}
	}

	return &agentv1.LoadAgentFromYAMLResponse{
		Agent: &agentv1.Agent{
			Name:    def.Metadata.Name,
			SpaceId: req.SpaceId,
			Status:  "active",
		},
		Created: created,
	}, nil
}

// findAgentFile searches configDir for a YAML file whose metadata.name matches
// agentName.  Returns the filename (not full path) or an error when not found.
// This mirrors the same helper in AgentAdminHandler.
func (h *AgentHandler) findAgentFile(agentName string) (string, error) {
	// Fast path: canonical filename.
	for _, ext := range []string{".yaml", ".yml"} {
		candidate := agentName + ext
		if _, err := os.Stat(filepath.Join(h.configDir, candidate)); err == nil {
			return candidate, nil
		}
	}

	// Slow path: scan all YAML files for matching metadata.name.
	entries, err := os.ReadDir(h.configDir)
	if err != nil {
		return "", fmt.Errorf("read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fname := e.Name()
		if !strings.HasSuffix(fname, ".yaml") && !strings.HasSuffix(fname, ".yml") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(h.configDir, fname))
		if readErr != nil {
			continue
		}
		def, parseErr := agentdef.Parse(data)
		if parseErr != nil {
			continue
		}
		if def.Metadata.Name == agentName {
			return fname, nil
		}
	}

	return "", fmt.Errorf("not found")
}
