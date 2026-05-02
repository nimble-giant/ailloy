package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
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
	withWorkflows        bool
	castGlobal           bool
	castSetFlags         []string
	castValFiles         []string
	castClaudePluginFlag bool
	castPluginName       string
	castPluginVer        string
	// castSilent suppresses interactive output from copyResolvedFiles and
	// related helpers. Set by CastMold (the programmatic core used by the
	// foundries TUI) so per-file "✅ Created" lines don't corrupt the
	// Bubble Tea alt-screen.
	castSilent atomic.Bool
)

func init() {
	rootCmd.AddCommand(castCmd)

	castCmd.Flags().BoolVarP(&castGlobal, "global", "g", false, "install into user home directory (~/) instead of current project")
	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks")
	castCmd.Flags().StringArrayVar(&castSetFlags, "set", nil, "override flux variable (format: key=value, can be repeated)")
	castCmd.Flags().StringArrayVarP(&castValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
	castCmd.Flags().BoolVar(&castClaudePluginFlag, "claude-plugin", false, "package the rendered mold as a Claude Code plugin instead of installing blanks at their cast destinations")
	castCmd.Flags().StringVar(&castPluginName, "plugin-name", "", "override the plugin name (defaults to the mold's name; requires a plugin output flag such as --claude-plugin)")
	castCmd.Flags().StringVar(&castPluginVer, "plugin-version", "", "override the plugin version (defaults to the mold's version; requires a plugin output flag such as --claude-plugin)")
}

func runCast(_ *cobra.Command, args []string) error {
	if err := validatePluginFlags(); err != nil {
		return err
	}
	reader, source, err := resolveMoldReader(args)
	if err != nil {
		return err
	}
	if castClaudePluginFlag {
		return castClaudePlugin(reader)
	}
	return castProject(reader, source)
}

// validatePluginFlags ensures plugin-specific overrides are only used when a
// plugin output flag is set.
func validatePluginFlags() error {
	if !castClaudePluginFlag {
		if castPluginName != "" {
			return fmt.Errorf("--plugin-name requires a plugin output flag (e.g. --claude-plugin)")
		}
		if castPluginVer != "" {
			return fmt.Errorf("--plugin-version requires a plugin output flag (e.g. --claude-plugin)")
		}
	}
	return nil
}

// resolvedRemote holds metadata about the most recently resolved remote mold.
// Used to populate the installed manifest after a successful cast.
var resolvedRemote *foundry.ResolveResult

// resolveMoldReader creates a MoldReader from args or the embedded mold.
// The returned source is the foundry cache key (e.g. "github.com/owner/repo")
// for remote references, or "" for local dirs and embedded molds.
func resolveMoldReader(args []string) (*blanks.MoldReader, string, error) {
	resolvedRemote = nil
	if len(args) >= 1 {
		if foundry.IsRemoteReference(args[0]) {
			var resolveOpts []foundry.ResolveOption
			if castGlobal {
				resolveOpts = append(resolveOpts, foundry.WithLockPath(globalLockPath()))
			}
			fsys, result, err := foundry.ResolveWithMetadata(args[0], resolveOpts...)
			if err != nil {
				return nil, "", fmt.Errorf("resolving remote mold: %w", err)
			}
			resolvedRemote = result
			return blanks.NewMoldReaderFromFS(fsys, result.Root), result.Ref.CacheKey(), nil
		}
		reader, err := blanks.NewMoldReaderFromPath(args[0])
		return reader, "", err
	}
	if smelt.HasEmbeddedMold() {
		fsys, err := smelt.OpenEmbeddedMold()
		if err != nil {
			return nil, "", fmt.Errorf("opening embedded mold: %w", err)
		}
		return blanks.NewMoldReader(fsys), "", nil
	}
	return nil, "", fmt.Errorf("mold directory is required: ailloy cast <mold-dir>")
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

func castProject(reader *blanks.MoldReader, source string) error {
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

	// Load ignore patterns from .ailloyignore and mold.yaml.
	ignorePatterns := mold.LoadIgnorePatterns(reader.FS(), manifest)

	var resolveOpts []mold.ResolveOption
	if len(ignorePatterns) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignorePatterns))
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS(), resolveOpts...)
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

	// Drop directories that ended up empty after skipped renders (#145).
	dirs = cleanupEmptyDirs(dirs, destPrefix)

	// Record the cast in the installed manifest (provenance for `recast` / `quench`).
	if resolvedRemote != nil {
		if err := recordInstalled(resolvedRemote, castGlobal); err != nil {
			log.Printf("warning: failed to update installed manifest: %v", err)
		}
	}

	// Record where blanks were installed (non-fatal if this fails).
	if destPrefix == "" {
		if err := writeInstallState(dirs); err != nil {
			log.Printf("warning: failed to write install state: %v", err)
		}

		// Backfill the lock entry's Files manifest so uninstall knows what to remove.
		if source != "" {
			installed := make([]foundry.InstalledFile, 0, len(filesToCast))
			for _, f := range filesToCast {
				sum, _ := hashFile(f.DestPath)
				installed = append(installed, foundry.InstalledFile{RelPath: f.DestPath, SHA256: sum})
			}
			if err := foundry.RecordInstalledFiles(foundry.LockFileName, source, installed); err != nil {
				log.Printf("warning: recording installed files: %v", err)
			}
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

// cleanupEmptyDirs removes any directories from dirs (and their ancestors up
// to destPrefix or the working-directory root) that ended up empty after the
// render pass. Multi-destination output mappings can leave behind empty
// directories when target guards skip every file at one destination (#145).
// Returns the subset of input dirs that still exist after cleanup.
func cleanupEmptyDirs(dirs []string, destPrefix string) []string {
	stop := destPrefix
	if stop == "" {
		stop = "."
	}

	candidates := make(map[string]bool)
	for _, d := range dirs {
		for cur := d; cur != stop && cur != "." && cur != string(filepath.Separator) && cur != ""; cur = filepath.Dir(cur) {
			candidates[cur] = true
		}
	}

	ordered := make([]string, 0, len(candidates))
	for d := range candidates {
		ordered = append(ordered, d)
	}
	// Deepest paths first so children are removed before their parents.
	sort.Slice(ordered, func(i, j int) bool {
		return len(ordered[i]) > len(ordered[j])
	})

	for _, d := range ordered {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = os.Remove(d)
		}
	}

	remaining := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			remaining = append(remaining, d)
		}
	}
	return remaining
}

