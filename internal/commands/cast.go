package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/smelt"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var castCmd = &cobra.Command{
	Use:     "cast [mold-dir]",
	Aliases: []string{"install"},
	Short:   "Cast Ailloy configuration into a project",
	Long: `Cast Ailloy configuration into a project (alias: install).

Installs rendered blanks from the given mold into the current repository.
If run from a stuffed binary (created by smelt -o binary), the embedded mold
is used automatically when no mold-dir is provided.
Use -f to layer additional flux value files (Helm-style).
Use -g/--global to install into the user's home directory (~/) instead.`,
	RunE: runCast,
}

var (
	withWorkflows bool
	castGlobal    bool
	castSetFlags  []string
	castValFiles  []string
)

func init() {
	rootCmd.AddCommand(castCmd)

	castCmd.Flags().BoolVarP(&castGlobal, "global", "g", false, "install into user home directory (~/) instead of current project")
	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks (e.g. Claude Code agent)")
	castCmd.Flags().StringArrayVar(&castSetFlags, "set", nil, "override flux variable (format: key=value, can be repeated)")
	castCmd.Flags().StringArrayVarP(&castValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
}

func runCast(_ *cobra.Command, args []string) error {
	reader, err := resolveMoldReader(args)
	if err != nil {
		return err
	}
	return castProject(reader)
}

// resolveMoldReader creates a MoldReader from args or the embedded mold.
func resolveMoldReader(args []string) (*blanks.MoldReader, error) {
	if len(args) >= 1 {
		return blanks.NewMoldReaderFromPath(args[0])
	}
	if smelt.HasEmbeddedMold() {
		fsys, err := smelt.OpenEmbeddedMold()
		if err != nil {
			return nil, fmt.Errorf("opening embedded mold: %w", err)
		}
		return blanks.NewMoldReader(fsys), nil
	}
	return nil, fmt.Errorf("mold directory is required: ailloy cast <mold-dir>")
}

// loadCastFlux loads layered flux values using Helm-style precedence:
// mold flux.yaml < mold.yaml schema defaults < -f files (left to right) < --set flags
func loadCastFlux(reader *blanks.MoldReader) (map[string]any, error) {
	// Layer 1: Load mold flux.yaml as base
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Layer 2: Apply mold.yaml schema defaults (for in-mold compatibility)
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		fluxDefaults = mold.ApplyFluxDefaults(manifest.Flux, fluxDefaults)
	}

	flux := make(map[string]any)
	for k, v := range fluxDefaults {
		flux[k] = v
	}

	// Layer 3: Layer -f files left-to-right (each overrides previous)
	if len(castValFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(castValFiles)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 4: Apply --set overrides (highest precedence)
	if err := mold.ApplySetOverrides(flux, castSetFlags); err != nil {
		return nil, err
	}

	return flux, nil
}

// resolveDestPrefix returns the destination directory prefix.
// When --global is set, files are installed under ~/ instead of the current directory,
// so mold output paths (e.g. .claude/commands) land in the user's home directory.
func resolveDestPrefix() (string, error) {
	if !castGlobal {
		return "", nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return homeDir, nil
}

func castProject(reader *blanks.MoldReader) error {
	// Welcome message
	fmt.Println(styles.WorkingBanner("Casting Ailloy project structure..."))
	fmt.Println()

	// Check runtime dependencies
	checkDependencies()

	destPrefix, err := resolveDestPrefix()
	if err != nil {
		return err
	}

	// Check if we're in a git repository (skip for global installs)
	if destPrefix == "" {
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			warning := styles.WarningStyle.Render("⚠️  Warning: ") +
				"Not in a Git repository. Consider running " +
				styles.CodeStyle.Render("git init") + " first."
			fmt.Println(warning)
			fmt.Println()
		}
	}

	// Load manifest and resolve output files.
	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		return fmt.Errorf("failed to resolve output files: %w", err)
	}

	// Filter out workflow files unless --with-workflows is set.
	var filesToCast []mold.ResolvedFile
	for _, rf := range resolved {
		if !withWorkflows && strings.HasPrefix(rf.DestPath, ".github/") {
			continue
		}
		// Prefix dest paths for global installs.
		if destPrefix != "" {
			rf.DestPath = filepath.Join(destPrefix, rf.DestPath)
		}
		filesToCast = append(filesToCast, rf)
	}

	// Collect unique output directories.
	dirSet := make(map[string]bool)
	for _, rf := range filesToCast {
		dirSet[filepath.Dir(rf.DestPath)] = true
	}
	var dirs []string
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	fmt.Println(styles.InfoStyle.Render("📁 Creating directory structure..."))
	for i, dir := range dirs {
		fmt.Print(styles.ProgressStep(i+1, len(dirs), "Creating "+dir))
		time.Sleep(100 * time.Millisecond) // Small delay for visual effect

		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- Project directories need group read access
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Print("\r" + styles.SuccessStyle.Render("✅ Created directory: ") + styles.CodeStyle.Render(dir) + "\n")
	}
	fmt.Println()

	// Copy resolved files from mold
	if err := copyResolvedFiles(reader, manifest, filesToCast); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	// Success celebration
	fmt.Println()
	successMessage := "Project casting complete!"
	fmt.Println(styles.SuccessBanner(successMessage))
	fmt.Println()

	// Summary box
	summaryContent := styles.SuccessStyle.Render("🎉 Setup Complete!\n\n")
	for _, dir := range dirs {
		summaryContent += styles.FoxBullet("Blanks: ") + styles.CodeStyle.Render(dir+"/") + "\n"
	}
	summaryContent += styles.FoxBullet("Ready for AI-powered development! 🚀")

	// Check if CLAUDE.md exists and suggest creating one if not (skip for global)
	if destPrefix == "" {
		if _, err := os.Stat("CLAUDE.md"); os.IsNotExist(err) {
			summaryContent += "\n\n" +
				styles.InfoStyle.Render("💡 Tip: ") +
				"No " + styles.CodeStyle.Render("CLAUDE.md") + " detected. " +
				"Run " + styles.CodeStyle.Render("/init") + " in Claude Code to create one."
		}
	}

	summary := styles.SuccessBoxStyle.Render(summaryContent)

	fmt.Println(summary)

	return nil
}

// copyResolvedFiles copies resolved mold files to the project, applying template
// processing where indicated by the output mapping.
func copyResolvedFiles(reader *blanks.MoldReader, manifest *mold.Mold, resolved []mold.ResolvedFile) error {
	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	// Validate: prefer flux.schema.yaml, fall back to mold.yaml flux: section
	var schema []mold.FluxVar
	if s, err := reader.LoadFluxSchema(); err == nil && s != nil {
		schema = s
	} else if manifest != nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	if err := mold.ValidateFlux(schema, flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	for _, rf := range resolved {
		content, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rf.SrcPath, err)
		}

		var outputContent []byte
		if rf.Process {
			processed, err := mold.ProcessTemplate(string(content), flux, opts...)
			if err != nil {
				return fmt.Errorf("failed to process %s: %w", rf.SrcPath, err)
			}
			outputContent = []byte(processed)
		} else {
			outputContent = content
		}

		if err := os.MkdirAll(filepath.Dir(rf.DestPath), 0750); err != nil { // #nosec G301
			return fmt.Errorf("failed to create directory for %s: %w", rf.DestPath, err)
		}

		//#nosec G306 -- Blanks need to be readable
		if err := os.WriteFile(rf.DestPath, outputContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", rf.DestPath, err)
		}

		fmt.Println(styles.SuccessStyle.Render("✅ Created: ") + styles.CodeStyle.Render(rf.DestPath))
	}

	return nil
}
