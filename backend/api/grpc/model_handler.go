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
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modelv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/model/v1"
	adminconfig "github.com/superagent-ai/superagent-base/backend/api/model/admin/config"
	"github.com/superagent-ai/superagent-base/backend/api/model/app/developer_api"
	"github.com/superagent-ai/superagent-base/backend/application/modelmgr"
	"github.com/superagent-ai/superagent-base/backend/bizpkg/config"
	bizmgr "github.com/superagent-ai/superagent-base/backend/bizpkg/config/modelmgr"
	"github.com/superagent-ai/superagent-base/backend/bizpkg/llm/modelbuilder"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
	"github.com/cloudwego/eino/schema"
)

// ModelHandler implements modelv1.ModelServiceServer by delegating to
// the modelmgr application service.
type ModelHandler struct {
	modelv1.UnimplementedModelServiceServer
	svc *modelmgr.ModelmgrApplicationService
}

// NewModelHandler creates a ModelHandler.
func NewModelHandler(svc *modelmgr.ModelmgrApplicationService) *ModelHandler {
	return &ModelHandler{svc: svc}
}

// modelConf returns the ModelConfig singleton, or nil if not yet initialised.
// config.ModelConf() dereferences a nil pointer when the config has not been
// initialised, so we guard with a named-return + recover.
func modelConf() (mc *bizmgr.ModelConfig) {
	defer func() {
		if r := recover(); r != nil {
			mc = nil
		}
	}()
	mc = config.ModelConf()
	return mc
}

// modelDoToProto converts an internal modelmgr.Model to a proto Model message.
func modelDoToProto(m *bizmgr.Model) *modelv1.Model {
	if m == nil || m.Model == nil {
		return nil
	}

	inner := m.Model

	pm := &modelv1.Model{
		Id:            fmt.Sprintf("%d", inner.ID),
		Enabled:       inner.Status != config.ModelStatus_StatusDeleted,
		ContextLength: int32(inner.DisplayInfo.GetMaxTokens()),
	}

	if inner.DisplayInfo != nil {
		pm.Name = inner.DisplayInfo.Name
		if inner.DisplayInfo.Description != nil {
			pm.Description = inner.DisplayInfo.Description.EnUs
		}
	}

	if inner.Provider != nil {
		pm.Provider = inner.Provider.ModelClass.String()
	}

	if inner.Capability != nil {
		caps := make([]string, 0, 4)
		if inner.Capability.FunctionCall != nil && *inner.Capability.FunctionCall {
			caps = append(caps, "function_call")
		}
		if inner.Capability.ImageUnderstanding != nil && *inner.Capability.ImageUnderstanding {
			caps = append(caps, "image_understanding")
		}
		if inner.Capability.VideoUnderstanding != nil && *inner.Capability.VideoUnderstanding {
			caps = append(caps, "video_understanding")
		}
		if inner.Capability.AudioUnderstanding != nil && *inner.Capability.AudioUnderstanding {
			caps = append(caps, "audio_understanding")
		}
		if inner.Capability.SupportMultiModal != nil && *inner.Capability.SupportMultiModal {
			caps = append(caps, "multimodal")
		}
		pm.Capabilities = caps
	}

	return pm
}

