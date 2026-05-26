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

package builtin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a minimal git repository in dir with one commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		// Suppress git UI hints.
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	// Create an initial commit so HEAD is valid.
	readme := filepath.Join(dir, "README.md")
	os.WriteFile(readme, []byte("# test\n"), 0644)
	run("add", "README.md")
	run("commit", "-m", "initial commit")
}

func TestGitStatusTool_Clean(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	tool := newGitStatusTool()
	args, _ := json.Marshal(gitStatusParams{Cwd: dir})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res gitResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("want exit_code 0, got %d (output: %q)", res.ExitCode, res.Output)
	}
}

func TestGitStatusTool_WithChanges(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create an untracked file.
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hi"), 0644)

	tool := newGitStatusTool()
	args, _ := json.Marshal(gitStatusParams{Cwd: dir})
	out, _ := tool.InvokableRun(context.Background(), string(args))

	var res gitResult
	json.Unmarshal([]byte(out), &res)
	if res.Output == "" {
		t.Error("expected non-empty status output for untracked file")
	}
}

func TestGitLogTool_Basic(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	tool := newGitLogTool()
	args, _ := json.Marshal(gitLogParams{Cwd: dir, Limit: 5})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res gitResult
	json.Unmarshal([]byte(out), &res)
	if res.ExitCode != 0 {
		t.Errorf("want exit_code 0, got %d", res.ExitCode)
	}
	if res.Output == "" {
		t.Error("expected non-empty log output")
	}
}

func TestGitDiffTool_NoChanges(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	tool := newGitDiffTool()
	args, _ := json.Marshal(gitDiffParams{Cwd: dir})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res gitResult
	json.Unmarshal([]byte(out), &res)
	if res.ExitCode != 0 {
		t.Errorf("want exit_code 0, got %d", res.ExitCode)
	}
}
