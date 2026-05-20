---
name: model-routing
description: >
  Guide for configuring and debugging the model routing layer in Superagent.
  TRIGGER when: adding a new model provider, configuring fallback chains,
  tuning cost/latency routing strategy, debugging "model not found" errors,
  or when the user asks "how does model routing work", "模型路由怎么配置",
  "add a new LLM provider".
  DO NOT TRIGGER when: writing agent YAML (use agent-yaml-authoring),
  implementing tool logic, or working on the UI.
origin: learned
tags: [model, routing, llm, provider, fallback, cost, latency]
---

# Model Routing

Source: `backend/pkg/modelrouter/` and `configs/models/routing-rules.yaml`.

## Routing Strategies

| Strategy | When to use |
|----------|-------------|
| `capability` | Route by required feature (vision, function-calling, long-context) |
| `cost` | Always pick cheapest model that meets the task |
| `latency` | Always pick fastest model (streaming first-token) |
| `fallback` | Try primary; on error/timeout move to next in chain |
| `round_robin` | Distribute load evenly across providers |

## routing-rules.yaml Structure

```yaml
strategies:
  default: fallback

providers:
  openai:
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_API_KEY}
    models:
      - id: gpt-4o
        capabilities: [vision, function_calling]
        cost_per_1k_tokens: 0.005
        avg_latency_ms: 800

  anthropic:
    base_url: https://api.anthropic.com
    api_key: ${ANTHROPIC_API_KEY}
    models:
      - id: claude-sonnet-4-6
        capabilities: [function_calling, long_context]
        cost_per_1k_tokens: 0.003

fallback_chains:
  default:
    - gpt-4o
    - claude-sonnet-4-6
    - gpt-4o-mini          # cheap fallback
```

## Agent-Level Override

```yaml
spec:
  model: gpt-4o            # explicit model
  # or
  model_strategy: cost     # let router pick cheapest capable model
  model_capabilities:
    - vision
    - function_calling
```

## Fallback Behavior

1. Primary model called; if error or timeout → next in chain.
2. All models in chain exhausted → returns last error to caller.
3. Timeout per-model configured at provider level (`timeout_ms`).

## Adding a New Provider

1. Add provider block to `routing-rules.yaml` with `base_url`, `api_key`, model list.
2. Ensure the model ID matches what the provider API expects.
3. Restart or hot-reload (router watches the config file).
4. Test: `curl -X POST /api/v1/chat -d '{"model":"new-model","messages":[...]}'`

## Debugging

```bash
# Check which model was selected for a request (log level DEBUG)
APP_LOG_LEVEL=debug make dev-server

# Grep for routing decisions
grep "model_router" logs/app.log
```
