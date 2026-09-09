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

// adapters.go previously contained chatModelAdapter that wrapped
// wfcompose.ToolCallingChatModel as eino's model.ToolCallingChatModel.
// Now that all facades use wfcompose types, the adapter is no longer needed.
// This file is kept as a compatibility shim for NewModelAdapter callers.
package einobridge

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// NewModelAdapter is a no-op: the native provider already implements
// wfcompose.ToolCallingChatModel. Returns the provider directly.
func NewModelAdapter(inner wfcompose.ToolCallingChatModel) wfcompose.ToolCallingChatModel {
	return inner
}
