package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

// rawVersion holds the unformatted version string (e.g. "v0.6.7") for semver comparison.
var rawVersion string

var rootCmd = &cobra.Command{
	Use:   "ailloy",
	Short: "The package manager for AI instructions",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		styles.Init()
	},
}

// RawVersion returns the unformatted version string set at build time.
func RawVersion() string {
	return rawVersion
}

// SetVersionInfo sets the version information injected via ldflags at build time.
func SetVersionInfo(version, commit, date string) {
	rawVersion = version
	if commit != "unknown" && date != "unknown" {
		rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	} else {
		rootCmd.Version = version
	}
	rootCmd.Long = buildLongDescription(rootCmd.Version)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styles.ErrorStyle.Render("Error: ")+err.Error())
		os.Exit(1)
	}
}

func buildLongDescription(version string) string {
	banner := styles.WelcomeBanner(version)

	description := styles.BoxStyle.Render(
		"Ailloy is the package manager for AI instructions.\n" +
			"Find, create, and share reusable AI workflow packages —\n" +
			"the same way Helm manages Kubernetes applications.\n\n" +
			"Like in metallurgy—where combining two elements yields a stronger alloy—\n" +
			"Ailloy fuses human creativity with AI precision.",
	)

	quickStart := styles.InfoBoxStyle.Render(
		"Quick Start:\n\n" +
			"🦊 ailloy cast            # Cast project (alias: install)\n" +
			"🦊 ailloy mold list       # View molds\n" +
			"🦊 ailloy anneal          # Anneal settings (alias: configure)",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		description,
		"",
		quickStart,
	)
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Register custom template function to render commands as a styled table
	cobra.AddTemplateFunc("commandTable", func(cmd *cobra.Command) string {
		// Check if any subcommand has aliases
		hasAliases := false
		for _, c := range cmd.Commands() {
			if (c.IsAvailableCommand() || c.Name() == "help") && len(c.Aliases) > 0 {
				hasAliases = true
				break
			}
		}

		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(styles.Primary1)).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return lipgloss.NewStyle().Bold(true).Foreground(styles.Primary1)
				}
				return lipgloss.NewStyle()
			})

		if hasAliases {
			t.Headers("Command", "Aliases", "Description")
		} else {
			t.Headers("Command", "Description")
		}

		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() && c.Name() != "help" {
				continue
			}
			if hasAliases {
				t.Row(c.Name(), strings.Join(c.Aliases, ", "), c.Short)
			} else {
				t.Row(c.Name(), c.Short)
			}
		}

		return t.Render()
	})

	rootCmd.SetUsageTemplate(usageTemplateWithAliases)
}

// usageTemplateWithAliases is cobra's default usage template modified to render
// available commands as a styled table with an Aliases column.
const usageTemplateWithAliases = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{commandTable .}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
