package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/assay"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var temperCmd = &cobra.Command{
	Use:     "temper [path]",
	Aliases: []string{"validate"},
	Short:   "Validate a mold or ingot package",
	Long: `Validate a mold or ingot package (alias: validate).

Checks structural integrity, manifest fields, file references,
template syntax, and flux schema consistency. Reports errors
(blocking) and warnings (informational).

Use --assay (or its alias --lint) to also render blanks and run assay on
the output. This catches content-level issues (line count, structure,
cross-references) before casting, without needing a separate cast + assay step.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemper,
}

var (
	temperLint      bool
	temperSetValues []string
	temperValFiles  []string
	temperFormat    string
	temperFailOn    string
	temperMaxLines  int
)

func init() {
	rootCmd.AddCommand(temperCmd)

	temperCmd.Flags().BoolVar(&temperLint, "assay", false, "render blanks and run assay (lint) on the output")
	temperCmd.Flags().BoolVar(&temperLint, "lint", false, "alias for --assay")
	_ = temperCmd.Flags().MarkHidden("lint")
	temperCmd.Flags().StringArrayVar(&temperSetValues, "set", nil, "set flux values for rendering (key=value)")
	temperCmd.Flags().StringArrayVarP(&temperValFiles, "values", "f", nil, "flux value files for rendering (can be repeated)")
	temperCmd.Flags().StringVar(&temperFormat, "format", "console", "assay output format: console, json, markdown")
	temperCmd.Flags().StringVar(&temperFailOn, "fail-on", "error", "assay exit threshold: error, warning, suggestion")
	temperCmd.Flags().IntVar(&temperMaxLines, "max-lines", 0, "override assay line-count threshold (default: 150)")
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

	// Run assay (lint) on rendered blanks if requested
	if temperLint {
		if result.ManifestKind != "mold" {
			fmt.Println()
			fmt.Println(styles.InfoStyle.Render("--assay is only supported for molds, skipping."))
			return nil
		}

		fmt.Println()
		if err := runTemperLint(moldDir); err != nil {
			return err
		}
	}

	return nil
}

// runTemperLint renders the mold blanks into a temp directory and runs assay on them.
func runTemperLint(moldDir string) error {
	fmt.Println(styles.WorkingBanner("Linting rendered blanks..."))
	fmt.Println()

	reader, err := blanks.NewMoldReaderFromPath(moldDir)
	if err != nil {
		return fmt.Errorf("reading mold: %w", err)
	}

	// Load flux with layering (same precedence as forge/cast)
	flux, err := loadTemperFlux(reader)
	if err != nil {
		return fmt.Errorf("loading flux: %w", err)
	}

	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Validate flux against schema
	schema, _ := reader.LoadFluxSchema()
	if schema == nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	if err := mold.ValidateFlux(schema, flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver and render files
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		return fmt.Errorf("resolving output files: %w", err)
	}

	// Render to temp directory
	tmpDir, err := os.MkdirTemp("", "ailloy-temper-lint-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //#nosec G104
	}()

	if err := writeRenderedFiles(resolved, reader.FS(), flux, opts, tmpDir); err != nil {
		return err
	}

	// Run assay on the rendered output
	return runAssayOnDir(tmpDir)
}

// writeRenderedFiles renders resolved files and writes them to outputDir.
func writeRenderedFiles(resolved []mold.ResolvedFile, moldFS fs.FS, flux map[string]any, opts []mold.TemplateOption, outputDir string) error {
	for _, rf := range resolved {
		content, err := fs.ReadFile(moldFS, rf.SrcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rf.SrcPath, err)
		}

		var rendered string
		if rf.Process {
			rendered, err = renderFile(rf.SrcPath, content, flux, opts...)
			if err != nil {
				return err
			}
		} else {
			rendered = string(content)
		}

		// Skip files that render to empty or whitespace-only content (#130)
		if rf.Process && strings.TrimSpace(rendered) == "" {
			log.Printf("skipping %s: rendered to empty content", rf.SrcPath)
			continue
		}

		dest := filepath.Join(outputDir, rf.DestPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil { // #nosec G301 -- Output directories need group read access
			return fmt.Errorf("creating directory for %s: %w", rf.DestPath, err)
		}
		//#nosec G306 -- Rendered blanks need to be readable
		if err := os.WriteFile(dest, []byte(rendered), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", rf.DestPath, err)
		}
	}
	return nil
}

// runAssayOnDir runs assay against the given directory and formats the output.
func runAssayOnDir(dir string) error {
	cfg := assay.DefaultConfig()

	if temperMaxLines > 0 {
		if cfg.Rules == nil {
			cfg.Rules = make(map[string]assay.RuleConfig)
		}
		rc := cfg.Rules["line-count"]
		if rc.Options == nil {
			rc.Options = make(map[string]any)
		}
		rc.Options["max-lines"] = temperMaxLines
		cfg.Rules["line-count"] = rc
	}

	result, err := assay.Assay(dir, cfg)
	if err != nil {
		return fmt.Errorf("assay: %w", err)
	}

	if result.FilesScanned == 0 {
		fmt.Println(styles.InfoStyle.Render("No AI instruction files found in rendered output."))
		return nil
	}

	// Format and print output
	formatter := assay.NewFormatter(temperFormat, dir)
	output := formatter.Format(result)
	if output != "" {
		fmt.Print(output)
	}

	// Print summary
	errs := len(result.Errors())
	warns := len(result.Warnings())
	suggestions := len(result.Suggestions())

	if errs > 0 || warns > 0 || suggestions > 0 {
		fmt.Println()
	}

	fmt.Printf("%s scanned, ", styles.InfoStyle.Render(fmt.Sprintf("%d file(s)", result.FilesScanned)))

	var failOnSeverity mold.DiagSeverity
	switch temperFailOn {
	case "suggestion":
		failOnSeverity = mold.SeveritySuggestion
	case "warning":
		failOnSeverity = mold.SeverityWarning
	default:
		failOnSeverity = mold.SeverityError
	}

	if result.HasFailures(failOnSeverity) {
		fmt.Println(styles.ErrorStyle.Render(fmt.Sprintf("%d error(s), %d warning(s), %d suggestion(s)",
			errs, warns, suggestions)))
		return fmt.Errorf("temper --assay: assay findings exceed --%s threshold", temperFailOn)
	}

	fmt.Println(styles.SuccessStyle.Render(fmt.Sprintf("%d error(s), %d warning(s), %d suggestion(s)",
		errs, warns, suggestions)))

	return nil
}

// loadTemperFlux loads layered flux values using the same Helm-style precedence as forge/cast.
func loadTemperFlux(reader *blanks.MoldReader) (map[string]any, error) {
	// Layer 1: Load mold flux.yaml as base
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Layer 2: Apply mold.yaml schema defaults
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		fluxDefaults = mold.ApplyFluxDefaults(manifest.Flux, fluxDefaults)
	}

	flux := make(map[string]any)
	for k, v := range fluxDefaults {
		flux[k] = v
	}

	// Layer 3: Layer -f files left-to-right
	if len(temperValFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(temperValFiles)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 4: Apply --set overrides (highest precedence)
	if err := mold.ApplySetOverrides(flux, temperSetValues); err != nil {
		return nil, err
	}

	return flux, nil
}
