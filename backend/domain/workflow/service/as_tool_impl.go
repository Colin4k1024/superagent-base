/*
 * Copyright 2025 coze-dev Authors
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

package service

import (
	"context"


	workflowModel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/compose"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/dagcompose"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

type asToolImpl struct {
	repo workflow.Repository
}

func (a *asToolImpl) WithMessagePipe() (wfcompose.Option, *wfcompose.StreamReader[*entity.Message], func()) {
	if useDAGEngine() {
		return dagcompose.WithMessagePipe()
	}
	return compose.WithMessagePipe()
}

func (a *asToolImpl) WithExecuteConfig(cfg workflowModel.ExecuteConfig) wfcompose.Option {
	if useDAGEngine() {
		return dagcompose.WithToolExecuteConfig(cfg)
	}
	return compose.WithToolExecuteConfig(cfg)
}

func (a *asToolImpl) WithResumeToolWorkflow(resumingEvent *entity.ToolInterruptEvent, resumeData string,
	allInterruptEvents map[string]*entity.ToolInterruptEvent) wfcompose.Option {
	if useDAGEngine() {
		return dagcompose.WithToolResume(resumingEvent, resumeData, allInterruptEvents)
	}
	toolCallID2ExeID := make(map[string]int64, len(allInterruptEvents))
	for callID, event := range allInterruptEvents {
		toolCallID2ExeID[callID] = event.ExecuteID
	}
	return compose.WithToolResume(resumingEvent, resumeData, allInterruptEvents)
}

func (a *asToolImpl) WorkflowAsModelTool(ctx context.Context, policies []*vo.GetPolicy) (tools []workflow.ToolFromWorkflow, err error) {
	for _, id := range policies {
		t, err := a.repo.WorkflowAsTool(ctx, *id, vo.WorkflowToolConfig{})
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}

	return tools, nil
}
