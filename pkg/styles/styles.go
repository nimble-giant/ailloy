package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
)

// Color scheme based on Ailloy branding
var (
	// Primary colors - Purple gradient
	Primary1 = lipgloss.Color("#667eea")
	Primary2 = lipgloss.Color("#764ba2")

	// Accent colors - Orange/Fox theme
	Accent1 = lipgloss.Color("#ff9800")
	Accent2 = lipgloss.Color("#ff6b6b")

	// Status colors
	Success = lipgloss.Color("#4CAF50")
	Info    = lipgloss.Color("#2196F3")
	Warning = lipgloss.Color("#FFC107")
	Error   = lipgloss.Color("#f44336")

	// Neutral colors
	Gray      = lipgloss.Color("#666666")
	LightGray = lipgloss.Color("#999999")
	White     = lipgloss.Color("#ffffff")
)

// Base styles
var (
	// Header style with gradient effect
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary1).
			Bold(true).
			MarginBottom(1)

	// Large header for welcome screens
	LargeHeaderStyle = lipgloss.NewStyle().
				Foreground(Primary1).
				Bold(true).
				MarginBottom(2).
				Padding(1, 2)

	// Success message style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	// Error message style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// Info message style
	InfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	// Warning message style
	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	// Subtle gray text
	SubtleStyle = lipgloss.NewStyle().
			Foreground(Gray)

	// Accent style for highlights
	AccentStyle = lipgloss.NewStyle().
			Foreground(Accent1).
			Bold(true)

	// Code/path style
	CodeStyle = lipgloss.NewStyle().
			Foreground(Primary2).
			Background(lipgloss.Color("#f5f5f5")).
			Padding(0, 1)
)

// Box styles
var (
	// Main content box
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary1).
			Padding(1, 2).
			AlignHorizontal(lipgloss.Left).
			MarginBottom(1)

	// Success box
	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Success).
			Padding(1, 2).
			AlignHorizontal(lipgloss.Left).
			MarginBottom(1)

	// Error box
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Error).
			Padding(1, 2).
			AlignHorizontal(lipgloss.Left).
			MarginBottom(1)

	// Info box
	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Info).
			Padding(1, 2).
			AlignHorizontal(lipgloss.Left).
			MarginBottom(1)
)

// Special gradient style for headers
func GradientHeader(text string) string {
	return lipgloss.NewStyle().
		Background(Primary1).
		Foreground(White).
		Bold(true).
		Padding(0, 2).
		Render(text)
}

// Fox-themed bullet points
func BulletPoint(text string, icon string) string {
	if icon == "" {
		icon = "🦊"
	}
	return AccentStyle.Render(icon+" ") + text
}

// Create a styled table
func NewTable() *table.Table {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(Primary1)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return HeaderStyle.
					Foreground(Primary1).
					Bold(true).
					Align(lipgloss.Center)
			case row%2 == 0:
				return lipgloss.NewStyle().Foreground(Gray)
			default:
				return lipgloss.NewStyle()
			}
		})
	return t
}

// Progress indicator
func ProgressStep(step, total int, description string) string {
	percentage := float64(step) / float64(total) * 100
	bar := ""
	barLength := 20
	filled := int(percentage / 100 * float64(barLength))

	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		AccentStyle.Render(bar),
		InfoStyle.Render(" "),
		InfoStyle.Render(description),
	)
}

// Check if terminal supports color
func SupportsColor() bool {
	return termenv.HasDarkBackground() || termenv.ColorProfile() != termenv.Ascii
}

// Initialize styles based on terminal capabilities
func Init() {
	if !SupportsColor() {
		// Fallback to simpler styles for limited terminals
		HeaderStyle = HeaderStyle.Foreground(lipgloss.NoColor{})
		AccentStyle = AccentStyle.Foreground(lipgloss.NoColor{})
		SuccessStyle = SuccessStyle.Foreground(lipgloss.NoColor{})
		ErrorStyle = ErrorStyle.Foreground(lipgloss.NoColor{})
	}
}
