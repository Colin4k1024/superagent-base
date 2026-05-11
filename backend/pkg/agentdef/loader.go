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
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

// Loader scans a directory for *.yaml / *.yml files and holds the resulting
// AgentDefinition map in memory.
type Loader struct {
	dir    string
	mu     sync.RWMutex
	agents map[string]*AgentDefinition
}

// NewLoader creates a Loader for dir.  Call LoadAll to populate it.
func NewLoader(dir string) *Loader {
	return &Loader{
		dir:    dir,
		agents: make(map[string]*AgentDefinition),
	}
}

// LoadAll scans dir for YAML files, parses and validates each one, and
// replaces the internal map atomically on success.
//
// Individual file errors are collected and returned together so that a single
// bad file does not abort loading the rest.  The Loader is updated with all
// successfully parsed definitions regardless of whether errors occurred.
func (l *Loader) LoadAll() (map[string]*AgentDefinition, error) {
	loaded := make(map[string]*AgentDefinition)
	var errs []string

	walkErr := filepath.WalkDir(l.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		def, parseErr := ParseFile(path)
		if parseErr != nil {
			errs = append(errs, parseErr.Error())
			return nil
		}
		if _, exists := loaded[def.Metadata.Name]; exists {
			errs = append(errs, fmt.Sprintf("duplicate agent name %q in %s", def.Metadata.Name, path))
			return nil
		}
		loaded[def.Metadata.Name] = def
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("agentdef: loader: walk dir %q: %w", l.dir, walkErr)
	}

	// Atomically replace the internal map.
	l.mu.Lock()
	l.agents = loaded
	l.mu.Unlock()

	if len(errs) > 0 {
		return loaded, fmt.Errorf("agentdef: loader: errors loading dir %q:\n  %s", l.dir, strings.Join(errs, "\n  "))
	}
	return loaded, nil
}

// Get returns the AgentDefinition for the given name, or false if not found.
func (l *Loader) Get(name string) (*AgentDefinition, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.agents[name]
	return d, ok
}

// List returns a snapshot of all loaded agent names.
func (l *Loader) List() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	names := make([]string, 0, len(l.agents))
	for n := range l.agents {
		names = append(names, n)
	}
	return names
}
