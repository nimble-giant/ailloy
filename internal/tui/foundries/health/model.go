package health

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

var (
	sevErrorStyle = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
	sevWarnStyle  = lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
	sevInfoStyle  = lipgloss.NewStyle().Foreground(styles.Info)
	sourceStyle   = lipgloss.NewStyle().Foreground(styles.Primary1)
	titleStyle    = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	detailStyle   = lipgloss.NewStyle().Foreground(styles.LightGray)
	metaStyle     = lipgloss.NewStyle().Foreground(styles.Gray)
	clearStyle    = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
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
		return metaStyle.Render("Running health checks…")
	}
	if len(m.findings) == 0 {
		return clearStyle.Render("All clear ✓") + "\n\n" + metaStyle.Render("r refresh")
	}
	var b strings.Builder
	for _, f := range m.findings {
		var sev string
		switch f.Severity {
		case data.SevError:
			sev = sevErrorStyle.Render("✗")
		case data.SevWarn:
			sev = sevWarnStyle.Render("⚠")
		default:
			sev = sevInfoStyle.Render("ℹ")
		}
		fmt.Fprintf(&b, "%s %s %s %s\n",
			sev,
			sourceStyle.Render("["+f.Source+"]"),
			titleStyle.Render(f.Title),
			detailStyle.Render("— "+f.Detail))
	}
	b.WriteString("\n" + metaStyle.Render("r refresh") + "\n")
	return b.String()
}
