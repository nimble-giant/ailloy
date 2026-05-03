package docs

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	clidocs "github.com/nimble-giant/ailloy/docs"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	tree := clidocs.Tree()
	if tree == nil || len(tree.Children) == 0 {
		t.Fatal("clidocs.Tree() returned an empty tree")
	}
	m := New(tree)
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return resized.(Model)
}

func keyRune(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestNew_StartsAtRootCursor(t *testing.T) {
	m := newTestModel(t)
	if m.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", m.cursor)
	}
}

func TestNew_StartsFocusedOnList(t *testing.T) {
	m := newTestModel(t)
	if m.Focus() != FocusList {
		t.Errorf("expected initial focus FocusList, got %v", m.Focus())
	}
}

func TestNew_AutoExpandsTopLevelDirectories(t *testing.T) {
	m := newTestModel(t)
	if !m.IsExpanded("topics") {
		t.Errorf("topics/ should be expanded by default for discoverability")
	}
	// Visible rows should include the nested tutorial since topics/ is open.
	found := false
	for _, r := range m.rows {
		if r.topic != nil && r.topic.Slug == "topics/tutorials/first-mold" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected nested tutorial slug to be visible after auto-expand")
	}
}

func TestUpdate_ArrowDownAdvancesCursor(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor to advance to 1, got %d", m.cursor)
	}
}

func TestUpdate_KKeyMovesUp(t *testing.T) {
	m := newTestModel(t)
	for range 2 {
		updated, _ := m.Update(keyRune('j'))
		m = updated.(Model)
	}
	if m.cursor != 2 {
		t.Fatalf("expected cursor at 2, got %d", m.cursor)
	}
	updated, _ := m.Update(keyRune('k'))
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected k to move cursor to 1, got %d", m.cursor)
	}
}

func TestUpdate_CursorClampsAtBounds(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", m.cursor)
	}
	for range len(m.rows) + 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.cursor != len(m.rows)-1 {
		t.Errorf("cursor should clamp to len-1=%d, got %d", len(m.rows)-1, m.cursor)
	}
}

func TestUpdate_LExpandsCollapsedDirectory(t *testing.T) {
	// Start by collapsing topics/, then re-expand with l.
	m := newTestModel(t)
	// Find topics/ row.
	var topicsIdx = -1
	for i, r := range m.rows {
		if r.isDir && r.dirPath == "topics" {
			topicsIdx = i
			break
		}
	}
	if topicsIdx == -1 {
		t.Fatal("expected a topics/ directory row")
	}
	m.cursor = topicsIdx
	// Collapse first.
	updated, _ := m.Update(keyRune('h'))
	m = updated.(Model)
	if m.IsExpanded("topics") {
		t.Fatal("h should have collapsed topics/")
	}
	// Now expand.
	updated, _ = m.Update(keyRune('l'))
	m = updated.(Model)
	if !m.IsExpanded("topics") {
		t.Error("l should have expanded topics/")
	}
}

func TestUpdate_HCollapsesAndJumpsToParent(t *testing.T) {
	m := newTestModel(t)
	// Move cursor onto the nested tutorial leaf.
	for i, r := range m.rows {
		if r.topic != nil && r.topic.Slug == "topics/tutorials/first-mold" {
			m.cursor = i
			break
		}
	}
	if m.rows[m.cursor].topic == nil {
		t.Fatal("test setup: cursor not on the nested topic")
	}
	startDepth := m.rows[m.cursor].depth
	updated, _ := m.Update(keyRune('h'))
	m = updated.(Model)
	if m.rows[m.cursor].depth >= startDepth {
		t.Errorf("h on a leaf should jump to a shallower row; before depth=%d after depth=%d",
			startDepth, m.rows[m.cursor].depth)
	}
}

