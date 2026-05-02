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
	OnProgress    func(stage, item string)

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

	reader, source, err := openMoldReaderForCore(ref, opts.Global)
	if err != nil {
		return res, err
	}
	res.Source = source

	manifest, err := reader.LoadManifest()
	if err != nil {
		return res, fmt.Errorf("loading mold manifest: %w", err)
	}
	if manifest != nil {
		res.MoldName = manifest.Name
	}

	flux, err := layerFluxForCore(reader, opts.ValueFiles, opts.SetOverrides)
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

	if err := copyResolvedFiles(reader, manifest, flux, filesToCast); err != nil {
		return res, fmt.Errorf("copying files: %w", err)
	}

	if destPrefix == "" && source != "" {
		installed := make([]foundry.InstalledFile, 0, len(filesToCast))
		for _, f := range filesToCast {
			sum, _ := hashFile(f.DestPath)
			installed = append(installed, foundry.InstalledFile{RelPath: f.DestPath, SHA256: sum})
		}
		res.FilesCast = installed
		_ = foundry.RecordInstalledFiles(foundry.LockFileName, source, installed)
	}

	return res, nil
}

// openMoldReaderForCore is the CastMold-flavored counterpart to
// resolveMoldReader; it accepts a single ref string instead of args.
func openMoldReaderForCore(ref string, global bool) (*blanks.MoldReader, string, error) {
	if ref == "" {
		return nil, "", fmt.Errorf("ref required")
	}
	if foundry.IsRemoteReference(ref) {
		var resolveOpts []foundry.ResolveOption
		if global {
			resolveOpts = append(resolveOpts, foundry.WithoutLock())
		}
		fsys, root, err := foundry.ResolveWithRoot(ref, resolveOpts...)
		if err != nil {
			return nil, "", fmt.Errorf("resolving remote mold: %w", err)
		}
		source := ""
		if parsed, perr := foundry.ParseReference(ref); perr == nil {
			source = parsed.CacheKey()
		}
		return blanks.NewMoldReaderFromFS(fsys, root), source, nil
	}
	reader, err := blanks.NewMoldReaderFromPath(ref)
	return reader, "", err
}

// layerFluxForCore mirrors loadCastFlux but is parameterized so CastMold
// doesn't depend on package-level cast flag vars.
func layerFluxForCore(reader *blanks.MoldReader, valueFiles, setOverrides []string) (map[string]any, error) {
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
