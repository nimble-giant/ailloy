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

	"github.com/charmbracelet/huh"
	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
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
	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks")
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
		if foundry.IsRemoteReference(args[0]) {
			fsys, err := foundry.Resolve(args[0])
			if err != nil {
				return nil, fmt.Errorf("resolving remote mold: %w", err)
			}
			return blanks.NewMoldReader(fsys), nil
		}
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
// so mold output paths land in the user's home directory.
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

	// Load manifest.
	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	// Load flux values and extract output mapping.
	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
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
	if err := copyResolvedFiles(reader, manifest, flux, filesToCast); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	// Record where blanks were installed (non-fatal if this fails).
	if destPrefix == "" {
		if err := writeInstallState(dirs); err != nil {
			log.Printf("warning: failed to write install state: %v", err)
		}
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

	// Check for AGENTS.md and CLAUDE.md integration (skip for global)
	if destPrefix == "" {
		agentsInstalled := hasDestFile(filesToCast, "AGENTS.md")
		_, claudeExists := os.Stat("CLAUDE.md")

		switch {
		case agentsInstalled && claudeExists == nil:
			// AGENTS.md was installed and CLAUDE.md exists — offer to add import
			if !claudeMDHasAgentsImport("CLAUDE.md") {
				summaryContent += "\n\n" +
					styles.InfoStyle.Render("💡 ") +
					styles.CodeStyle.Render("AGENTS.md") + " installed."
			}
		case agentsInstalled:
			// AGENTS.md was installed but no CLAUDE.md
			summaryContent += "\n\n" +
				styles.InfoStyle.Render("💡 Tip: ") +
				styles.CodeStyle.Render("AGENTS.md") + " installed. " +
				"Add " + styles.CodeStyle.Render("@AGENTS.md") + " to your " +
				styles.CodeStyle.Render("CLAUDE.md") + " to load it in Claude Code."
		default:
			// No AGENTS.md in this mold — check for any AI instruction file
			instructionFiles := []string{"CLAUDE.md", "AGENTS.md", ".cursorrules", ".windsurfrules"}
			hasInstructions := false
			for _, f := range instructionFiles {
				if _, err := os.Stat(f); err == nil {
					hasInstructions = true
					break
				}
			}
			if !hasInstructions {
				summaryContent += "\n\n" +
					styles.InfoStyle.Render("💡 Tip: ") +
					"No AI instruction file detected. " +
					"Consider adding one for your AI coding tool (e.g., " +
					styles.CodeStyle.Render("CLAUDE.md") + ", " +
					styles.CodeStyle.Render("AGENTS.md") + ", " +
					styles.CodeStyle.Render(".cursorrules") + ")."
			}
		}
	}

	summary := styles.SuccessBoxStyle.Render(summaryContent)

	fmt.Println(summary)

	// Prompt to add @AGENTS.md import to CLAUDE.md (after summary box)
	if destPrefix == "" {
		agentsInstalled := hasDestFile(filesToCast, "AGENTS.md")
		if agentsInstalled {
			if _, err := os.Stat("CLAUDE.md"); err == nil {
				if !claudeMDHasAgentsImport("CLAUDE.md") {
					offerAgentsImport("CLAUDE.md")
				}
			}
		}
	}

	return nil
}

// installState represents the .ailloy/state.yaml file that records where blanks were installed.
type installState struct {
	BlankDirs    []string `yaml:"blankDirs,omitempty"`
	WorkflowDirs []string `yaml:"workflowDirs,omitempty"`
}

// writeInstallState records where blanks were installed so `mold list` can find them.
func writeInstallState(dirs []string) error {
	state := installState{}
	for _, d := range dirs {
		if strings.HasPrefix(d, ".github/") {
			state.WorkflowDirs = append(state.WorkflowDirs, d)
		} else {
			state.BlankDirs = append(state.BlankDirs, d)
		}
	}
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(".ailloy", 0750); err != nil { // #nosec G301
		return err
	}
	return os.WriteFile(".ailloy/state.yaml", data, 0644) // #nosec G306
}

// hasDestFile checks if any resolved file targets the given destination path.
func hasDestFile(files []mold.ResolvedFile, destPath string) bool {
	for _, rf := range files {
		if rf.DestPath == destPath {
			return true
		}
	}
	return false
}

// claudeMDHasAgentsImport checks if a CLAUDE.md file already contains an @AGENTS.md import.
func claudeMDHasAgentsImport(path string) bool {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a known constant
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "@agents.md")
}

// offerAgentsImport prompts the user to add @AGENTS.md import to their CLAUDE.md.
func offerAgentsImport(claudePath string) {
	fmt.Println()

	var confirm bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add @AGENTS.md import to your CLAUDE.md?").
				Description("This makes Claude Code load your AGENTS.md instructions.").
				Affirmative("Yes").
				Negative("No").
				Value(&confirm),
		),
	).WithTheme(ailloyTheme())

	if err := form.Run(); err != nil {
		return
	}

	if !confirm {
		return
	}

	data, err := os.ReadFile(claudePath) // #nosec G304 -- path is a known constant
	if err != nil {
		fmt.Println(styles.WarningStyle.Render("⚠️  Could not read " + claudePath + ": " + err.Error()))
		return
	}

	newContent := "@AGENTS.md\n\n" + string(data)
	//#nosec G306 -- CLAUDE.md needs to be readable
	if err := os.WriteFile(claudePath, []byte(newContent), 0644); err != nil {
		fmt.Println(styles.WarningStyle.Render("⚠️  Could not update " + claudePath + ": " + err.Error()))
		return
	}

	fmt.Println(styles.SuccessStyle.Render("✅ Added ") + styles.CodeStyle.Render("@AGENTS.md") +
		styles.SuccessStyle.Render(" import to ") + styles.CodeStyle.Render(claudePath))
}

// copyResolvedFiles copies resolved mold files to the project, applying template
// processing where indicated by the output mapping.
func copyResolvedFiles(reader *blanks.MoldReader, manifest *mold.Mold, flux map[string]any, resolved []mold.ResolvedFile) error {
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
