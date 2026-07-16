package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/huh"
	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/internal/tui/ceremony"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/merge"
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
	withWorkflows                bool
	castGlobal                   bool
	castSetFlags                 []string
	castValFiles                 []string
	castClaudePluginFlag         bool
	castPluginName               string
	castPluginVer                string
	castForceReplaceOnParseError bool
	// castFrozen, when true, blocks auto-install of declared ingot/ore deps
	// at cast time. Missing deps surface as errors instead of being fetched
	// silently. Intended for CI; see --frozen flag.
	castFrozen bool
	// castLatestOnNoTags, when true, automatically casts from the default
	// branch HEAD when the foundry has no semver tags. Skips the interactive
	// prompt; intended for CI contexts.
	castLatestOnNoTags bool
	// castOffline, when true, disables all network operations. Tag resolution
	// and bare-clone fetches are served from the local cache; fails with an
	// actionable error if the cache is cold. Intended for air-gapped builds.
	castOffline bool
)

// copyOpts configures copyResolvedFiles. Centralising these as a struct lets
// callers like CastMold (used by the foundries TUI) request a fully silent
// run — no per-file stdout writes, all warnings to a discarding logger — so
// concurrent casts can't race on global silencing flags.
type copyOpts struct {
	ForceReplaceOnParseError bool
	// Silent suppresses the per-file "✅ Created" stdout lines. The Bubble
	// Tea alt-screen is corrupted by any stray Println, so the TUI path sets
	// this true.
	Silent bool
	// Logger receives non-fatal warnings (template validation, skipped empty
	// renders). Nil falls back to log.Default(); the TUI path passes a
	// discarding logger so concurrent casts can't race on log.SetOutput.
	Logger *log.Logger
}

// logger returns opts.Logger or log.Default() when unset.
func (o copyOpts) logger() *log.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return log.Default()
}

