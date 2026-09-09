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

// components_facade re-exports wfcompose component types as prefixed aliases.
// This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ---------------------------------------------------------------------------
// Component types
// ---------------------------------------------------------------------------

type Component = wfcompose.Component

const (
	ComponentOfPrompt        Component = wfcompose.ComponentOfPrompt
	ComponentOfAgenticPrompt Component = wfcompose.ComponentOfAgenticPrompt
	ComponentOfChatModel     Component = wfcompose.ComponentOfChatModel
	ComponentOfAgenticModel  Component = wfcompose.ComponentOfAgenticModel
	ComponentOfEmbedding     Component = wfcompose.ComponentOfEmbedding
	ComponentOfIndexer       Component = wfcompose.ComponentOfIndexer
	ComponentOfRetriever     Component = wfcompose.ComponentOfRetriever
	ComponentOfLoader        Component = wfcompose.ComponentOfLoader
	ComponentOfTransformer   Component = wfcompose.ComponentOfTransformer
	ComponentOfTool          Component = wfcompose.ComponentOfTool
)

func IsCallbacksEnabled(component any) bool {
	type callbacksEnabler interface {
		IsCallbacksEnabled() bool
	}
	if ce, ok := component.(callbacksEnabler); ok {
		return ce.IsCallbacksEnabled()
	}
	return false
}

// ---------------------------------------------------------------------------
// Model types
// ---------------------------------------------------------------------------

type BaseChatModel = wfcompose.ChatModel
type ToolCallingChatModel = wfcompose.ToolCallingChatModel
type ModelOption = wfcompose.ModelOption
type ModelCallbackInput = wfcompose.ModelCallbackInput
type ModelCallbackOutput = wfcompose.ModelCallbackOutput
type ModelTokenUsage = wfcompose.TokenUsage
type PromptTokenDetails = wfcompose.PromptTokenDetails

func ModelConvCallbackInput(src CallbackInput) *ModelCallbackInput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*ModelCallbackInput); ok {
		return v
	}
	if v, ok := src.(ModelCallbackInput); ok {
		return &v
	}
	return nil
}

func ModelConvCallbackOutput(src CallbackOutput) *ModelCallbackOutput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*ModelCallbackOutput); ok {
		return v
	}
	if v, ok := src.(ModelCallbackOutput); ok {
		return &v
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tool types
// ---------------------------------------------------------------------------

type BaseTool = wfcompose.BaseTool
type InvokableTool = wfcompose.InvokableTool
type ToolOption = wfcompose.ToolOption
type ToolCallbackInput struct {
	ArgumentsInJSON string
	Extra           map[string]any
}
type ToolCallbackOutput struct {
	ArgumentsInJSON string
	Response        string
	Extra           map[string]any
}

func ToolConvCallbackInput(src CallbackInput) *ToolCallbackInput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*ToolCallbackInput); ok {
		return v
	}
	if v, ok := src.(ToolCallbackInput); ok {
		return &v
	}
	return nil
}

func ToolConvCallbackOutput(src CallbackOutput) *ToolCallbackOutput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*ToolCallbackOutput); ok {
		return v
	}
	if v, ok := src.(ToolCallbackOutput); ok {
		return &v
	}
	return nil
}

func ToolGetImplSpecificOptions[T any](base *T, opts ...ToolOption) *T {
	return base
}

func ToolWrapImplSpecificOptFn[T any](optFn func(*T)) ToolOption {
	return wfcompose.NewToolOption(func(m *map[string]any) {})
}

// ---------------------------------------------------------------------------
// Prompt types
// ---------------------------------------------------------------------------

type ChatTemplate = wfcompose.ChatTemplate
type PromptOption = wfcompose.PromptOption
type DefaultChatTemplate = wfcompose.DefaultChatTemplate

func PromptFromMessages(formatType FormatType, templates ...MessagesTemplate) *DefaultChatTemplate {
	return wfcompose.FromMessages(wfcompose.FormatType(formatType), templates...)
}

