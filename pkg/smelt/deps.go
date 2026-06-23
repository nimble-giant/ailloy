package smelt

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// DepEntry is one record in the embedded dep manifest.
type DepEntry struct {
	Source      string `json:"source"`
	Subpath     string `json:"subpath,omitempty"`
	Version     string `json:"version"`
	Commit      string `json:"commit,omitempty"`
	MoldVersion string `json:"mold_version,omitempty"`
}

// DepManifest is written to /deps/manifest.json in the smelted binary so the
// cast side can enumerate bundled deps without scanning the FS.
type DepManifest struct {
	Molds  []DepEntry `json:"molds"`
	Ores   []DepEntry `json:"ores"`
	Ingots []DepEntry `json:"ingots"`
}

// smeltDepFetcher is the subset of depgraph.ProdFetcher that collectDepsWith
// needs: graph construction + per-node cached FS retrieval.
// Tests inject a fake; production uses depgraph.NewProdFetcher().
type smeltDepFetcher interface {
	depgraph.Fetcher
	CacheEntry(depgraph.NodeKey) *depgraph.ProdFetchCacheEntry
}

// artifactResolverFunc resolves a raw ore/ingot reference to an fs.FS plus
// provenance. Matches the signature of foundry.ResolveWithMetadata.
type artifactResolverFunc func(rawRef string, opts ...foundry.ResolveOption) (fs.FS, *foundry.ResolveResult, error)

// collectDeps resolves and embeds all transitive mold, ore, and ingot deps for
// the mold rooted at moldDir. Returns the extra archiveFiles to stuff into the
// binary and the manifest summarising what was bundled. When the mold has no
// dependencies both slices are empty (leaf-mold case: no /deps/ subtree).
func collectDeps(moldDir string, m *mold.Mold) ([]archiveFile, *DepManifest, error) {
	var resolveOpts []foundry.ResolveOption
	lockPath := filepath.Join(moldDir, foundry.LockFileName)
	if _, err := os.Stat(lockPath); err == nil {
		resolveOpts = append(resolveOpts, foundry.WithLockPath(lockPath))
	}

	fetcher := depgraph.NewProdFetcher()
	if len(resolveOpts) > 0 {
		fetcher.LockPath = lockPath
	}

	return collectDepsWith(m, resolveOpts, fetcher, foundry.ResolveWithMetadata)
}

// collectDepsWith is the testable core: callers inject a fetcher and artifact
// resolver so unit tests can drive the pipeline without network access.
func collectDepsWith(
	m *mold.Mold,
	resolveOpts []foundry.ResolveOption,
	fetcher smeltDepFetcher,
	resolveArtifact artifactResolverFunc,
) ([]archiveFile, *DepManifest, error) {
	manifest := &DepManifest{
		Molds:  []DepEntry{},
		Ores:   []DepEntry{},
		Ingots: []DepEntry{},
	}
	var files []archiveFile

	seenOres := map[string]bool{}
	seenIngots := map[string]bool{}

	// Collect ore/ingot deps declared by the root mold.
	af, err := collectArtifactDeps(m, resolveOpts, resolveArtifact, seenOres, seenIngots, manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("collecting root ore/ingot deps: %w", err)
	}
	files = append(files, af...)

	// Skip the dep-graph walk when the root has no mold-kind deps (leaf mold).
	if !hasMoldDeps(m) {
		return files, manifest, nil
	}

	name := m.Name
	if name == "" {
		name = "local"
	}
	rootRef := &foundry.Reference{
		Host:  "local",
		Owner: "dir",
		Repo:  name,
	}

	graph, err := depgraph.New(fetcher).Build(m, rootRef)
	if err != nil {
		return nil, nil, fmt.Errorf("building dep graph: %w", err)
	}

	rootKey := depgraph.NodeKey{Source: rootRef.CacheKey(), Subpath: rootRef.Subpath}

	for _, node := range graph.Nodes {
		if node.Key == rootKey {
			continue
		}

		entry := fetcher.CacheEntry(node.Key)
		if entry == nil || entry.FS == nil {
			return nil, nil, fmt.Errorf("dep graph node %s missing cached fs", node.Key)
		}

		moldVersion := ""
		if entry.Mold != nil {
			moldVersion = entry.Mold.Version
		}
		manifest.Molds = append(manifest.Molds, DepEntry{
			Source:      node.Key.Source,
			Subpath:     node.Key.Subpath,
			Version:     node.Version,
			Commit:      node.Commit,
			MoldVersion: moldVersion,
		})

		base := depFSPath("molds", node.Key.Source, node.Key.Subpath)
		moldFiles, err := walkIntoFiles(entry.FS, base)
		if err != nil {
			return nil, nil, fmt.Errorf("collecting files for dep %s: %w", node.Key, err)
		}
		files = append(files, moldFiles...)

		af, err := collectArtifactDeps(entry.Mold, resolveOpts, resolveArtifact, seenOres, seenIngots, manifest)
		if err != nil {
			return nil, nil, fmt.Errorf("collecting ore/ingot deps for %s: %w", node.Key, err)
		}
		files = append(files, af...)
	}

	return files, manifest, nil
}

