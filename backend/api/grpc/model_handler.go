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

	modelv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/model/v1"
	"github.com/superagent-ai/superagent-base/backend/application/modelmgr"
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

// ListModels lists available model configurations.
func (h *ModelHandler) ListModels(_ context.Context, _ *modelv1.ListModelsRequest) (*modelv1.ListModelsResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "model service not initialised")
	}
	// TODO: delegate to h.svc.GetModelList and map results.
	return &modelv1.ListModelsResponse{Models: []*modelv1.Model{}}, nil
}

// GetModel retrieves a model by ID.
func (h *ModelHandler) GetModel(_ context.Context, req *modelv1.GetModelRequest) (*modelv1.GetModelResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "model service not initialised")
	}
	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model_id is required")
	}
	// TODO: delegate to h.svc.
	return &modelv1.GetModelResponse{
		Model: &modelv1.Model{Id: req.ModelId},
	}, nil
}

// CreateModel registers a new model configuration.
func (h *ModelHandler) CreateModel(_ context.Context, req *modelv1.CreateModelRequest) (*modelv1.CreateModelResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "model service not initialised")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	// TODO: persist via h.svc.
	return &modelv1.CreateModelResponse{
		Model: &modelv1.Model{
			Name:        req.Name,
			Provider:    req.Provider,
			Description: req.Description,
		},
	}, nil
}

// TestModel tests connectivity to a model endpoint.
func (h *ModelHandler) TestModel(_ context.Context, req *modelv1.TestModelRequest) (*modelv1.TestModelResponse, error) {
	if h.svc == nil {
		return nil, status.Error(codes.Unavailable, "model service not initialised")
	}
	if req.ModelId == "" {
		return nil, status.Error(codes.InvalidArgument, "model_id is required")
	}
	// TODO: invoke model and measure latency.
	return &modelv1.TestModelResponse{
		Success:   true,
		Response:  "[placeholder]",
		LatencyMs: 0,
	}, nil
}
