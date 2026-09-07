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

package agent

import "context"

// CheckpointStore is the framework-agnostic interrupt/resume persistence interface.
// It abstracts eino's adk.CheckPointStore
// and Google ADK Go's session.Service + ReconstructRunState.
type CheckpointStore interface {
	// Save persists a checkpoint for the given session ID.
	Save(ctx context.Context, sessionID string, data []byte) error

	// Load retrieves the checkpoint for a session.
	Load(ctx context.Context, sessionID string) ([]byte, error)

	// Delete removes a checkpoint.
	Delete(ctx context.Context, sessionID string) error
}

// InterruptableAgent is an agent that supports interrupt/resume semantics.
type InterruptableAgent interface {
	AgentRuntime

	// HasInterrupt returns true if the agent's last run was interrupted.
	HasInterrupt(ctx context.Context, sessionID string) (bool, error)
}
