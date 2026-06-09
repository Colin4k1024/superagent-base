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

	"github.com/superagent-ai/superagent-base/backend/pkg/tool/sandbox"
)

// resolveADKHandlers converts the middleware specs from the YAML definition
// into ADK ChatModelAgentMiddleware instances for use with ChatModelAgent.
// Only ADK-native middlewares (reduction, sandbox, etc.) are resolved here;
// legacy middleware wrappers (timeout, retry, rate_limit, cache) are handled
// separately via applyMiddleware.
//
// When sandboxSpec is non-nil and enabled, a sandbox middleware is automatically
// injected even if not explicitly listed in specs.
func resolveADKHandlers(ctx context.Context, specs []MiddlewareSpec, sandboxSpec *SandboxSpec, toolRefs []ToolRef) ([]adk.ChatModelAgentMiddleware, error) {
	var handlers []adk.ChatModelAgentMiddleware

	// Auto-inject sandbox middleware when spec.sandbox.enabled is true.
	if sandboxSpec != nil && sandboxSpec.Enabled {
		h, err := buildSandboxHandler(ctx, sandboxSpec, toolRefs)
		if err != nil {
			return nil, fmt.Errorf("agentdef: sandbox middleware: %w", err)
		}
		handlers = append(handlers, h)
	}

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

// buildSandboxHandler creates a sandbox middleware from the agent's SandboxSpec.
func buildSandboxHandler(ctx context.Context, spec *SandboxSpec, toolRefs []ToolRef) (adk.ChatModelAgentMiddleware, error) {
	defaultPolicy := policyFromSandboxSpec(spec)

	// Create sandbox backend with auto-fallback.
	backend, err := sandbox.NewBackend(ctx, spec.Backend, defaultPolicy)
	if err != nil {
		return nil, fmt.Errorf("create sandbox backend %q: %w", spec.Backend, err)
	}

	// Extract per-tool sandbox overrides from ToolRef.Config.
	perToolPolicy := extractPerToolPolicies(toolRefs)

	return newSandboxMiddleware(backend, defaultPolicy, perToolPolicy), nil
}

// extractPerToolPolicies reads sandbox policy overrides from each ToolRef's Config map.
func extractPerToolPolicies(toolRefs []ToolRef) map[string]*sandbox.Policy {
	policies := make(map[string]*sandbox.Policy)
	for _, ref := range toolRefs {
		if ref.Config == nil {
			continue
		}
		sbCfg, ok := ref.Config["sandbox"]
		if !ok {
			continue
		}
		cfgMap, ok := sbCfg.(map[string]any)
		if !ok {
			continue
		}
		p := &sandbox.Policy{TimeoutSeconds: 30, MemoryLimitMB: 256}
		if v, ok := cfgMap["timeout_seconds"]; ok {
			switch t := v.(type) {
			case int:
				p.TimeoutSeconds = t
			case float64:
				p.TimeoutSeconds = int(t)
			}
		}
		if v, ok := cfgMap["memory_limit_mb"]; ok {
			switch t := v.(type) {
			case int:
				p.MemoryLimitMB = int64(t)
			case float64:
				p.MemoryLimitMB = int64(t)
			case int64:
				p.MemoryLimitMB = t
			}
		}
		if v, ok := cfgMap["allow_net"]; ok {
			if arr, ok := v.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						p.AllowNet = append(p.AllowNet, s)
					}
				}
			}
		}
		if v, ok := cfgMap["allow_read"]; ok {
			if arr, ok := v.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						p.AllowRead = append(p.AllowRead, s)
					}
				}
			}
		}
		if v, ok := cfgMap["allow_write"]; ok {
			if arr, ok := v.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						p.AllowWrite = append(p.AllowWrite, s)
					}
				}
			}
		}
		if v, ok := cfgMap["allow_env"]; ok {
			if arr, ok := v.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						p.AllowEnv = append(p.AllowEnv, s)
					}
				}
			}
		}
		// Extract tool name from ref URI.
		toolName := extractToolName(ref.Ref)
		if toolName != "" {
			policies[toolName] = p
		}
	}
	return policies
}

// extractToolName returns the tool name from a ref URI.
func extractToolName(ref string) string {
	// builtin/web_search → web_search
	// mcp://server/tool → tool
	// skill://name → name
	for _, prefix := range []string{"builtin/", "mcp://", "skill://"} {
		if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
			remainder := ref[len(prefix):]
			// For mcp://, take the last part after /
			for i := len(remainder) - 1; i >= 0; i-- {
				if remainder[i] == '/' {
					return remainder[i+1:]
				}
			}
			return remainder
		}
	}
	return ref
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
