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

package dagbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// RunnerOption configures the NodeRunner.
type RunnerOption func(*runnerConfig)

type runnerConfig struct {
	timeoutMS      int64
	maxRetry       int64
	errProcessType vo.ErrorProcessType
	dataOnErr      func(ctx context.Context) map[string]any
	preProcessors  []func(context.Context, map[string]any) (map[string]any, error)
}

// WithTimeout sets the per-node execution timeout in milliseconds.
func WithTimeout(ms int64) RunnerOption {
	return func(c *runnerConfig) { c.timeoutMS = ms }
}

// WithMaxRetry sets the maximum number of retries on failure.
func WithMaxRetry(n int64) RunnerOption {
	return func(c *runnerConfig) { c.maxRetry = n }
}

// WithErrorProcessType sets the error handling strategy.
func WithErrorProcessType(t vo.ErrorProcessType) RunnerOption {
	return func(c *runnerConfig) { c.errProcessType = t }
}

// WithDataOnErr sets a function that returns default output on error.
func WithDataOnErr(f func(ctx context.Context) map[string]any) RunnerOption {
	return func(c *runnerConfig) { c.dataOnErr = f }
}

// WithPreProcessor adds a pre-processor that transforms node input.
func WithPreProcessor(f func(context.Context, map[string]any) (map[string]any, error)) RunnerOption {
	return func(c *runnerConfig) { c.preProcessors = append(c.preProcessors, f) }
}

// NodeRunner wraps a dag.NodeExecutor with timeout, retry, pre-processors,
// and error handling. It mirrors the behavior of compose.nodeRunConfig
// but without any eino dependency.
type NodeRunner struct {
	exec dag.NodeExecutor
	cfg  runnerConfig
}

// NewNodeRunner creates a NodeRunner that decorates the given executor.
func NewNodeRunner(exec dag.NodeExecutor, opts ...RunnerOption) *NodeRunner {
	cfg := runnerConfig{
		timeoutMS:      30000, // default 30s
		maxRetry:       0,
		errProcessType: vo.ErrorProcessTypeThrow,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &NodeRunner{exec: exec, cfg: cfg}
}

// Execute runs the node with timeout, retry, and pre-processors.
func (nr *NodeRunner) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	// Apply pre-processors.
	processed := input
	for _, pp := range nr.cfg.preProcessors {
		var err error
		processed, err = pp(ctx, processed)
		if err != nil {
			return nr.handleError(ctx, err)
		}
	}

	// Execute with timeout + retry.
	var lastErr error
	attempts := nr.cfg.maxRetry + 1
	for i := int64(0); i < attempts; i++ {
		output, err := nr.executeWithTimeout(ctx, processed)
		if err == nil {
			return output, nil
		}
		lastErr = err
		if i < nr.cfg.maxRetry {
			logs.CtxWarnf(ctx, "dagbridge: node failed (attempt %d/%d), retrying: %v",
				i+1, attempts, err)
		}
	}

	return nr.handleError(ctx, lastErr)
}

func (nr *NodeRunner) executeWithTimeout(ctx context.Context, input map[string]any) (map[string]any, error) {
	if nr.cfg.timeoutMS <= 0 {
		return nr.exec.Execute(ctx, input)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(nr.cfg.timeoutMS)*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	var output map[string]any
	var err error
	go func() {
		defer close(done)
		output, err = nr.exec.Execute(timeoutCtx, input)
	}()

	select {
	case <-done:
		return output, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("dagbridge: node timed out after %dms", nr.cfg.timeoutMS)
	}
}

func (nr *NodeRunner) handleError(ctx context.Context, err error) (map[string]any, error) {
	if nr.cfg.dataOnErr != nil && nr.cfg.errProcessType != vo.ErrorProcessTypeThrow {
		return nr.cfg.dataOnErr(ctx), nil
	}
	return nil, err
}

// Compile-time check that NodeRunner implements dag.NodeExecutor.
var _ dag.NodeExecutor = (*NodeRunner)(nil)