func init() {
	rootCmd.AddCommand(castCmd)

	castCmd.Flags().BoolVarP(&castGlobal, "global", "g", false, "install into user home directory (~/) instead of current project")
	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks")
	castCmd.Flags().StringArrayVar(&castSetFlags, "set", nil, "override flux variable (format: key=value, can be repeated)")
	castCmd.Flags().StringArrayVarP(&castValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
	castCmd.Flags().BoolVar(&castClaudePluginFlag, "claude-plugin", false, "package the rendered mold as a Claude Code plugin instead of installing blanks at their cast destinations")
	castCmd.Flags().StringVar(&castPluginName, "plugin-name", "", "override the plugin name (defaults to the mold's name; requires a plugin output flag such as --claude-plugin)")
	castCmd.Flags().StringVar(&castPluginVer, "plugin-version", "", "override the plugin version (defaults to the mold's version; requires a plugin output flag such as --claude-plugin)")
	castCmd.Flags().BoolVar(&castForceReplaceOnParseError,
		"force-replace-on-parse-error",
		false,
		"if a destination uses strategy: merge but is unparseable, replace it instead of erroring")
	castCmd.Flags().BoolVar(&castFrozen,
		"frozen",
		false,
		"fail (do not auto-install) when a declared ingot/ore dep is missing from .ailloy/; intended for CI")
	castCmd.Flags().BoolVar(&castLatestOnNoTags,
		"latest-on-no-tags",
		false,
		"cast from default branch HEAD when the foundry has no semver tags (skips interactive prompt; intended for CI)")
	castCmd.Flags().BoolVar(&castOffline,
		"offline",
		false,
		"resolve all dependencies from the local cache only; fails if the cache is cold (run without --offline first to warm it)")
}

func runCast(_ *cobra.Command, args []string) error {
	if err := validatePluginFlags(); err != nil {
		return err
	}
	// A smelted binary carries its mold embedded; network resolution of
	// transitive deps is unnecessary and breaks air-gapped environments.
	// Auto-enable offline mode so the binary works without --offline.
	if len(args) == 0 && smelt.HasEmbeddedMold() {
		castOffline = true
	}
	reader, source, err := resolveMoldReader(args)
	if err != nil {
		return err
	}
	if err := checkAilloyRequirement(reader); err != nil {
		return err
	}
	if castClaudePluginFlag {
		return castClaudePlugin(reader, source)
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

// checkAilloyRequirement enforces a mold's `requires.ailloy` constraint before
// any casting work begins. Without this gate an old binary silently ignores
// the constraint and proceeds with a degraded cast (e.g. skipping transitive
// dependency installation), leaving the user with a partially installed mold
// and no explanation (#229).
//
// Manifest load failures are swallowed here: the cast pipeline reloads the
// manifest downstream and reports those errors with better context.
func checkAilloyRequirement(reader *blanks.MoldReader) error {
	manifest, err := reader.LoadManifest()
	if err != nil || manifest == nil {
		return nil
	}
	return enforceAilloyVersion(manifest.Requires.Ailloy)
}

// enforceAilloyVersion checks the running ailloy build against a mold's
// requires.ailloy constraint, returning an actionable error when it is not
// satisfied. It passes (returns nil) when there is no constraint, when the
// constraint is unparseable (`temper` flags malformed constraints — a cast
// should not double as a linter), or when the running binary has no release
// version to compare (dev builds, where evolveCurrentVersion is empty/"dev").
func enforceAilloyVersion(requires string) error {
	requires = strings.TrimSpace(requires)
	if requires == "" {
		return nil
	}
	current := strings.TrimSpace(evolveCurrentVersion)
	if current == "" || current == "dev" {
		return nil
	}
	constraint, err := semver.NewConstraint(requires)
	if err != nil {
		return nil
	}
	v, err := semver.NewVersion(strings.TrimPrefix(current, "v"))
	if err != nil {
		return nil
	}
	if constraint.Check(v) {
		return nil
	}
	return fmt.Errorf(
		"this mold requires ailloy %s, but you are running v%s\nRun `brew upgrade ailloy` to update",
		requires, strings.TrimPrefix(current, "v"))
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
			if castOffline {
				resolveOpts = append(resolveOpts, foundry.WithOffline())
			}
			fsys, result, err := foundry.ResolveWithMetadata(args[0], resolveOpts...)
			if err != nil {
				if errors.Is(err, foundry.ErrNoSemverTags) {
					return resolveMoldReaderWithDefaultBranch(args[0])
				}
				return nil, "", fmt.Errorf("resolving remote mold: %w", err)
			}
			resolvedRemote = result
			return blanks.NewMoldReaderFromFS(fsys, result.Root), result.Ref.OverrideKey(), nil
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

// resolveMoldReaderWithDefaultBranch handles the fallback path when a foundry
// has no semver tags. It prompts the user interactively (or auto-accepts when
// --latest-on-no-tags is set) then resolves the default branch HEAD commit and
// fetches the mold from it.
func resolveMoldReaderWithDefaultBranch(rawRef string) (*blanks.MoldReader, string, error) {
	ref, err := foundry.ParseReference(rawRef)
	if err != nil {
		return nil, "", fmt.Errorf("parsing reference: %w", err)
	}

	if !castLatestOnNoTags {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("No semver tags found for %s.", ref.CacheKey())).
					Description("Cast from latest commit on default branch instead?").
					Affirmative("Yes").
					Negative("No").
					Value(&confirm),
			),
		).WithTheme(ailloyTheme())
		if err := form.Run(); err != nil {
			return nil, "", fmt.Errorf("prompt failed: %w", err)
		}
		if !confirm {
			return nil, "", fmt.Errorf(
				"cast aborted: %s has no semver tags\n"+
					"Add a version tag to the foundry, or use --latest-on-no-tags to cast from HEAD",
				ref.CacheKey())
		}
	}

	git := foundry.DefaultGitRunner()
	resolved, err := foundry.ResolveDefaultBranchHead(ref, git)
	if err != nil {
		return nil, "", fmt.Errorf("resolving default branch HEAD: %w", err)
	}

	fetcher, err := foundry.NewFetcher(git)
	if err != nil {
		return nil, "", fmt.Errorf("creating fetcher: %w", err)
	}
	fsys, root, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		return nil, "", fmt.Errorf("fetching mold: %w", err)
	}

	result := &foundry.ResolveResult{Ref: ref, Resolved: *resolved, Root: root}
	resolvedRemote = result
	return blanks.NewMoldReaderFromFS(fsys, root), ref.OverrideKey(), nil
}

