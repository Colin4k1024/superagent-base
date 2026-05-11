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

package skill_test

import (
	"context"
	"testing"

	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
)

// stubHubClient is a minimal HubClient that returns pre-canned SkillMeta values.
type stubHubClient struct {
	metas map[string]*skill.SkillMeta
}

func (s *stubHubClient) Get(_ context.Context, name, _ string) (*skill.SkillMeta, error) {
	m, ok := s.metas[name]
	if !ok {
		return nil, &notFoundError{name: name}
	}
	return m, nil
}

func (s *stubHubClient) Search(_ context.Context, _ string, _ skill.SearchOpts) ([]skill.SkillMeta, error) {
	return nil, nil
}
func (s *stubHubClient) Install(_ context.Context, _ string, _ string) (*skill.SkillInstance, error) {
	return nil, nil
}
func (s *stubHubClient) Uninstall(_ context.Context, _ string) error { return nil }
func (s *stubHubClient) List(_ context.Context) ([]skill.SkillInstance, error) {
	return nil, nil
}
func (s *stubHubClient) CheckHealth(_ context.Context, _ string) (bool, error) { return true, nil }

type notFoundError struct{ name string }

func (e *notFoundError) Error() string { return "skill not found: " + e.name }

// stubInvoker records invocations and returns a fixed response.
type stubInvoker struct {
	called []string
}

func (s *stubInvoker) Invoke(_ context.Context, name string, _ map[string]any) (map[string]any, error) {
	s.called = append(s.called, name)
	return map[string]any{"ok": true}, nil
}

// ─── Manager tests ────────────────────────────────────────────────────────────

func TestManager_InstallAndGetTool(t *testing.T) {
	hub := &stubHubClient{metas: map[string]*skill.SkillMeta{
		"greet": {Name: "greet", Version: "1.0", Description: "says hello"},
	}}
	inv := &stubInvoker{}
	mgr := skill.NewManager(hub, inv)

	if err := mgr.Install(context.Background(), "greet", "1.0"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	tool, ok := mgr.GetTool("greet")
	if !ok {
		t.Fatal("expected GetTool to return true after Install")
	}
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
}

func TestManager_GetTool_NotInstalled(t *testing.T) {
	hub := &stubHubClient{metas: map[string]*skill.SkillMeta{}}
	mgr := skill.NewManager(hub, &stubInvoker{})

	_, ok := mgr.GetTool("ghost")
	if ok {
		t.Error("expected GetTool to return false for uninstalled skill")
	}
}

func TestManager_ListInstalled(t *testing.T) {
	hub := &stubHubClient{metas: map[string]*skill.SkillMeta{
		"a": {Name: "a", Version: "1"},
		"b": {Name: "b", Version: "1"},
	}}
	mgr := skill.NewManager(hub, &stubInvoker{})

	mgr.Install(context.Background(), "a", "1")
	mgr.Install(context.Background(), "b", "1")

	list := mgr.ListInstalled()
	if len(list) != 2 {
		t.Errorf("expected 2 installed skills, got %d", len(list))
	}
}

func TestManager_Uninstall(t *testing.T) {
	hub := &stubHubClient{metas: map[string]*skill.SkillMeta{
		"temp": {Name: "temp", Version: "1"},
	}}
	mgr := skill.NewManager(hub, &stubInvoker{})

	mgr.Install(context.Background(), "temp", "1")
	if _, ok := mgr.GetTool("temp"); !ok {
		t.Fatal("skill should be installed before uninstall")
	}

	mgr.Uninstall("temp")
	if _, ok := mgr.GetTool("temp"); ok {
		t.Error("skill should not be available after Uninstall")
	}
}

func TestManager_Install_HubError(t *testing.T) {
	hub := &stubHubClient{metas: map[string]*skill.SkillMeta{}} // "missing" not in map
	mgr := skill.NewManager(hub, &stubInvoker{})

	err := mgr.Install(context.Background(), "missing", "1")
	if err == nil {
		t.Fatal("expected error when hub returns not-found")
	}
}
