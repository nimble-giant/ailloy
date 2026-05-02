package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/plugin"
)

func TestSlugifyPluginName(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"my-mold", "my-mold", false},
		{"My Cool Mold", "my-cool-mold", false},
		{"Foo___Bar  Baz", "foo-bar-baz", false},
		{"  leading-and-trailing!!!", "leading-and-trailing", false},
		{"already-good", "already-good", false},
		{"!!!", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		got, err := slugifyPluginName(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("slugifyPluginName(%q): expected error, got slug %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("slugifyPluginName(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("slugifyPluginName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidatePluginFlags(t *testing.T) {
	t.Run("name without --claude-plugin errors", func(t *testing.T) {
		resetCastFlags()
		castPluginName = "x"
		if err := validatePluginFlags(); err == nil {
			t.Error("expected error, got nil")
		}
	})
	t.Run("version without --claude-plugin errors", func(t *testing.T) {
		resetCastFlags()
		castPluginVer = "1.0.0"
		if err := validatePluginFlags(); err == nil {
			t.Error("expected error, got nil")
		}
	})
	t.Run("with --claude-plugin OK", func(t *testing.T) {
		resetCastFlags()
		castClaudePluginFlag = true
		castPluginName = "x"
		castPluginVer = "1.0.0"
		if err := validatePluginFlags(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("no plugin flags OK", func(t *testing.T) {
		resetCastFlags()
		if err := validatePluginFlags(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFilterForPlugin(t *testing.T) {
	files := []plugin.RenderedFile{
		{CastDest: ".claude/commands/a.md"},
		{CastDest: ".github/workflows/ci.yml"},
		{CastDest: "AGENTS.md"},
		{CastDest: ".github/workflows/release.yml"},
	}
	out, hadWorkflows := filterForPlugin(files)
	if !hadWorkflows {
		t.Error("expected hadWorkflows true")
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out))
	}
	for _, f := range out {
		if strings.HasPrefix(f.CastDest, ".github/workflows/") {
			t.Errorf("workflow leaked through: %s", f.CastDest)
		}
	}
}

func TestCastClaudePlugin_FullPipeline(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	defer resetCastFlags()

	tmp := t.TempDir()
	chdir(t, tmp)

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	// Slug derived from mold name "fixture-mold".
	pluginDir := filepath.Join(tmp, ".claude", "plugins", "fixture-mold")
	mustFile(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	mustFile(t, filepath.Join(pluginDir, "commands", "hello.md"))
	mustFile(t, filepath.Join(pluginDir, "skills", "demo.md"))
	mustFile(t, filepath.Join(pluginDir, "AGENTS.md"))
	mustFile(t, filepath.Join(pluginDir, "README.md"))

	// Verify the manifest reflects mold.yaml.
	data, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["name"] != "fixture-mold" {
		t.Errorf("manifest name = %v, want fixture-mold", raw["name"])
	}
	if raw["version"] != "1.2.3" {
		t.Errorf("manifest version = %v, want 1.2.3", raw["version"])
	}

	// Flux variable should have been substituted in the templated command.
	helloBytes, _ := os.ReadFile(filepath.Join(pluginDir, "commands", "hello.md"))
	if !strings.Contains(string(helloBytes), "Hello, world!") {
		t.Errorf("expected flux substitution in hello.md, got: %s", helloBytes)
	}
}

func TestCastClaudePlugin_FluxOverride(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	castSetFlags = []string{"greeting=Howdy"}
	defer resetCastFlags()

	tmp := t.TempDir()
	chdir(t, tmp)

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	pluginDir := filepath.Join(tmp, ".claude", "plugins", "fixture-mold")
	helloBytes, _ := os.ReadFile(filepath.Join(pluginDir, "commands", "hello.md"))
	if !strings.Contains(string(helloBytes), "Howdy, world!") {
		t.Errorf("expected --set override in rendered file, got: %s", helloBytes)
	}
}

func TestCastClaudePlugin_WithWorkflowsWarnsAndSkips(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	withWorkflows = true
	defer resetCastFlags()

	tmp := t.TempDir()
	chdir(t, tmp)

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	pluginDir := filepath.Join(tmp, ".claude", "plugins", "fixture-mold")
	if _, err := os.Stat(filepath.Join(pluginDir, "workflows")); !os.IsNotExist(err) {
		t.Error("expected no workflows dir in plugin output")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, ".github")); !os.IsNotExist(err) {
		t.Error("expected no .github dir in plugin output")
	}
}

func TestCastClaudePlugin_RecastReplacesContents(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	defer resetCastFlags()

	tmp := t.TempDir()
	chdir(t, tmp)

	pluginDir := filepath.Join(tmp, ".claude", "plugins", "fixture-mold")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "stale.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	siblingDir := filepath.Join(tmp, ".claude", "plugins", "other-plugin")
	if err := os.MkdirAll(siblingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siblingDir, "keep.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "stale.md")); !os.IsNotExist(err) {
		t.Error("expected stale.md to be removed")
	}
	mustFile(t, filepath.Join(siblingDir, "keep.md"))
	mustFile(t, filepath.Join(pluginDir, "commands", "hello.md"))
}

func TestCastClaudePlugin_GlobalRoutesToHome(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	castGlobal = true
	defer resetCastFlags()

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	pluginDir := filepath.Join(tmp, ".claude", "plugins", "fixture-mold")
	mustFile(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
}

func TestCastClaudePlugin_PluginNameOverride(t *testing.T) {
	resetCastFlags()
	castClaudePluginFlag = true
	castPluginName = "Custom Plugin Name"
	castPluginVer = "9.9.9"
	defer resetCastFlags()

	tmp := t.TempDir()
	chdir(t, tmp)

	reader := fixtureMoldReader()
	if err := castClaudePlugin(reader); err != nil {
		t.Fatalf("castClaudePlugin: %v", err)
	}

	// Override slugified to "custom-plugin-name".
	pluginDir := filepath.Join(tmp, ".claude", "plugins", "custom-plugin-name")
	mustFile(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))

	data, _ := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	var raw map[string]any
	_ = json.Unmarshal(data, &raw)
	if raw["name"] != "Custom Plugin Name" {
		t.Errorf("manifest name = %v, want Custom Plugin Name", raw["name"])
	}
	if raw["version"] != "9.9.9" {
		t.Errorf("manifest version = %v, want 9.9.9", raw["version"])
	}
}

// --- helpers ---

// resetCastFlags resets cast's package-level flag vars to their zero values so
// tests are isolated.
func resetCastFlags() {
	withWorkflows = false
	castGlobal = false
	castSetFlags = nil
	castValFiles = nil
	castClaudePluginFlag = false
	castPluginName = ""
	castPluginVer = ""
}

// chdir switches into dir for the duration of the test, restoring the original
// working directory afterward.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

func mustFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected %s: %v", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("expected %s to be a file, got directory", path)
	}
}

// fixtureMoldReader builds an in-memory mold with commands, a skill, AGENTS.md,
// README.md, a workflow, and a flux variable used in one of the commands.
func fixtureMoldReader() *blanks.MoldReader {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`apiVersion: v1
kind: mold
name: fixture-mold
version: 1.2.3
description: A fixture mold for testing cast --claude-plugin
author:
  name: Fixture Author
flux:
  - name: greeting
    type: string
    required: false
    default: Hello
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`greeting: Hello
output:
  commands: .claude/commands
  skills: .claude/skills
  workflows:
    dest: .github/workflows
    process: false
`)},
		"commands/hello.md": &fstest.MapFile{Data: []byte("# Hello\n{{ .greeting }}, world!\n")},
		"skills/demo.md":    &fstest.MapFile{Data: []byte("# Demo skill\nA demo.\n")},
		"workflows/ci.yml":  &fstest.MapFile{Data: []byte("name: CI\non: push\n")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Fixture Agents\nInstructions for {{ .greeting }}.\n")},
		"README.md":         &fstest.MapFile{Data: []byte("# Fixture Mold README\nA fixture for tests.\n")},
	}
	return blanks.NewMoldReader(fsys)
}
