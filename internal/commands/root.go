package commands

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "ailloy",
	Short:   "AI-assisted development methodology and toolchain",
	Long:    buildLongDescription(),
	Version: "0.1.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		styles.Init()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildLongDescription() string {
	banner := styles.WelcomeBanner()

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
