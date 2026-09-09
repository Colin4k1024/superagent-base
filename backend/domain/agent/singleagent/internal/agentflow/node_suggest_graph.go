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

package agentflow

import (
	"context"


	"github.com/superagent-ai/superagent-base/backend/api/model/app/bot_common"
	"github.com/superagent-ai/superagent-base/backend/bizpkg/llm/modelbuilder"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"github.com/superagent-ai/superagent-base/backend/pkg/lang/ptr"
)

const (
	keyOfSuggestPromptVariables = "suggest_prompt_variables"
	keyOfSuggestGraph           = "suggest_graph"
	keyOfSuggestPreInputParse   = "suggest_pre_input_parse"
	keyOfSuggestPersonParse     = "suggest_persona"
	keyOfSuggestChatModel       = "suggest_chat_model"
	keyOfSuggestParser          = "suggest_parser"
	keyOfSuggestTemplate        = "suggest_template"
)

func newSuggestGraph(_ context.Context, conf *Config, chatModel modelbuilder.ToolCallingChatModel) (*einobridge.Graph[[]*einobridge.Message, *einobridge.Message], bool) {

	isNeedGenerateSuggest := false
	agentSuggestionSetting := conf.Agent.SuggestReply

	sp := &suggestPersonaRender{}
	if agentSuggestionSetting != nil && ptr.From(agentSuggestionSetting.SuggestReplyMode) != bot_common.SuggestReplyMode_Disable {
		isNeedGenerateSuggest = true
		if ptr.From(agentSuggestionSetting.SuggestReplyMode) == bot_common.SuggestReplyMode_Custom {
			sp.persona = ptr.From(agentSuggestionSetting.CustomizedSuggestPrompt)
		}
	}

	if !isNeedGenerateSuggest {
		return nil, isNeedGenerateSuggest
	}
	suggestPrompt := einobridge.PromptFromMessages(einobridge.Jinja2,
		einobridge.SystemMessage(SUGGESTION_PROMPT_JINJA2),
		einobridge.UserMessage("Based on the contextual information, provide three recommended questions"),
	)

	suggestGraph := einobridge.NewGraph[[]*einobridge.Message, *einobridge.Message]()
	suggestPromptVars := &suggestPromptVariables{}
	_ = suggestGraph.AddLambdaNode(keyOfSuggestPromptVariables,
		einobridge.InvokableLambda[[]*einobridge.Message, map[string]any](func(ctx context.Context, vb []*einobridge.Message) (map[string]any, error) {
		return suggestPromptVars.AssembleSuggestPromptVariables(ctx, einobridge.WrapMessageSlice(vb))
	}))

	_ = suggestGraph.AddLambdaNode(keyOfSuggestPersonParse,
		einobridge.InvokableLambda[[]*einobridge.Message, string](func(ctx context.Context, vb []*einobridge.Message) (string, error) {
		return sp.RenderPersona(ctx, einobridge.WrapMessageSlice(vb))
	}),
		einobridge.WithOutputKey(keyOfSuggestPersonParse),
	)

	_ = suggestGraph.AddChatTemplateNode(keyOfSuggestTemplate, suggestPrompt)
	_ = suggestGraph.AddChatModelNode(keyOfSuggestChatModel, chatModel, einobridge.WithNodeName(keyOfSuggestChatModel))
	_ = suggestGraph.AddLambdaNode(keyOfSuggestParser, einobridge.InvokableLambda[*einobridge.Message, *einobridge.Message](func(ctx context.Context, msg *einobridge.Message) (*einobridge.Message, error) {
		wfMsg, err := suggestParser(ctx, einobridge.WrapMessage(msg))
		if err != nil {
			return nil, err
		}
		return einobridge.UnwrapMessage(wfMsg), nil
	}), einobridge.WithNodeName(keyOfSuggestParser))

	_ = suggestGraph.AddEdge(einobridge.START, keyOfSuggestPromptVariables)
	_ = suggestGraph.AddEdge(einobridge.START, keyOfSuggestPersonParse)
	_ = suggestGraph.AddEdge(keyOfSuggestPersonParse, keyOfSuggestTemplate)
	_ = suggestGraph.AddEdge(keyOfSuggestPromptVariables, keyOfSuggestTemplate)
	_ = suggestGraph.AddEdge(keyOfSuggestTemplate, keyOfSuggestChatModel)
	_ = suggestGraph.AddEdge(keyOfSuggestChatModel, keyOfSuggestParser)
	_ = suggestGraph.AddEdge(keyOfSuggestParser, einobridge.END)

	return suggestGraph, isNeedGenerateSuggest
}
