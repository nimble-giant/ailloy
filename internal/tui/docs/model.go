// Package docs implements a bubbletea TUI for browsing ailloy's embedded
// documentation. The screen is a thin branded header on top, a left-hand
// collapsible tree of topics, and a right-hand scrollable viewport that
// renders the selected topic via glamour. It is launched by `ailloy docs`
// when stdin/stdout is a TTY.
package docs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	clidocs "github.com/nimble-giant/ailloy/docs"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Layout bounds. The TUI grows to fill the terminal but never collapses
// either pane below these widths.
const (
	minViewportWidth = 32
	minListWidth     = 22
	listWidthRatio   = 32 // percent of total width allocated to the tree
	maxListWidth     = 38
	footerHeight     = 1
	headerHeight     = 2
)

// Focus identifies which pane currently receives input.
type Focus int

const (
	FocusList Focus = iota
	FocusBody
)

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Expand    key.Binding // h on a folder collapses; l on a folder expands. On a file, l focuses body.
	Collapse  key.Binding // h on a file collapses parent and moves there.
	OpenBody  key.Binding // enter on a file focuses the body.
	FocusList key.Binding
	Quit      key.Binding
	Help      key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Expand:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "expand/read")),
		Collapse:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "collapse/back")),
		OpenBody:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		FocusList: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back to tree")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

// row is one rendered line in the tree pane. It holds either a directory
// (with depth + path) or a topic leaf.
type row struct {
	depth   int
	isDir   bool
	dirPath string // for directories, the FS path used as the expanded-set key
	name    string // display name (last segment)
	topic   *clidocs.Topic
}

// renderCacheKey combines a slug with the render width so cached output is
// reused when the user revisits a topic at the same window size and
// invalidated automatically on resize.
type renderCacheKey struct {
	slug  string
	width int
}

// topicRenderedMsg is dispatched when an async glamour render completes.
type topicRenderedMsg struct {
	key      renderCacheKey
	rendered string
	err      error
}

// Model is the bubbletea model for the docs browser.
type Model struct {
	tree      *clidocs.Node
	rows      []row // flattened, only-visible rows
	cursor    int
	expanded  map[string]bool
	rendered  string
	renderErr error
	loadedFor string
	cache     map[renderCacheKey]string
	loading   bool   // a render is currently in flight
	pending   string // slug whose render is in flight
	spinner   spinner.Model
	focus     Focus
	width     int
	height    int
	viewport  viewport.Model
	keys      keyMap
	showHelp  bool
	ready     bool
}

// New constructs a fresh docs Model from a clidocs.Tree.
func New(tree *clidocs.Node) Model {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	sp := spinner.New(spinner.WithSpinner(brandSpinner))
	sp.Style = spinnerStyle

	m := Model{
		tree:     tree,
		expanded: map[string]bool{},
		cache:    map[renderCacheKey]string{},
		spinner:  sp,
		viewport: vp,
		keys:     defaultKeys(),
		focus:    FocusList,
	}
	// Auto-expand every directory so the user sees the full tree on first
	// open. They can still collapse anything they don't want with h/←.
	expandAll(tree, m.expanded)
	m.rebuildRows()
	// Prefer "getting-started" so first-time users land on the quickstart
	// rather than a nested tutorial. Fall back to the first available leaf.
	startCursor := -1
	for i, r := range m.rows {
		if r.topic == nil {
			continue
		}
		if r.topic.Slug == "getting-started" {
			startCursor = i
			break
		}
		if startCursor < 0 {
			startCursor = i
		}
	}
	if startCursor >= 0 {
		m.cursor = startCursor
	}
	return m
}

// Init kicks the spinner so it animates while a render is in flight. The
// spinner is harmless when no render is loading.
func (m Model) Init() tea.Cmd { return m.spinner.Tick }

