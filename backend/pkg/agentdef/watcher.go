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
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

const (
	// debounceDuration controls how long to wait after the last filesystem
	// event before triggering a reload.  This prevents rapid successive
	// reloads when editors write files in multiple steps.
	debounceDuration = 2 * time.Second
)

// Watcher monitors a directory for YAML file changes and notifies a Reloader
// when add/modify/delete events occur.
type Watcher struct {
	dir      string
	reloader *Reloader
	w        *fsnotify.Watcher
}

// NewWatcher creates a Watcher for dir.  Call Start to begin watching.
func NewWatcher(dir string, reloader *Reloader) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(dir); err != nil {
		_ = fw.Close()
		return nil, err
	}
	return &Watcher{dir: dir, reloader: reloader, w: fw}, nil
}

// Start begins the watch loop.  It blocks until ctx is cancelled.
// Typically called in a dedicated goroutine.
func (w *Watcher) Start(ctx context.Context) {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	var pending bool

	for {
		select {
		case <-ctx.Done():
			_ = w.w.Close()
			return

		case event, ok := <-w.w.Events:
			if !ok {
				return
			}
			if !isYAMLFile(event.Name) {
				continue
			}
			logs.Infof("agentdef watcher: event %s on %s", event.Op, event.Name)

			// Reset the debounce timer on every new event.
			if pending && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(debounceDuration)
			pending = true

		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			logs.Warnf("agentdef watcher: fsnotify error: %v", err)

		case <-timer.C:
			pending = false
			if err := w.reloader.ReloadDir(ctx, w.dir); err != nil {
				logs.Warnf("agentdef watcher: reload error: %v", err)
			}
		}
	}
}

// Close releases watcher resources.  It is safe to call after Start returns.
func (w *Watcher) Close() error {
	return w.w.Close()
}

// isYAMLFile returns true when the path has a .yaml or .yml extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
