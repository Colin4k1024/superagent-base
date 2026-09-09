/*
 * License: Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package modelbuilder

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"


	"github.com/superagent-ai/superagent-base/backend/api/model/admin/config"
	"github.com/superagent-ai/superagent-base/backend/pkg/lang/ptr"
)

type claudeModelBuilder struct {
	cfg *config.Model
}

func newClaudeModelBuilder(cfg *config.Model) Service {
	return &claudeModelBuilder{
		cfg: cfg,
	}
}

func (c *claudeModelBuilder) getDefaultClaudeConfig() *einobridge.ClaudeConfig {
	return &einobridge.ClaudeConfig{}
}

func (c *claudeModelBuilder) applyParamsToChatModelConfig(conf *einobridge.ClaudeConfig, params *LLMParams) {
	if params == nil {
		return
	}

	conf.TopP = params.TopP
	conf.TopK = params.TopK

	if params.Temperature != nil {
		conf.Temperature = ptr.Of(*params.Temperature)
	}

	if params.MaxTokens != 0 {
		conf.MaxTokens = params.MaxTokens
	}

	if params.EnableThinking != nil {
		conf.Thinking = &einobridge.ClaudeThinking{
			Enable: *params.EnableThinking,
		}
	}
}

func (c *claudeModelBuilder) Build(ctx context.Context, params *LLMParams) (ToolCallingChatModel, error) {
	base := c.cfg.Connection.BaseConnInfo

	conf := c.getDefaultClaudeConfig()
	conf.APIKey = base.APIKey
	conf.Model = base.Model
	if base.BaseURL != "" {
		conf.BaseURL = &base.BaseURL
	}

	switch base.ThinkingType {
	case config.ThinkingType_Enable:
		conf.Thinking = &einobridge.ClaudeThinking{
			Enable: true,
		}
	case config.ThinkingType_Disable:
		conf.Thinking = &einobridge.ClaudeThinking{
			Enable: false,
		}
	}

	c.applyParamsToChatModelConfig(conf, params)

	return einobridge.ClaudeNewChatModel(ctx, conf)
}
