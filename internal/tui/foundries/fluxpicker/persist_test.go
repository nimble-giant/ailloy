package fluxpicker

import (
	"os"
	"path/filepath"
	"testing"

	yaml "github.com/goccy/go-yaml"
)

func TestWriteFluxFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flux.yaml")
	overrides := map[string]any{"agents.targets": []string{"opencode"}}
	if err := writeFluxFile(path, overrides); err != nil {
		t.Fatalf("writeFluxFile: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got map[string]any
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	agents, _ := got["agents"].(map[string]any)
	if agents == nil {
		t.Fatalf("expected nested agents map, got %+v", got)
	}
	targets, _ := agents["targets"].([]any)
	if len(targets) != 1 || targets[0] != "opencode" {
		t.Fatalf("targets = %v want [opencode]", targets)
	}
}

func TestWriteFluxFile_MergesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flux.yaml")
	if err := os.WriteFile(path, []byte("agents:\n  parallel: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overrides := map[string]any{"agents.targets": []string{"opencode"}}
	if err := writeFluxFile(path, overrides); err != nil {
		t.Fatalf("writeFluxFile: %v", err)
	}
	b, _ := os.ReadFile(path)
	var got map[string]any
	_ = yaml.Unmarshal(b, &got)
	agents := got["agents"].(map[string]any)
	if agents["parallel"] != true {
		t.Fatalf("expected existing parallel:true preserved, got %+v", agents)
	}
	if _, ok := agents["targets"]; !ok {
		t.Fatalf("expected targets to be merged in, got %+v", agents)
	}
}

func TestPersistOverrides_Session(t *testing.T) {
	path, err := persistOverrides("agents", SaveTargetSession, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path for session target, got %q", path)
	}
}

func TestPersistOverrides_ProjectWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	path, err := persistOverrides("agents", SaveTargetProject, map[string]any{"agents.targets": []string{"opencode"}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if path != ".ailloy/flux/agents.yaml" {
		t.Fatalf("path = %q want .ailloy/flux/agents.yaml", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s, got %v", path, err)
	}
}

func TestPersistOverrides_GlobalWritesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := persistOverrides("agents", SaveTargetGlobal, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := filepath.Join(home, ".ailloy", "flux", "agents.yaml")
	if path != want {
		t.Fatalf("path = %q want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s, got %v", path, err)
	}
}

func TestLastPathSegment(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"agents":          "agents",
		"official/agents": "agents",
		"a/b/c/d":         "d",
		"trailing/":       "",
	}
	for in, want := range cases {
		if got := lastPathSegment(in); got != want {
			t.Errorf("lastPathSegment(%q) = %q want %q", in, got, want)
		}
	}
}
