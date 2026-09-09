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

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"github.com/cloudwego/eino/components/document/parser"
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

type Embedder = wfcompose.Embedder
type EmbeddingOption = wfcompose.EmbeddingOption

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

// ---------------------------------------------------------------------------
// Generic function wrappers for tool package
// ---------------------------------------------------------------------------

func ToolGetImplSpecificOptions[T any](base *T, opts ...ToolOption) *T {
	return tool.GetImplSpecificOptions[T](base, opts...)
}

func ToolWrapImplSpecificOptFn[T any](optFn func(*T)) ToolOption {
	return tool.WrapImplSpecificOptFn[T](optFn)
}

// ---------------------------------------------------------------------------
// Generic function wrappers for retriever package
// ---------------------------------------------------------------------------

func RetrieverGetImplSpecificOptions[T any](base *T, opts ...RetrieverOption) *T {
	return retriever.GetImplSpecificOptions[T](base, opts...)
}

func RetrieverGetCommonOptions(base *RetrieverOptions, opts ...RetrieverOption) *RetrieverOptions {
	return retriever.GetCommonOptions(base, opts...)
}

func RetrieverWrapImplSpecificOptFn[T any](optFn func(*T)) RetrieverOption {
	return retriever.WrapImplSpecificOptFn[T](optFn)
}

// ---------------------------------------------------------------------------
// Generic function wrappers for indexer package
// ---------------------------------------------------------------------------

func IndexerGetImplSpecificOptions[T any](base *T, opts ...IndexerOption) *T {
	return indexer.GetImplSpecificOptions[T](base, opts...)
}

func IndexerWrapImplSpecificOptFn[T any](optFn func(*T)) IndexerOption {
	return indexer.WrapImplSpecificOptFn[T](optFn)
}

// ---------------------------------------------------------------------------
// Generic function wrappers for embedding package
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Generic function wrappers for parser package
// ---------------------------------------------------------------------------

func ParserGetCommonOptions(base *ParserOptions, opts ...ParserOption) *ParserOptions {
	return parser.GetCommonOptions(base, opts...)
}

func ParserGetImplSpecificOptions[T any](base *T, opts ...ParserOption) *T {
	return parser.GetImplSpecificOptions[T](base, opts...)
}

// ---------------------------------------------------------------------------
// Additional wrappers for document parser
// ---------------------------------------------------------------------------

const MetaKeySource = parser.MetaKeySource

type ExtParser = parser.ExtParser
type ExtParserConfig = parser.ExtParserConfig
type TextParser = parser.TextParser

// RetrieverConvCallbackOutput converts a callback output to a retriever callback output.
func RetrieverConvCallbackOutput(src CallbackOutput) *retriever.CallbackOutput {
	return retriever.ConvCallbackOutput(src)
}

func RetrieverConvCallbackInput(src CallbackInput) *retriever.CallbackInput {
	return retriever.ConvCallbackInput(src)
}

// EmbeddingConvCallbackOutput and EmbeddingConvCallbackInput removed (unused, eino decoupling).
