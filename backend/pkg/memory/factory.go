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

package memory

import "fmt"

// backends holds registered backend constructors keyed by type name.
var backends = map[string]func() Backend{}

// Register registers a named backend constructor. Call this from an init()
// function in each backend sub-package.
func Register(name string, constructor func() Backend) {
	backends[name] = constructor
}

// New creates a new Backend of the type named in config.Type.
// The caller must call Backend.Init with the same config before use.
func New(config BackendConfig) (Backend, error) {
	constructor, ok := backends[config.Type]
	if !ok {
		return nil, fmt.Errorf("memory: unknown backend type %q", config.Type)
	}
	return constructor(), nil
}