// loadCastFlux loads layered flux values using Helm-style precedence:
// mold flux.yaml < mold.yaml schema defaults < persisted ~/.ailloy/flux/<slug>.yaml
// < persisted ./.ailloy/flux/<slug>.yaml < -f files (left to right) < --set flags.
//
// Schema and defaults are loaded via LoadMoldFluxWithOres so installed ore
// overlays (mold-local → project → global) participate in the merge before
// any persisted/-f/--set layers run.
//
// `source` is the mold ref used to derive the persisted-file slug (typically the
// foundry cache key for remote refs). Empty source skips persisted-file lookup.
//
// Returns the resolved flux map plus the merged schema (used downstream by
// copyResolvedFiles for ValidateFlux).
func loadCastFlux(reader *blanks.MoldReader, source string) (map[string]any, []mold.FluxVar, error) {
	// Layers 1+2: ore-aware merge of mold.yaml flux schema, mold flux.yaml,
	// and any installed ore overlays (mold-local → project → global).
	mergedSchema, fluxDefaults, _, err := mold.LoadMoldFluxWithOres(reader.FS(), readerSearchPaths(reader, castGlobal))
	if err != nil {
		// Fall back to the legacy single-mold path so an ore-loading hiccup
		// doesn't break ore-less casts.
		fluxDefaults, err = reader.LoadFluxDefaults()
		if err != nil {
			fluxDefaults = make(map[string]any)
		}
	}
	// Merge mold.yaml's in-line flux: schema in. LoadMoldFluxWithOres only
	// reads the standalone flux.schema.yaml file; molds that declare their
	// schema inline (no flux.schema.yaml on disk) still need their defaults.
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		fluxDefaults = mold.ApplyFluxDefaults(manifest.Flux, fluxDefaults)
		if len(mergedSchema) == 0 {
			mergedSchema = manifest.Flux
		}
	}

	flux := make(map[string]any)
	for k, v := range fluxDefaults {
		flux[k] = v
	}
	mold.ApplyManifestOutputDefault(flux, manifest)

	// Layer 3: persisted flux files written by the foundries TUI (global, then
	// project — project wins on conflict). Layered before user-supplied -f so
	// explicit -f still overrides saved values.
	persisted := mold.PersistedFluxPaths(source)
	if len(persisted) > 0 {
		overlay, err := mold.LayerFluxFiles(persisted)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 4: Layer -f files left-to-right (each overrides previous)
	if len(castValFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(castValFiles)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 5: Apply --set overrides (highest precedence)
	if err := mold.ApplySetOverrides(flux, castSetFlags); err != nil {
		return nil, nil, err
	}

	return flux, mergedSchema, nil
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
	// Welcome message — themed entrance flourish wraps the canonical banner.
	ceremony.Open(ceremony.Cast)

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

	// Auto-install declared ingot/ore deps before flux merge so the next
	// LoadMoldFluxWithOres call sees the just-installed overlays.
	moldKey := ""
	if resolvedRemote != nil {
		moldKey = resolvedRemote.Ref.CacheKey()
		if resolvedRemote.Ref.Subpath != "" {
			moldKey += "@" + resolvedRemote.Ref.Subpath
		}
	}
	// Local-path deps are only safe when the parent mold is itself local —
	// otherwise a malicious foundry could declare e.g. `- ore: /etc` and
	// have it copied into the project tree.
	allowLocalDeps := resolvedRemote == nil
	if err := installDeclaredDeps(manifest, moldKey, castGlobal, allowLocalDeps, castFrozen, false, nil); err != nil {
		return fmt.Errorf("installing declared dependencies: %w", err)
	}

	// Load flux values and merged schema (mold + ore overlays).
	flux, mergedSchema, err := loadCastFlux(reader, source)
	if err != nil {
		flux = make(map[string]any)
		mergedSchema = manifest.Flux
	}

	// Load ignore patterns from .ailloyignore and mold.yaml.
	ignorePatterns := mold.LoadIgnorePatterns(reader.FS(), manifest)

	var resolveOpts []mold.ResolveOption
	if len(ignorePatterns) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignorePatterns))
	}

	// Resolve ore deps ephemerally to get OreSource records (fs handles +
	// extracted output overlays). The on-disk schema/defaults path
	// (loadCastFlux → LoadMoldFluxWithOres) doesn't expose FS handles, so
	// we layer this read-only resolution on top. installDeclaredDeps above
	// already pulled the deps; ResolveDepsEphemeral hits the same foundry
	// cache and produces matching content.
	depResolver, derr := ResolveDepsEphemeral(manifest, allowLocalDeps)
	if derr != nil {
		return fmt.Errorf("resolving ore output overlays: %w", derr)
	}
	resolved, err := mold.ResolveFilesWithOreSources(flux["output"], reader.FS(), depResolver.OreSources(), resolveOpts...)
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

	// Copy resolved files from mold (using the ore-merged schema for validation).
	if err := copyResolvedFilesWithSchema(reader, manifest, mergedSchema, flux, filesToCast, copyOpts{
		ForceReplaceOnParseError: castForceReplaceOnParseError,
	}); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	// Drop directories that ended up empty after skipped renders (#145).
	dirs = cleanupEmptyDirs(dirs, destPrefix)

	// Record where blanks were installed (non-fatal if this fails).
	if destPrefix == "" {
		if err := writeInstallState(dirs); err != nil {
			log.Printf("warning: failed to write install state: %v", err)
		}
	}

	// Record the cast in the installed manifest and backfill the Files list
	// so uninstall knows what to remove. The manifest is always written
	// (regardless of --global), so this works for both project and global
	// installs.
	if resolvedRemote != nil {
		installed := make([]foundry.InstalledFile, 0, len(filesToCast))
		for _, f := range filesToCast {
			sum, _ := hashFile(f.DestPath)
			rel := f.DestPath
			if destPrefix != "" {
				if r, rerr := filepath.Rel(destPrefix, f.DestPath); rerr == nil {
					rel = r
				}
			}
			installed = append(installed, foundry.InstalledFile{RelPath: rel, SHA256: sum})
		}
		castOpts := &foundry.CastOptionsRecord{
			WithWorkflows: withWorkflows,
			ValueFiles:    castValFiles,
			SetOverrides:  castSetFlags,
		}
		if err := recordCastedFiles(resolvedRemote, installed, castGlobal, castOpts, nil); err != nil {
			log.Printf("warning: failed to record installed files: %v", err)
		}
	}

	// Cast transitive mold deps (mold-on-mold dependencies). No-op when the
	// root has no mold-kind deps. Runs after the root is recorded so cycles
	// or conflicts surface alongside the root cast result.
	// resolvedRemote is nil for local-dir and embedded casts; castTransitiveDeps
	// handles that by synthesizing a local sentinel reference.
	if err := castTransitiveDeps(resolvedRemote, manifest, flux, destPrefix); err != nil {
		return fmt.Errorf("casting transitive dependencies: %w", err)
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
	ceremony.Stamp(ceremony.Cast, fmt.Sprintf("%d blank dir(s) installed", len(dirs)))

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

const installStatePath = ".ailloy/state.yaml"

// writeInstallState records where blanks were installed so `mold list` can find them.
//
// Reads the existing state.yaml first and unions the new dirs into it, so
// repeated casts (e.g. installing several molds from a foundry) accumulate
// rather than overwriting each other.
func writeInstallState(dirs []string) error {
	state := installState{}
	if existing, err := readInstallState(installStatePath); err == nil && existing != nil {
		state = *existing
	}

	blankSet := make(map[string]struct{}, len(state.BlankDirs)+len(dirs))
	workflowSet := make(map[string]struct{}, len(state.WorkflowDirs)+len(dirs))
	for _, d := range state.BlankDirs {
		blankSet[d] = struct{}{}
	}
	for _, d := range state.WorkflowDirs {
		workflowSet[d] = struct{}{}
	}
	for _, d := range dirs {
		if strings.HasPrefix(d, ".github/") {
			workflowSet[d] = struct{}{}
		} else {
			blankSet[d] = struct{}{}
		}
	}

	state.BlankDirs = sortedKeys(blankSet)
	state.WorkflowDirs = sortedKeys(workflowSet)

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(".ailloy", 0750); err != nil { // #nosec G301
		return err
	}
	return os.WriteFile(installStatePath, data, 0644) // #nosec G306
}

func readInstallState(path string) (*installState, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a known constant
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s installState
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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

// chooseFS returns rf.SrcFS when non-nil, falling back to the mold's primary
// fs. Resolved files originating from an ore (or a consumer `from: ore/...`
// selector) carry a non-nil SrcFS; mold-origin files carry nil and fall back
// to the reader.
func chooseFS(rf mold.ResolvedFile, primary fs.FS) fs.FS {
	if rf.SrcFS != nil {
		return rf.SrcFS
	}
	return primary
}

// copyResolvedFiles copies resolved mold files to the project, applying template
// processing where indicated by the output mapping. Schema for validation is
// inferred from the reader / mold manifest. Cast-time callers should prefer
// copyResolvedFilesWithSchema so ore-merged schema entries participate in
// ValidateFlux.
func copyResolvedFiles(reader *blanks.MoldReader, manifest *mold.Mold, flux map[string]any, resolved []mold.ResolvedFile, opts copyOpts) error {
	var schema []mold.FluxVar
	if s, err := reader.LoadFluxSchema(); err == nil && s != nil {
		schema = s
	} else if manifest != nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	return copyResolvedFilesWithSchema(reader, manifest, schema, flux, resolved, opts)
}

// copyResolvedFilesWithSchema is copyResolvedFiles with an explicit schema
// parameter. Callers that have already merged ore overlays (cast/recast)
// pass the merged schema so ValidateFlux sees the full ore.<name>.* surface.
func copyResolvedFilesWithSchema(reader *blanks.MoldReader, manifest *mold.Mold, schema []mold.FluxVar, flux map[string]any, resolved []mold.ResolvedFile, opts copyOpts) error {
	logger := opts.logger()

	// Validate: ore-merged schema preferred; fall back to flux.schema.yaml /
	// mold.yaml's flux: block when caller didn't supply one.
	if len(schema) == 0 {
		if s, err := reader.LoadFluxSchema(); err == nil && s != nil {
			schema = s
		} else if manifest != nil && len(manifest.Flux) > 0 {
			schema = manifest.Flux
		}
	}
	if err := mold.ValidateFlux(schema, flux); err != nil {
		logger.Printf("warning: %v", err)
	}

	// Build ingot resolver. Also search the mold's own FS (the embedded
	// filesystem for stuffed-binary casts, where the mold's ingots live off-disk).
	resolver := buildIngotResolver(flux, reader.Root())
	resolver.FS = reader.FS()
	tplOpts := []mold.TemplateOption{
		mold.WithIngotResolver(resolver),
		mold.WithLogger(logger),
	}

	for _, rf := range resolved {
		content, err := fs.ReadFile(chooseFS(rf, reader.FS()), rf.SrcPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rf.SrcPath, err)
		}

		var outputContent []byte
		if rf.Process {
			fluxForFile := flux
			if len(rf.Set) > 0 {
				fluxForFile = mold.MergeSet(flux, rf.Set)
			}
			processed, err := mold.ProcessTemplate(string(content), fluxForFile, tplOpts...)
			if err != nil {
				return fmt.Errorf("failed to process %s: %w", rf.SrcPath, err)
			}
			outputContent = []byte(processed)
		} else {
			outputContent = content
		}

		// Skip files that render to empty or whitespace-only content (#130)
		if rf.Process && strings.TrimSpace(string(outputContent)) == "" {
			logger.Printf("skipping %s: rendered to empty content", rf.SrcPath)
			continue
		}

		switch rf.Strategy {
		case "merge":
			err := merge.MergeFile(rf.DestPath, outputContent, merge.Options{
				ForceReplaceOnParseError: opts.ForceReplaceOnParseError,
			})
			if err != nil {
				var pe *merge.ParseError
				if errors.As(err, &pe) {
					return fmt.Errorf(
						"failed to merge into %s: existing %s file could not be parsed: %w. "+
							"Re-run with --force-replace-on-parse-error to overwrite",
						pe.Path, pe.Format, pe.Err)
				}
				return fmt.Errorf("failed to merge %s: %w", rf.DestPath, err)
			}
		case "append":
			if manifest == nil {
				return fmt.Errorf("append strategy requires a mold manifest with a name (dest %s)", rf.DestPath)
			}
			err := merge.AppendFile(rf.DestPath, outputContent, merge.AppendOptions{
				MoldName: manifest.Name,
			})
			if err != nil {
				return fmt.Errorf("failed to append into %s: %w", rf.DestPath, err)
			}
		case "", "replace":
			if err := os.MkdirAll(filepath.Dir(rf.DestPath), 0750); err != nil { // #nosec G301
				return fmt.Errorf("failed to create directory for %s: %w", rf.DestPath, err)
			}
			//#nosec G306 -- Blanks need to be readable
			if err := os.WriteFile(rf.DestPath, outputContent, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", rf.DestPath, err)
			}
		default:
			return fmt.Errorf("unknown strategy %q on output for %s", rf.Strategy, rf.DestPath)
		}

		if !opts.Silent {
			fmt.Println(styles.SuccessStyle.Render("✅ Created: ") + styles.CodeStyle.Render(rf.DestPath))
		}
	}

	return nil
}

// recordInstalled upserts the just-cast mold into the installed manifest,
// preserving the option-shaped flags that drove this cast so a future
// `recast` can replay them. logger receives the "corrupt manifest, resetting"
// warning; pass a discarding logger from TUI callers to keep the alt-screen
// clean.
//
// installedAs is "direct" for top-level casts (the user typed `ailloy cast
// <ref>`) and "transitive" for molds pulled in by another mold's
// dependencies. installedBy is the list of parent source[@subpath] strings
// for transitives — empty/ignored for direct casts.
func recordInstalled(result *foundry.ResolveResult, global bool, opts *foundry.CastOptionsRecord, installedAs string, installedBy []string, logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}
	path := manifestPathFor(global)
	if path == "" {
		return nil // global path unresolvable — silently skip rather than write to cwd
	}
	manifest, err := foundry.ReadInstalledManifest(path)
	if err != nil {
		logger.Printf("warning: corrupt installed manifest at %s, resetting: %v", path, err)
		manifest = nil
	}
	if manifest == nil {
		manifest = &foundry.InstalledManifest{APIVersion: "v1"}
	}
	// Preserve any pre-existing InstalledBy entries when a transitive is also
	// reached via additional parents, so multi-parent reverse edges accumulate
	// like ArtifactEntry.Dependents already does.
	mergedBy := append([]string(nil), installedBy...)
	if existing := manifest.FindBySource(result.Ref.CacheKey(), result.Ref.Subpath); existing != nil {
		for _, p := range existing.InstalledBy {
			if !containsString(mergedBy, p) {
				mergedBy = append(mergedBy, p)
			}
		}
		// A mold cast directly stays direct even if it was previously seen as
		// transitive; the user explicitly asked for it now.
		if installedAs == "" {
			installedAs = existing.InstalledAs
		}
	}
	entry := foundry.InstalledEntry{
		Name:        result.Ref.Repo,
		Source:      result.Ref.CacheKey(),
		Subpath:     result.Ref.Subpath,
		Version:     result.Resolved.Tag,
		Commit:      result.Resolved.Commit,
		CastAt:      time.Now().UTC(),
		InstalledAs: installedAs,
		InstalledBy: mergedBy,
	}
	if opts != nil && (opts.WithWorkflows || len(opts.ValueFiles) > 0 || len(opts.SetOverrides) > 0) {
		// Copy to detach from caller's slice ownership.
		copied := *opts
		copied.ValueFiles = append([]string(nil), opts.ValueFiles...)
		copied.SetOverrides = append([]string(nil), opts.SetOverrides...)
		entry.CastOptions = &copied
	}
	manifest.UpsertEntry(entry)
	return foundry.WriteInstalledManifest(path, manifest)
}

