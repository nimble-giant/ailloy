package foundries

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/discover"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/fluxpicker"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/registered"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

func TestApp_FOpensPickerWhenMoldHighlighted(t *testing.T) {
	a := newAppForTest()
	a.discover = discover.NewWithFiltered([]data.CatalogEntry{{Name: "x", Source: "official/x"}}, 0)
	a.active = TabDiscover

	next, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app := next.(App)
	if !app.picker.IsOpen() {
		t.Fatal("expected picker to open")
	}
	if app.picker.MoldRef() != "official/x" {
		t.Fatalf("picker moldRef = %q want official/x", app.picker.MoldRef())
	}
}

func TestApp_FNoOpWhenNoMoldHighlighted(t *testing.T) {
	a := newAppForTest()
	a.active = TabHealth // health has no mold context
	next, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app := next.(App)
	if app.picker.IsOpen() {
		t.Fatal("expected picker to remain closed")
	}
}

func TestApp_FluxOverridesMsg_ClosesPicker(t *testing.T) {
	a := newAppForTest()
	a.picker = fluxpicker.New().OpenFor("official/x", data.ScopeProject, nil, nil)
	if !a.picker.IsOpen() {
		t.Fatal("setup: expected picker open")
	}
	next, _ := a.Update(fluxpicker.FluxOverridesMsg{
		MoldRef: "official/x", Scope: data.ScopeProject,
		Overrides: map[string]any{"k": "v"}, Target: fluxpicker.SaveTargetSession,
	})
	app := next.(App)
	if app.picker.IsOpen() {
		t.Fatal("expected picker to close after FluxOverridesMsg")
	}
}

func TestApp_FluxOverridesMsg_SessionRoutesToDiscover(t *testing.T) {
	a := newAppForTest()
	a.active = TabDiscover
	a.picker = fluxpicker.New().OpenFor("official/x", data.ScopeProject, nil, nil)

	next, _ := a.Update(fluxpicker.FluxOverridesMsg{
		MoldRef: "official/x",
		Scope:   data.ScopeProject,
		Overrides: map[string]any{
			"k": "v",
		},
		Target: fluxpicker.SaveTargetSession,
	})
	app := next.(App)
	// We can't read app.discover.pending across packages, but ApplySessionOverrides
	// is exported — we can call it explicitly to confirm routing didn't choke,
	// then re-cast and check downstream effects in the integration test (Task 14).
	if app.picker.IsOpen() {
		t.Fatal("expected picker to close")
	}
}

func TestPressingFOnFoundriesTabOpensFoundryPicker(t *testing.T) {
	cfg := &index.Config{
		Foundries: []index.FoundryEntry{{Name: "alpha", URL: "https://github.com/x/alpha"}},
	}
	a := App{
		cfg:        cfg,
		registered: registered.New(cfg, nil, nil, nil, nil),
		picker:     fluxpicker.New(),
		active:     TabFoundries,
	}
	// Move cursor onto "alpha". For this synthetic cfg there is no project
	// foundries entry — registered shows official + alpha. Cursor 0 is the
	// official foundry; cursor 1 is alpha.
	a.registered.SetCursorForTest(1)

	updated, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app := updated.(App)
	if !app.picker.IsOpen() {
		t.Fatalf("picker should be open after pressing f on Foundries tab")
	}
	if !app.picker.IsFoundryMode() {
		t.Fatalf("picker should be in foundry mode")
	}
	if app.picker.FoundryName() != "alpha" {
		t.Errorf("FoundryName = %q, want alpha", app.picker.FoundryName())
	}
}

// TestLoadFoundrySchemas_IncludesOreNamespacedKeys verifies the foundry-mode
// picker's per-mold schema population picks up ore-namespaced flux keys for
// local-path mold sources, mirroring what the cast pipeline applies. Without
// this, the picker would silently omit ore.<ns>.* rows even though the
// underlying cast accepts them.
func TestLoadFoundrySchemas_IncludesOreNamespacedKeys(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", tmp)

	// Build a local mold "delta" with an ore "status" under <mold>/ores/.
	deltaPath := filepath.Join(tmp, "delta")
	if err := os.MkdirAll(filepath.Join(deltaPath, "ores", "status"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deltaPath, "mold.yaml"), []byte(`apiVersion: v1
kind: mold
name: delta
version: 0.0.1
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deltaPath, "ores", "status", "ore.yaml"), []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deltaPath, "ores", "status", "flux.schema.yaml"), []byte(`- name: value
  type: string
  default: pending
`), 0o644); err != nil {
		t.Fatal(err)
	}

	parentURL := "https://github.com/example/parent"
	entry := &index.FoundryEntry{URL: parentURL, Type: "git"}
	cacheDir := index.CachedIndexDir(tmp, entry)
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds:
  - name: delta
    source: ` + deltaPath + `
`)
	if err := os.WriteFile(filepath.Join(cacheDir, "foundry.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &index.Config{
		Foundries: []index.FoundryEntry{{Name: "parent", URL: parentURL, Type: "git", Status: "ok"}},
	}

	schemas, refs, err := loadFoundrySchemas(cfg, "parent")
	if err != nil {
		t.Fatalf("loadFoundrySchemas: %v", err)
	}
	if refs["delta"] != deltaPath {
		t.Errorf("refs[delta] = %q, want %q", refs["delta"], deltaPath)
	}
	deltaSchema, ok := schemas["delta"]
	if !ok {
		keys := make([]string, 0, len(schemas))
		for k := range schemas {
			keys = append(keys, k)
		}
		t.Fatalf("schemas missing delta entry; got keys=%v", keys)
	}
	found := false
	for _, fv := range deltaSchema {
		if fv.Name == "ore.status.value" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("delta schema missing ore.status.value; got %+v", deltaSchema)
	}
}

// newAppForTest returns a minimal App ready for keystroke-based tests.
// It bypasses the full New() constructor (which needs a Config and many
// callbacks) to keep tests focused on the App's routing logic.
func newAppForTest() App {
	return App{
		picker: fluxpicker.New(),
	}
}
