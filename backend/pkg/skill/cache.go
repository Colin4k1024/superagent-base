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

import "sync"

// Cache is a thread-safe in-memory store for installed SkillInstance records.
type Cache struct {
	mu     sync.RWMutex
	skills map[string]*SkillInstance
}

// NewCache returns an empty Cache.
func NewCache() *Cache {
	return &Cache{
		skills: make(map[string]*SkillInstance),
	}
}

// Set inserts or replaces the SkillInstance for the given name.
func (c *Cache) Set(name string, instance *SkillInstance) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skills[name] = instance
}

// Get retrieves the SkillInstance for name, returning nil and false if absent.
func (c *Cache) Get(name string) (*SkillInstance, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	inst, ok := c.skills[name]
	return inst, ok
}

// Delete removes the SkillInstance associated with name.
func (c *Cache) Delete(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.skills, name)
}

// All returns a snapshot of all cached SkillInstances.
func (c *Cache) All() []SkillInstance {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]SkillInstance, 0, len(c.skills))
	for _, inst := range c.skills {
		out = append(out, *inst)
	}
	return out
}
