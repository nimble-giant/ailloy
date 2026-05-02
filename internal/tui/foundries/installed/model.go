package installed

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
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
		res, err := foundry.UninstallMold(it.LockPath, it.Entry.Source, foundry.UninstallOptions{})
		return uninstallDoneMsg{source: it.Entry.Source, res: res, err: err}
	}
}

func (m Model) updateCmd(it data.InventoryItem) tea.Cmd {
	cast := m.cast
	return func() tea.Msg {
		if cast == nil {
			return updateDoneMsg{source: it.Entry.Source, err: fmt.Errorf("no cast function configured")}
		}
		err := cast(context.Background(), it.Entry.Source, CastOptions{Global: it.Scope == data.ScopeGlobal})
		return updateDoneMsg{source: it.Entry.Source, err: err}
	}
}

func (m Model) View() string {
	if m.loading {
		return "Loading inventory…"
	}
	if m.loadErr != nil {
		return "Error: " + m.loadErr.Error()
	}
	var b strings.Builder
	if m.flash != "" {
		b.WriteString(m.flash + "\n\n")
	}
	b.WriteString("Casted molds (Installed):\n\n")
	if len(m.items) == 0 {
		b.WriteString("(none — Discover tab to install)\n")
		return b.String()
	}
	for i, it := range m.items {
		caret := "  "
		if i == m.cursor {
			caret = "▶ "
		}
		legacy := ""
		if it.Entry.Files == nil {
			legacy = "  ⚠ legacy (re-cast to enable safe uninstall)"
		}
		verified := ""
		if it.Verified {
			verified = " ✓"
		}
		b.WriteString(fmt.Sprintf("%s%s%s  %s  [%s]  %s%s\n",
			caret, it.Entry.Name, verified, it.Entry.Version, it.Scope, it.Entry.Source, legacy))
	}
	b.WriteString("\nu update · x uninstall · r refresh · j/k move\n")
	return b.String()
}
