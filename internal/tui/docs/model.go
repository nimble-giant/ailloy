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
	headerHeight     = 1
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

// Model is the bubbletea model for the docs browser.
type Model struct {
	tree      *clidocs.Node
	rows      []row // flattened, only-visible rows
	cursor    int
	expanded  map[string]bool
	rendered  string
	renderErr error
	loadedFor string
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
	m := Model{
		tree:     tree,
		expanded: map[string]bool{},
		viewport: vp,
		keys:     defaultKeys(),
		focus:    FocusList,
	}
	// Auto-expand every directory so the user sees the full tree on first
	// open. They can still collapse anything they don't want with h/←.
	expandAll(tree, m.expanded)
	m.rebuildRows()
	return m
}

// Init satisfies tea.Model. The TUI does not start any commands.
func (m Model) Init() tea.Cmd { return nil }

// Update handles input and resize events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.renderCurrent(true)
		m.ready = true
		return m, nil

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
	switch {
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.Expand):
		m.expandOrFocus()
	case key.Matches(msg, m.keys.Collapse):
		m.collapseOrJumpToParent()
	case key.Matches(msg, m.keys.OpenBody):
		m.expandOrFocus()
	}
	return m, nil
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
	right := m.viewport.View()

	leftPane := paneStyle(m.focus == FocusList).Width(listW).Height(bodyH).Render(left)
	rightPane := paneStyle(m.focus == FocusBody).Width(bodyW).Height(bodyH).Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
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

func (m *Model) moveCursor(delta int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	m.renderCurrent(false)
}

// expandOrFocus is bound to →/l/enter on the list. Behavior:
//   - If the row is a collapsed directory: expand it.
//   - If the row is an expanded directory: jump to its first child.
//   - If the row is a file: focus the body pane.
func (m *Model) expandOrFocus() {
	if len(m.rows) == 0 {
		return
	}
	r := m.rows[m.cursor]
	if r.isDir {
		if !m.expanded[r.dirPath] {
			m.expanded[r.dirPath] = true
			m.rebuildRows()
			return
		}
		// Already expanded — step into the first child if there is one.
		if m.cursor+1 < len(m.rows) && m.rows[m.cursor+1].depth > r.depth {
			m.cursor++
			m.renderCurrent(false)
		}
		return
	}
	m.focus = FocusBody
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
	// Find the closest ancestor row above us with a lower depth.
	for i := m.cursor - 1; i >= 0; i-- {
		if m.rows[i].depth < r.depth {
			m.cursor = i
			m.renderCurrent(false)
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

// renderCurrent ensures rendered+viewport reflect the highlighted row.
func (m *Model) renderCurrent(force bool) {
	if len(m.rows) == 0 || m.rows[m.cursor].topic == nil {
		// Directory row — leave the body intact rather than blanking it.
		return
	}
	slug := m.rows[m.cursor].topic.Slug
	if !force && slug == m.loadedFor {
		return
	}
	body, err := clidocs.Read(slug)
	if err != nil {
		m.renderErr = err
		m.rendered = err.Error()
		m.loadedFor = slug
		m.viewport.SetContent(m.rendered)
		m.viewport.GotoTop()
		return
	}
	rendered, rerr := renderMarkdown(string(body), m.bodyContentWidth())
	if rerr != nil {
		m.renderErr = rerr
		m.rendered = rerr.Error()
	} else {
		m.renderErr = nil
		m.rendered = rendered
	}
	m.loadedFor = slug
	m.viewport.SetContent(m.rendered)
	m.viewport.GotoTop()
}

func (m *Model) layout() {
	_, bodyW := m.paneWidths()
	m.viewport.Width = bodyW - 2
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
	w := bodyW - 4
	if w < minViewportWidth {
		w = minViewportWidth
	}
	return w
}

// renderList draws the collapsible tree into the left pane.
func (m Model) renderList(width int) string {
	if len(m.rows) == 0 {
		return styles.SubtleStyle.Render("(no topics)")
	}
	cellWidth := width - 4 // border + inner padding
	if cellWidth < 8 {
		cellWidth = 8
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
		label := r.name
		row := indent + marker + label
		row = clipLine(row, cellWidth)

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

// renderHeader draws the thin branded top bar with logo + active topic.
func (m Model) renderHeader() string {
	logo := headerLogoStyle.Render(" 🦊 Ailloy Docs ")

	var center string
	if cur := m.currentRow(); cur != nil {
		if cur.topic != nil {
			center = headerCenterStyle.Render(cur.topic.Title)
		} else {
			center = headerCenterStyle.Render(cur.name + "/")
		}
	}

	right := headerRightStyle.Render(m.headerStatus())

	gap := m.width - lipgloss.Width(logo) - lipgloss.Width(center) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	leftPad := gap / 2
	rightPad := gap - leftPad
	return logo +
		strings.Repeat(" ", leftPad) +
		center +
		strings.Repeat(" ", rightPad) +
		right
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

var (
	headerLogoStyle = lipgloss.NewStyle().
			Foreground(styles.White).
			Background(styles.Accent1).
			Bold(true)

	headerCenterStyle = lipgloss.NewStyle().
				Foreground(styles.Accent1).
				Bold(true)

	headerRightStyle = lipgloss.NewStyle().
				Foreground(styles.Gray).
				Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(styles.Gray)

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
