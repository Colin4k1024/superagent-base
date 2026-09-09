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

package gemini

import "google.golang.org/genai"

// Config is the configuration for a Gemini chat model backed by the
// google.golang.org/genai SDK. The caller is responsible for creating
// the *genai.Client (e.g. via genai.NewClient) and passing it in.
type Config struct {
	// Client is a pre-created genai SDK client.
	Client *genai.Client
	// Model is the Gemini model identifier (e.g. "gemini-2.5-flash").
	Model string
	// TopK controls top-k sampling.
	TopK *int
	// TopP controls nucleus sampling.
	TopP *float32
	// Temperature controls randomness.
	Temperature *float64
	// MaxTokens limits the number of output tokens.
	MaxTokens *int
	// ThinkingConfig configures Gemini thinking/reasoning behaviour.
	ThinkingConfig *genai.ThinkingConfig
}
