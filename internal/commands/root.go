package commands

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ailloy",
	Short: "AI-assisted development methodology and toolchain",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		styles.Init()
	},
}

// SetVersionInfo sets the version information injected via ldflags at build time.
func SetVersionInfo(version, commit, date string) {
	if commit != "unknown" && date != "unknown" {
		rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	} else {
		rootCmd.Version = version
	}
	rootCmd.Long = buildLongDescription(rootCmd.Version)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildLongDescription(version string) string {
	banner := styles.WelcomeBanner(version)

	description := styles.BoxStyle.Render(
		"Ailloy is a modern AI-assisted development methodology and toolchain\n" +
			"that fuses human creativity with AI precision.\n\n" +
			"Like in metallurgy—where combining two elements yields a stronger alloy—\n" +
			"Ailloy represents the fusion of traditional development practices with\n" +
			"AI assistance to create more efficient engineering workflows.",
	)

	quickStart := styles.InfoBoxStyle.Render(
		"Quick Start:\n\n" +
			"🦊 ailloy init            # Set up project\n" +
			"🦊 ailloy template list   # View templates\n" +
			"🦊 ailloy customize       # Configure settings",
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
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