// hashFile returns the hex-encoded sha256 of a file's contents.
func hashFile(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304 -- path under user control by design
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
	//#nosec G306,G703 -- claudePath is a hardcoded constant ("CLAUDE.md"), not user input
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
	resolver := buildIngotResolver(flux, reader.Root())
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	for _, rf := range resolved {
		content, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rf.SrcPath, err)
		}

		var outputContent []byte
		if rf.Process {
			fluxForFile := flux
			if len(rf.Set) > 0 {
				fluxForFile = mold.MergeSet(flux, rf.Set)
			}
			processed, err := mold.ProcessTemplate(string(content), fluxForFile, opts...)
			if err != nil {
				return fmt.Errorf("failed to process %s: %w", rf.SrcPath, err)
			}
			outputContent = []byte(processed)
		} else {
			outputContent = content
		}

		// Skip files that render to empty or whitespace-only content (#130)
		if rf.Process && strings.TrimSpace(string(outputContent)) == "" {
			log.Printf("skipping %s: rendered to empty content", rf.SrcPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(rf.DestPath), 0750); err != nil { // #nosec G301
			return fmt.Errorf("failed to create directory for %s: %w", rf.DestPath, err)
		}

		//#nosec G306 -- Blanks need to be readable
		if err := os.WriteFile(rf.DestPath, outputContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", rf.DestPath, err)
		}

		if !castSilent.Load() {
			fmt.Println(styles.SuccessStyle.Render("✅ Created: ") + styles.CodeStyle.Render(rf.DestPath))
		}
	}

	return nil
}

// recordInstalled upserts the just-cast mold into the installed manifest.
func recordInstalled(result *foundry.ResolveResult, global bool) error {
	path := manifestPathFor(global)
	if path == "" {
		return nil // global path unresolvable — silently skip rather than write to cwd
	}
	manifest, err := foundry.ReadInstalledManifest(path)
	if err != nil {
		log.Printf("warning: corrupt installed manifest at %s, resetting: %v", path, err)
		manifest = nil
	}
	if manifest == nil {
		manifest = &foundry.InstalledManifest{APIVersion: "v1"}
	}
	manifest.UpsertEntry(foundry.InstalledEntry{
		Name:    result.Ref.Repo,
		Source:  result.Ref.CacheKey(),
		Subpath: result.Ref.Subpath,
		Version: result.Resolved.Tag,
		Commit:  result.Resolved.Commit,
		CastAt:  time.Now().UTC(),
	})
	return foundry.WriteInstalledManifest(path, manifest)
}