// collectArtifactDeps resolves and embeds ore/ingot deps declared by m.
// Already-seen (source+subpath) pairs are skipped to deduplicate when the same
// artifact is referenced by multiple molds in the graph.
func collectArtifactDeps(
	m *mold.Mold,
	resolveOpts []foundry.ResolveOption,
	resolveArtifact artifactResolverFunc,
	seenOres, seenIngots map[string]bool,
	manifest *DepManifest,
) ([]archiveFile, error) {
	if m == nil {
		return nil, nil
	}
	var files []archiveFile
	for _, dep := range m.Dependencies {
		kind, err := dep.Kind()
		if err != nil || (kind != "ore" && kind != "ingot") {
			continue
		}

		var raw string
		var seen map[string]bool
		switch kind {
		case "ore":
			raw = dep.Ore
			seen = seenOres
		case "ingot":
			raw = dep.Ingot
			seen = seenIngots
		}

		if !foundry.IsRemoteReference(raw) {
			// Local ore/ingot deps are resolved at cast time from disk.
			continue
		}

		ref, err := foundry.ParseReference(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing %s ref %q: %w", kind, raw, err)
		}
		dedupeKey := ref.CacheKey() + "//" + ref.Subpath
		if seen[dedupeKey] {
			continue
		}
		seen[dedupeKey] = true

		fsys, result, err := resolveArtifact(raw, resolveOpts...)
		if err != nil {
			return nil, fmt.Errorf("resolving %s %q: %w", kind, raw, err)
		}

		base := depFSPath(kind+"s", result.Ref.CacheKey(), result.Ref.Subpath)
		af, err := walkIntoFiles(fsys, base)
		if err != nil {
			return nil, fmt.Errorf("collecting files for %s %s: %w", kind, raw, err)
		}
		files = append(files, af...)

		entry := DepEntry{
			Source:  result.Ref.CacheKey(),
			Subpath: result.Ref.Subpath,
			Version: result.Resolved.Tag,
			Commit:  result.Resolved.Commit,
		}
		switch kind {
		case "ore":
			manifest.Ores = append(manifest.Ores, entry)
		case "ingot":
			manifest.Ingots = append(manifest.Ingots, entry)
		}
	}
	return files, nil
}

// hasMoldDeps reports whether m declares any mold-kind dependencies.
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

// depFSPath constructs the embedded FS path for a bundled dep artifact:
//
//	deps/<kind>/<source>[/<subpath>]
func depFSPath(kind, source, subpath string) string {
	p := "deps/" + kind + "/" + source
	if sp := strings.Trim(subpath, "/"); sp != "" {
		p += "/" + sp
	}
	return p
}

// walkIntoFiles walks fsys and returns every regular file as an archiveFile
// with path prefixed under base.
func walkIntoFiles(fsys fs.FS, base string) ([]archiveFile, error) {
	var files []archiveFile
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		files = append(files, archiveFile{
			path: base + "/" + path,
			data: data,
		})
		return nil
	})
	return files, err
}

// marshalDepManifest returns the JSON encoding of m, or nil when empty.
func marshalDepManifest(m *DepManifest) []byte {
	if m == nil || (len(m.Molds) == 0 && len(m.Ores) == 0 && len(m.Ingots) == 0) {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return data
}
