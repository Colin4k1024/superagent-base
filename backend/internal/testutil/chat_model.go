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

package testutil

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"

	"github.com/superagent-ai/superagent-base/backend/api/model/admin/config"
	"github.com/superagent-ai/superagent-base/backend/api/model/app/developer_api"
	"github.com/superagent-ai/superagent-base/backend/bizpkg/config/modelmgr"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

type UTChatModel struct {
	InvokeResultProvider func(index int, in []*einobridge.Message) (*einobridge.Message, error)
	StreamResultProvider func(index int, in []*einobridge.Message) (*einobridge.StreamReader[*einobridge.Message], error)
	Modals               *developer_api.ModelAbility
	Index                int
	ModelType            string
	mu                   sync.Mutex
}

func (q *UTChatModel) Generate(ctx context.Context, in []*einobridge.Message, _ ...model.Option) (*einobridge.Message, error) {
	ctx = einobridge.EnsureRunInfo(ctx, "ut_chat_model", components.ComponentOfChatModel)
	ctx = einobridge.OnStart(ctx, in)
	defer func() {
		q.mu.Lock()
		q.Index++
		q.mu.Unlock()
	}()
	defer func() {
		if panicErr := recover(); panicErr != nil {
			logs.CtxErrorf(ctx, "ut Chat Model: %s, panic: %v, stack: %s", q.ModelType, panicErr, string(debug.Stack()))
			einobridge.OnError(ctx, fmt.Errorf("model: %s, panic: %v, stack: %s", q.ModelType, panicErr, string(debug.Stack())))
		}
	}()

	if q.InvokeResultProvider == nil {
		return nil, fmt.Errorf("invoke result provider is nil")
	}

	q.mu.Lock()
	msg, err := q.InvokeResultProvider(q.Index, in)
	q.mu.Unlock()
	if err != nil {
		einobridge.OnError(ctx, err)
		return nil, err
	}

	callbackOut := &model.CallbackOutput{
		Message: msg,
	}

	if msg.ResponseMeta != nil {
		callbackOut.TokenUsage = &model.TokenUsage{
			PromptTokens: msg.ResponseMeta.Usage.PromptTokens,
			PromptTokenDetails: model.PromptTokenDetails{
				CachedTokens: msg.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
			},
			CompletionTokens: msg.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      msg.ResponseMeta.Usage.TotalTokens,
		}
	}

	_ = einobridge.OnEnd(ctx, callbackOut)
	return msg, nil
}

func (q *UTChatModel) Stream(ctx context.Context, in []*einobridge.Message, _ ...model.Option) (*einobridge.StreamReader[*einobridge.Message], error) {
	ctx = einobridge.EnsureRunInfo(ctx, "ut_chat_model", components.ComponentOfChatModel)
	ctx = einobridge.OnStart(ctx, in)
	defer func() {
		q.mu.Lock()
		q.Index++
		q.mu.Unlock()
	}()
	defer func() {
		if panicErr := recover(); panicErr != nil {
			logs.CtxErrorf(ctx, "ut Chat Model: %s, panic: %v, stack: %s", q.ModelType, panicErr, string(debug.Stack()))
			einobridge.OnError(ctx, fmt.Errorf("model: %s, panic: %v, stack: %s", q.ModelType, panicErr, string(debug.Stack())))
		}
	}()

	if q.StreamResultProvider == nil {
		return nil, fmt.Errorf("stream result provider is nil")
	}

	q.mu.Lock()
	outS, err := q.StreamResultProvider(q.Index, in)
	q.mu.Unlock()
	if err != nil {
		einobridge.OnError(ctx, err)
		return nil, err
	}

	callbackStream := einobridge.StreamReaderWithConvert(outS, func(t *einobridge.Message) (*model.CallbackOutput, error) {
		callbackOut := &model.CallbackOutput{
			Message: t,
		}

		if t.ResponseMeta != nil {
			callbackOut.TokenUsage = &model.TokenUsage{
				PromptTokens: t.ResponseMeta.Usage.PromptTokens,
				PromptTokenDetails: model.PromptTokenDetails{
					CachedTokens: t.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
				},
				CompletionTokens: t.ResponseMeta.Usage.CompletionTokens,
				TotalTokens:      t.ResponseMeta.Usage.TotalTokens,
			}
		}

		return callbackOut, nil
	})
	_, s := einobridge.OnEndWithStreamOutput(ctx, callbackStream)
	return einobridge.StreamReaderWithConvert(s, func(t *model.CallbackOutput) (*einobridge.Message, error) {
		return t.Message, nil
	}), nil
}

func (q *UTChatModel) WithTools(tools []*einobridge.ToolInfo) (model.ToolCallingChatModel, error) {
	return q, nil
}

func (q *UTChatModel) IsCallbacksEnabled() bool {
	return true
}

func (q *UTChatModel) Reset() {
	q.Index = 0
}

func (q *UTChatModel) Info(_ context.Context) *modelmgr.Model {
	return &modelmgr.Model{
		Model: &config.Model{
			Capability: q.Modals,
		},
	}
}