func TestUpdate_EnterOnFileFocusesBody(t *testing.T) {
	m := newTestModel(t)
	// Find the first file row and put cursor there.
	for i, r := range m.rows {
		if r.topic != nil {
			m.cursor = i
			break
		}
	}
	if m.rows[m.cursor].topic == nil {
		t.Fatal("test setup: no file row in tree")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.Focus() != FocusBody {
		t.Errorf("enter on file should focus body, got %v", m.Focus())
	}
}

func TestUpdate_EnterOnDirectoryDoesNotFocusBody(t *testing.T) {
	m := newTestModel(t)
	// Find a directory row.
	for i, r := range m.rows {
		if r.isDir {
			m.cursor = i
			break
		}
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.Focus() == FocusBody {
		t.Error("enter on a directory should not focus body")
	}
}

func TestUpdate_EscReturnsToList(t *testing.T) {
	m := newTestModel(t)
	for i, r := range m.rows {
		if r.topic != nil {
			m.cursor = i
			break
		}
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.Focus() != FocusList {
		t.Errorf("esc should return focus to list, got %v", m.Focus())
	}
}

func TestUpdate_QuitReturnsTeaQuit(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("expected non-nil cmd for q press")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", cmd())
	}
}

func TestUpdate_HelpToggles(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(keyRune('?'))
	m = updated.(Model)
	if !m.showHelp {
		t.Error("expected ? to enable help")
	}
	updated, _ = m.Update(keyRune('?'))
	m = updated.(Model)
	if m.showHelp {
		t.Error("expected ? to toggle help off")
	}
}

func TestUpdate_JOnBodyFocusScrollsViewport(t *testing.T) {
	m := newTestModel(t)
	// Focus body on a long topic (foundry) to guarantee scrollable content.
	moveCursorToSlug(t, &m, "foundry")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.Focus() != FocusBody {
		t.Fatalf("expected body focus before scroll test")
	}
	before := m.viewport.YOffset
	updated, _ = m.Update(keyRune('j'))
	m = updated.(Model)
	if m.viewport.YOffset == before {
		t.Errorf("j on body focus should scroll; YOffset unchanged at %d", before)
	}
}

func TestView_ContainsLogoAndCurrentTopic(t *testing.T) {
	m := newTestModel(t)
	out := m.View()
	if !strings.Contains(out, "Ailloy Docs") {
		t.Errorf("View should include the brand logo; got:\n%s", out)
	}
	cur := m.currentRow()
	if cur == nil {
		t.Fatal("currentRow returned nil")
	}
	want := cur.name
	if cur.topic != nil {
		want = cur.topic.Title
	}
	if !strings.Contains(out, want) {
		t.Errorf("View should mention current row %q; got:\n%s", want, out)
	}
}

func TestView_EmptyBeforeResize(t *testing.T) {
	m := New(clidocs.Tree())
	if m.View() != "" {
		t.Error("expected empty View() before WindowSizeMsg")
	}
}

func TestView_FooterAdaptsToFocus(t *testing.T) {
	m := newTestModel(t)
	listFooter := m.View()
	if !strings.Contains(listFooter, "expand") && !strings.Contains(listFooter, "collapse") {
		t.Errorf("list footer should mention expand/collapse hints; got:\n%s", listFooter)
	}
	moveCursorToSlug(t, &m, "flux")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	bodyFooter := m.View()
	if !strings.Contains(bodyFooter, "scroll") {
		t.Errorf("body footer should mention scroll; got:\n%s", bodyFooter)
	}
}

func TestPaneWidths_RespectMinima(t *testing.T) {
	m := New(clidocs.Tree())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = updated.(Model)
	list, body := m.paneWidths()
	if list < minListWidth {
		t.Errorf("list width %d below minimum %d", list, minListWidth)
	}
	if body < minViewportWidth {
		t.Errorf("body width %d below minimum %d", body, minViewportWidth)
	}
}

// moveCursorToSlug positions the cursor onto the row whose topic.Slug matches
// the given value. Fails the test if no such row is visible (caller may need
// to expand a folder first).
func moveCursorToSlug(t *testing.T, m *Model, slug string) {
	t.Helper()
	for i, r := range m.rows {
		if r.topic != nil && r.topic.Slug == slug {
			m.cursor = i
			m.renderCurrent(false)
			return
		}
	}
	t.Fatalf("slug %q not visible in tree rows", slug)
}