// Update handles input, resize, async render results, and spinner ticks.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		// Width changed → invalidate the cache so renders match the new
		// glamour word-wrap width.
		m.cache = map[renderCacheKey]string{}
		cmd := m.renderCurrent(true)
		m.ready = true
		return m, cmd

	case topicRenderedMsg:
		m.cache[msg.key] = msg.rendered
		// Apply only if it still matches what the user is looking at.
		if msg.key.slug == m.pending {
			m.loading = false
			m.pending = ""
			m.loadedFor = msg.key.slug
			m.renderErr = msg.err
			m.rendered = msg.rendered
			m.viewport.SetContent(msg.rendered)
			m.viewport.GotoTop()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Globals.
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

		if m.focus == FocusList {
			return m.updateList(msg)
		}
		return m.updateBody(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case key.Matches(msg, m.keys.Up):
		cmd = m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		cmd = m.moveCursor(1)
	case key.Matches(msg, m.keys.Expand):
		cmd = m.expandOrFocus()
	case key.Matches(msg, m.keys.Collapse):
		m.collapseOrJumpToParent()
	case key.Matches(msg, m.keys.OpenBody):
		cmd = m.expandOrFocus()
	}
	return m, cmd
}

func (m Model) updateBody(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusList), key.Matches(msg, m.keys.Collapse):
		m.focus = FocusList
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.viewport.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.viewport.ScrollDown(1)
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the current screen.
func (m Model) View() string {
	if !m.ready || m.width == 0 {
		return ""
	}

	listW, bodyW := m.paneWidths()
	bodyH := m.bodyHeight()

	left := m.renderList(listW)
	right := m.bodyContent(bodyW, bodyH)

	leftPane := paneStyle(m.focus == FocusList).Width(listW).Height(bodyH).Render(left)
	rightPane := paneStyle(m.focus == FocusBody).Width(bodyW).Height(bodyH).Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// bodyContent returns either the rendered viewport (with a thin scrollbar
// down the right edge) or, when a render is in flight, a centered branded
// spinner so users see immediate feedback for long-loading docs.
func (m Model) bodyContent(width, height int) string {
	if !m.loading {
		return m.composeViewportWithScrollbar()
	}
	innerW := width - 2
	innerH := height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	title := m.pending
	if r := m.currentRow(); r != nil && r.topic != nil {
		title = r.topic.Title
	}
	msg := lipgloss.JoinVertical(
		lipgloss.Center,
		spinnerLineStyle.Render(m.spinner.View()+"  Forging "+title+"…"),
		spinnerHintStyle.Render("rendering with glamour"),
	)
	return lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, msg)
}

// composeViewportWithScrollbar renders the viewport's current view and
// joins it horizontally with a 1-column scrollbar. The track is dimmed,
// the thumb is the brand orange, and both vanish when the content fits
// entirely on screen.
func (m Model) composeViewportWithScrollbar() string {
	view := m.viewport.View()
	bar := m.renderScrollbar(m.viewport.Height)
	if bar == "" {
		return view
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, view, bar)
}

// renderScrollbar returns a vertical scrollbar of the requested height,
// or "" when there's nothing to scroll. The thumb height is proportional
// to the visible fraction of the document, and its position to the
// scroll percentage — the same math used by mainstream readers.
func (m Model) renderScrollbar(height int) string {
	if height <= 0 {
		return ""
	}
	total := m.viewport.TotalLineCount()
	visible := m.viewport.VisibleLineCount()
	if total <= visible || total == 0 {
		return ""
	}

	thumbSize := visible * height / total
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > height {
		thumbSize = height
	}
	travel := height - thumbSize
	thumbStart := 0
	if travel > 0 {
		thumbStart = int(float64(travel)*m.viewport.ScrollPercent() + 0.5)
		if thumbStart > travel {
			thumbStart = travel
		}
	}

	var b strings.Builder
	for i := range height {
		if i >= thumbStart && i < thumbStart+thumbSize {
			b.WriteString(scrollbarThumbStyle.Render("█"))
		} else {
			b.WriteString(scrollbarTrackStyle.Render("│"))
		}
		if i < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// CurrentTopic returns the slug currently highlighted, or "" if the
// cursor sits on a directory row.
func (m Model) CurrentTopic() string {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return ""
	}
	r := m.rows[m.cursor]
	if r.topic == nil {
		return ""
	}
	return r.topic.Slug
}

// Focus reports which pane currently has input focus.
func (m Model) Focus() Focus { return m.focus }

// Rendered returns the cached glamour output for the current topic.
func (m Model) Rendered() string { return m.rendered }

// IsExpanded reports whether the given directory path is expanded. Exposed
// for tests.
func (m Model) IsExpanded(dir string) bool { return m.expanded[dir] }

func (m *Model) moveCursor(delta int) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	return m.renderCurrent(false)
}

