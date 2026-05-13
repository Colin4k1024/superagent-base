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

package checkpoint

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
)

const (
	maxCheckpointEntries = 1000
	checkpointTTL        = 30 * time.Minute
)

type memEntry struct {
	data      []byte
	createdAt time.Time
}

type inMemoryStore struct {
	m  map[string]memEntry
	mu sync.RWMutex
}

func (i *inMemoryStore) Get(_ context.Context, checkPointID string) ([]byte, bool, error) {
	i.mu.RLock()
	e, ok := i.m[checkPointID]
	i.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	// TTL check.
	if time.Since(e.createdAt) > checkpointTTL {
		i.mu.Lock()
		delete(i.m, checkPointID)
		i.mu.Unlock()
		return nil, false, nil
	}
	return e.data, true, nil
}

func (i *inMemoryStore) Set(_ context.Context, checkPointID string, checkPoint []byte) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Evict expired entries when at capacity.
	if len(i.m) >= maxCheckpointEntries {
		now := time.Now()
		for k, e := range i.m {
			if now.Sub(e.createdAt) > checkpointTTL {
				delete(i.m, k)
			}
		}
		// If still at capacity, evict oldest.
		if len(i.m) >= maxCheckpointEntries {
			var oldestKey string
			var oldestTime time.Time
			for k, e := range i.m {
				if oldestKey == "" || e.createdAt.Before(oldestTime) {
					oldestKey = k
					oldestTime = e.createdAt
				}
			}
			delete(i.m, oldestKey)
		}
	}

	i.m[checkPointID] = memEntry{data: checkPoint, createdAt: time.Now()}
	return nil
}

func NewInMemoryStore() compose.CheckPointStore {
	return &inMemoryStore{
		m: make(map[string]memEntry),
	}
}
