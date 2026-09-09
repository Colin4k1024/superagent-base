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

package modelbuilder

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"

	"google.golang.org/genai"

	"github.com/superagent-ai/superagent-base/backend/api/model/admin/config"
	"github.com/superagent-ai/superagent-base/backend/pkg/lang/ptr"
)

type geminiModelBuilder struct {
	cfg *config.Model
}

func newGeminiModelBuilder(cfg *config.Model) Service {
	return &geminiModelBuilder{
		cfg: cfg,
	}
}

func (g *geminiModelBuilder) getDefaultGeminiConfig() *einobridge.GeminiModelConfig {
	return &einobridge.GeminiModelConfig{}
}

func (g *geminiModelBuilder) getDefaultGenaiConfig() *genai.ClientConfig {
	return &genai.ClientConfig{
		HTTPOptions: genai.HTTPOptions{
			BaseURL: "https://generativelanguage.googleapis.com/",
		},
	}
}

func (g *geminiModelBuilder) applyParamsToGeminiConfig(conf *einobridge.GeminiModelConfig, params *LLMParams) {
	if params == nil {
		return
	}

	if params.TopK != nil {
		k := int(*params.TopK)
		conf.TopK = &k
	}
	conf.TopP = params.TopP

	if params.Temperature != nil {
		t := float64(*params.Temperature)
		conf.Temperature = &t
	}

	if params.MaxTokens != 0 {
		conf.MaxTokens = ptr.Of(params.MaxTokens)
	}

	if params.EnableThinking != nil {
		conf.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: *params.EnableThinking,
		}
	}
}

func (g *geminiModelBuilder) Build(ctx context.Context, params *LLMParams) (ToolCallingChatModel, error) {
	base := g.cfg.Connection.BaseConnInfo

	clientCfg := g.getDefaultGenaiConfig()
	if base.BaseURL != "" {
		clientCfg.HTTPOptions.BaseURL = base.BaseURL
	}

	clientCfg.APIKey = base.APIKey
	if g.cfg.Connection.Gemini != nil {
		clientCfg.Backend = genai.Backend(g.cfg.Connection.Gemini.Backend)
		clientCfg.Project = g.cfg.Connection.Gemini.Project
		clientCfg.Location = g.cfg.Connection.Gemini.Location
	}

	client, err := genai.NewClient(ctx, clientCfg)
	if err != nil {
		return nil, err
	}

	conf := g.getDefaultGeminiConfig()
	conf.Client = client
	conf.Model = base.Model

	switch base.ThinkingType {
	case config.ThinkingType_Enable:
		conf.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
		}
	case config.ThinkingType_Disable:
		conf.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: false,
		}
	}

	g.applyParamsToGeminiConfig(conf, params)

	return einobridge.GeminiNewChatModel(ctx, conf)
}
