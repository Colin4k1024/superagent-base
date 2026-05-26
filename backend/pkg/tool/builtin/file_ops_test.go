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
	"path/filepath"
	"strings"
	"testing"
)

// helper: create a temp workspace with a file
func newWorkspace(t *testing.T) (string, FileOpsConfig) {
	t.Helper()
	dir := t.TempDir()
	return dir, FileOpsConfig{WorkspaceDir: dir, AllowWrite: true}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return p
}

// ─── safeJoinPath ─────────────────────────────────────────────────────────

func TestSafeJoinPath_Normal(t *testing.T) {
	root := "/workspace"
	got, err := safeJoinPath(root, "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/workspace/src/main.go" {
		t.Errorf("got %q", got)
	}
}

func TestSafeJoinPath_Traversal(t *testing.T) {
	root := "/workspace"
	_, err := safeJoinPath(root, "../secret/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestSafeJoinPath_AbsoluteInRoot(t *testing.T) {
	root := "/workspace"
	got, err := safeJoinPath(root, "/workspace/foo.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/workspace/foo.go" {
		t.Errorf("got %q", got)
	}
}

func TestSafeJoinPath_AbsoluteOutsideRoot(t *testing.T) {
	root := "/workspace"
	_, err := safeJoinPath(root, "/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path outside root")
	}
}

// ─── FileReadTool ─────────────────────────────────────────────────────────

func TestFileReadTool_Basic(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "hello.txt", "line1\nline2\nline3\n")

	tool := newFileReadTool(cfg)
	args, _ := json.Marshal(fileReadParams{Path: "hello.txt"})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res fileReadResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.Lines != 3 {
		t.Errorf("want 3 lines, got %d", res.Lines)
	}
	if !strings.Contains(res.Content, "line2") {
		t.Errorf("content missing 'line2': %q", res.Content)
	}
	if res.Truncated {
		t.Error("should not be truncated")
	}
}

func TestFileReadTool_LineRange(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "file.txt", "a\nb\nc\nd\ne\n")

	tool := newFileReadTool(cfg)
	args, _ := json.Marshal(fileReadParams{Path: "file.txt", Offset: 2, Limit: 2})
	out, _ := tool.InvokableRun(context.Background(), string(args))

	var res fileReadResult
	json.Unmarshal([]byte(out), &res)
	// Expect lines b and c (offset=2 means start at line 2)
	if !strings.Contains(res.Content, "b") {
		t.Errorf("want 'b' in content, got %q", res.Content)
	}
	if strings.Contains(res.Content, "d") {
		t.Errorf("unexpected 'd' in content: %q", res.Content)
	}
}

func TestFileReadTool_NotFound(t *testing.T) {
	dir, cfg := newWorkspace(t)
	_ = dir
	tool := newFileReadTool(cfg)
	args, _ := json.Marshal(fileReadParams{Path: "nonexistent.txt"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileReadTool_PathTraversal(t *testing.T) {
	_, cfg := newWorkspace(t)
	tool := newFileReadTool(cfg)
	args, _ := json.Marshal(fileReadParams{Path: "../passwd"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

// ─── FileWriteTool ────────────────────────────────────────────────────────

func TestFileWriteTool_Create(t *testing.T) {
	dir, cfg := newWorkspace(t)
	tool := newFileWriteTool(cfg)
	args, _ := json.Marshal(fileWriteParams{Path: "new.txt", Content: "hello"})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res fileWriteResult
	json.Unmarshal([]byte(out), &res)
	if !res.Success {
		t.Error("expected success=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
	if string(data) != "hello" {
		t.Errorf("file content mismatch: %q", data)
	}
}

func TestFileWriteTool_DisabledWrite(t *testing.T) {
	_, cfg := newWorkspace(t)
	cfg.AllowWrite = false
	tool := newFileWriteTool(cfg)
	args, _ := json.Marshal(fileWriteParams{Path: "x.txt", Content: "y"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error when AllowWrite=false")
	}
}

func TestFileWriteTool_CreatesParentDirs(t *testing.T) {
	dir, cfg := newWorkspace(t)
	tool := newFileWriteTool(cfg)
	args, _ := json.Marshal(fileWriteParams{Path: "a/b/c/file.txt", Content: "deep"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a/b/c/file.txt"))
	if string(data) != "deep" {
		t.Errorf("content mismatch: %q", data)
	}
}

// ─── FileEditTool ─────────────────────────────────────────────────────────

func TestFileEditTool_Replace(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "edit.txt", "foo bar baz")

	tool := newFileEditTool(cfg)
	args, _ := json.Marshal(fileEditParams{Path: "edit.txt", OldString: "bar", NewString: "QUX"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "edit.txt"))
	if string(data) != "foo QUX baz" {
		t.Errorf("content mismatch: %q", data)
	}
}

func TestFileEditTool_OldStringNotFound(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "edit.txt", "hello world")

	tool := newFileEditTool(cfg)
	args, _ := json.Marshal(fileEditParams{Path: "edit.txt", OldString: "NOTEXIST", NewString: "x"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error when old_string not found")
	}
}

func TestFileEditTool_DuplicateOldString(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "dup.txt", "abc abc abc")

	tool := newFileEditTool(cfg)
	args, _ := json.Marshal(fileEditParams{Path: "dup.txt", OldString: "abc", NewString: "X"})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error for duplicate old_string")
	}
}

// ─── FileGlobTool ─────────────────────────────────────────────────────────

func TestFileGlobTool_Match(t *testing.T) {
	dir, cfg := newWorkspace(t)
	writeFile(t, dir, "a.go", "")
	writeFile(t, dir, "b.go", "")
	writeFile(t, dir, "c.txt", "")

	tool := newFileGlobTool(cfg)
	args, _ := json.Marshal(fileGlobParams{Pattern: "*.go"})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res fileGlobResult
	json.Unmarshal([]byte(out), &res)
	if res.Count != 2 {
		t.Errorf("want 2 matches, got %d: %v", res.Count, res.Matches)
	}
}
