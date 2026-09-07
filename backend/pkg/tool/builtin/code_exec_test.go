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

package builtin

import (
	"context"
	"testing"
)

func TestCodeExecTool_Info(t *testing.T) {
	tool := newCodeExecTool()
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Info() returned nil")
	}
	if info.Name != "code_execute" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "code_execute")
	}
	if info.Desc == "" {
		t.Error("Info().Desc is empty")
	}
}

// TestCodeExecTool_Run_NoRunner verifies that when coderunner is not
// initialized (nil runner), Run returns an error rather than panicking.
// The python branch is the one that checks coderunner.GetCodeRunner().
func TestCodeExecTool_Run_NoRunner(t *testing.T) {
	tool := newCodeExecTool()
	// coderunner.GetCodeRunner() returns nil in a bare test environment.
	_, err := tool.Run(context.Background(),
		`{"language":"python","code":"print(1)"}`)
	if err == nil {
		t.Error("expected error when code runner is not initialized, got nil")
	}
}

// TestCodeExecTool_Run_UnsupportedLanguage verifies that an unsupported
// language returns an error.
func TestCodeExecTool_Run_UnsupportedLanguage(t *testing.T) {
	tool := newCodeExecTool()
	_, err := tool.Run(context.Background(),
		`{"language":"ruby","code":"puts 1"}`)
	if err == nil {
		t.Error("expected error for unsupported language, got nil")
	}
}

// TestCodeExecTool_Run_MissingLanguage verifies that missing language
// field returns an error.
func TestCodeExecTool_Run_MissingLanguage(t *testing.T) {
	tool := newCodeExecTool()
	_, err := tool.Run(context.Background(), `{"code":"echo hi"}`)
	if err == nil {
		t.Error("expected error for missing language, got nil")
	}
}

// TestCodeExecTool_Run_JavaScriptNotSupported verifies the explicit
// "not yet supported" error for the javascript language.
func TestCodeExecTool_Run_JavaScriptNotSupported(t *testing.T) {
	tool := newCodeExecTool()
	_, err := tool.Run(context.Background(),
		`{"language":"javascript","code":"console.log(1)"}`)
	if err == nil {
		t.Error("expected error for javascript (not yet supported), got nil")
	}
}