// ListModels lists available model configurations.
func (h *ModelHandler) ListModels(ctx context.Context, req *modelv1.ListModelsRequest) (*modelv1.ListModelsResponse, error) {
	mc := modelConf()
	if mc == nil {
		logs.CtxWarnf(ctx, "model config not yet initialised, returning empty list")
		return &modelv1.ListModelsResponse{Models: []*modelv1.Model{}, Total: 0}, nil
	}

	modelList, err := mc.GetOnlineModelList(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list models: %v", err)
	}

	protos := make([]*modelv1.Model, 0, len(modelList))
	for _, m := range modelList {
		pm := modelDoToProto(m)
		if pm == nil {
			continue
		}
		// Apply provider filter if requested.
		if req.FilterProvider != "" && pm.Provider != req.FilterProvider {
			continue
		}
		// Apply capability filter if requested.
		if req.FilterCapability != "" {
			matched := false
			for _, c := range pm.Capabilities {
				if c == req.FilterCapability {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		protos = append(protos, pm)
	}

	return &modelv1.ListModelsResponse{
		Models: protos,
		Total:  int32(len(protos)),
	}, nil
}

// GetModel retrieves a model by ID.
func (h *ModelHandler) GetModel(ctx context.Context, req *modelv1.GetModelRequest) (*modelv1.GetModelResponse, error) {
	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model_id is required")
	}

	modelID, err := strconv.ParseInt(req.ModelId, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model_id must be a numeric ID, got %q", req.ModelId)
	}

	mc := modelConf()
	if mc == nil {
		return nil, status.Error(codes.Unavailable, "model config not yet initialised")
	}

	m, err := mc.GetModelByID(ctx, modelID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "model %d not found: %v", modelID, err)
	}

	return &modelv1.GetModelResponse{Model: modelDoToProto(m)}, nil
}

// CreateModel registers a new model configuration and persists it to the DB.
// It mirrors the HTTP CreateModel handler in config_service.go:
//  1. Validate inputs and map proto fields to internal types.
//  2. Build a transient LLM client and verify connectivity with a test prompt.
//  3. Persist via ModelConfig.CreateModel and return the saved record.
func (h *ModelHandler) CreateModel(ctx context.Context, req *modelv1.CreateModelRequest) (*modelv1.CreateModelResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	// Map provider string → ModelClass.
	modelClass, err := developer_api.ModelClassFromString(req.Provider)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported provider %q: %v", req.Provider, err)
	}

	// Build a Connection from the optional Config struct.
	conn := &adminconfig.Connection{
		BaseConnInfo: &adminconfig.BaseConnectionInfo{},
	}
	if req.Config != nil {
		fields := req.Config.GetFields()
		if v, ok := fields["base_url"]; ok {
			conn.BaseConnInfo.BaseURL = v.GetStringValue()
		}
		if v, ok := fields["api_key"]; ok {
			conn.BaseConnInfo.APIKey = v.GetStringValue()
		}
		if v, ok := fields["model"]; ok {
			conn.BaseConnInfo.Model = v.GetStringValue()
		}
	}

	// Verify connectivity: build a transient LLM client and run a trivial prompt.
	builder, err := modelbuilder.NewModelBuilder(modelClass, &adminconfig.Model{Connection: conn})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "build model builder failed: %v", err)
	}
	chatModel, err := builder.Build(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "build model failed: %v", err)
	}
	if _, err = chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage("1+1=?,Just answer with a number, no explanation."),
	}); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model connectivity check failed: %v", err)
	}

	// Persist.
	mc := modelConf()
	if mc == nil {
		return nil, status.Error(codes.Unavailable, "model config not yet initialised")
	}
	id, err := mc.CreateModel(ctx, modelClass, req.Name, conn, &bizmgr.ModelExtra{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "persist model failed: %v", err)
	}

	// Fetch the newly created record so we can return a fully populated proto.
	saved, err := mc.GetModelByID(ctx, id)
	if err != nil {
		logs.CtxWarnf(ctx, "model created (id=%d) but re-fetch failed: %v", id, err)
		// Return a minimal proto rather than failing the whole call.
		return &modelv1.CreateModelResponse{
			Model: &modelv1.Model{
				Id:            fmt.Sprintf("%d", id),
				Name:          req.Name,
				Provider:      req.Provider,
				Description:   req.Description,
				Capabilities:  req.Capabilities,
				ContextLength: req.ContextLength,
				Enabled:       true,
			},
		}, nil
	}

	return &modelv1.CreateModelResponse{Model: modelDoToProto(saved)}, nil
}

// TestModel tests connectivity to a model by sending a short prompt and
// measuring the round-trip latency. The method always returns a non-error
// gRPC response; failures are reported via success=false + error string.
func (h *ModelHandler) TestModel(ctx context.Context, req *modelv1.TestModelRequest) (*modelv1.TestModelResponse, error) {
	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model_id is required")
	}

	modelID, err := strconv.ParseInt(req.ModelId, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model_id must be numeric, got %q", req.ModelId)
	}

	// Build model from persisted config.
	chatModel, _, err := modelbuilder.BuildModelByID(ctx, modelID, nil)
	if err != nil {
		return &modelv1.TestModelResponse{
			Success: false,
			Error:   fmt.Sprintf("build model failed: %v", err),
		}, nil
	}

	prompt := req.Prompt
	if prompt == "" {
		prompt = "1+1=?,Just answer with a number, no explanation."
	}

	start := time.Now()
	resp, err := chatModel.Generate(ctx, []*schema.Message{schema.SystemMessage(prompt)})
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return &modelv1.TestModelResponse{
			Success:   false,
			LatencyMs: latencyMs,
			Error:     err.Error(),
		}, nil
	}

	responseText := ""
	if resp != nil {
		responseText = resp.Content
	}

	return &modelv1.TestModelResponse{
		Success:   true,
		Response:  responseText,
		LatencyMs: latencyMs,
	}, nil
}
