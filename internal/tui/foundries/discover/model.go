package discover

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// CastOptions decouples this package from internal/commands. The parent
// translates between this and commands.CastOptions.
type CastOptions struct {
	Global        bool
	WithWorkflows bool
	ValueFiles    []string
	SetOverrides  []string
}

// CastFunc is the install operation injected by the parent.
type CastFunc func(ctx context.Context, source string, opts CastOptions) (foundry.InstalledFile, error)

// CastResultFn signature matches what the parent provides; we discard the
// detail and just look at err for status display.
type CastFn func(ctx context.Context, source string, opts CastOptions) error

// Model is the Discover tab.
type Model struct {
	cfg      *index.Config
	cast     CastFn
	catalog  []data.CatalogEntry
	filtered []data.CatalogEntry
	selected map[string]bool
	cursor   int
	filter   textinput.Model
	loading  bool
	loadErr  error
	castStat map[string]string
}

type catalogLoadedMsg struct {
	catalog []data.CatalogEntry
	err     error
}

type castDoneMsg struct {
	source string
	err    error
}

// New initializes the Discover model and kicks off catalog loading. cast is
// the install operation; pass nil during tests.
func New(cfg *index.Config, cast CastFn) Model {
	ti := textinput.New()
	ti.Placeholder = "filter (name / desc / tag / source)"
	ti.Prompt = "/ "
	return Model{
		cfg:      cfg,
		cast:     cast,
		selected: map[string]bool{},
		filter:   ti,
		loading:  true,
		castStat: map[string]string{},
	}
}

func (m Model) Init() tea.Cmd { return loadCatalogCmd(m.cfg) }

func loadCatalogCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		c, err := data.LoadCatalog(cfg)
		return catalogLoadedMsg{catalog: c, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case catalogLoadedMsg:
		m.loading = false
		m.catalog = msg.catalog
		m.loadErr = msg.err
		m.applyFilter()
		return m, nil
	case castDoneMsg:
		if msg.err != nil {
			m.castStat[msg.source] = "err: " + msg.err.Error()
		} else {
			m.castStat[msg.source] = "ok"
		}
		return m, nil
	case tea.KeyMsg:
		if m.filter.Focused() {
			switch msg.String() {
			case "esc", "enter":
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.applyFilter()
			return m, cmd
		}
		switch msg.String() {
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case " ":
			if m.cursor < len(m.filtered) {
				src := m.filtered[m.cursor].Source
				if m.selected[src] {
					delete(m.selected, src)
				} else {
					m.selected[src] = true
				}
			}
		case "c":
			m.selected = map[string]bool{}
		case "enter":
			cmds := []tea.Cmd{}
			for src := range m.selected {
				m.castStat[src] = "casting"
				cmds = append(cmds, m.castCmd(src))
			}
			return m, tea.Batch(cmds...)
		case "r":
			m.loading = true
			return m, loadCatalogCmd(m.cfg)
		}
	}
	return m, nil
}

func (m Model) castCmd(source string) tea.Cmd {
	cast := m.cast
	return func() tea.Msg {
		if cast == nil {
			return castDoneMsg{source: source, err: fmt.Errorf("no cast function configured")}
		}
		return castDoneMsg{source: source, err: cast(context.Background(), source, CastOptions{})}
	}
}

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		m.filtered = append([]data.CatalogEntry(nil), m.catalog...)
		return
	}
	var out []data.CatalogEntry
	for _, e := range m.catalog {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Description), q) ||
			strings.Contains(strings.ToLower(e.Source), q) {
			out = append(out, e)
			continue
		}
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				out = append(out, e)
				break
			}
		}
	}
	m.filtered = out
	if m.cursor >= len(out) {
		m.cursor = 0
	}
}

func (m Model) View() string {
	if m.loading {
		return "Loading catalog…"
	}
	if m.loadErr != nil {
		return "Error: " + m.loadErr.Error()
	}

	var b strings.Builder
	b.WriteString(m.filter.View() + "\n\n")

	if m.filter.Value() == "" {
		recent := data.Recent(m.catalog, 7*24*time.Hour, 10)
		if len(recent) > 0 {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent (last 7 days)") + "\n")
			for _, e := range recent {
				fmt.Fprintf(&b, "  · %s — %s\n", e.Name, e.Description)
			}
			b.WriteString("\n")
		}
	}

	visible := append([]data.CatalogEntry(nil), m.filtered...)
	sort.Slice(visible, func(i, j int) bool { return visible[i].Name < visible[j].Name })
	for i, e := range visible {
		mark := "[ ]"
		if m.selected[e.Source] {
			mark = "[x]"
		}
		caret := "  "
		if i == m.cursor {
			caret = "▶ "
		}
		verified := ""
		if e.Verified {
			verified = " ✓"
		}
		status := ""
		if s, ok := m.castStat[e.Source]; ok {
			status = "  (" + s + ")"
		}
		fmt.Fprintf(&b, "%s%s %s%s — %s%s\n", caret, mark, e.Name, verified, e.Description, status)
	}

	fmt.Fprintf(&b, "\n%d selected · %d shown · %d total\n", len(m.selected), len(visible), len(m.catalog))
	b.WriteString("space toggle · enter cast all · / search · c clear · r refresh · j/k move\n")
	return b.String()
}
