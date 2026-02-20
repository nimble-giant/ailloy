package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/safepath"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var temperCmd = &cobra.Command{
	Use:     "temper [path]",
	Aliases: []string{"lint"},
	Short:   "Validate a mold or ingot package",
	Long: `Validate a mold or ingot package for structural and template errors (alias: lint).

Checks manifest fields, file references, template syntax, and flux schema
consistency. Reports errors (blocking) and warnings (informational).

Exit code 0 if valid, non-zero if errors are found.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemper,
}

func init() {
	rootCmd.AddCommand(temperCmd)
}

func runTemper(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	cleanDir, err := safepath.Clean(dir)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Resolve to absolute path so we can split into parent (FS root) and base name.
	// fs.FS paths cannot start with "." or "/", so we use the parent directory as
	// the FS root and the directory name as the base path prefix.
	absDir, err := filepath.Abs(cleanDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	parentDir := filepath.Dir(absDir)
	baseName := filepath.Base(absDir)

	moldPath := filepath.Join(absDir, "mold.yaml")
	ingotPath := filepath.Join(absDir, "ingot.yaml")

	var result *mold.ValidationResult

	switch {
	case fileExists(moldPath):
		m, err := mold.LoadMold(moldPath)
		if err != nil {
			return fmt.Errorf("loading mold manifest: %w", err)
		}
		fmt.Println(styles.HeaderStyle.Render("Tempering mold: " + m.Name))
		result = mold.TemperMold(m, os.DirFS(parentDir), baseName)

	case fileExists(ingotPath):
		i, err := mold.LoadIngot(ingotPath)
		if err != nil {
			return fmt.Errorf("loading ingot manifest: %w", err)
		}
		fmt.Println(styles.HeaderStyle.Render("Tempering ingot: " + i.Name))
		result = mold.TemperIngot(i, os.DirFS(parentDir), baseName)

	default:
		return fmt.Errorf("no mold.yaml or ingot.yaml found in %s", cleanDir)
	}

	printTemperResults(result)

	if result.HasErrors() {
		return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
	}
	return nil
}

func printTemperResults(result *mold.ValidationResult) {
	for _, e := range result.Errors {
		loc := ""
		if e.File != "" {
			loc = e.File + ": "
		}
		fmt.Println(styles.ErrorStyle.Render("ERROR ") + loc + e.Message)
	}

	for _, w := range result.Warnings {
		loc := ""
		if w.File != "" {
			loc = w.File + ": "
		}
		fmt.Println(styles.WarningStyle.Render("WARNING ") + loc + w.Message)
	}

	if !result.HasErrors() && len(result.Warnings) == 0 {
		fmt.Println(styles.SuccessStyle.Render("OK") + " — no issues found")
	} else if !result.HasErrors() {
		fmt.Println(styles.SuccessStyle.Render("OK") + fmt.Sprintf(" — %d warning(s), no errors", len(result.Warnings)))
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
