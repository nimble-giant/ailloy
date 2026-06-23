package commands

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/smelt"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// hasMoldDeps reports whether the given mold declares any kind=="mold"
// dependencies (i.e. transitive mold-on-mold edges). Used to short-circuit
// the depgraph build for the common case of leaf-only molds.
func hasMoldDeps(m *mold.Mold) bool {
	if m == nil {
		return false
	}
	for _, d := range m.Dependencies {
		if k, _ := d.Kind(); k == "mold" {
			return true
		}
	}
	return false
}

// depFetcher is the subset of depgraph behaviors castTransitiveDeps needs:
// graph construction + per-node fs.FS retrieval. The production wiring uses
// depgraph.ProdFetcher; tests inject a fake.
type depFetcher interface {
	depgraph.Fetcher
	CacheEntry(depgraph.NodeKey) *depgraph.ProdFetchCacheEntry
}

// castTransitiveDeps resolves and casts the transitive mold dependencies of
// `root`. It is a no-op if root has no mold-kind deps.
//
// Each transitive node renders into the same project (or global) destination
// as the root, with Helm-style flux propagation: dep defaults <- the parent's
// `with:` block <- the root cast's `--set <depAlias>.*` and `-f` overrides.
//
// `--with-workflows` cascades to transitives; transitives' .github/ files are
// emitted only when the user passed --with-workflows for the root.
//
// Failures during dep resolution or per-dep casting bubble up unchanged so
// the cast as a whole is reported as failed (rather than silently leaving a
// half-installed graph).
func castTransitiveDeps(rootResult *foundry.ResolveResult, root *mold.Mold, rootFlux map[string]any, destPrefix string) error {
	if root == nil || !hasMoldDeps(root) {
		return nil
	}
	// For local-dir and embedded casts resolvedRemote is nil, but the mold may
	// still declare mold-kind dependencies. Synthesize a local sentinel reference
	// so the dep graph has a stable root key; the transitive nodes are all remote
	// and will be fetched normally.
	if rootResult == nil {
		name := root.Name
		if name == "" {
			name = "local"
		}
		rootResult = &foundry.ResolveResult{
			Ref: &foundry.Reference{
				Host:  "local",
				Owner: "dir",
				Repo:  name,
			},
		}
	}

	prodFetcher := depgraph.NewProdFetcher()
	if castGlobal {
		prodFetcher.LockPath = globalLockPath()
	}
	prodFetcher.Offline = castOffline

	// When running as a smelted binary with embedded deps, prefer the
	// embedded dep store over the network so offline casts work end-to-end.
	var fetcher depFetcher = prodFetcher
	if smelt.HasEmbeddedMold() {
		if embFetcher, err := smelt.NewEmbeddedDepFetcher(prodFetcher); err == nil {
			fetcher = embFetcher
		}
	}
	return castTransitiveDepsWith(fetcher, rootResult, root, rootFlux, destPrefix)
}

