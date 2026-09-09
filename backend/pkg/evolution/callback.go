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

package evolution

import (
	"context"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// startKey is a context key for storing component start time.
type startKey struct{}

// NewEvolutionCallback builds an eino einobridge.Handler that feeds
// Tool and Model execution events into the Engine's SignalCollector.
//
// The callback is designed to be registered globally:
//
//	einobridge.AppendGlobalHandlers(evolution.NewEvolutionCallback(engine))
//
// It is safe to call with a nil engine — all hooks become no-ops.
func NewEvolutionCallback(engine *Engine) einobridge.Handler {
	cb := &evolutionCallback{engine: engine}
	return einobridge.NewHandlerBuilder().
		OnStartFn(cb.onStart).
		OnEndFn(cb.onEnd).
		OnErrorFn(cb.onError).
		Build()
}

type evolutionCallback struct {
	engine *Engine
}

func (c *evolutionCallback) onStart(ctx context.Context, _ *einobridge.RunInfo, _ einobridge.CallbackInput) context.Context {
	return context.WithValue(ctx, startKey{}, time.Now())
}

func (c *evolutionCallback) onEnd(ctx context.Context, info *einobridge.RunInfo, output einobridge.CallbackOutput) context.Context {
	if c.engine == nil || info == nil {
		return ctx
	}
	start, _ := ctx.Value(startKey{}).(time.Time)
	// Convert eino RunInfo to ACL CallbackRunInfo for framework-agnostic signal building.
	aclInfo := einoToEvolutionRunInfo(info)
	sig := buildSuccessSignal(aclInfo, output, start)
	c.engine.Collector().Collect(ctx, sig)
	return ctx
}

func (c *evolutionCallback) onError(ctx context.Context, info *einobridge.RunInfo, err error) context.Context {
	if c.engine == nil || info == nil {
		return ctx
	}
	start, _ := ctx.Value(startKey{}).(time.Time)
	aclInfo := einoToEvolutionRunInfo(info)
	sig := buildErrorSignal(aclInfo, err, start)
	c.engine.Collector().Collect(ctx, sig)
	return ctx
}

// einoToEvolutionRunInfo converts eino's einobridge.RunInfo to observe.CallbackRunInfo.
func einoToEvolutionRunInfo(info *einobridge.RunInfo) observe.CallbackRunInfo {
	comp := observe.ComponentOther
	switch info.Component {
	case components.ComponentOfChatModel:
		comp = observe.ComponentChatModel
	case components.ComponentOfTool:
		comp = observe.ComponentTool
	}
	return observe.CallbackRunInfo{Component: comp, Name: info.Name}
}

// buildSuccessSignal constructs a Signal from ACL run info and eino callback output.
func buildSuccessSignal(info observe.CallbackRunInfo, output einobridge.CallbackOutput, start time.Time) Signal {
	sig := Signal{
		Component: info.Name,
		Timestamp: time.Now(),
		Duration:  durationSince(start),
	}
	switch info.Component {
	case observe.ComponentTool:
		sig.Type = "tool_success"
		sig.Output = fmt.Sprintf("%v", output)
	case observe.ComponentChatModel:
		sig.Type = "model_invoke"
		if cbOut := model.ConvCallbackOutput(output); cbOut != nil && cbOut.Message != nil {
			if cbOut.Message.Content != "" {
				sig.Output = Truncate(cbOut.Message.Content, 200)
			}
		}
	default:
		sig.Type = fmt.Sprintf("%s_success", info.Component)
	}
	return sig
}

// buildErrorSignal constructs a Signal from ACL run info and an error.
func buildErrorSignal(info observe.CallbackRunInfo, err error, start time.Time) Signal {
	sig := Signal{
		Component: info.Name,
		Timestamp: time.Now(),
		Duration:  durationSince(start),
	}
	if err != nil {
		sig.Error = err.Error()
	}
	switch info.Component {
	case observe.ComponentTool:
		sig.Type = "tool_error"
	case observe.ComponentChatModel:
		sig.Type = "model_error"
	default:
		sig.Type = fmt.Sprintf("%s_error", info.Component)
	}
	return sig
}

func durationSince(t time.Time) time.Duration {
	if t.IsZero() {
		return 0
	}
	return time.Since(t)
}

// Truncate shortens s to max runes, appending "..." if truncated.
// Uses rune conversion to avoid splitting multi-byte UTF-8 characters.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
