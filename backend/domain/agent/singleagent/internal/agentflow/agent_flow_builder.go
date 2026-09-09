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
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"context"
	"fmt"
	"regexp"
	"strings"


	"github.com/superagent-ai/superagent-base/backend/bizpkg/llm/modelbuilder"
	"github.com/superagent-ai/superagent-base/backend/domain/agent/singleagent/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow"
	"github.com/superagent-ai/superagent-base/backend/pkg/lang/slices"
)

type Config struct {
	Agent    *entity.SingleAgent
	UserID   string
	Identity *entity.AgentIdentity
	CPStore  einobridge.CheckPointStore

	CustomVariables map[string]string

	ConversationID int64
}

const (
	keyOfPersonRender           = "persona_render"
	keyOfKnowledgeRetriever     = "knowledge_retriever"
	keyOfKnowledgeRetrieverPack = "knowledge_retriever_pack"
	keyOfPromptVariables        = "prompt_variables"
	keyOfPromptTemplate         = "prompt_template"
	keyOfReActAgent             = "react_agent"
	keyOfReActAgentToolsNode    = "agent_tool"
	keyOfReActAgentChatModel    = "re_act_chat_model"
	keyOfLLM                    = "llm"
	keyOfToolsPreRetriever      = "tools_pre_retriever"
)

