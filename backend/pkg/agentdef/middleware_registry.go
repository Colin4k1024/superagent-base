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

package agentdef

import (
	"context"
	"fmt"
	"sync"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
)

// MiddlewareFactory constructs a framework-agnostic agent.Middleware from YAML config.
type MiddlewareFactory func(ctx context.Context, config map[string]any) (aclagent.Middleware, error)

var (
	mwRegistryMu sync.RWMutex
	mwRegistry   = make(map[string]MiddlewareFactory)
)

// RegisterMiddleware registers a named middleware factory for use in agent YAML definitions.
func RegisterMiddleware(name string, factory MiddlewareFactory) {
	if name == "" {
		panic("agentdef: RegisterMiddleware called with empty name")
	}
	if factory == nil {
		panic(fmt.Sprintf("agentdef: RegisterMiddleware %q called with nil factory", name))
	}
	mwRegistryMu.Lock()
	defer mwRegistryMu.Unlock()
	mwRegistry[name] = factory
}

// GetMiddlewareFactory retrieves a registered middleware factory by name.
func GetMiddlewareFactory(name string) (MiddlewareFactory, bool) {
	mwRegistryMu.RLock()
	defer mwRegistryMu.RUnlock()
	f, ok := mwRegistry[name]
	return f, ok
}

// ListMiddleware returns all registered middleware names.
func ListMiddleware() []string {
	mwRegistryMu.RLock()
	defer mwRegistryMu.RUnlock()
	names := make([]string, 0, len(mwRegistry))
	for name := range mwRegistry {
		names = append(names, name)
	}
	return names
}
