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
package llm

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"


	crossplugin "github.com/superagent-ai/superagent-base/backend/crossdomain/plugin"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/execute"
)

type pluginInvokableTool struct {
	pluginInvokableTool crossplugin.InvokableTool
}

func newInvokableTool(pl crossplugin.InvokableTool) einobridge.InvokableTool {
	return &pluginInvokableTool{
		pluginInvokableTool: pl,
	}
}

func (p pluginInvokableTool) Info(ctx context.Context) (*einobridge.ToolInfo, error) {
	return p.pluginInvokableTool.Info(ctx)
}

func (p pluginInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einobridge.ToolOption) (string, error) {
	execCfg := execute.GetExecuteConfig(opts...)
	return p.pluginInvokableTool.PluginInvoke(ctx, argumentsInJSON, execCfg)
}
