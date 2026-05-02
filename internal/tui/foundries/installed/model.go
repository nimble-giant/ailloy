package installed

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

var (
	headingStyle  = lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	moldNameStyle = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	verifiedStyle = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	scopeStyle    = lipgloss.NewStyle().Foreground(styles.Primary2)
	versionStyle  = lipgloss.NewStyle().Foreground(styles.Info)
	metaStyle     = lipgloss.NewStyle().Foreground(styles.Gray)
	warnStyle     = lipgloss.NewStyle().Foreground(styles.Warning)
	flashOK       = lipgloss.NewStyle().Foreground(styles.Success)
	flashErr      = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
)

// CastOptions decouples this package from internal/commands.
type CastOptions struct {
	Global        bool
	WithWorkflows bool
	ValueFiles    []string
	SetOverrides  []string
}

// CastFn is the install operation injected by the parent (used for `u` update).
type CastFn func(ctx context.Context, source string, opts CastOptions) error

// Model is the Installed tab.
type Model struct {
	cfg     *index.Config
	cast    CastFn
	items   []data.InventoryItem
	cursor  int
	loading bool
	loadErr error
	flash   string
	pending map[string][]string // moldRef → encoded "--set" overrides
}

type loadedMsg struct {
	items []data.InventoryItem
	err   error
}

type uninstallDoneMsg struct {
	source string
	res    foundry.UninstallResult
	err    error
}

type updateDoneMsg struct {
	source string
	err    error
}

func New(cfg *index.Config, cast CastFn) Model {
	return Model{cfg: cfg, cast: cast, loading: true}
}

func (m Model) Init() tea.Cmd { return loadCmd(m.cfg) }

func loadCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		items, err := data.LoadInventory(cfg)
		return loadedMsg{items: items, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		m.loading = false
		m.items = msg.items
		m.loadErr = msg.err
		return m, nil
	case uninstallDoneMsg:
		if msg.err != nil {
			m.flash = "uninstall error: " + msg.err.Error()
		} else {
			m.flash = fmt.Sprintf("removed %d files (skipped %d, retained %d)",
				len(msg.res.Deleted), len(msg.res.SkippedModified), len(msg.res.Retained))
		}
		return m, loadCmd(m.cfg)
	case updateDoneMsg:
		if msg.err != nil {
			m.flash = "update error: " + msg.err.Error()
		} else {
			m.flash = "updated " + msg.source
			delete(m.pending, msg.source)
		}
		return m, loadCmd(m.cfg)
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "x":
			if m.cursor < len(m.items) {
				return m, uninstallCmd(m.items[m.cursor])
			}
		case "u":
			if m.cursor < len(m.items) {
				return m, m.updateCmd(m.items[m.cursor])
			}
		case "r":
			m.loading = true
			return m, loadCmd(m.cfg)
		}
	}
	return m, nil
}

func uninstallCmd(it data.InventoryItem) tea.Cmd {
	return func() tea.Msg {
		res, err := foundry.UninstallMold(it.ManifestPath, it.Entry.Source, foundry.UninstallOptions{})
		return uninstallDoneMsg{source: it.Entry.Source, res: res, err: err}
	}
}

func (m Model) updateCmd(it data.InventoryItem) tea.Cmd {
	cast := m.cast
	// Capture pending overrides at Cmd build time so concurrent picker edits
	// don't change what's already in flight.
	extra := append([]string(nil), m.pending[it.Entry.Source]...)
	return func() tea.Msg {
		if cast == nil {
			return updateDoneMsg{source: it.Entry.Source, err: fmt.Errorf("no cast function configured")}
		}
		opts := CastOptions{Global: it.Scope == data.ScopeGlobal}
		if len(extra) > 0 {
			opts.SetOverrides = append(opts.SetOverrides, extra...)
		}
		err := cast(context.Background(), it.Entry.Source, opts)
		return updateDoneMsg{source: it.Entry.Source, err: err}
	}
}

// CurrentMold returns the highlighted installed mold's source and its scope
// (project or global). Returns ok=false when no item is highlighted.
func (m Model) CurrentMold() (ref string, scope data.Scope, ok bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return "", "", false
	}
	it := m.items[m.cursor]
	return it.Entry.Source, it.Scope, true
}

// ApplySessionOverrides records session overrides for the given mold ref.
// They will be applied as --set overrides on the next cast of that mold.
// Last-write-wins: re-applying replaces any prior pending overrides for the
// same ref rather than merging.
func (m Model) ApplySessionOverrides(moldRef string, overrides map[string]any) Model {
	if m.pending == nil {
		m.pending = map[string][]string{}
	}
	m.pending[moldRef] = encodeSetOverrides(overrides)
	return m
}

// encodeSetOverrides converts a typed override map into the --set k=v strings
// that CastOptions.SetOverrides expects. Slices are emitted as YAML flow
// sequences ([a,b,c]) so the cast core's YAML-aware --set parser produces a
// real list rather than parsing the Go-default "[a b c]" form as a single
// string. Result is sorted for deterministic ordering.
func encodeSetOverrides(overrides map[string]any) []string {
	out := make([]string, 0, len(overrides))
	for k, v := range overrides {
		out = append(out, fmt.Sprintf("%s=%s", k, formatSetValue(v)))
	}
	sort.Strings(out)
	return out
}

func formatSetValue(v any) string {
	switch x := v.(type) {
	case []string:
		return "[" + strings.Join(x, ",") + "]"
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (m Model) View() string {
	if m.loading {
		return metaStyle.Render("Loading inventory…")
	}
	if m.loadErr != nil {
		return flashErr.Render("Error: " + m.loadErr.Error())
	}
	var b strings.Builder
	if m.flash != "" {
		style := flashOK
		if strings.Contains(m.flash, "error") {
			style = flashErr
		}
		b.WriteString(style.Render(m.flash) + "\n\n")
	}
	b.WriteString(headingStyle.Render("Casted molds (Installed):") + "\n\n")
	if len(m.items) == 0 {
		b.WriteString(metaStyle.Render("(none — Discover tab to install)") + "\n")
		return b.String()
	}
	for i, it := range m.items {
		caret := "  "
		if i == m.cursor {
			caret = cursorStyle.Render("▶ ")
		}
		legacy := ""
		if it.Entry.Files == nil {
			legacy = "  " + warnStyle.Render("⚠ legacy (re-cast to enable safe uninstall)")
		}
		verified := ""
		if it.Verified {
			verified = " " + verifiedStyle.Render("✓")
		}
		fmt.Fprintf(&b, "%s%s%s  %s  %s  %s%s\n",
			caret,
			moldNameStyle.Render(it.Entry.Name),
			verified,
			versionStyle.Render(it.Entry.Version),
			scopeStyle.Render("["+string(it.Scope)+"]"),
			metaStyle.Render(it.Entry.Source),
			legacy)
	}
	b.WriteString("\n" + metaStyle.Render("u update · x uninstall · r refresh · j/k move") + "\n")
	return b.String()
}
