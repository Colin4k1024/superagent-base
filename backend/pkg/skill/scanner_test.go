package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseClaudeFrontmatter(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantFM      string
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid",
			input: `---
name: test
description: a test skill
---
# Test Skill
Body content here.`,
			wantFM:      "name: test\ndescription: a test skill",
			wantContent: "# Test Skill\nBody content here.",
		},
		{
			name:    "missing open delimiter",
			input:   "name: test\n---\nbody",
			wantErr: true,
		},
		{
			name:    "missing close delimiter",
			input:   "---\nname: test\nbody",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fm, content, err := parseClaudeFrontmatter(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fm != tc.wantFM {
				t.Errorf("frontmatter mismatch\n got: %q\nwant: %q", fm, tc.wantFM)
			}
			if content != tc.wantContent {
				t.Errorf("content mismatch\n got: %q\nwant: %q", content, tc.wantContent)
			}
		})
	}
}

func TestScanClaudeSkills(t *testing.T) {
	dir := t.TempDir()

	writeSkill := func(subdir, content string) {
		p := filepath.Join(dir, subdir)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeSkill("obs", "---\nname: observability\ndescription: OTel tracing\ntags: [otel]\n---\n# Observability\nContent here.")
	writeSkill("bad", "no frontmatter here") // malformed — should be skipped

	skills, err := ScanClaudeSkills(dir)
	if err != nil {
		t.Fatalf("ScanClaudeSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	s := skills[0]
	if s.Name != "observability" {
		t.Errorf("name: got %q, want %q", s.Name, "observability")
	}
	if s.Description != "OTel tracing" {
		t.Errorf("description: got %q, want %q", s.Description, "OTel tracing")
	}
	if len(s.Tags) != 1 || s.Tags[0] != "otel" {
		t.Errorf("tags: got %v, want [otel]", s.Tags)
	}
	if s.Content != "# Observability\nContent here." {
		t.Errorf("content: got %q", s.Content)
	}
}

func TestRegisterClaudeSkills(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "myskill")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nname: myskill\ndescription: My skill\ntags: [test]\n---\n# My Skill\nInstructions."
	if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	invoker := NewLocalInvoker()
	mgr := NewManager(nil, invoker)

	n, err := RegisterClaudeSkills(mgr, invoker, dir)
	if err != nil {
		t.Fatalf("RegisterClaudeSkills: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 registered skill, got %d", n)
	}

	// Verify it is invocable.
	result, err := invoker.Invoke(context.Background(), "myskill", nil)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	content, ok := result["content"].(string)
	if !ok || content == "" {
		t.Errorf("expected non-empty content, got %v", result)
	}

	// Verify it is accessible via manager.
	toolObj, found := mgr.GetTool("myskill")
	if !found {
		t.Fatal("skill not found in manager")
	}
	if toolObj == nil {
		t.Fatal("tool is nil")
	}
}
