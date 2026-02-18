package commands

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// ailloyTheme returns a custom huh theme matching Ailloy's branding
func ailloyTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Group header styles
	t.Group.Title = lipgloss.NewStyle().
		Foreground(styles.Accent1).
		Bold(true).
		MarginBottom(1)

	t.Group.Description = lipgloss.NewStyle().
		Foreground(styles.Gray)

	// Focused field styles
	t.Focused.Title = lipgloss.NewStyle().
		Foreground(styles.Primary1).
		Bold(true)

	t.Focused.Description = lipgloss.NewStyle().
		Foreground(styles.Gray)

	t.Focused.SelectSelector = lipgloss.NewStyle().
		Foreground(styles.Accent1).
		SetString("> ")

	t.Focused.SelectedOption = lipgloss.NewStyle().
		Foreground(styles.Accent1)

	t.Focused.SelectedPrefix = lipgloss.NewStyle().
		Foreground(styles.Success).
		SetString("[x] ")

	t.Focused.UnselectedPrefix = lipgloss.NewStyle().
		Foreground(styles.Gray).
		SetString("[ ] ")

	t.Focused.FocusedButton = lipgloss.NewStyle().
		Background(styles.Primary1).
		Foreground(styles.White).
		Bold(true).
		Padding(0, 2)

	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(styles.Gray).
		Padding(0, 2)

	t.Focused.TextInput.Prompt = lipgloss.NewStyle().
		Foreground(styles.Accent1)

	t.Focused.TextInput.Cursor = lipgloss.NewStyle().
		Foreground(styles.Accent1)

	t.Focused.NoteTitle = lipgloss.NewStyle().
		Foreground(styles.Primary1).
		Bold(true).
		MarginBottom(1)

	t.Focused.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary1).
		Padding(1, 2)

	// Blurred field styles
	t.Blurred.Title = lipgloss.NewStyle().
		Foreground(styles.Gray)

	t.Blurred.TextInput.Text = lipgloss.NewStyle().
		Foreground(styles.LightGray)

	t.Blurred.SelectSelector = lipgloss.NewStyle().
		SetString("  ")

	return t
}