// ---------------------------------------------------------------------------
// Retriever types
// ---------------------------------------------------------------------------

type Retriever = wfcompose.Retriever
type RetrieverOption = wfcompose.RetrieverOption
type RetrieverOptions = wfcompose.RetrieverOptions

func WithDSLInfo(dsl map[string]any) RetrieverOption {
	return wfcompose.WithDSLInfo(dsl)
}

func RetrieverGetImplSpecificOptions[T any](base *T, opts ...RetrieverOption) *T {
	return base
}

func RetrieverGetCommonOptions(base *RetrieverOptions, opts ...RetrieverOption) *RetrieverOptions {
	return wfcompose.GetRetrieverOptions(base, opts...)
}

func RetrieverWrapImplSpecificOptFn[T any](optFn func(*T)) RetrieverOption {
	return wfcompose.NewRetrieverOption(func(o *wfcompose.RetrieverOptions) {})
}



// ---------------------------------------------------------------------------
// Indexer types
// ---------------------------------------------------------------------------

type Indexer = wfcompose.Indexer
type IndexerOption = wfcompose.IndexerOption

func IndexerGetImplSpecificOptions[T any](base *T, opts ...IndexerOption) *T {
	return base
}

func IndexerWrapImplSpecificOptFn[T any](optFn func(*T)) IndexerOption {
	return wfcompose.NewIndexerOption(func(any) {})
}

// ---------------------------------------------------------------------------
// Embedding types
// ---------------------------------------------------------------------------

type Embedder = wfcompose.Embedder
type EmbeddingOption = wfcompose.EmbeddingOption

// ---------------------------------------------------------------------------
// Document Parser types
// ---------------------------------------------------------------------------

type DocParser = wfcompose.DocParser
type ParserOption = wfcompose.DocParserOption
type ParserOptions = wfcompose.DocParserOptions

func WithExtraMeta(meta map[string]any) ParserOption {
	return wfcompose.WithExtraMeta(meta)
}

func ParserGetCommonOptions(base *ParserOptions, opts ...ParserOption) *ParserOptions {
	return base
}

func ParserGetImplSpecificOptions[T any](base *T, opts ...ParserOption) *T {
	return base
}

const MetaKeySource = wfcompose.MetaKeySource

type ExtParser = wfcompose.ExtParser
type ExtParserConfig = wfcompose.ExtParserConfig
type TextParser = struct{}

// ---------------------------------------------------------------------------
// Tool utils types (stubs)
// ---------------------------------------------------------------------------

type SchemaModifierFn = any
type ToolUtilsOption = func(any)

func WithSchemaModifier(modifier any) ToolUtilsOption {
	return func(any) {}
}

type InvokeFuncT[I, D any] = func(ctx context.Context, input I) (D, error)

// InferTool creates an invokable tool from a name, description, and invoke function.
func InferTool[I, D any](toolName, toolDesc string, i InvokeFuncT[I, D], opts ...ToolUtilsOption) (InvokableTool, error) {
	return nil, nil
}

// RetrieverCallbackOutput is the callback output for retriever operations.
type RetrieverCallbackOutput struct {
	Docs []*wfcompose.Document
}

// RetrieverConvCallbackOutput converts a CallbackOutput to *RetrieverCallbackOutput.
func RetrieverConvCallbackOutput(src CallbackOutput) *RetrieverCallbackOutput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*RetrieverCallbackOutput); ok {
		return v
	}
	if v, ok := src.(RetrieverCallbackOutput); ok {
		return &v
	}
	return nil
}

// RetrieverConvCallbackInput converts a CallbackInput to *RetrieverCallbackInput.
type RetrieverCallbackInput struct {
	Query  string
	Extras map[string]any
}

func RetrieverConvCallbackInput(src CallbackInput) *RetrieverCallbackInput {
	if src == nil {
		return nil
	}
	if v, ok := src.(*RetrieverCallbackInput); ok {
		return v
	}
	if v, ok := src.(RetrieverCallbackInput); ok {
		return &v
	}
	return nil
}
