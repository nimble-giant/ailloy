package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var ingotCmd = &cobra.Command{
	Use:   "ingot",
	Short: "Work with Ailloy ingots (reusable components)",
	Long: `Commands for managing Ailloy ingots.

Ingots are reusable template components that can be included in molds
via the {{ingot "name"}} template function.`,
}

var ingotGetCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Download an ingot without installing",
	Long: `Download an ingot to the local cache without installing it.

The reference follows the standard format: <host>/<owner>/<repo>[@<version>][//<subpath>]
After download, the cache path is printed.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngotGet,
}

var (
	ingotAddGlobal bool
)

var ingotAddCmd = &cobra.Command{
	Use:   "add <reference>",
	Short: "Download and register an ingot",
	Long: `Download an ingot and register it in the project's .ailloy/ingots/ directory.

If the reference points to a multi-ingot repo (containing ingots/<name>/ingot.yaml
entries) and no //subpath is given, every ingot in the repo is installed. To install
just one, use the //ingots/<name> subpath suffix.

The ingot files are copied into .ailloy/ingots/<name>/ for use by the template engine.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngotAdd,
}

func init() {
	rootCmd.AddCommand(ingotCmd)
	ingotCmd.AddCommand(ingotGetCmd)
	ingotCmd.AddCommand(ingotAddCmd)

	ingotAddCmd.Flags().BoolVar(&ingotAddGlobal, "global", false, "install under ~/.ailloy/ingots/ instead of ./.ailloy/ingots/")
}

func runIngotGet(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Downloading ingot..."))
	fmt.Println()

	parsed, err := foundry.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing reference: %w", err)
	}

	fsys, err := foundry.Resolve(ref)
	if err != nil {
		return fmt.Errorf("resolving ingot: %w", err)
	}

	// Validate ingot.yaml exists.
	ingot, err := mold.LoadIngotFromFS(fsys, "ingot.yaml")
	if err != nil {
		return fmt.Errorf("invalid ingot manifest: %w", err)
	}

	// Print cache path.
	cacheDir, err := foundry.CacheDir()
	if err != nil {
		return fmt.Errorf("determining cache directory: %w", err)
	}

	lock, _ := foundry.ReadLockFile(foundry.LockFileName)
	version := "latest"
	if entry := lock.FindEntry(parsed.CacheKey(), parsed.Subpath); entry != nil {
		version = entry.Version
	}

	cachePath := foundry.VersionDir(cacheDir, parsed, version)
	if parsed.Subpath != "" {
		cachePath = filepath.Join(cachePath, parsed.Subpath)
	}

	fmt.Println(styles.SuccessStyle.Render("Downloaded: ") + styles.AccentStyle.Render(ingot.Name+" "+ingot.Version))
	if ingot.Description != "" {
		fmt.Println(styles.SubtleStyle.Render("  " + ingot.Description))
	}
	fmt.Println(styles.InfoStyle.Render("Cache path: ") + styles.CodeStyle.Render(cachePath))

	return nil
}

func runIngotAdd(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Adding ingot..."))
	fmt.Println()

	fsys, result, err := foundry.ResolveWithMetadata(ref)
	if err != nil {
		return fmt.Errorf("resolving ingot: %w", err)
	}

	return installIngotsFromFS(fsys, result, ingotAddGlobal)
}

// runIngotAddFromLocal is a test-only seam that drives the multi-package install
// pipeline against a local fs.FS without going through the foundry resolver.
// We synthesize a ResolveResult that recordInstalledArtifact / lock writes will
// accept; the source string uses the local path so identity is stable across
// re-installs from the same source.
func runIngotAddFromLocal(localDir string, global bool) error {
	fsys := os.DirFS(localDir)
	result := &foundry.ResolveResult{
		Ref: &foundry.Reference{
			Host:  "local",
			Owner: "fs",
			Repo:  filepath.Base(localDir),
		},
		Resolved: foundry.ResolvedVersion{
			Tag:    "local",
			Commit: "local",
		},
	}
	return installIngotsFromFS(fsys, result, global)
}

