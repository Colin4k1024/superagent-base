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

// components_facade re-exports cloudwego/eino/components and its
// sub-packages as type aliases so callers avoid importing eino directly.
package einobridge

import (
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// ---------------------------------------------------------------------------
// eino/components
// ---------------------------------------------------------------------------

type Component = components.Component

const (
	ComponentOfPrompt        Component = components.ComponentOfPrompt
	ComponentOfAgenticPrompt Component = components.ComponentOfAgenticPrompt
	ComponentOfChatModel     Component = components.ComponentOfChatModel
	ComponentOfAgenticModel  Component = components.ComponentOfAgenticModel
	ComponentOfEmbedding     Component = components.ComponentOfEmbedding
	ComponentOfIndexer       Component = components.ComponentOfIndexer
	ComponentOfRetriever     Component = components.ComponentOfRetriever
	ComponentOfLoader        Component = components.ComponentOfLoader
	ComponentOfTransformer   Component = components.ComponentOfTransformer
	ComponentOfTool           Component = components.ComponentOfTool
)

func IsCallbacksEnabled(component any) bool {
	return components.IsCallbacksEnabled(component)
}

// ---------------------------------------------------------------------------
// eino/components/model
// ---------------------------------------------------------------------------

type BaseChatModel = model.BaseChatModel
type ToolCallingChatModel = model.ToolCallingChatModel
type ModelOption = model.Option
type ModelCallbackInput = model.CallbackInput
type ModelCallbackOutput = model.CallbackOutput
type ModelTokenUsage = model.TokenUsage
type PromptTokenDetails = model.PromptTokenDetails

func ModelConvCallbackInput(src CallbackInput) *ModelCallbackInput {
	return model.ConvCallbackInput(src)
}

func ModelConvCallbackOutput(src CallbackOutput) *ModelCallbackOutput {
	return model.ConvCallbackOutput(src)
}

// ---------------------------------------------------------------------------
// eino/components/tool
// ---------------------------------------------------------------------------

type BaseTool = tool.BaseTool
type InvokableTool = tool.InvokableTool
type ToolOption = tool.Option
type ToolCallbackInput = tool.CallbackInput
type ToolCallbackOutput = tool.CallbackOutput

func ToolConvCallbackInput(src CallbackInput) *ToolCallbackInput {
	return tool.ConvCallbackInput(src)
}

func ToolConvCallbackOutput(src CallbackOutput) *ToolCallbackOutput {
	return tool.ConvCallbackOutput(src)
}

// ---------------------------------------------------------------------------
// eino/components/prompt
// ---------------------------------------------------------------------------

type ChatTemplate = prompt.ChatTemplate
type PromptOption = prompt.Option
type DefaultChatTemplate = prompt.DefaultChatTemplate

func PromptFromMessages(formatType FormatType, templates ...MessagesTemplate) *DefaultChatTemplate {
	return prompt.FromMessages(formatType, templates...)
}

// ---------------------------------------------------------------------------
// eino/components/retriever
// ---------------------------------------------------------------------------

type Retriever = retriever.Retriever
type RetrieverOption = retriever.Option
type RetrieverOptions = retriever.Options

func WithDSLInfo(dsl map[string]any) RetrieverOption {
	return retriever.WithDSLInfo(dsl)
}

// ---------------------------------------------------------------------------
// eino/components/indexer
// ---------------------------------------------------------------------------

type Indexer = indexer.Indexer
type IndexerOption = indexer.Option

// ---------------------------------------------------------------------------
// eino/components/embedding
// ---------------------------------------------------------------------------

type Embedder = embedding.Embedder
type EmbeddingOption = embedding.Option

// ---------------------------------------------------------------------------
// eino/components/document/parser
// ---------------------------------------------------------------------------

type DocParser = parser.Parser
type ParserOption = parser.Option
type ParserOptions = parser.Options

func WithExtraMeta(meta map[string]any) ParserOption {
	return parser.WithExtraMeta(meta)
}

// ---------------------------------------------------------------------------
// eino/components/tool/utils
// ---------------------------------------------------------------------------

type SchemaModifierFn = toolutils.SchemaModifierFn
type ToolUtilsOption = toolutils.Option

func WithSchemaModifier(modifier SchemaModifierFn) ToolUtilsOption {
	return toolutils.WithSchemaModifier(modifier)
}

func InferTool[T, D any](toolName, toolDesc string, i toolutils.InvokeFunc[T, D], opts ...ToolUtilsOption) (tool.InvokableTool, error) {
	return toolutils.InferTool[T, D](toolName, toolDesc, i, opts...)
}
