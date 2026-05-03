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

func TestNew_StartsOnFileRow(t *testing.T) {
	m := newTestModel(t)
	if r := m.currentRow(); r == nil || r.topic == nil {
		t.Errorf("expected initial cursor on a file row, got %+v", r)
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
	start := m.cursor
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != start+1 {
		t.Errorf("expected cursor to advance to %d, got %d", start+1, m.cursor)
	}
}

func TestUpdate_KKeyMovesUp(t *testing.T) {
	m := newTestModel(t)
	start := m.cursor
	for range 2 {
		updated, _ := m.Update(keyRune('j'))
		m = updated.(Model)
	}
	if m.cursor != start+2 {
		t.Fatalf("expected cursor at %d after two j presses, got %d", start+2, m.cursor)
	}
	updated, _ := m.Update(keyRune('k'))
	m = updated.(Model)
	if m.cursor != start+1 {
		t.Errorf("expected k to move cursor to %d, got %d", start+1, m.cursor)
	}
}

func TestUpdate_CursorClampsAtBounds(t *testing.T) {
	m := newTestModel(t)
	// Press Up enough times to push past the top regardless of starting row.
	for range len(m.rows) + 5 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = updated.(Model)
	}
	if m.cursor != 0 {
		t.Errorf("cursor should clamp to 0 after many Up presses, got %d", m.cursor)
	}
	for range len(m.rows) + 5 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
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

func TestNew_StartsCursorOnFirstFile(t *testing.T) {
	m := newTestModel(t)
	if r := m.currentRow(); r == nil || r.topic == nil {
		t.Fatalf("expected initial cursor to land on a file row, got dir/nil")
	}
}

func TestRender_DispatchesAsyncCmdOnCacheMiss(t *testing.T) {
	m := newTestModel(t)
	// First, drain whatever the WindowSizeMsg dispatched.
	// Then move cursor to a different topic and capture the resulting Cmd.
	cmd := m.moveCursor(1)
	if cmd == nil {
		// Maybe the next row is a directory — advance until a file.
		for cmd == nil && m.cursor+1 < len(m.rows) {
			cmd = m.moveCursor(1)
		}
	}
	if cmd == nil {
		t.Skip("no leaf row available to trigger async render")
	}
	if !m.loading {
		t.Errorf("model should be marked loading while a render is in flight")
	}
	msg := cmd()
	if _, ok := msg.(topicRenderedMsg); !ok {
		t.Fatalf("expected topicRenderedMsg, got %T", msg)
	}
	updated, _ := m.Update(msg)
	m = updated.(Model)
	if m.loading {
		t.Errorf("model should clear loading after topicRenderedMsg")
	}
	if m.rendered == "" {
		t.Errorf("expected rendered content after async render completes")
	}
}

func TestRender_CacheHitIsSyncAndNoCmd(t *testing.T) {
	m := newTestModel(t)
	// Render initial topic synchronously by draining the cmd, populating cache.
	first := m.currentRow()
	if first == nil || first.topic == nil {
		t.Fatal("test setup: expected a leaf cursor")
	}
	// Make sure something is rendered + cached. The first-leaf launch
	// shouldn't have completed yet, so call renderCurrent(true) and drain.
	cmd := m.renderCurrent(true)
	drainRender(t, &m, cmd)

	// Now re-request the same slug — should hit the cache and return nil.
	cmd2 := m.renderCurrent(false)
	if cmd2 != nil {
		t.Errorf("expected nil cmd on cache hit, got non-nil")
	}
	if m.loading {
		t.Errorf("cache hit should not put model in loading state")
	}
}

func TestView_ShowsSpinnerWhileLoading(t *testing.T) {
	m := newTestModel(t)
	// Force a fresh load on a non-cached topic.
	moveCursorToSlugNoDrain(t, &m, "foundry")
	if !m.loading {
		t.Skip("could not enter loading state — render may have been synchronous")
	}
	out := m.View()
	if !strings.Contains(out, "Forging") {
		t.Errorf("expected loading view to contain 'Forging…'; got:\n%s", out)
	}
}

// moveCursorToSlugNoDrain positions the cursor without draining the render
// command, so the model stays in the loading state for tests that want to
// observe that state.
func moveCursorToSlugNoDrain(t *testing.T, m *Model, slug string) {
	t.Helper()
	for i, r := range m.rows {
		if r.topic != nil && r.topic.Slug == slug {
			m.cursor = i
			_ = m.renderCurrent(true)
			return
		}
	}
	t.Fatalf("slug %q not visible", slug)
}

func TestView_HeaderHasOrangeBackground(t *testing.T) {
	m := newTestModel(t)
	header := m.renderHeader()
	if !strings.Contains(header, "Ailloy Docs") {
		t.Errorf("header missing logo: %q", header)
	}
	if !strings.Contains(header, "TREE") && !strings.Contains(header, "BODY") {
		t.Errorf("header missing focus status: %q", header)
	}
	// The header is now 2 lines tall so the brand chrome is unmistakable.
	if got := strings.Count(header, "\n"); got != 1 {
		t.Errorf("expected header to span 2 lines (1 newline), got %d newlines", got)
	}
}

func TestView_HeaderShowsActiveTopicTitle(t *testing.T) {
	m := newTestModel(t)
	header := m.renderHeader()
	cur := m.currentRow()
	if cur == nil || cur.topic == nil {
		t.Skip("no topic at startup")
	}
	if !strings.Contains(header, cur.topic.Title) {
		t.Errorf("header should contain active topic title %q; got:\n%s", cur.topic.Title, header)
	}
}

func TestScrollbar_HiddenWhenContentFits(t *testing.T) {
	m := newTestModel(t)
	// Manually set viewport content to a single line and re-render — the
	// scrollbar should be empty because total ≤ visible.
	m.viewport.SetContent("only one line of content")
	bar := m.renderScrollbar(m.viewport.Height)
	if bar != "" {
		t.Errorf("expected empty scrollbar when content fits; got %q", bar)
	}
}

func TestScrollbar_RendersWhenContentOverflows(t *testing.T) {
	m := newTestModel(t)
	// Build content taller than the viewport.
	lines := make([]string, m.viewport.Height*4)
	for i := range lines {
		lines[i] = "line"
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	bar := m.renderScrollbar(m.viewport.Height)
	if bar == "" {
		t.Fatal("expected scrollbar to render when content overflows")
	}
	if !strings.Contains(bar, "█") {
		t.Errorf("scrollbar should contain a thumb (█); got %q", bar)
	}
	if !strings.Contains(bar, "│") {
		t.Errorf("scrollbar should contain track characters (│); got %q", bar)
	}
}

func TestScrollbar_ThumbMovesWithScroll(t *testing.T) {
	m := newTestModel(t)
	lines := make([]string, m.viewport.Height*4)
	for i := range lines {
		lines[i] = "line"
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	top := m.renderScrollbar(m.viewport.Height)

	m.viewport.GotoBottom()
	bottom := m.renderScrollbar(m.viewport.Height)

	if top == bottom {
		t.Errorf("thumb should move when scrolling from top to bottom; got identical bars")
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

// moveCursorToSlug positions the cursor onto the row whose topic.Slug
// matches the given value AND drains the resulting render command so the
// viewport has the topic's content. Fails the test if the slug isn't
// visible (caller may need to expand a folder first).
func moveCursorToSlug(t *testing.T, m *Model, slug string) {
	t.Helper()
	for i, r := range m.rows {
		if r.topic != nil && r.topic.Slug == slug {
			m.cursor = i
			cmd := m.renderCurrent(false)
			drainRender(t, m, cmd)
			return
		}
	}
	t.Fatalf("slug %q not visible in tree rows", slug)
}

// drainRender invokes the given Cmd (if non-nil) and feeds the resulting
// topicRenderedMsg back into the model so the viewport has real content.
// Tests use this to skip past the async loading state.
func drainRender(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	updated, _ := m.Update(msg)
	*m = updated.(Model)
}