// castTransitiveDepsWith is the testable core: it accepts an injected fetcher
// so unit tests can drive the cast pipeline without touching git.
func castTransitiveDepsWith(fetcher depFetcher, rootResult *foundry.ResolveResult, root *mold.Mold, rootFlux map[string]any, destPrefix string) error {
	if rootResult == nil || root == nil || !hasMoldDeps(root) {
		return nil
	}
	graph, err := depgraph.New(fetcher).Build(root, rootResult.Ref)
	if err != nil {
		return fmt.Errorf("resolving dependency graph: %w", err)
	}

	rootKey := depgraph.NodeKey{Source: rootResult.Ref.CacheKey(), Subpath: rootResult.Ref.Subpath}
	parentLabel := rootKey.String()

	for _, node := range graph.Nodes {
		if node.Key == rootKey {
			continue
		}

		fmt.Println(styles.InfoStyle.Render("📦 Casting dependency: ") + styles.CodeStyle.Render(node.Key.String()) + " " + styles.CodeStyle.Render(node.Version))

		entry := fetcher.CacheEntry(node.Key)
		if entry == nil || entry.FS == nil {
			return fmt.Errorf("internal error: dep graph node %s missing cached fs", node.Key)
		}

		// foundry.Fetch already returns a per-mold fs rooted at the mold dir
		// (subpath included), and entry.Root points at that dir on disk —
		// mirror what runCast does for the root mold.
		//
		// For embedded deps entry.Root is empty (no on-disk path). Stage the
		// dep's ingots/ subtree to a temp dir so the disk-based IngotResolver
		// used during template rendering can find them.
		moldRoot := entry.Root
		if moldRoot == "" {
			if tmpDir, serr := stageIngotsFromFS(entry.FS); serr == nil && tmpDir != "" {
				defer func() { _ = os.RemoveAll(tmpDir) }()
				moldRoot = tmpDir
			}
		}
		reader := blanks.NewMoldReaderFromFS(entry.FS, moldRoot)

		manifest, err := reader.LoadManifest()
		if err != nil {
			return fmt.Errorf("loading mold.yaml for %s: %w", node.Key, err)
		}

		// Recurse the ingot/ore install pass for this transitive's own
		// non-mold deps. Mold-kind deps are handled by the depgraph walk.
		moldKey := node.Key.Source
		if node.Key.Subpath != "" {
			moldKey += "@" + node.Key.Subpath
		}
		if err := installDeclaredDeps(manifest, moldKey, castGlobal, false, castFrozen, false, nil); err != nil {
			return fmt.Errorf("installing declared deps of %s: %w", node.Key, err)
		}

		// Compute effective flux: dep defaults < parent's `with:` block <
		// root's --set/-f overrides scoped to <depAlias>.*. The dep's alias
		// defaults to its mold name when no `as:` was set.
		flux, schema, err := loadDepFlux(reader, manifest, &node, rootFlux)
		if err != nil {
			return fmt.Errorf("loading flux for %s: %w", node.Key, err)
		}

		ignorePatterns := mold.LoadIgnorePatterns(reader.FS(), manifest)
		var resolveOpts []mold.ResolveOption
		if len(ignorePatterns) > 0 {
			resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignorePatterns))
		}

		resolved, err := mold.ResolveFiles(flux["output"], reader.FS(), resolveOpts...)
		if err != nil {
			return fmt.Errorf("resolving output files for %s: %w", node.Key, err)
		}

		var filesToCast []mold.ResolvedFile
		for _, rf := range resolved {
			if !withWorkflows && strings.HasPrefix(rf.DestPath, ".github/") {
				continue
			}
			if destPrefix != "" {
				rf.DestPath = filepath.Join(destPrefix, rf.DestPath)
			}
			filesToCast = append(filesToCast, rf)
		}

		// Create destination directories for this dep's outputs.
		dirSet := make(map[string]bool)
		for _, rf := range filesToCast {
			dirSet[filepath.Dir(rf.DestPath)] = true
		}
		for d := range dirSet {
			if err := os.MkdirAll(d, 0750); err != nil { // #nosec G301
				return fmt.Errorf("creating directory %s for dep %s: %w", d, node.Key, err)
			}
		}

		if err := copyResolvedFilesWithSchema(reader, manifest, schema, flux, filesToCast, copyOpts{
			ForceReplaceOnParseError: castForceReplaceOnParseError,
		}); err != nil {
			return fmt.Errorf("copying files for %s: %w", node.Key, err)
		}

		// Record the transitive entry. Use the same destPrefix-stripping that
		// the root cast uses so .Files paths are project-relative.
		installedFiles := make([]foundry.InstalledFile, 0, len(filesToCast))
		for _, f := range filesToCast {
			sum, _ := hashFileForDeps(f.DestPath)
			rel := f.DestPath
			if destPrefix != "" {
				if r, rerr := filepath.Rel(destPrefix, f.DestPath); rerr == nil {
					rel = r
				}
			}
			installedFiles = append(installedFiles, foundry.InstalledFile{RelPath: rel, SHA256: sum})
		}

		// Synthesize a ResolveResult-shaped record from the cached fetch. The
		// cached Reference already has the post-resolution version baked in.
		depResult := &foundry.ResolveResult{
			Ref: entry.Reference,
			Resolved: foundry.ResolvedVersion{
				Tag:    node.Version,
				Commit: node.Commit,
			},
			Root: entry.Root,
		}
		// Build the parents list. For now we record the immediate parents
		// captured during graph construction.
		var parents []string
		seen := map[string]struct{}{}
		for _, e := range node.Parents {
			lbl := e.Source
			if e.Subpath != "" {
				lbl += "@" + e.Subpath
			}
			if _, dup := seen[lbl]; dup {
				continue
			}
			seen[lbl] = struct{}{}
			parents = append(parents, lbl)
		}
		if len(parents) == 0 {
			parents = []string{parentLabel}
		}

		if err := recordCastedFilesWithProvenance(depResult, installedFiles, castGlobal, nil, "transitive", parents, nil); err != nil {
			log.Printf("warning: failed to record transitive dep %s: %v", node.Key, err)
		}
	}
	return nil
}

