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

package modelrouter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfigFromFile reads a YAML file at path and unmarshals it into a
// RouterConfig. Returns an error when the file cannot be read or is malformed.
func LoadConfigFromFile(path string) (*RouterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("modelrouter: read config file %q: %w", path, err)
	}
	return LoadConfigFromBytes(data)
}

// LoadConfigFromBytes unmarshals YAML bytes into a RouterConfig.
func LoadConfigFromBytes(data []byte) (*RouterConfig, error) {
	var cfg RouterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("modelrouter: parse config: %w", err)
	}
	return &cfg, nil
}