func BuildAgent(ctx context.Context, conf *Config) (r *AgentRunner, err error) {
	persona := conf.Agent.Prompt.GetPrompt()

	avConf := &variableConf{
		Agent:       conf.Agent,
		UserID:      conf.UserID,
		ConnectorID: conf.Identity.ConnectorID,
	}
	avs, err := loadAgentVariables(ctx, avConf)
	if err != nil {
		return nil, err
	}
	if conf.CustomVariables != nil {
		for k, v := range conf.CustomVariables {
			avs[k] = v
		}
	}

	promptVars := &promptVariables{
		Agent: conf.Agent,
		avs:   avs,
	}

	personaVars := &personaRender{
		personaVariableNames: extractJinja2Placeholder(persona),
		persona:              persona,
		variables:            avs,
	}

	kr, err := newKnowledgeRetriever(ctx, &retrieverConfig{
		knowledgeConfig: conf.Agent.Knowledge,
	})
	if err != nil {
		return nil, err
	}

	chatModel, modelInfo, err := modelbuilder.BuildModelBySettings(ctx, conf.Agent.ModelInfo)
	if err != nil {
		return nil, err
	}

	requireCheckpoint := false
	pluginTools, err := newPluginTools(ctx, &toolConfig{
		spaceID:       conf.Agent.SpaceID,
		userID:        conf.UserID,
		agentIdentity: conf.Identity,
		toolConf:      conf.Agent.Plugin,

		conversationID: conf.ConversationID,
	})
	if err != nil {
		return nil, err
	}
	tr := newPreToolRetriever(&toolPreCallConf{})

	wfTools, returnDirectlyTools, err := newWorkflowTools(ctx, &workflowConfig{
		wfInfos: conf.Agent.Workflow,
	})
	if err != nil {
		return nil, err
	}

	var dbTools []einobridge.InvokableTool
	if len(conf.Agent.Database) > 0 {
		dbTools, err = newDatabaseTools(ctx, &databaseConfig{
			spaceID:       conf.Agent.SpaceID,
			userID:        conf.UserID,
			agentIdentity: conf.Identity,
			databaseConf:  conf.Agent.Database,
		})
		if err != nil {
			return nil, err
		}
	}

	var avTools []einobridge.InvokableTool
	if len(avs) > 0 {
		avTools, err = newAgentVariableTools(ctx, avConf)
		if err != nil {
			return nil, err
		}
	}
	containWfTool := false

	if len(wfTools) > 0 {
		containWfTool = true
	}
	agentTools := make([]einobridge.BaseTool, 0, len(pluginTools)+len(wfTools)+len(dbTools)+len(avTools))
	agentTools = append(agentTools, slices.Transform(pluginTools, func(a einobridge.InvokableTool) einobridge.BaseTool {
		return a
	})...)
	agentTools = append(agentTools, slices.Transform(wfTools, func(a workflow.ToolFromWorkflow) einobridge.BaseTool { return a.(einobridge.BaseTool) })...)
	agentTools = append(agentTools, slices.Transform(dbTools, func(a einobridge.InvokableTool) einobridge.BaseTool {
		return a
	})...)

	agentTools = append(agentTools, slices.Transform(avTools, func(a einobridge.InvokableTool) einobridge.BaseTool {
		return a
	})...)

	var isReActAgent bool
	if len(agentTools) > 0 {
		isReActAgent = true
		requireCheckpoint = true
		if modelInfo.Capability != nil && !modelInfo.Capability.GetFunctionCall() {
			return nil, fmt.Errorf("model %v does not support function call", modelInfo.DisplayInfo.Name)
		}
	}

	var agentGraph einobridge.AnyGraph
	var agentNodeOpts []einobridge.GraphAddNodeOpt
	var agentNodeName string
	if isReActAgent {
		result, err := buildReActGraph(ctx, chatModel, agentTools, returnDirectlyTools,
			keyOfReActAgentChatModel, keyOfReActAgentToolsNode)
		if err != nil {
			return nil, err
		}
		agentGraph = result.graph
		agentNodeOpts = result.nodeOpts

		agentNodeName = keyOfReActAgent
	} else {
		agentNodeName = keyOfLLM
	}

	suggestGraph, nsg := newSuggestGraph(ctx, conf, chatModel)

	g := einobridge.NewGraph[*AgentRequest, *wfcompose.Message](
		einobridge.WithGenLocalState(func(ctx context.Context) (state *AgentState) {
			return &AgentState{}
		}))

	_ = g.AddLambdaNode(keyOfPersonRender,
		einobridge.InvokableLambda[*AgentRequest, string](personaVars.RenderPersona),
		einobridge.WithStatePreHandler(func(ctx context.Context, ar *AgentRequest, state *AgentState) (*AgentRequest, error) {
			state.UserInput = ar.Input
			return ar, nil
		}),
		einobridge.WithOutputKey(placeholderOfPersona))

	_ = g.AddLambdaNode(keyOfPromptVariables,
		einobridge.InvokableLambda[*AgentRequest, map[string]any](promptVars.AssemblePromptVariables))

	_ = g.AddLambdaNode(keyOfKnowledgeRetriever,
		einobridge.InvokableLambda[*AgentRequest, []*wfcompose.Document](kr.Retrieve),
		einobridge.WithNodeName(keyOfKnowledgeRetriever))

	_ = g.AddLambdaNode(keyOfToolsPreRetriever,
		einobridge.InvokableLambda[*AgentRequest, []*wfcompose.Message](tr.toolPreRetrieve),
		einobridge.WithOutputKey(keyOfToolsPreRetriever),
		einobridge.WithNodeName(keyOfToolsPreRetriever),
	)
	_ = g.AddLambdaNode(keyOfKnowledgeRetrieverPack,
		einobridge.InvokableLambda[[]*wfcompose.Document, string](kr.PackRetrieveResultInfo),
		einobridge.WithOutputKey(placeholderOfKnowledge),
	)
	_ = g.AddChatTemplateNode(keyOfPromptTemplate, chatPrompt)

	agentNodeOpts = append(agentNodeOpts, einobridge.WithNodeName(agentNodeName))

	if isReActAgent {
		_ = g.AddGraphNode(agentNodeName, agentGraph, agentNodeOpts...)
	} else {
		_ = g.AddChatModelNode(agentNodeName, chatModel, agentNodeOpts...)
	}

	if nsg {
		_ = g.AddLambdaNode(keyOfSuggestPreInputParse, einobridge.ToList[*wfcompose.Message](),
			einobridge.WithStatePostHandler(func(ctx context.Context, out []*wfcompose.Message, state *AgentState) ([]*wfcompose.Message, error) {
				out = append(out, state.UserInput)
				return out, nil
			}),
		)
		_ = g.AddGraphNode(keyOfSuggestGraph, suggestGraph)
	}

	_ = g.AddEdge(einobridge.START, keyOfPersonRender)
	_ = g.AddEdge(einobridge.START, keyOfPromptVariables)
	_ = g.AddEdge(einobridge.START, keyOfKnowledgeRetriever)
	_ = g.AddEdge(einobridge.START, keyOfToolsPreRetriever)

	_ = g.AddEdge(keyOfPersonRender, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfPromptVariables, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfKnowledgeRetriever, keyOfKnowledgeRetrieverPack)
	_ = g.AddEdge(keyOfKnowledgeRetrieverPack, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfToolsPreRetriever, keyOfPromptTemplate)

	_ = g.AddEdge(keyOfPromptTemplate, agentNodeName)

	if nsg {
		_ = g.AddEdge(agentNodeName, keyOfSuggestPreInputParse)
		_ = g.AddEdge(keyOfSuggestPreInputParse, keyOfSuggestGraph)
		_ = g.AddEdge(keyOfSuggestGraph, einobridge.END)
	} else {
		_ = g.AddEdge(agentNodeName, einobridge.END)
	}

	var opts []einobridge.GraphCompileOption
	if requireCheckpoint {
		opts = append(opts, einobridge.WithCheckPointStore(conf.CPStore))
	}
	opts = append(opts, einobridge.WithNodeTriggerMode(einobridge.AllPredecessor))
	runner, err := einobridge.Compile(g, ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &AgentRunner{
		runner:              runner,
		requireCheckpoint:   requireCheckpoint,
		modelInfo:           modelInfo,
		containWfTool:       containWfTool,
		returnDirectlyTools: returnDirectlyTools,
	}, nil
}

func extractJinja2Placeholder(persona string) (variableNames []string) {
	re := regexp.MustCompile(`{{([^}]*)}}`)
	matches := re.FindAllStringSubmatch(persona, -1)
	variables := make([]string, 0, len(matches))
	for _, match := range matches {
		val := strings.TrimSpace(match[1])
		if val != "" {
			variables = append(variables, match[1])
		}
	}
	return variables
}