// expandOrFocus is bound to →/l/enter on the list. Behavior:
//   - If the row is a collapsed directory: expand it.
//   - If the row is an expanded directory: jump to its first child.
//   - If the row is a file: focus the body pane.
func (m *Model) expandOrFocus() tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	r := m.rows[m.cursor]
	if r.isDir {
		if !m.expanded[r.dirPath] {
			m.expanded[r.dirPath] = true
			m.rebuildRows()
			return nil
		}
		// Already expanded — step into the first child if there is one.
		if m.cursor+1 < len(m.rows) && m.rows[m.cursor+1].depth > r.depth {
			m.cursor++
			return m.renderCurrent(false)
		}
		return nil
	}
	m.focus = FocusBody
	return nil
}

// collapseOrJumpToParent is bound to ←/h on the list. Behavior:
//   - If the row is an expanded directory: collapse it.
//   - Otherwise: jump the cursor to the parent directory row, if any.
func (m *Model) collapseOrJumpToParent() {
	if len(m.rows) == 0 {
		return
	}
	r := m.rows[m.cursor]
	if r.isDir && m.expanded[r.dirPath] {
		m.expanded[r.dirPath] = false
		m.rebuildRows()
		return
	}
	for i := m.cursor - 1; i >= 0; i-- {
		if m.rows[i].depth < r.depth {
			m.cursor = i
			_ = m.renderCurrent(false)
			return
		}
	}
}

// expandAll marks every directory in the tree as expanded.
func expandAll(n *clidocs.Node, set map[string]bool) {
	if n == nil {
		return
	}
	for _, c := range n.Children {
		if c.IsDir {
			set[c.Path] = true
			expandAll(c, set)
		}
	}
}

