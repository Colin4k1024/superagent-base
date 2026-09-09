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

// Package observe provides observability (tracing + metrics) for agent
// execution.  This file is a thin adapter that bridges eino's callback
// system to the framework-agnostic ObservabilityCallback interface.
// The core logic lives in observe_callback_impl.go.

package observe

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
)

// EinoObserveCallback adapts the ACL ObserveCallback to eino's callbacks.Handler.
type EinoObserveCallback struct {
	acl *ObserveCallback
}

// NewEinoObserveCallback creates an eino callbacks.Handler backed by the
// framework-agnostic ObserveCallback.  The returned handler is registered
// with the eino framework; all events are converted to ACL types and
// delegated to ObserveCallback.
func NewEinoObserveCallback() callbacks.Handler {
	acl := NewObserveCallback()
	cb := &EinoObserveCallback{acl: acl}
	SetCallback(acl) // register as global ACL callback
	return callbacks.NewHandlerBuilder().
		OnStartFn(cb.OnStart).
		OnEndFn(cb.OnEnd).
		OnEndWithStreamOutputFn(cb.OnEndWithStream).
		OnErrorFn(cb.OnError).
		Build()
}

// einoToACLRunInfo converts eino's callbacks.RunInfo to the ACL CallbackRunInfo.
func einoToACLRunInfo(info *callbacks.RunInfo) CallbackRunInfo {
	comp := ComponentOther
	switch info.Component {
	case components.ComponentOfChatModel:
		comp = ComponentChatModel
	case components.ComponentOfTool:
		comp = ComponentTool
	}
	return CallbackRunInfo{Component: comp, Name: info.Name}
}

// einoToACLInput converts eino's callbacks.CallbackInput to the ACL CallbackInput.
func einoToACLInput(input callbacks.CallbackInput) CallbackInput {
	cbIn := model.ConvCallbackInput(input)
	if cbIn != nil && cbIn.Messages != nil {
		msgs := make([]*einoMsg, len(cbIn.Messages))
		for i, m := range cbIn.Messages {
			msgs[i] = (*einoMsg)(m)
		}
		return CallbackInput{Raw: msgs}
	}
	return CallbackInput{Raw: input}
}

// einoToACLOutput converts eino's callbacks.CallbackOutput to the ACL CallbackOutput.
func einoToACLOutput(output callbacks.CallbackOutput) CallbackOutput {
	cbOut := model.ConvCallbackOutput(output)
	if cbOut != nil && cbOut.Message != nil {
		out := CallbackOutput{Raw: output}
		if cbOut.Message.ResponseMeta != nil && cbOut.Message.ResponseMeta.Usage != nil {
			out.PromptTokens = cbOut.Message.ResponseMeta.Usage.PromptTokens
			out.CompletionTokens = cbOut.Message.ResponseMeta.Usage.CompletionTokens
		}
		return out
	}
	return CallbackOutput{Raw: output}
}

func (c *EinoObserveCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	return c.acl.OnStart(ctx, einoToACLRunInfo(info), einoToACLInput(input))
}

func (c *EinoObserveCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	return c.acl.OnEnd(ctx, einoToACLRunInfo(info), einoToACLOutput(output))
}

func (c *EinoObserveCallback) OnEndWithStream(ctx context.Context, info *callbacks.RunInfo, output *einobridge.StreamReader[callbacks.CallbackOutput]) context.Context {
	// For streaming, we drain the eino stream reader to collect token counts.
	// The ACL OnEndWithStream expects an llm.StreamReader, but we can't
	// directly convert the eino stream reader.  Instead, we drain it here
	// and call OnEnd with the collected data.
	go func() {
		var promptTokens, completionTokens int
		if output != nil {
			for {
				chunk, err := output.Recv()
				if err != nil {
					break
				}
				cbOut := model.ConvCallbackOutput(chunk)
				if cbOut != nil && cbOut.Message != nil {
					if cbOut.Message.ResponseMeta != nil && cbOut.Message.ResponseMeta.Usage != nil {
						promptTokens = cbOut.Message.ResponseMeta.Usage.PromptTokens
						completionTokens = cbOut.Message.ResponseMeta.Usage.CompletionTokens
					}
					completionTokens++
				}
			}
		}
		aclInfo := einoToACLRunInfo(info)
		out := CallbackOutput{PromptTokens: promptTokens, CompletionTokens: completionTokens}
		c.acl.OnEnd(ctx, aclInfo, out)
	}()
	return ctx
}

func (c *EinoObserveCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	return c.acl.OnError(ctx, einoToACLRunInfo(info), err)
}

// einoMsg is a type alias for einobridge.Message to avoid direct schema import
// in the CallbackInput.Raw field.  Callers that need the actual message
// should type-assert to *einobridge.Message.
type einoMsg = einobridge.Message
