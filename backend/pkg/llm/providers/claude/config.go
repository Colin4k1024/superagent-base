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

package claude

// ThinkingConfig controls Claude's extended-thinking feature.
type ThinkingConfig struct {
	Enable bool
}

// Config is the configuration for a Claude chat model backed by the
// Anthropic SDK directly (no eino dependency).
type Config struct {
	APIKey      string          `json:"api_key"`
	BaseURL      *string         `json:"base_url,omitempty"`
	Model        string          `json:"model"`
	Temperature  *float32        `json:"temperature,omitempty"`
	TopP         float32         `json:"top_p,omitempty"`
	TopK         int             `json:"top_k,omitempty"`
	MaxTokens    int             `json:"max_tokens"`
	Thinking     *ThinkingConfig `json:"thinking,omitempty"`
}
