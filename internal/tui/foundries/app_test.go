package foundries

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/discover"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/fluxpicker"
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

// newAppForTest returns a minimal App ready for keystroke-based tests.
// It bypasses the full New() constructor (which needs a Config and many
// callbacks) to keep tests focused on the App's routing logic.
func newAppForTest() App {
	return App{
		picker: fluxpicker.New(),
	}
}