// rebuildRows flattens the tree into the currently-visible rows based on
// the expanded-set, then clamps the cursor so it always points at a row.
func (m *Model) rebuildRows() {
	m.rows = m.rows[:0]
	if m.tree == nil {
		return
	}
	prevPath := ""
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		if m.rows[m.cursor].topic != nil {
			prevPath = m.rows[m.cursor].topic.File
		} else {
			prevPath = m.rows[m.cursor].dirPath
		}
	}
	var walk func(n *clidocs.Node, depth int)
	walk = func(n *clidocs.Node, depth int) {
		for _, c := range n.Children {
			if c.IsDir {
				m.rows = append(m.rows, row{
					depth:   depth,
					isDir:   true,
					dirPath: c.Path,
					name:    c.Name,
				})
				if m.expanded[c.Path] {
					walk(c, depth+1)
				}
			} else {
				topic := c.Topic
				m.rows = append(m.rows, row{
					depth: depth,
					isDir: false,
					name:  topic.Title,
					topic: &topic,
				})
			}
		}
	}
	walk(m.tree, 0)

	// Restore cursor to the previously-selected path when possible.
	if prevPath != "" {
		for i, r := range m.rows {
			if r.isDir && r.dirPath == prevPath {
				m.cursor = i
				return
			}
			if r.topic != nil && r.topic.File == prevPath {
				m.cursor = i
				return
			}
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// renderCurrent ensures the viewport reflects the highlighted row. If the
// rendered output is already cached, it is applied synchronously and no
// command is returned. Otherwise a tea.Cmd is returned that performs the
// glamour render off the Update goroutine and dispatches a topicRenderedMsg
// when complete; the spinner shows in the body until the message arrives.
func (m *Model) renderCurrent(force bool) tea.Cmd {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	r := m.rows[m.cursor]
	if r.topic == nil {
		// Directory row — leave the body intact rather than blanking it.
		return nil
	}
	slug := r.topic.Slug
	if !force && slug == m.loadedFor {
		return nil
	}
	key := renderCacheKey{slug: slug, width: m.bodyContentWidth()}
	if cached, ok := m.cache[key]; ok {
		m.loading = false
		m.pending = ""
		m.loadedFor = slug
		m.renderErr = nil
		m.rendered = cached
		m.viewport.SetContent(cached)
		m.viewport.GotoTop()
		return nil
	}
	// Cache miss — render asynchronously so the UI stays responsive on big
	// docs. The body shows the branded spinner until the result arrives.
	m.loading = true
	m.pending = slug
	return func() tea.Msg {
		body, err := clidocs.Read(slug)
		if err != nil {
			return topicRenderedMsg{key: key, rendered: err.Error(), err: err}
		}
		rendered, rerr := renderMarkdown(string(body), key.width)
		if rerr != nil {
			return topicRenderedMsg{key: key, rendered: rerr.Error(), err: rerr}
		}
		return topicRenderedMsg{key: key, rendered: rendered}
	}
}

func (m *Model) layout() {
	_, bodyW := m.paneWidths()
	// Inner viewport width = body pane minus rounded border (2) and scrollbar (1).
	m.viewport.Width = bodyW - 3
	if m.viewport.Width < 1 {
		m.viewport.Width = 1
	}
	m.viewport.Height = m.bodyHeight() - 2
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
}

func (m Model) paneWidths() (list, body int) {
	list = m.width * listWidthRatio / 100
	if list > maxListWidth {
		list = maxListWidth
	}
	if list < minListWidth {
		list = minListWidth
	}
	body = m.width - list
	if body < minViewportWidth {
		body = minViewportWidth
		if list+body > m.width && m.width > minViewportWidth {
			list = m.width - body
			if list < minListWidth {
				list = minListWidth
			}
		}
	}
	return list, body
}

func (m Model) bodyHeight() int {
	h := m.height - headerHeight - footerHeight
	if h < 3 {
		h = 3
	}
	return h
}

func (m Model) bodyContentWidth() int {
	_, bodyW := m.paneWidths()
	// Body pane − border (2) − inner padding (2) − scrollbar column (1).
	w := bodyW - 5
	if w < minViewportWidth {
		w = minViewportWidth
	}
	return w
}

// renderList draws the collapsible tree into the left pane. Content is
// hard-clipped so long titles never wrap onto a second line (the row
// styles use horizontal padding internally; we account for that here so
// the styled cells match the content width exactly).
func (m Model) renderList(width int) string {
	if len(m.rows) == 0 {
		return styles.SubtleStyle.Render("(no topics)")
	}
	// `width` is the outer pane width. The pane chrome (rounded border
	// + 1ch padding on each side) eats 4 cells, so the row cells span
	// width - 4. Each row style then adds its own 1ch padding on each
	// side, leaving cellWidth - 2 cells for actual text. Clipping to
	// that exact width prevents wrapping onto a second line.
	cellWidth := width - 4
	if cellWidth < 8 {
		cellWidth = 8
	}
	contentWidth := cellWidth - 2
	if contentWidth < 4 {
		contentWidth = 4
	}

	var b strings.Builder
	for i, r := range m.rows {
		marker := "  "
		switch {
		case r.isDir && m.expanded[r.dirPath]:
			marker = "▾ "
		case r.isDir:
			marker = "▸ "
		}
		indent := strings.Repeat("  ", r.depth)
		row := clipLine(indent+marker+treeLabel(r), contentWidth)

		switch {
		case i == m.cursor && m.focus == FocusList:
			row = listRowActiveStyle.Width(cellWidth).Render(row)
		case i == m.cursor:
			row = listRowFocusedDimStyle.Width(cellWidth).Render(row)
		case r.isDir:
			row = listRowDirStyle.Width(cellWidth).Render(row)
		default:
			row = listRowFileStyle.Width(cellWidth).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// treeLabel picks the row label rendered in the tree pane. Directories
// use their folder name; files use the slug basename (e.g. "anneal",
// "first-mold") because slugs are short and predictable, and the full
// human title stays visible in the header above the body.
func treeLabel(r row) string {
	if r.isDir || r.topic == nil {
		return r.name
	}
	if base := slugBase(r.topic.Slug); base != "" {
		return base
	}
	return r.topic.Title
}

func slugBase(slug string) string {
	idx := strings.LastIndexByte(slug, '/')
	if idx < 0 {
		return slug
	}
	return slug[idx+1:]
}

// renderHeader draws a 2-line full-width branded top bar. The entire bar
// has an orange background so it's unmistakable on any terminal theme.
//
//	[ORANGE BAR]  🦊 Ailloy Docs                              TREE / BODY · 42%
//	[ORANGE BAR]  Configuration Wizard (`ailloy anneal`)
//
// We pre-pad each line to the full width and feed the result to a style
// that only sets foreground/background/bold — no Width() trickery — so
// nothing about the bar can be silently swallowed by terminal quirks.
func (m Model) renderHeader() string {
	logo := "🦊 Ailloy Docs"

	var topicTitle string
	if cur := m.currentRow(); cur != nil {
		if cur.topic != nil {
			topicTitle = cur.topic.Title
		} else {
			topicTitle = cur.name + "/"
		}
	}

	right := m.headerStatus()

	line1 := layoutHeaderLine(m.width, logo, "", right)
	line2 := layoutHeaderLine(m.width, "  "+topicTitle, "", "")

	return headerBarStyle.Render(line1) + "\n" + headerBarStyle.Render(line2)
}

// layoutHeaderLine builds a single header line of exactly `width` runes
// with `left` flush-left, `center` centered, and `right` flush-right.
// Empty sections collapse cleanly. Anything that doesn't fit is clipped.
func layoutHeaderLine(width int, left, center, right string) string {
	if width < 1 {
		return ""
	}
	const sidePad = 1
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	// Clip left/right if they alone overflow the line.
	if leftW+rightW+sidePad*2 > width {
		left = clipLine(left, width-rightW-sidePad*2)
		leftW = lipgloss.Width(left)
	}

	free := width - leftW - rightW - sidePad*2
	if free < 0 {
		free = 0
	}
	if lipgloss.Width(center) > free {
		center = clipLine(center, free)
	}
	centerW := lipgloss.Width(center)

	totalPad := width - leftW - centerW - rightW - sidePad*2
	if totalPad < 0 {
		totalPad = 0
	}
	leftGap := totalPad / 2
	rightGap := totalPad - leftGap

	return " " + left +
		strings.Repeat(" ", leftGap) +
		center +
		strings.Repeat(" ", rightGap) +
		right + " "
}

// headerStatus is a short right-aligned indicator: focused pane + (in body
// focus) the scroll percentage.
func (m Model) headerStatus() string {
	if m.focus == FocusBody {
		return fmt.Sprintf(" BODY · %3.0f%% ", m.viewport.ScrollPercent()*100)
	}
	return " TREE "
}

// renderFooter shows context-aware key hints, swapping to a verbose help
// row when the user presses ?.
func (m Model) renderFooter() string {
	if m.showHelp {
		help := []string{
			m.keys.Up.Help().Key + " " + m.keys.Up.Help().Desc,
			m.keys.Down.Help().Key + " " + m.keys.Down.Help().Desc,
			m.keys.Expand.Help().Key + " " + m.keys.Expand.Help().Desc,
			m.keys.Collapse.Help().Key + " " + m.keys.Collapse.Help().Desc,
			m.keys.OpenBody.Help().Key + " " + m.keys.OpenBody.Help().Desc,
			"pgup/pgdn page",
			m.keys.Quit.Help().Key + " " + m.keys.Quit.Help().Desc,
		}
		return footerStyle.Render(" " + strings.Join(help, "  ·  ") + " ")
	}
	var hint string
	if m.focus == FocusList {
		hint = "j/k navigate  ·  l/→ expand / read  ·  h/← collapse / back  ·  enter read  ·  ?: help  ·  q: quit"
	} else {
		hint = "j/k or ↑/↓ scroll  ·  pgup/pgdn page  ·  esc/h back to tree  ·  q: quit"
	}
	return footerStyle.Render(" " + hint + " ")
}

func (m Model) currentRow() *row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	r := m.rows[m.cursor]
	return &r
}

// clipLine truncates a string to n display columns with an ellipsis. Pure
// byte length is fine here because all our slugs/titles are ASCII; if
// non-ASCII titles appear later, swap to runewidth.
func clipLine(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	out := []rune(s)
	for len(out) > 0 && lipgloss.Width(string(out)+"…") > n {
		out = out[:len(out)-1]
	}
	return string(out) + "…"
}

// Run launches the docs TUI in alternate-screen mode and blocks until the
// user quits.
func Run() error {
	tree := clidocs.Tree()
	if tree == nil || len(tree.Children) == 0 {
		return fmt.Errorf("no embedded docs topics available")
	}
	p := tea.NewProgram(New(tree), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// renderMarkdown is a small wrapper around glamour with auto style and
// word-wrap pinned to the requested width.
func renderMarkdown(md string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()
	return r.Render(md)
}

// ----------------------------------------------------------------------
// Branded styles. Orange (Accent1) is Ailloy's primary highlight color;
// purple (Primary1/Primary2) provides supporting structure.
// ----------------------------------------------------------------------

// brandSpinner is a fox/forge themed spinner. The frames evoke ailloy's
// metalwork motif so the loading state ties into the rest of the brand.
var brandSpinner = spinner.Spinner{
	Frames: []string{
		"🦊 ⚒  ",
		"🦊 ⚒ ✦",
		"🦊  ⚒ ",
		"🦊 ⚒  ",
		"🦊✦⚒  ",
		"🦊 ⚒ ✦",
	},
	FPS: 8,
}

var (
	// headerBarStyle paints the entire header line with the brand orange
	// background so it's visible at a glance no matter the terminal theme.
	headerBarStyle = lipgloss.NewStyle().
			Foreground(styles.White).
			Background(styles.Accent1).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(styles.Gray)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(styles.Accent1).
			Bold(true)

	spinnerLineStyle = lipgloss.NewStyle().
				Foreground(styles.Accent1).
				Bold(true)

	spinnerHintStyle = lipgloss.NewStyle().
				Foreground(styles.Gray).
				Italic(true)

	// Reading-pane scrollbar. Thumb uses the brand orange so it stays
	// consistent with focus borders and list highlights; the track is a
	// muted column that disappears against most terminal backgrounds.
	scrollbarThumbStyle = lipgloss.NewStyle().
				Foreground(styles.Accent1)

	scrollbarTrackStyle = lipgloss.NewStyle().
				Foreground(styles.Gray).
				Faint(true)

	listRowActiveStyle = lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Accent1).
				Bold(true).
				Padding(0, 1)

	listRowFocusedDimStyle = lipgloss.NewStyle().
				Foreground(styles.Accent1).
				Bold(true).
				Padding(0, 1)

	listRowDirStyle = lipgloss.NewStyle().
			Foreground(styles.Primary1).
			Padding(0, 1)

	listRowFileStyle = lipgloss.NewStyle().
				Foreground(styles.LightGray).
				Padding(0, 1)
)

func paneStyle(focused bool) lipgloss.Style {
	border := lipgloss.RoundedBorder()
	color := styles.Gray
	if focused {
		color = styles.Accent1
	}
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(color).
		Padding(0, 1)
}
