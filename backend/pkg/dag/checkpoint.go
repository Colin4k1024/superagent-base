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

package dag

import (
	"context"
	"encoding/json"
	"fmt"
)

// CheckpointStore persists graph execution state for interrupt/resume.
type CheckpointStore interface {
	Get(ctx context.Context, checkpointID string) ([]byte, bool, error)
	Set(ctx context.Context, checkpointID string, data []byte) error
	Delete(ctx context.Context, checkpointID string) error
}

// CheckpointData serializes the execution state of a graph run.
type CheckpointData struct {
	Outputs  map[NodeKey]map[string]any `json:"outputs"`
	State    map[string]any             `json:"state"`
	Usage    Usage                      `json:"usage"`
	Executed []NodeKey                  `json:"executed"`
}

// Snapshot serializes the current execution state for checkpointing.
func (ec *ExecutionContext) Snapshot(executed map[NodeKey]bool) ([]byte, error) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	cp := CheckpointData{
		Outputs: make(map[NodeKey]map[string]any),
		State:   make(map[string]any),
		Usage:   ec.usage,
	}
	for k, v := range ec.outputs {
		cp.Outputs[k] = v
	}
	for k, v := range ec.state {
		cp.State[k] = v
	}
	for k := range executed {
		if executed[k] {
			cp.Executed = append(cp.Executed, k)
		}
	}
	return json.Marshal(cp)
}

// Restore deserializes checkpoint data into the execution context.
func (ec *ExecutionContext) Restore(data []byte) (map[NodeKey]bool, error) {
	var cp CheckpointData
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("dag: failed to unmarshal checkpoint: %w", err)
	}

	ec.mu.Lock()
	defer ec.mu.Unlock()

	for k, v := range cp.Outputs {
		ec.outputs[k] = v
	}
	for k, v := range cp.State {
		ec.state[k] = v
	}
	ec.usage = cp.Usage

	executed := make(map[NodeKey]bool)
	for _, k := range cp.Executed {
		executed[k] = true
	}
	return executed, nil
}

// InMemoryCheckpointStore is a simple in-memory implementation for testing.
type InMemoryCheckpointStore struct {
	data map[string][]byte
}

// NewInMemoryCheckpointStore creates a new in-memory checkpoint store.
func NewInMemoryCheckpointStore() *InMemoryCheckpointStore {
	return &InMemoryCheckpointStore{data: make(map[string][]byte)}
}

func (s *InMemoryCheckpointStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	d, ok := s.data[id]
	return d, ok, nil
}

func (s *InMemoryCheckpointStore) Set(_ context.Context, id string, data []byte) error {
	s.data[id] = data
	return nil
}

func (s *InMemoryCheckpointStore) Delete(_ context.Context, id string) error {
	delete(s.data, id)
	return nil
}
