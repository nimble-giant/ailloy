package foundries

import "github.com/charmbracelet/lipgloss"

var (
	tabActive   = lipgloss.NewStyle().Bold(true).Underline(true).Padding(0, 1)
	tabInactive = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	statusBar   = lipgloss.NewStyle().Faint(true).MarginTop(1)
	bodyBox     = lipgloss.NewStyle().Padding(1, 2)
)
