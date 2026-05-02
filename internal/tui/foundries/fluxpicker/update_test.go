package fluxpicker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestUpdate_EscClosesPicker(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "k", Type: "string"}}, nil)
	m.filter.Blur()
	m, _ = m.Update(keyMsg("esc"))
	if m.IsOpen() {
		t.Fatal("expected esc to close picker")
	}
}

func TestUpdate_DownArrowMovesCursor(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		}, nil)
	m.filter.Blur()
	m, _ = m.Update(keyMsg("down"))
	if m.cursor != 1 {
		t.Fatalf("cursor = %d want 1", m.cursor)
	}
}

func TestUpdate_TabCommitsTopMatchToEditor(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "agents.targets", Type: "string"},
		{Name: "runtime.profile", Type: "string"},
	}
	m := New().OpenFor("a", data.ScopeProject, schema, nil)
	m.filter.SetValue("agents")
	m, _ = m.Update(keyMsg("tab"))
	if m.focus != focusEditor {
		t.Fatalf("expected focus to move to editor; got %v", m.focus)
	}
	if m.editor.key != "agents.targets" {
		t.Fatalf("editor.key = %q want agents.targets", m.editor.key)
	}
}

func TestUpdate_DClearsHighlightedOverride(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "k", Type: "string"}}, nil).
		SetOverride("k", "v")
	m.filter.Blur()
	m, _ = m.Update(keyMsg("d"))
	if _, ok := m.Overrides()["k"]; ok {
		t.Fatal("expected 'd' to clear override on highlighted key")
	}
}

func TestUpdate_RResetsAllOverrides(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "k1"}, {Name: "k2"}}, nil).
		SetOverride("k1", 1).
		SetOverride("k2", 2)
	m.filter.Blur()
	m, _ = m.Update(keyMsg("R"))
	if len(m.Overrides()) != 0 {
		t.Fatalf("expected R to reset all; got %+v", m.Overrides())
	}
}

func TestUpdate_SOpensSavePrompt(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "k"}}, nil)
	m.filter.Blur()
	m, _ = m.Update(keyMsg("s"))
	if m.focus != focusSavePrompt {
		t.Fatalf("expected save prompt focus, got %v", m.focus)
	}
	if !m.saving.active {
		t.Fatal("expected saving.active=true")
	}
}

func TestUpdate_TypingIntoFilterUpdatesQuery(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "agents.targets"}}, nil)
	// filter is focused by OpenFor; type "a"
	m, _ = m.Update(keyMsg("a"))
	if v := m.filter.Value(); v != "a" {
		t.Fatalf("filter value = %q want a", v)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor reset expected; got %d", m.cursor)
	}
}

func TestUpdate_ShortcutKeysGoToFilterWhenFocused(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject,
		[]mold.FluxVar{{Name: "k"}}, nil).
		SetOverride("k", "v")
	// filter is focused by OpenFor; 'd' must NOT clear the override.
	m, _ = m.Update(keyMsg("d"))
	if _, ok := m.Overrides()["k"]; !ok {
		t.Fatal("'d' on focused filter should not clear override")
	}
	if m.filter.Value() != "d" {
		t.Fatalf("filter value = %q want d", m.filter.Value())
	}

	// 's' must NOT open save prompt while filter is focused.
	m, _ = m.Update(keyMsg("s"))
	if m.focus == focusSavePrompt {
		t.Fatal("'s' on focused filter should not open save prompt")
	}
	if m.filter.Value() != "ds" {
		t.Fatalf("filter value = %q want ds", m.filter.Value())
	}
}
