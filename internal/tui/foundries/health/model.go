package health

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// Model is the Health tab. Drift checks come from data.DriftFindings;
// assay findings come from data.AssayFindings over .ailloy/state.yaml's
// blank dirs.
type Model struct {
	cfg      *index.Config
	findings []data.Finding
	loading  bool
}

type findingsMsg struct{ findings []data.Finding }

func New(cfg *index.Config) Model { return Model{cfg: cfg, loading: true} }

func (m Model) Init() tea.Cmd { return runChecks(m.cfg) }

func runChecks(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		catalog, _ := data.LoadCatalog(cfg)
		items, _ := data.LoadInventory(cfg)
		out := data.DriftFindings(cfg, items, catalog)
		out = append(out, data.AssayFindings(data.ReadBlankDirs(".ailloy/state.yaml"))...)
		return findingsMsg{findings: out}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case findingsMsg:
		m.loading = false
		m.findings = msg.findings
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.loading = true
			return m, runChecks(m.cfg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return "Running health checks…"
	}
	if len(m.findings) == 0 {
		return "All clear ✓\n\nr refresh"
	}
	var b strings.Builder
	for _, f := range m.findings {
		sev := "ℹ"
		switch f.Severity {
		case data.SevError:
			sev = "✗"
		case data.SevWarn:
			sev = "⚠"
		}
		b.WriteString(fmt.Sprintf("%s [%s] %s — %s\n", sev, f.Source, f.Title, f.Detail))
	}
	b.WriteString("\nr refresh\n")
	return b.String()
}
