package fluxpicker

import (
	"os"
	"path/filepath"
	"testing"

	yaml "github.com/goccy/go-yaml"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestEndToEnd_SaveToProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	schema := []mold.FluxVar{
		{Name: "agents.targets", Type: "list"},
		{Name: "agents.parallel", Type: "bool", Default: "true"},
	}
	m := New().OpenFor("official/agents", data.ScopeProject, schema, nil)

	// User filters by typing.
	m.filter.SetValue("agents.tar")

	// Tab commits the top match into the editor.
	m, _ = m.Update(keyMsg("tab"))
	if m.editor.key != "agents.targets" {
		t.Fatalf("editor.key = %q want agents.targets", m.editor.key)
	}

	// Simulate the user typing into the editor by setting the form-pointer
	// directly (the huh.Form would otherwise need a real terminal).
	if m.editor.rawValue == nil {
		t.Fatal("expected editor.rawValue pointer to be initialized")
	}
	*m.editor.rawValue = "opencode, claude"

	// Enter commits the value back to overrides.
	m, _ = m.Update(keyMsg("enter"))
	got, ok := m.Overrides()["agents.targets"].([]string)
	if !ok || len(got) != 2 {
		t.Fatalf("override not applied: %+v", m.Overrides())
	}

	// Blur the filter so that the 's' shortcut key is recognized (not typed
	// into the filter text input).
	m.filter.Blur()

	// Open save prompt and save to project.
	m, _ = m.Update(keyMsg("s"))
	if m.focus != focusSavePrompt {
		t.Fatalf("expected save prompt, got focus=%v", m.focus)
	}
	m, cmd := m.Update(keyMsg("p"))
	if m.err != nil {
		t.Fatalf("save error: %v", m.err)
	}
	if cmd == nil {
		t.Fatal("expected emit cmd from save")
	}

	wantPath := filepath.Join(".ailloy", "flux", "agents.yaml")
	b, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read written file at %s: %v", wantPath, err)
	}
	var fileBack map[string]any
	if err := yaml.Unmarshal(b, &fileBack); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	agents, _ := fileBack["agents"].(map[string]any)
	if agents == nil {
		t.Fatalf("expected nested agents map, got %+v", fileBack)
	}
	targets, _ := agents["targets"].([]any)
	if len(targets) != 2 || targets[0] != "opencode" || targets[1] != "claude" {
		t.Fatalf("targets = %v want [opencode claude]", targets)
	}
}

func TestEndToEnd_SessionTargetSkipsDisk(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	schema := []mold.FluxVar{{Name: "k", Type: "string"}}
	m := New().OpenFor("ref", data.ScopeProject, schema, nil)
	m = m.SetOverride("k", "v")

	// Blur the filter so that the 's' shortcut key is recognized (not typed
	// into the filter text input).
	m.filter.Blur()

	m, _ = m.Update(keyMsg("s"))
	if m.focus != focusSavePrompt {
		t.Fatalf("expected save prompt focus, got %v", m.focus)
	}
	m, cmd := m.Update(keyMsg("o"))
	if cmd == nil {
		t.Fatal("expected emit cmd from session save")
	}

	// Confirm no flux file was written.
	if _, err := os.Stat(filepath.Join(".ailloy", "flux", "ref.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no flux file on session save; stat err = %v", err)
	}

	// Confirm the emitted message has Target=Session.
	msg := cmd()
	fm, ok := msg.(FluxOverridesMsg)
	if !ok {
		t.Fatalf("got %T want FluxOverridesMsg", msg)
	}
	if fm.Target != SaveTargetSession {
		t.Fatalf("target = %v want SaveTargetSession", fm.Target)
	}
	if fm.Overrides["k"] != "v" {
		t.Fatalf("overrides not propagated: %+v", fm.Overrides)
	}
}
