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

package skill

// SkillMeta represents a skill's metadata from SkillsHub.
type SkillMeta struct {
	Name         string        `json:"name"         yaml:"name"`
	Version      string        `json:"version"      yaml:"version"`
	Author       string        `json:"author"       yaml:"author"`
	Description  string        `json:"description"  yaml:"description"`
	Tags         []string      `json:"tags"         yaml:"tags"`
	Triggers     []string      `json:"triggers"     yaml:"triggers"`
	Input        *JSONSchema   `json:"input"        yaml:"input"`
	Output       *JSONSchema   `json:"output"       yaml:"output"`
	Runtime      RuntimeConfig `json:"runtime"      yaml:"runtime"`
	Dependencies []string      `json:"dependencies" yaml:"dependencies"`
}

// RuntimeConfig describes how a skill's runtime process is managed.
type RuntimeConfig struct {
	// Type identifies the runtime protocol: grpc, http, wasm, or embedded.
	Type        string `json:"type"         yaml:"type"`
	Image       string `json:"image"        yaml:"image"`
	Port        int    `json:"port"         yaml:"port"`
	HealthCheck string `json:"health_check" yaml:"health_check"`
}

// JSONSchema is a minimal JSON Schema representation used for skill I/O definitions.
type JSONSchema struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
}

// SkillInstance pairs a skill's metadata with its current lifecycle status.
type SkillInstance struct {
	Meta SkillMeta
	// Status is one of: installed, running, stopped, error.
	Status string
}
