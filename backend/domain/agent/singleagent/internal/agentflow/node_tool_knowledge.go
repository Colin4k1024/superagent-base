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
	"fmt"
	"reflect"
	"strconv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"

	"github.com/superagent-ai/superagent-base/backend/api/model/app/bot_common"
	crossknowledge "github.com/superagent-ai/superagent-base/backend/crossdomain/knowledge"
	knowledgeEntity "github.com/superagent-ai/superagent-base/backend/domain/knowledge/entity"
)

const (
	knowledgeToolName = "recallKnowledge"
	knowledgeDesc     = `Provides multiple retrieval methods to search for content fragments stored in the knowledge base, helping the large model obtain more accurate and reliable context information`
)

type knowledgeConfig struct {
	knowledgeInfos  []*knowledgeEntity.Knowledge
	knowledgeConfig *bot_common.Knowledge
	Input           *schema.Message
	GetHistory      func() []*schema.Message
}

func newKnowledgeTool(ctx context.Context, conf *knowledgeConfig) (tool.InvokableTool, error) {
	kl := &knowledgeTool{
		knowledgeConfig: conf.knowledgeConfig,
		Input:           conf.Input,
		GetHistory:      conf.GetHistory,
	}

	customTagsFn := func(jsonTagName string, t reflect.Type, tag reflect.StructTag,
		sc *jsonschema.Schema,
	) {
		// Process KnowledgeIDs field only
		if jsonTagName != "KnowledgeIDs" {
			return
		}

		// Build knowledge base description
		desc := "Available Knowledge Base List as format knowledge_id: knowledge_name - knowledge_description: \n"
		for _, k := range conf.knowledgeInfos {
			desc += fmt.Sprintf("- %d: %s - %s\n", k.ID, k.Name, k.Description)
		}

		sc.Type = "array"
		sc.Items = &jsonschema.Schema{
			Type: "integer",
		}
		// Set field descriptions and enumeration values
		sc.Description = desc
		sc.Enum = make([]interface{}, 0, len(conf.knowledgeInfos))
		for _, k := range conf.knowledgeInfos {
			sc.Enum = append(sc.Enum, strconv.FormatInt(k.ID, 10))
		}
	}

	return utils.InferTool(knowledgeToolName, knowledgeDesc, kl.Retrieve, utils.WithSchemaModifier(customTagsFn))
}

type RetrieveRequest struct {
	KnowledgeIDs []int64 `json:"knowledge_ids" jsonschema:"description="`
}

type knowledgeTool struct {
	knowledgeConfig *bot_common.Knowledge
	Input           *schema.Message
	GetHistory      func() []*schema.Message
}

func (k *knowledgeTool) Retrieve(ctx context.Context, req *RetrieveRequest) ([]*schema.Document, error) {
	rr, err := genKnowledgeRequest(ctx, req.KnowledgeIDs, k.knowledgeConfig, k.Input.Content, k.GetHistory())
	if err != nil {
		return nil, err
	}

	resp, err := crossknowledge.DefaultSVC().Retrieve(ctx, rr)
	if err != nil {
		return nil, err
	}

	docs, err := convertDocument(ctx, resp.RetrieveSlices)
	if err != nil {
		return nil, err
	}

	return docs, nil
}
