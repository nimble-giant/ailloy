package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// CastOptions configures a CastMold call. All fields are optional.
type CastOptions struct {
	Global        bool     // install under $HOME instead of cwd
	WithWorkflows bool     // include .github/ workflow blanks
	ValueFiles    []string // -f layered flux value files
	SetOverrides  []string // --set key=val overrides
	// ForceReplaceOnParseError, when true, allows merge-strategy
	// destinations whose existing on-disk file is unparseable to be
	// replaced rather than erroring. Mirrors the
	// --force-replace-on-parse-error CLI flag.
	ForceReplaceOnParseError bool
	OnProgress               func(stage, item string)

	// ClaudePlugin packages the rendered mold as a Claude Code plugin under
	// .claude/plugins/<slug>/ (or ~/.claude/plugins/<slug>/ when Global is set)
	// instead of installing blanks at their cast destinations. When set,
	// PluginName/PluginVersion override the values derived from mold.yaml.
	ClaudePlugin  bool
	PluginName    string
	PluginVersion string
}

// CastResult summarizes a CastMold call for programmatic consumers.
type CastResult struct {
	Source     string                  // resolved source identifier (foundry cache key)
	MoldName   string                  // from mold.yaml
	FilesCast  []foundry.InstalledFile // files installed (with sha256 hashes)
	Dirs       []string                // unique parent directories created
	GlobalRoot string                  // populated when Global=true
}

// CastMold performs the cast install pipeline as a callable function with
// no terminal output. Returns structured results suitable for the TUI to
// render. The cobra `runCast` keeps its richer CLI presentation.
//
// ctx is currently unused but reserved for future cancellation.
func CastMold(_ context.Context, ref string, opts CastOptions) (CastResult, error) {
	var res CastResult

	// Silence per-file "✅ Created" lines from copyResolvedFiles so the
	// Bubble Tea alt-screen doesn't get clobbered. Also redirect stderr
	// so log.Printf warnings (e.g. install-state, lock-file write) don't
	// bleed through.
	castSilent.Store(true)
	defer castSilent.Store(false)
	prevLog := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prevLog)

	reader, remoteResult, err := openMoldReaderForCore(ref, opts.Global)
	if err != nil {
		return res, err
	}
	source := ""
	if remoteResult != nil {
		source = remoteResult.Ref.OverrideKey()
	}
	res.Source = source

	manifest, err := reader.LoadManifest()
	if err != nil {
		return res, fmt.Errorf("loading mold manifest: %w", err)
	}
	if manifest != nil {
		res.MoldName = manifest.Name
	}

	flux, err := layerFluxForCore(reader, source, opts.ValueFiles, opts.SetOverrides)
	if err != nil {
		return res, err
	}

	if opts.ClaudePlugin {
		pluginRes, perr := packageMoldAsClaudePlugin(reader, flux, pluginPackageOpts{
			Global:          opts.Global,
			WithWorkflows:   opts.WithWorkflows,
			NameOverride:    opts.PluginName,
			VersionOverride: opts.PluginVersion,
		})
		if perr != nil {
			return res, perr
		}
		res.Dirs = []string{pluginRes.TargetDir}
		if opts.Global {
			home, _ := os.UserHomeDir()
			res.GlobalRoot = home
		}
		return res, nil
	}

	destPrefix := ""
	if opts.Global {
		home, err := os.UserHomeDir()
		if err != nil {
			return res, fmt.Errorf("cannot determine home directory: %w", err)
		}
		destPrefix = home
		res.GlobalRoot = home
	}

	ignore := mold.LoadIgnorePatterns(reader.FS(), manifest)
	var resolveOpts []mold.ResolveOption
	if len(ignore) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignore))
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS(), resolveOpts...)
	if err != nil {
		return res, fmt.Errorf("resolving output files: %w", err)
	}

	var filesToCast []mold.ResolvedFile
	for _, rf := range resolved {
		if !opts.WithWorkflows && strings.HasPrefix(rf.DestPath, ".github/") {
			continue
		}
		if destPrefix != "" {
			rf.DestPath = filepath.Join(destPrefix, rf.DestPath)
		}
		filesToCast = append(filesToCast, rf)
	}

	dirSet := map[string]struct{}{}
	for _, rf := range filesToCast {
		dirSet[filepath.Dir(rf.DestPath)] = struct{}{}
	}
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	res.Dirs = dirs

	for i, dir := range dirs {
		if opts.OnProgress != nil {
			opts.OnProgress(fmt.Sprintf("mkdir %d/%d", i+1, len(dirs)), dir)
		}
		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- project directories need group read access
			return res, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := copyResolvedFiles(reader, manifest, flux, filesToCast, opts.ForceReplaceOnParseError); err != nil {
		return res, fmt.Errorf("copying files: %w", err)
	}

	// Drop directories that ended up empty after skipped renders (#145, #195).
	dirs = cleanupEmptyDirs(dirs, destPrefix)
	res.Dirs = dirs

	// Mirror what cast.go does: record install dirs in .ailloy/state.yaml so
	// `mold list` can find blanks installed via the foundries TUI.
	if destPrefix == "" {
		if err := writeInstallState(dirs); err != nil {
			log.Printf("warning: failed to write install state: %v", err)
		}
	}

	if remoteResult != nil {
		manifestPath := manifestPathFor(opts.Global)
		if manifestPath != "" {
			// Upsert the manifest entry with provenance, then backfill Files
			// + FileHashes. Mirrors what cast.go does via recordInstalled +
			// RecordInstalledFiles.
			if err := recordInstalled(remoteResult, opts.Global); err != nil {
				log.Printf("warning: failed to update installed manifest: %v", err)
			} else {
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
				res.FilesCast = installed
				_ = foundry.RecordInstalledFiles(manifestPath, source, remoteResult.Ref.Subpath, installed)
			}
		}
	}

	return res, nil
}

