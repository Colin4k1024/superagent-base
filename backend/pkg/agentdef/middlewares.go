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

package agentdef

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
)

// resolveADKHandlers converts the middleware specs from the YAML definition
// into ADK ChatModelAgentMiddleware instances for use with ChatModelAgent.
// Only ADK-native middlewares (reduction, etc.) are resolved here; legacy
// middleware wrappers (timeout, retry, rate_limit, cache) are handled separately
// via applyMiddleware.
func resolveADKHandlers(ctx context.Context, specs []MiddlewareSpec) ([]adk.ChatModelAgentMiddleware, error) {
	var handlers []adk.ChatModelAgentMiddleware
	for _, spec := range specs {
		switch spec.Name {
		case "reduction":
			h, err := buildReductionHandler(ctx, spec.Config)
			if err != nil {
				return nil, fmt.Errorf("agentdef: middleware %q: %w", spec.Name, err)
			}
			handlers = append(handlers, h)
		}
	}
	return handlers, nil
}

func buildReductionHandler(ctx context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	config := &reduction.Config{
		SkipTruncation: true,
	}

	if v, ok := cfg["max_tokens"]; ok {
		switch t := v.(type) {
		case int:
			config.MaxTokensForClear = int64(t)
		case float64:
			config.MaxTokensForClear = int64(t)
		}
	}
	if v, ok := cfg["max_length_for_trunc"]; ok {
		config.SkipTruncation = false
		switch t := v.(type) {
		case int:
			config.MaxLengthForTrunc = t
		case float64:
			config.MaxLengthForTrunc = int(t)
		}
	}
	if v, ok := cfg["retention_suffix"]; ok {
		switch t := v.(type) {
		case int:
			config.ClearRetentionSuffixLimit = t
		case float64:
			config.ClearRetentionSuffixLimit = int(t)
		}
	}

	return reduction.New(ctx, config)
}