// loadDepFlux computes the effective flux map for a transitive dep, mirroring
// loadCastFlux's ordering but rooted in the dep's own defaults and seeded by
// the parent's `with:` block + root --set/-f scoped to <depAlias>.*.
func loadDepFlux(reader *blanks.MoldReader, manifest *mold.Mold, node *depgraph.Node, rootFlux map[string]any) (map[string]any, []mold.FluxVar, error) {
	defaults, err := reader.LoadFluxDefaults()
	if err != nil {
		defaults = map[string]any{}
	}
	if manifest != nil && len(manifest.Flux) > 0 {
		defaults = mold.ApplyFluxDefaults(manifest.Flux, defaults)
	}

	flux := make(map[string]any, len(defaults))
	for k, v := range defaults {
		flux[k] = v
	}
	mold.ApplyManifestOutputDefault(flux, manifest)

	// Layer parent-supplied `with:` values.
	for k, v := range node.With {
		flux[k] = v
	}

	// Layer root --set scoped to this dep's alias. Treat <alias>.<key>=value
	// as setting <key> on the dep's flux. A child mold may also explicitly
	// reference root flux via direct keys — we don't try to do that magic
	// here; users opt-in by listing the keys in `with:` on the parent.
	alias := depAlias(node, manifest)
	if alias != "" {
		prefix := alias + "."
		for _, raw := range castSetFlags {
			parts := strings.SplitN(raw, "=", 2)
			if len(parts) != 2 || !strings.HasPrefix(parts[0], prefix) {
				continue
			}
			scoped := strings.TrimPrefix(parts[0], prefix) + "=" + parts[1]
			if err := mold.ApplySetOverrides(flux, []string{scoped}); err != nil {
				return nil, nil, fmt.Errorf("applying scoped --set %q: %w", raw, err)
			}
		}
	}

	schema := manifest.Flux
	if s, _ := reader.LoadFluxSchema(); len(s) > 0 {
		schema = s
	}
	return flux, schema, nil
}

// depAlias returns the alias the parent declared for this node, falling back
// to the mold's own name when the dep entry didn't set `as:`.
func depAlias(node *depgraph.Node, manifest *mold.Mold) string {
	for _, e := range node.Parents {
		if e.As != "" {
			return e.As
		}
	}
	if manifest != nil && manifest.Name != "" {
		return manifest.Name
	}
	return ""
}

// stageIngotsFromFS extracts the ingots/ subtree from fsys into a temporary
// directory, returning the temp dir path. Used when casting a transitive dep
// served from the embedded binary FS (entry.Root == "") so the disk-based
// IngotResolver can locate ingots via the temp dir. Returns ("", nil) when
// fsys has no ingots/ directory.
func stageIngotsFromFS(fsys fs.FS) (string, error) {
	if _, err := fs.Stat(fsys, "ingots"); err != nil {
		return "", nil
	}
	tmpDir, err := os.MkdirTemp("", "ailloy-dep-ingots-*")
	if err != nil {
		return "", err
	}
	werr := fs.WalkDir(fsys, "ingots", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		dest := filepath.Join(tmpDir, path)
		if d.IsDir() {
			return os.MkdirAll(dest, 0750) // #nosec G301
		}
		if merr := os.MkdirAll(filepath.Dir(dest), 0750); merr != nil { // #nosec G301
			return merr
		}
		data, rerr := fs.ReadFile(fsys, path)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(dest, data, 0644) // #nosec G306
	})
	if werr != nil {
		_ = os.RemoveAll(tmpDir)
		return "", werr
	}
	return tmpDir, nil
}

// hashFileForDeps duplicates cast.go's hashFile to avoid an import cycle on
// shared helpers; we only need a sha256 hex of the file contents.
func hashFileForDeps(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304 -- path under user control by design
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
