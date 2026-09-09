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

// adk_facade re-exports cloudwego/eino/adk types as aliases so callers
// avoid importing eino directly (S4 migration).
package einobridge

import (
	"github.com/cloudwego/eino/adk"
)

type AdkChatModelAgent = adk.ChatModelAgent
type AdkCheckPointStore = adk.CheckPointStore
type AdkAgentInput = adk.AgentInput
type AdkAgentEvent = adk.AgentEvent

func NewAdkAsyncIteratorPair[T any]() (*adk.AsyncIterator[T], *adk.AsyncGenerator[T]) {
	return adk.NewAsyncIteratorPair[T]()
}

// AsyncIterator re-exports adk.AsyncIterator.
type AdkAsyncIterator[T any] = adk.AsyncIterator[T]