// recordCastedFiles upserts the just-cast mold into the installed manifest
// and backfills its Files list in one place so the lookup key cannot drift
// from the write key. The lookup must use Ref.CacheKey() (host/owner/repo) —
// not OverrideKey, which inlines the subpath and would not match the entry
// recordInstalled just wrote for monorepo foundries. opts persists the cast
// arguments for `recast`; logger is forwarded so TUI callers can keep the
// alt-screen clean.
func recordCastedFiles(result *foundry.ResolveResult, files []foundry.InstalledFile, global bool, opts *foundry.CastOptionsRecord, logger *log.Logger) error {
	return recordCastedFilesWithProvenance(result, files, global, opts, "direct", nil, logger)
}

// recordCastedFilesWithProvenance is recordCastedFiles with explicit
// InstalledAs / InstalledBy plumbing for transitive mold deps. The default
// recordCastedFiles call sets installedAs="direct" and no installedBy.
func recordCastedFilesWithProvenance(result *foundry.ResolveResult, files []foundry.InstalledFile, global bool, opts *foundry.CastOptionsRecord, installedAs string, installedBy []string, logger *log.Logger) error {
	if err := recordInstalled(result, global, opts, installedAs, installedBy, logger); err != nil {
		return err
	}
	path := manifestPathFor(global)
	if path == "" {
		return nil
	}
	return foundry.RecordInstalledFiles(path, result.Ref.CacheKey(), result.Ref.Subpath, files)
}

