package docs

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	clidocs "github.com/nimble-giant/ailloy/docs"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	topics := clidocs.List()
	if len(topics) == 0 {
		t.Fatal("no embedded topics available — clidocs.List() returned nothing")
	}
	m := New(topics)
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return resized.(Model)
}

func TestNew_StartsOnFirstTopic(t *testing.T) {
	m := newTestModel(t)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if got := m.CurrentTopic(); got != m.topics[0].Slug {
		t.Errorf("CurrentTopic() = %q, want %q", got, m.topics[0].Slug)
	}
}

func TestNew_StartsFocusedOnList(t *testing.T) {
	m := newTestModel(t)
	if m.Focus() != FocusList {
		t.Errorf("expected initial focus FocusList, got %v", m.Focus())
	}
}

func TestUpdate_ArrowDownAdvancesCursor(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor to advance to 1 after KeyDown, got %d", m.cursor)
	}
}

func TestUpdate_KKeyMovesUp(t *testing.T) {
	m := newTestModel(t)
	// Move down twice then back up with k.
	for range 2 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
	}
	if m.cursor != 2 {
		t.Fatalf("expected cursor at 2 after two j presses, got %d", m.cursor)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected k to move cursor to 1, got %d", m.cursor)
	}
}

func TestUpdate_CursorClampsAtBounds(t *testing.T) {
	m := newTestModel(t)
	// Page up at top should stay at 0.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", m.cursor)
	}
	// Press down past the end.
	for range len(m.topics) + 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.cursor != len(m.topics)-1 {
		t.Errorf("cursor should clamp to len-1 = %d, got %d", len(m.topics)-1, m.cursor)
	}
}

func TestUpdate_EnterSwitchesFocusToBody(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.Focus() != FocusBody {
		t.Errorf("expected enter to focus body, got %v", m.Focus())
	}
}

func TestUpdate_EscReturnsToList(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.Focus() != FocusList {
		t.Errorf("expected esc to return focus to list, got %v", m.Focus())
	}
}

func TestUpdate_QuitReturnsTeaQuit(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for q press")
	}
	// Calling cmd should return a tea.QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestUpdate_HelpToggles(t *testing.T) {
	m := newTestModel(t)
	if m.showHelp {
		t.Fatal("help should start hidden")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if !m.showHelp {
		t.Error("expected ? to enable help")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if m.showHelp {
		t.Error("expected ? to toggle help off")
	}
}

func TestRenderCurrent_PopulatesViewport(t *testing.T) {
	m := newTestModel(t)
	if m.Rendered() == "" {
		t.Fatal("expected viewport to be populated after WindowSizeMsg")
	}
}

func TestView_ContainsHighlightedTopic(t *testing.T) {
	m := newTestModel(t)
	out := m.View()
	if !strings.Contains(out, m.topics[0].Slug) {
		t.Errorf("expected View() to contain first topic slug %q", m.topics[0].Slug)
	}
	if !strings.Contains(out, "ailloy docs") {
		t.Errorf("expected header to contain 'ailloy docs'; got:\n%s", out)
	}
}

func TestView_EmptyBeforeResize(t *testing.T) {
	m := New(clidocs.List())
	if m.View() != "" {
		t.Error("expected empty View() before WindowSizeMsg")
	}
}

func TestMoveCursor_TriggersRerender(t *testing.T) {
	m := newTestModel(t)
	first := m.Rendered()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.Rendered() == first {
		t.Errorf("expected rendered content to change when cursor moves; both equal")
	}
}

func TestPaneWidths_RespectMinima(t *testing.T) {
	m := New(clidocs.List())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = updated.(Model)
	list, body := m.paneWidths()
	if list < minListWidth {
		t.Errorf("list width %d below minimum %d", list, minListWidth)
	}
	if body < minViewportWidth {
		t.Errorf("body width %d below minimum %d", body, minViewportWidth)
	}
}
