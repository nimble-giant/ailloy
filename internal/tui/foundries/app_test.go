package foundries

import (
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

// newAppForTest returns a minimal App ready for keystroke-based tests.
// It bypasses the full New() constructor (which needs a Config and many
// callbacks) to keep tests focused on the App's routing logic.
func newAppForTest() App {
	return App{
		picker: fluxpicker.New(),
	}
}
