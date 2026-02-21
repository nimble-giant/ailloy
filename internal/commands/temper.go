package commands

import (
	"fmt"
	"os"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var temperCmd = &cobra.Command{
	Use:     "temper [path]",
	Aliases: []string{"lint"},
	Short:   "Validate and lint a mold or ingot package",
	Long: `Validate and lint a mold or ingot package (alias: lint).

Checks structural integrity, manifest fields, file references,
template syntax, and flux schema consistency. Reports errors
(blocking) and warnings (informational).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemper,
}

func init() {
	rootCmd.AddCommand(temperCmd)
}

func runTemper(_ *cobra.Command, args []string) error {
	fmt.Println(styles.WorkingBanner("Tempering..."))
	fmt.Println()

	moldDir := "."
	if len(args) > 0 {
		moldDir = args[0]
	}

	fsys := os.DirFS(moldDir)
	result := mold.Temper(fsys)

	if result.Name != "" {
		fmt.Println(styles.InfoStyle.Render("Package: ") +
			styles.CodeStyle.Render(result.Name) +
			styles.SubtleStyle.Render(fmt.Sprintf(" (%s, %s)", result.ManifestKind, result.Version)))
		fmt.Println()
	}

	warnings := result.Warnings()
	errors := result.Errors()

	for _, d := range warnings {
		loc := ""
		if d.File != "" {
			loc = styles.SubtleStyle.Render(d.File + ": ")
		}
		fmt.Println(styles.WarningStyle.Render("WARNING: ") + loc + d.Message)
	}

	for _, d := range errors {
		loc := ""
		if d.File != "" {
			loc = styles.SubtleStyle.Render(d.File + ": ")
		}
		fmt.Println(styles.ErrorStyle.Render("ERROR: ") + loc + d.Message)
	}

	if len(warnings) > 0 || len(errors) > 0 {
		fmt.Println()
	}

	if result.HasErrors() {
		fmt.Println(styles.ErrorStyle.Render(fmt.Sprintf("Validation failed: %d error(s), %d warning(s)",
			len(errors), len(warnings))))
		return fmt.Errorf("temper: %d error(s) found", len(errors))
	}

	msg := fmt.Sprintf("Validation passed: 0 errors, %d warning(s)", len(warnings))
	fmt.Println(styles.SuccessStyle.Render(msg))
	return nil
}
