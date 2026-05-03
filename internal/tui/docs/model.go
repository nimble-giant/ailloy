// Package docs implements a bubbletea TUI for browsing ailloy's embedded
// documentation. The screen is split into a left-hand topic list and a
// right-hand scrollable viewport that renders the selected topic via
// glamour. It is launched by `ailloy docs` when stdin/stdout is a TTY.
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

// Bounds for the rendered markdown width inside the right pane.
const (
	minViewportWidth = 32
	minListWidth     = 18
	listWidthRatio   = 30 // percent of total width allocated to the list
	maxListWidth     = 32
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
	FocusBody key.Binding
	FocusList key.Binding
	Quit      key.Binding
	Help      key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		FocusBody: key.NewBinding(key.WithKeys("right", "l", "enter"), key.WithHelp("→/enter", "read")),
		FocusList: key.NewBinding(key.WithKeys("left", "h", "esc"), key.WithHelp("←/esc", "list")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

// Model is the bubbletea model for the docs browser.
type Model struct {
	topics    []clidocs.Topic
	cursor    int    // currently highlighted index in the list
	rendered  string // glamour output for the current topic, cached
	renderErr error
	loadedFor string // slug whose render is in `rendered`
	focus     Focus
	width     int
	height    int
	viewport  viewport.Model
	keys      keyMap
	showHelp  bool
	ready     bool
}

// New constructs a fresh docs Model. The provided topics drive the list;
// pass clidocs.List() in production.
func New(topics []clidocs.Topic) Model {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return Model{
		topics:   topics,
		viewport: vp,
		keys:     defaultKeys(),
		focus:    FocusList,
	}
}

// Init is required by tea.Model. We do not start any commands.
func (m Model) Init() tea.Cmd { return nil }

// Update handles input and resize events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		// Re-render the current topic against the new viewport width.
		m.renderCurrent(true)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Focus-toggle keys work in both states so vim-style h/l always
		// produces visible movement instead of becoming a no-op when the
		// user is already on the destination pane.
		switch {
		case key.Matches(msg, m.keys.FocusBody):
			if m.focus == FocusList {
				m.focus = FocusBody
				return m, nil
			}
		case key.Matches(msg, m.keys.FocusList):
			if m.focus == FocusBody {
				m.focus = FocusList
				return m, nil
			}
		}

		if m.focus == FocusList {
			switch {
			case key.Matches(msg, m.keys.Up):
				m.moveCursor(-1)
				return m, nil
			case key.Matches(msg, m.keys.Down):
				m.moveCursor(1)
				return m, nil
			}
			return m, nil
		}

		// Body focus: handle vim scroll keys explicitly so we don't depend
		// on viewport's default keymap matching, then forward the rest
		// (pgup/pgdn/space/u/d/etc) to the viewport.
		switch {
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

	return m, nil
}

// View renders the current screen. Returns an empty string before the first
// WindowSizeMsg so we don't paint with zero dimensions.
func (m Model) View() string {
	if !m.ready || m.width == 0 {
		return ""
	}

	left := m.renderList()
	right := m.viewport.View()

	listStyle := paneStyle(m.focus == FocusList)
	bodyStyle := paneStyle(m.focus == FocusBody)

	listW, bodyW := m.paneWidths()
	bodyH := m.bodyHeight()

	leftPane := listStyle.Width(listW).Height(bodyH).Render(left)
	rightPane := bodyStyle.Width(bodyW).Height(bodyH).Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// CurrentTopic returns the slug currently highlighted in the list. Useful
// for tests and for callers that want to know what the user picked.
func (m Model) CurrentTopic() string {
	if len(m.topics) == 0 {
		return ""
	}
	return m.topics[m.cursor].Slug
}

// Focus reports which pane currently has input focus. Exposed for tests.
func (m Model) Focus() Focus { return m.focus }

// Rendered returns the cached glamour output for the current topic. Exposed
// for tests so they can verify the body without hitting the viewport API.
func (m Model) Rendered() string { return m.rendered }

func (m *Model) moveCursor(delta int) {
	if len(m.topics) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.topics) {
		m.cursor = len(m.topics) - 1
	}
	m.renderCurrent(false)
}

// renderCurrent ensures m.rendered/viewport reflect the highlighted topic.
// Pass force=true to bypass the cache (e.g. on resize).
func (m *Model) renderCurrent(force bool) {
	if len(m.topics) == 0 {
		m.rendered = ""
		m.viewport.SetContent("")
		return
	}
	slug := m.topics[m.cursor].Slug
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
	width := m.bodyContentWidth()
	rendered, rerr := renderMarkdown(string(body), width)
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

// layout sizes the viewport based on the current window dimensions.
func (m *Model) layout() {
	_, bodyW := m.paneWidths()
	m.viewport.Width = bodyW - 2 // subtract the rounded-border columns
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
	}
	if list+body > m.width {
		list = m.width - body
		if list < minListWidth {
			list = minListWidth
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

// bodyContentWidth is the rendering width passed to glamour. Accounts for
// the rounded border + padding on the right pane.
func (m Model) bodyContentWidth() int {
	_, bodyW := m.paneWidths()
	w := bodyW - 4 // border + padding
	if w < minViewportWidth {
		w = minViewportWidth
	}
	return w
}

func (m Model) renderList() string {
	if len(m.topics) == 0 {
		return styles.SubtleStyle.Render("(no topics)")
	}
	var b strings.Builder
	for i, t := range m.topics {
		row := t.Slug
		switch {
		case i == m.cursor && m.focus == FocusList:
			row = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(styles.Primary1).
				Bold(true).
				Padding(0, 1).
				Render("▸ " + row)
		case i == m.cursor:
			row = lipgloss.NewStyle().
				Foreground(styles.Primary2).
				Bold(true).
				Padding(0, 1).
				Render("▸ " + row)
		default:
			row = lipgloss.NewStyle().Padding(0, 1).Render("  " + row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary1).
		Render(" ailloy docs ")

	listLabel := paneLabel("LIST", m.focus == FocusList)
	bodyLabel := paneLabel("BODY", m.focus == FocusBody)
	focusInfo := listLabel + " " + bodyLabel

	right := ""
	if len(m.topics) > 0 {
		t := m.topics[m.cursor]
		summary := fmt.Sprintf(" %s — %s ", t.Title, t.Summary)
		if m.focus == FocusBody {
			summary = fmt.Sprintf(" %s — %3.0f%% ", t.Title, m.viewport.ScrollPercent()*100)
		}
		right = styles.SubtleStyle.Render(summary)
	}

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(focusInfo) - lipgloss.Width(right) - 1
	if gap < 1 {
		gap = 1
	}
	return title + " " + focusInfo + strings.Repeat(" ", gap) + right
}

// paneLabel renders a small "[LIST]" / "[BODY]" pill that highlights the
// currently focused pane so the user always knows which keys do what.
func paneLabel(name string, active bool) string {
	style := lipgloss.NewStyle().Padding(0, 1)
	if active {
		style = style.
			Foreground(lipgloss.Color("0")).
			Background(styles.Primary1).
			Bold(true)
	} else {
		style = style.
			Foreground(styles.Primary2).
			Faint(true)
	}
	return style.Render(name)
}

func (m Model) renderFooter() string {
	if m.showHelp {
		help := []string{
			m.keys.Up.Help().Key + " " + m.keys.Up.Help().Desc,
			m.keys.Down.Help().Key + " " + m.keys.Down.Help().Desc,
			m.keys.FocusBody.Help().Key + " " + m.keys.FocusBody.Help().Desc,
			m.keys.FocusList.Help().Key + " " + m.keys.FocusList.Help().Desc,
			"pgup/pgdn scroll",
			m.keys.Quit.Help().Key + " " + m.keys.Quit.Help().Desc,
		}
		return styles.SubtleStyle.Render(" " + strings.Join(help, "  ·  ") + " ")
	}
	var hint string
	if m.focus == FocusList {
		hint = "j/k or ↑/↓ pick topic  ·  l/→/enter read  ·  ?: help  ·  q: quit"
	} else {
		hint = "j/k or ↑/↓ scroll  ·  pgup/pgdn page  ·  h/←/esc back to list  ·  q: quit"
	}
	return styles.SubtleStyle.Render(" " + hint + " ")
}

func paneStyle(focused bool) lipgloss.Style {
	border := lipgloss.RoundedBorder()
	color := styles.Primary2
	if focused {
		color = styles.Primary1
	}
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(color).
		Padding(0, 1)
}

// Run launches the docs TUI in alternate-screen mode and blocks until the
// user quits.
func Run() error {
	topics := clidocs.List()
	if len(topics) == 0 {
		return fmt.Errorf("no embedded docs topics available")
	}
	m := New(topics)
	p := tea.NewProgram(m, tea.WithAltScreen())
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