// openMoldReaderForCore is the CastMold-flavored counterpart to
// resolveMoldReader; it accepts a single ref string instead of args.
// The returned *foundry.ResolveResult is populated for remote refs (so the
// caller can record provenance in the installed manifest) and nil for local
// refs.
func openMoldReaderForCore(ref string, global bool) (*blanks.MoldReader, *foundry.ResolveResult, error) {
	if ref == "" {
		return nil, nil, fmt.Errorf("ref required")
	}
	if foundry.IsRemoteReference(ref) {
		var resolveOpts []foundry.ResolveOption
		if global {
			resolveOpts = append(resolveOpts, foundry.WithLockPath(globalLockPath()))
		}
		fsys, result, err := foundry.ResolveWithMetadata(ref, resolveOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving remote mold: %w", err)
		}
		return blanks.NewMoldReaderFromFS(fsys, result.Root), result, nil
	}
	reader, err := blanks.NewMoldReaderFromPath(ref)
	return reader, nil, err
}

// layerFluxForCore mirrors loadCastFlux but is parameterized so CastMold
// doesn't depend on package-level cast flag vars. `source` is the resolved
// mold ref used to pick up persisted flux files (~/.ailloy/flux/<slug>.yaml
// then ./.ailloy/flux/<slug>.yaml). Empty source skips persisted-file lookup.
func layerFluxForCore(reader *blanks.MoldReader, source string, valueFiles, setOverrides []string) (map[string]any, error) {
	defaults, err := reader.LoadFluxDefaults()
	if err != nil {
		defaults = make(map[string]any)
	}
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		defaults = mold.ApplyFluxDefaults(manifest.Flux, defaults)
	}
	flux := make(map[string]any, len(defaults))
	for k, v := range defaults {
		flux[k] = v
	}
	if persisted := mold.PersistedFluxPaths(source); len(persisted) > 0 {
		overlay, err := mold.LayerFluxFiles(persisted)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}
	if len(valueFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(valueFiles)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}
	if err := mold.ApplySetOverrides(flux, setOverrides); err != nil {
		return nil, err
	}
	return flux, nil
}
