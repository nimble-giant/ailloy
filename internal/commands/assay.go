package commands

import (
	"fmt"
	"os"

	"github.com/nimble-giant/ailloy/pkg/assay"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var assayCmd = &cobra.Command{
	Use:     "assay [path]",
	Aliases: []string{"lint"},
	Short:   "Lint AI instruction files against best practices",
	Long: `Lint AI instruction files against best practices (alias: lint).

Validates rendered AI instruction files (CLAUDE.md, AGENTS.md, Cursor rules,
Codex instructions, Copilot instructions, etc.) for structure, cross-references,
and platform-specific schema correctness.

Auto-detects platforms by file presence and applies platform-specific rules.
Reports errors, warnings, and suggestions with three output formats (console,
json, markdown) for terminal and CI use.

Use --init to generate a starter .ailloyrc.yaml configuration file.
Use --fail-on to control the exit code threshold for CI pipelines.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAssay,
}

var (
	assayFix      bool
	assayPlatform string
	assayFormat   string
	assayFailOn   string
	assayInit     bool
	assayMaxLines int
)

func init() {
	rootCmd.AddCommand(assayCmd)
	assayCmd.Flags().BoolVar(&assayFix, "fix", false, "auto-fix fixable issues")
	assayCmd.Flags().StringVar(&assayPlatform, "platform", "", "limit linting to a specific platform (claude, cursor, codex, copilot)")
	assayCmd.Flags().StringVar(&assayFormat, "format", "console", "output format: console, json, markdown")
	assayCmd.Flags().StringVar(&assayFailOn, "fail-on", "error", "exit non-zero on: error, warning, suggestion")
	assayCmd.Flags().BoolVar(&assayInit, "init", false, "generate a starter .ailloyrc.yaml config file")
	assayCmd.Flags().IntVar(&assayMaxLines, "max-lines", 0, "override line-count threshold (default: 150)")
}

func runAssay(_ *cobra.Command, args []string) error {
	// Handle --init
	if assayInit {
		content := assay.GenerateStarterConfig()
		path := ".ailloyrc.yaml"
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; remove it first to regenerate", path)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil { //#nosec G306
			return err
		}
		fmt.Println(styles.SuccessStyle.Render("Created ") + styles.CodeStyle.Render(path))
		return nil
	}

	fmt.Println(styles.WorkingBanner("Assaying..."))
	fmt.Println()

	// Resolve path
	startDir := "."
	if len(args) > 0 {
		startDir = args[0]
	}

	// Find project root
	rootDir, err := assay.FindProjectRoot(startDir)
	if err != nil {
		return fmt.Errorf("finding project root: %w", err)
	}

	// Load config
	cfg, err := assay.LoadConfig(rootDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI overrides
	if assayPlatform != "" {
		cfg.Platforms = []string{assayPlatform}
	}
	if assayMaxLines > 0 {
		if cfg.Rules == nil {
			cfg.Rules = make(map[string]assay.RuleConfig)
		}
		rc := cfg.Rules["line-count"]
		if rc.Options == nil {
			rc.Options = make(map[string]any)
		}
		rc.Options["max-lines"] = assayMaxLines
		cfg.Rules["line-count"] = rc
	}

	// Run assay
	result, err := assay.Assay(rootDir, cfg)
	if err != nil {
		return err
	}

	// Handle no files found
	if result.FilesScanned == 0 {
		fmt.Println(styles.InfoStyle.Render("No AI instruction files found."))
		return nil
	}

	// Format and print output
	formatter := assay.NewFormatter(assayFormat)
	output := formatter.Format(result)
	if output != "" {
		fmt.Print(output)
	}

	// Print summary
	errors := len(result.Errors())
	warnings := len(result.Warnings())
	suggestions := len(result.Suggestions())

	if errors > 0 || warnings > 0 || suggestions > 0 {
		fmt.Println()
	}

	fmt.Printf("%s scanned, ", styles.InfoStyle.Render(fmt.Sprintf("%d file(s)", result.FilesScanned)))

	// Determine exit code based on --fail-on
	var failOnSeverity mold.DiagSeverity
	switch assayFailOn {
	case "suggestion":
		failOnSeverity = mold.SeveritySuggestion
	case "warning":
		failOnSeverity = mold.SeverityWarning
	default:
		failOnSeverity = mold.SeverityError
	}

	if result.HasFailures(failOnSeverity) {
		fmt.Println(styles.ErrorStyle.Render(fmt.Sprintf("%d error(s), %d warning(s), %d suggestion(s)",
			errors, warnings, suggestions)))
		return fmt.Errorf("assay: findings exceed --%s threshold", assayFailOn)
	}

	fmt.Println(styles.SuccessStyle.Render(fmt.Sprintf("%d error(s), %d warning(s), %d suggestion(s)",
		errors, warnings, suggestions)))

	// Handle --fix
	if assayFix && len(result.Diagnostics) > 0 {
		fmt.Println()
		fmt.Println(styles.InfoStyle.Render("Auto-fix is not yet implemented for any rules."))
	}

	return nil
}
