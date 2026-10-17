package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCastMold_TargetsFanOut is the acceptance test for agent-target
// fan-out:
//
//	Given a mold whose flux maps agents/ and AGENTS.md with `targets:`
//	  and an ingot that branches on agent.format
//	When it is cast with agent.targets=[claude,codex,opencode]
//	Then each harness gets its native layout from the built-in registry,
//	  Codex agents are written as .toml with TOML content,
//	  and AGENTS.md (shared by Codex and OpenCode) is written once.
func TestCastMold_TargetsFanOut(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", t.TempDir())

	moldDir := filepath.Join(projectDir, "mold")
	files := map[string]string{
		"mold.yaml": "apiVersion: v1\nkind: mold\nname: fanout\nversion: 0.1.0\n",
		"flux.yaml": `output:
  agents:
    targets: agents
  INSTRUCTIONS.md:
    targets: instructions
`,
		"ingots/agent-header.md": `{{- if eq .agent.format "codex-toml" -}}
name = "reviewer"
description = "Reviews code"
developer_instructions = """
{{- else -}}
---
name: reviewer
description: Reviews code
---
{{- end }}`,
		"agents/reviewer.md": `{{ ingot "agent-header" }}
Review the diff for {{ .agent.current_target }}.
{{ if eq .agent.format "codex-toml" }}"""{{ end }}
`,
		"INSTRUCTIONS.md": "# Instructions for {{ .agent.current_targets }}\n",
	}
	for name, body := range files {
		p := filepath.Join(moldDir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := CastMold(t.Context(), moldDir, CastOptions{
		SetOverrides: []string{"agent.targets=[claude,codex,opencode]"},
	}); err != nil {
		t.Fatalf("CastMold: %v", err)
	}

	read := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
		return string(b)
	}

	if got := read(".claude/agents/reviewer.md"); !strings.HasPrefix(got, "---\nname: reviewer") {
		t.Errorf("claude agent not YAML frontmatter:\n%s", got)
	}
	if got := read(".codex/agents/reviewer.toml"); !strings.HasPrefix(got, `name = "reviewer"`) || !strings.Contains(got, "for codex") {
		t.Errorf("codex agent not TOML:\n%s", got)
	}
	if got := read(".opencode/agents/reviewer.md"); !strings.Contains(got, "for opencode") {
		t.Errorf("opencode agent wrong:\n%s", got)
	}
	if got := read("CLAUDE.md"); !strings.Contains(got, "[claude]") {
		t.Errorf("CLAUDE.md = %q", got)
	}
	if got := read("AGENTS.md"); !strings.Contains(got, "[codex opencode]") {
		t.Errorf("AGENTS.md should be shared by codex and opencode, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "INSTRUCTIONS.md")); err == nil {
		t.Error("INSTRUCTIONS.md source must not be installed at its identity path")
	}
}