// recordInstalledArtifact upserts an installed ingot or ore into the manifest.
// kind must be "ingot" or "ore". alias is "" for ingots and the --as value
// for ores (or "" if no alias). When invoked from `ailloy ore add` /
// `ailloy ingot add`, dependents includes the "user" sentinel so the artifact
// survives mold uninstalls.
func recordInstalledArtifact(kind string, result *foundry.ResolveResult, alias string, global bool) error {
	path := manifestPathFor(global)
	if path == "" {
		return nil
	}
	manifest, err := foundry.ReadInstalledManifest(path)
	if err != nil {
		log.Printf("warning: corrupt installed manifest at %s, resetting: %v", path, err)
		manifest = nil
	}
	if manifest == nil {
		manifest = &foundry.InstalledManifest{APIVersion: "v1"}
	}
	manifest.UpsertArtifact(kind, foundry.ArtifactEntry{
		Name:        result.Ref.Repo,
		Source:      result.Ref.CacheKey(),
		Subpath:     result.Ref.Subpath,
		Version:     result.Resolved.Tag,
		Commit:      result.Resolved.Commit,
		InstalledAt: time.Now().UTC(),
		Alias:       alias,
		Dependents:  []string{"user"},
	})
	return foundry.WriteInstalledManifest(path, manifest)
}
