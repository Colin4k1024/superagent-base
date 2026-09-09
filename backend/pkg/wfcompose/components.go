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

package wfcompose

import "context"

// Component is the type identifier for a pipeline component.
type Component string

const (
	ComponentOfPrompt       Component = "prompt"
	ComponentOfAgenticPrompt Component = "agentic_prompt"
	ComponentOfChatModel    Component = "chat_model"
	ComponentOfAgenticModel Component = "agentic_model"
	ComponentOfEmbedding    Component = "embedding"
	ComponentOfIndexer      Component = "indexer"
	ComponentOfRetriever    Component = "retriever"
	ComponentOfLoader       Component = "loader"
	ComponentOfTransformer  Component = "transformer"
	ComponentOfTool        Component = "tool"
	ComponentOfGraph        Component = "graph"
	ComponentOfToolsNode    Component = "tools_node"
	ComponentOfLambda       Component = "lambda"
	ComponentOfWorkflow     Component = "workflow"
	ComponentOfChain        Component = "chain"
	ComponentOfUnknown      Component = "unknown"
)

// ModelOptions holds call-time options for a ChatModel.
type ModelOptions struct {
	Temperature      *float32
	Model            *string
	TopP             *float32
	Tools            []*ToolInfo
	MaxTokens        *int
	Stop             []string
	ToolChoice       *string
	AllowedToolNames []string
}

// ModelOption is a call-time option for a ChatModel.
type ModelOption struct {
	apply func(opts *ModelOptions)
}

func NewModelOption(apply func(*ModelOptions)) ModelOption {
	return ModelOption{apply: apply}
}

func (o ModelOption) Apply(opts *ModelOptions) {
	if o.apply != nil {
		o.apply(opts)
	}
}

// WithTemperature sets the temperature for the model call.
func WithTemperature(temp float32) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.Temperature = &temp })
}

// WithMaxTokens sets the max tokens for the model call.
func WithMaxTokens(maxTokens int) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.MaxTokens = &maxTokens })
}

// WithModel sets the model name for the model call.
func WithModel(name string) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.Model = &name })
}

// WithTopP sets the top-p for the model call.
func WithTopP(topP float32) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.TopP = &topP })
}

// WithTools sets the tools for the model call.
func WithTools(tools []*ToolInfo) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.Tools = tools })
}

// WithStop sets the stop words for the model call.
func WithStop(stop []string) ModelOption {
	return NewModelOption(func(o *ModelOptions) { o.Stop = stop })
}

// WithToolChoice sets the tool choice for the model call.
func WithToolChoice(choice string, allowedNames ...string) ModelOption {
	return NewModelOption(func(o *ModelOptions) {
		o.ToolChoice = &choice
		o.AllowedToolNames = allowedNames
	})
}

// GetModelOptions applies ModelOptions to a base and returns the merged result.
func GetModelOptions(base *ModelOptions, opts ...ModelOption) *ModelOptions {
	if base == nil {
		base = &ModelOptions{}
	}
	for _, opt := range opts {
		opt.Apply(base)
	}
	return base
}

// ChatModel is the interface for a non-streaming chat model.
type ChatModel interface {
	Generate(ctx context.Context, input []*Message, opts ...ModelOption) (*Message, error)
	Stream(ctx context.Context, input []*Message, opts ...ModelOption) (*StreamReader[*Message], error)
}

// ToolCallingChatModel extends ChatModel with tool-binding support.
type ToolCallingChatModel interface {
	ChatModel
	WithTools(tools []*ToolInfo) (ToolCallingChatModel, error)
}

// BindToolser is an optional interface that ChatModel implementations may
// satisfy to bind tools to the model.
type BindToolser interface {
	BindTools(tools []*ToolInfo) error
}

// ToolOption is a call-time option for a Tool.
type ToolOption struct {
	apply func(*map[string]any)
}

// ToolInfo describes a tool's metadata.
// (Defined in schema_types.go; here we add the tool component interface.)

// InvokableTool is a tool that can be invoked synchronously.
type InvokableTool interface {
	Info(ctx context.Context) (*ToolInfo, error)
	InvokableRun(ctx context.Context, argsJSON string, opts ...ToolOption) (string, error)
}

// BaseTool is the base interface for all tools.
type BaseTool interface {
	Info(ctx context.Context) (*ToolInfo, error)
}

// Embedder is the interface for an embedding model.
type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string, opts ...EmbeddingOption) ([][]float64, error)
	EmbedQueries(ctx context.Context, queries []string, opts ...EmbeddingOption) ([][]float64, error)
}

// EmbeddingOption is a call-time option for an Embedder.
type EmbeddingOption struct {
	apply func(*EmbeddingOptions)
}

type EmbeddingOptions struct {
	Model string
}

func NewEmbeddingOption(apply func(*EmbeddingOptions)) EmbeddingOption {
	return EmbeddingOption{apply: apply}
}

func (o EmbeddingOption) Apply(opts *EmbeddingOptions) {
	if o.apply != nil {
		o.apply(opts)
	}
}
