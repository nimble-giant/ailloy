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
	"github.com/nimble-giant/ailloy/pkg/styles"
)

var (
	headingStyle    = lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	moldNameStyle   = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	verifiedStyle   = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	installedStyle  = lipgloss.NewStyle().Foreground(styles.Info).Bold(true)
	descStyle       = lipgloss.NewStyle().Foreground(styles.LightGray)
	metaStyle       = lipgloss.NewStyle().Foreground(styles.Gray)
	statusOKStyle   = lipgloss.NewStyle().Foreground(styles.Success)
	statusErrStyle  = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
	statusWaitStyle = lipgloss.NewStyle().Foreground(styles.Warning)
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

// UpdateFn fetches every effective foundry's index into the local cache.
// Used to bootstrap an empty Discover view (e.g. on a fresh install where
// the verified default has never been fetched).
type UpdateFn func(cfg *index.Config) error

// Model is the Discover tab.
type Model struct {
	cfg         *index.Config
	cast        CastFn
	update      UpdateFn
	catalog     []data.CatalogEntry
	filtered    []data.CatalogEntry
	installed   map[string]string // source -> installed version (for the "installed" badge)
	selected    map[string]bool
	cursor      int
	filter      textinput.Model
	loading     bool
	loadErr     error
	castStat    map[string]string
	autoFetched bool // guard so we only auto-fetch once per session
}

type catalogLoadedMsg struct {
	catalog []data.CatalogEntry
	err     error
}

type inventoryLoadedMsg struct {
	installed map[string]string // source -> version
}

type catalogFetchedMsg struct{ err error }

type castDoneMsg struct {
	source string
	err    error
}

// New initializes the Discover model and kicks off catalog loading. cast is
// the install operation, update is the foundry-index fetch operation; pass
// nil for either during tests.
func New(cfg *index.Config, cast CastFn, update UpdateFn) Model {
	ti := textinput.New()
	ti.Placeholder = "filter (name / desc / tag / source)"
	ti.Prompt = "/ "
	return Model{
		cfg:       cfg,
		cast:      cast,
		update:    update,
		installed: map[string]string{},
		selected:  map[string]bool{},
		filter:    ti,
		loading:   true,
		castStat:  map[string]string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadCatalogCmd(m.cfg), loadInventoryCmd(m.cfg))
}

func loadCatalogCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		c, err := data.LoadCatalog(cfg)
		return catalogLoadedMsg{catalog: c, err: err}
	}
}

func loadInventoryCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		items, _ := data.LoadInventory(cfg)
		out := make(map[string]string, len(items))
		for _, it := range items {
			// First scope wins (project before global). Both are useful info,
			// but the badge just signals "you have this".
			if _, exists := out[it.Entry.Source]; !exists {
				out[it.Entry.Source] = it.Entry.Version
			}
		}
		return inventoryLoadedMsg{installed: out}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case catalogLoadedMsg:
		m.loading = false
		m.catalog = msg.catalog
		m.loadErr = msg.err
		m.applyFilter()
		// Auto-bootstrap: if the catalog is empty and we have an update fn,
		// fetch all foundry indexes once and reload. Common on first run
		// where the verified default has never been cached.
		if len(m.catalog) == 0 && m.loadErr == nil && m.update != nil && !m.autoFetched {
			m.autoFetched = true
			m.loading = true
			return m, m.fetchCmd()
		}
		return m, nil
	case catalogFetchedMsg:
		// After fetch (success or error), reload from cache.
		return m, loadCatalogCmd(m.cfg)
	case inventoryLoadedMsg:
		m.installed = msg.installed
		return m, nil
	case castDoneMsg:
		if msg.err != nil {
			m.castStat[msg.source] = "err: " + msg.err.Error()
		} else {
			m.castStat[msg.source] = "ok"
		}
		// Refresh inventory so the "installed" badge updates immediately.
		return m, loadInventoryCmd(m.cfg)
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
			return m, tea.Batch(loadCatalogCmd(m.cfg), loadInventoryCmd(m.cfg))
		}
	}
	return m, nil
}

func (m Model) fetchCmd() tea.Cmd {
	update := m.update
	cfg := m.cfg
	return func() tea.Msg {
		if update == nil {
			return catalogFetchedMsg{}
		}
		return catalogFetchedMsg{err: update(cfg)}
	}
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

// CurrentMold returns the highlighted catalog entry's mold reference and the
// scope to use when casting. Returns ok=false when no entry is highlighted.
func (m Model) CurrentMold() (ref string, scope data.Scope, ok bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return "", "", false
	}
	return m.filtered[m.cursor].Source, data.ScopeProject, true
}

// NewWithFiltered constructs a Model directly with a known filtered slice
// and cursor — exported for cross-package tests in the foundries app.
func NewWithFiltered(entries []data.CatalogEntry, cursor int) Model {
	return Model{filtered: entries, cursor: cursor}
}

func (m Model) View() string {
	if m.loading {
		return metaStyle.Render("Loading catalog…")
	}
	if m.loadErr != nil {
		return statusErrStyle.Render("Error: " + m.loadErr.Error())
	}

	var b strings.Builder
	b.WriteString(m.filter.View() + "\n\n")

	if m.filter.Value() == "" {
		recent := data.Recent(m.catalog, 7*24*time.Hour, 10)
		if len(recent) > 0 {
			b.WriteString(headingStyle.Render("Recent (last 7 days)") + "\n")
			for _, e := range recent {
				fmt.Fprintf(&b, "  %s %s %s\n",
					metaStyle.Render("·"),
					moldNameStyle.Render(e.Name),
					descStyle.Render("— "+e.Description))
			}
			b.WriteString("\n")
		}
	}

	visible := append([]data.CatalogEntry(nil), m.filtered...)
	sort.Slice(visible, func(i, j int) bool { return visible[i].Name < visible[j].Name })
	for i, e := range visible {
		mark := "[ ]"
		if m.selected[e.Source] {
			mark = cursorStyle.Render("[x]")
		}
		caret := "  "
		if i == m.cursor {
			caret = cursorStyle.Render("▶ ")
		}
		verified := ""
		if e.Verified {
			verified = " " + verifiedStyle.Render("✓")
		}
		installedTag := ""
		if v, ok := m.installed[e.Source]; ok {
			label := "● installed"
			if v != "" {
				label = "● installed " + v
			}
			installedTag = " " + installedStyle.Render(label)
		}
		status := ""
		if s, ok := m.castStat[e.Source]; ok {
			switch s {
			case "ok":
				status = "  " + statusOKStyle.Render("("+s+")")
			case "casting":
				status = "  " + statusWaitStyle.Render("("+s+")")
			default:
				status = "  " + statusErrStyle.Render("("+s+")")
			}
		}
		fmt.Fprintf(&b, "%s%s %s%s%s %s%s\n",
			caret, mark,
			moldNameStyle.Render(e.Name),
			verified,
			installedTag,
			descStyle.Render("— "+e.Description),
			status)
	}

	fmt.Fprintf(&b, "\n%s\n", metaStyle.Render(fmt.Sprintf("%d selected · %d shown · %d total",
		len(m.selected), len(visible), len(m.catalog))))
	b.WriteString(metaStyle.Render("space toggle · enter cast all · / search · c clear · r refresh · j/k move") + "\n")
	return b.String()
}