// installIngotsFromFS handles single-ingot and multi-ingot layouts. Per
// issue #200:
//   - Bare ref into multi-layout: install every ingot.
//   - //subpath ref: foundry.Resolve already roots fsys at the subpath, so
//     this looks like single-at-root and installs just that one.
//   - Root manifest in fsys: install the single ingot.
//   - No manifests anywhere: error.
func installIngotsFromFS(fsys fs.FS, result *foundry.ResolveResult, global bool) error {
	pkgs, err := mold.DiscoverIngotPackages(fsys)
	if err != nil {
		return fmt.Errorf("discovering ingots: %w", err)
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no ingot.yaml found at root or under ingots/<name>/")
	}

	for _, p := range pkgs {
		if err := installSingleIngot(fsys, p, result, global); err != nil {
			return err
		}
	}

	fmt.Println()
	if len(pkgs) == 1 {
		fmt.Println(styles.SuccessStyle.Render("Ingot added: ") + styles.AccentStyle.Render(pkgs[0].Name+" "+pkgs[0].Version))
	} else {
		fmt.Println(styles.SuccessStyle.Render(fmt.Sprintf("Installed %d ingots from %s", len(pkgs), result.Ref.CacheKey())))
	}
	return nil
}

func installSingleIngot(fsys fs.FS, pkg mold.IngotPackage, result *foundry.ResolveResult, global bool) error {
	// Effective subpath: the resolver may already have rooted fsys at a //subpath
	// (in which case result.Ref.Subpath is set and pkg.Subpath is "" because pkg
	// was discovered at the root of that already-narrowed fsys). When we fan out
	// over a multi-package fsys, pkg.Subpath carries "ingots/<name>" and joins
	// with result.Ref.Subpath which is "". We pick whichever is non-empty.
	effectiveSubpath := result.Ref.Subpath
	if effectiveSubpath == "" {
		effectiveSubpath = pkg.Subpath
	}

	baseRoot := filepath.Join(".ailloy", "ingots")
	if global {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return fmt.Errorf("determining home directory: %w", herr)
		}
		baseRoot = filepath.Join(home, ".ailloy", "ingots")
	}
	destDir := filepath.Join(baseRoot, pkg.Name)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("creating ingot directory: %w", err)
	}

	pkgFS := fsys
	if pkg.Root != "." {
		sub, serr := fs.Sub(fsys, pkg.Root)
		if serr != nil {
			return fmt.Errorf("scoping fs to %s: %w", pkg.Root, serr)
		}
		pkgFS = sub
	}

	if err := fs.WalkDir(pkgFS, ".", func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		destPath := filepath.Join(destDir, p)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o750)
		}
		content, rerr := fs.ReadFile(pkgFS, p)
		if rerr != nil {
			return fmt.Errorf("reading %s: %w", p, rerr)
		}
		if err := os.WriteFile(destPath, content, 0o644); err != nil { //#nosec G306
			return fmt.Errorf("writing %s: %w", destPath, err)
		}
		fmt.Println(styles.SuccessStyle.Render("  + ") + styles.CodeStyle.Render(destPath))
		return nil
	}); err != nil {
		return fmt.Errorf("copying ingot files: %w", err)
	}

	// Synthesize a per-package ResolveResult so the recordInstalledArtifact +
	// lock writes carry the correct (Source, Subpath) identity. Ref is a
	// pointer, so we deep-copy it before mutating to avoid clobbering shared
	// state across pkgs in the same fan-out.
	perPkgResult := *result
	refCopy := *result.Ref
	refCopy.Subpath = effectiveSubpath
	refCopy.Repo = pkg.Name
	perPkgResult.Ref = &refCopy

	if err := recordInstalledArtifact("ingot", &perPkgResult, "", global); err != nil {
		log.Printf("warning: failed to update installed manifest: %v", err)
	}

	if !global {
		if lock, _ := foundry.ReadLockFile(foundry.LockFileName); lock != nil {
			lock.UpsertArtifactLock("ingot", foundry.LockEntry{
				Name:      pkg.Name,
				Source:    result.Ref.CacheKey(),
				Subpath:   effectiveSubpath,
				Version:   result.Resolved.Tag,
				Commit:    result.Resolved.Commit,
				Timestamp: time.Now().UTC(),
			})
			if werr := foundry.WriteLockFile(foundry.LockFileName, lock); werr != nil {
				log.Printf("warning: failed to update lock file: %v", werr)
			}
		}
	}
	return nil
}
