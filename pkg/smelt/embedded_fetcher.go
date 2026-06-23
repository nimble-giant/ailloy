package smelt

import (
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// EmbeddedDepFetcher implements depgraph.Fetcher backed by the smelted
// binary's embedded dep store. Mold deps present in the embedded manifest are
// served from the embedded FS without network access; deps absent from the
// manifest fall through to the wrapped ProdFetcher so non-smelted or
// partially-smelted binaries continue to work.
type EmbeddedDepFetcher struct {
	embFS    fs.FS
	manifest *DepManifest
	fallback *depgraph.ProdFetcher
	cache    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry
}

// NewEmbeddedDepFetcher opens the embedded FS from the current binary and
// reads the dep manifest. fallback is used for deps absent from the embedded
// manifest. Returns ErrNoEmbeddedMold if the binary has no embedded mold.
func NewEmbeddedDepFetcher(fallback *depgraph.ProdFetcher) (*EmbeddedDepFetcher, error) {
	embFS, err := OpenEmbeddedMold()
	if err != nil {
		return nil, err
	}

	var manifest DepManifest
	if data, err := fs.ReadFile(embFS, "deps/manifest.json"); err == nil {
		if jerr := json.Unmarshal(data, &manifest); jerr != nil {
			return nil, fmt.Errorf("parsing embedded dep manifest: %w", jerr)
		}
	}
	// Missing deps/manifest.json is fine — leaf mold with no dep subtree.

	return &EmbeddedDepFetcher{
		embFS:    embFS,
		manifest: &manifest,
		fallback: fallback,
		cache:    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}, nil
}

// moldEntry returns the manifest entry for the given (source, subpath), or nil.
func (e *EmbeddedDepFetcher) moldEntry(source, subpath string) *DepEntry {
	for i := range e.manifest.Molds {
		if e.manifest.Molds[i].Source == source && e.manifest.Molds[i].Subpath == subpath {
			return &e.manifest.Molds[i]
		}
	}
	return nil
}

// Fetch serves the mold from the embedded FS when the manifest lists it;
// otherwise falls through to the ProdFetcher.
func (e *EmbeddedDepFetcher) Fetch(ref *foundry.Reference) (depgraph.FetchResult, error) {
	key := depgraph.NodeKey{Source: ref.CacheKey(), Subpath: ref.Subpath}

	entry := e.moldEntry(ref.CacheKey(), ref.Subpath)
	if entry == nil {
		result, err := e.fallback.Fetch(ref)
		if err != nil {
			return depgraph.FetchResult{}, err
		}
		if ce := e.fallback.CacheEntry(key); ce != nil {
			e.cache[key] = ce
		}
		return result, nil
	}

	moldPath := depFSPath("molds", ref.CacheKey(), ref.Subpath)
	subFS, err := fs.Sub(e.embFS, moldPath)
	if err != nil {
		return depgraph.FetchResult{}, fmt.Errorf("opening embedded mold %s: %w", key, err)
	}

	m, err := mold.LoadMoldFromFS(subFS, "mold.yaml")
	if err != nil {
		return depgraph.FetchResult{}, fmt.Errorf("loading mold.yaml from embedded %s: %w", key, err)
	}

	pinnedRef := *ref
	pinnedRef.Version = entry.Version

	e.cache[key] = &depgraph.ProdFetchCacheEntry{
		FS:       subFS,
		Root:     "",
		Mold:     m,
		Resolved: foundry.ResolvedVersion{Tag: entry.Version, Commit: entry.Commit},
		// Reference carries the pinned version for downstream provenance recording.
		Reference: &pinnedRef,
	}

	return depgraph.FetchResult{
		Mold:    m,
		Version: entry.Version,
		Commit:  entry.Commit,
	}, nil
}

// Tags returns the single pinned version for the given source+subpath from the
// embedded manifest, bypassing constraint solving (the version was already
// resolved at smelt time). Falls through to ProdFetcher when not embedded.
func (e *EmbeddedDepFetcher) Tags(source, subpath string) (map[string]depgraph.TagInfo, error) {
	if entry := e.moldEntry(source, subpath); entry != nil {
		return map[string]depgraph.TagInfo{
			entry.Version: {SHA: entry.Commit, MoldVersion: entry.MoldVersion},
		}, nil
	}
	return e.fallback.Tags(source, subpath)
}

// CacheEntry returns the cached fetch result for the given node key.
func (e *EmbeddedDepFetcher) CacheEntry(key depgraph.NodeKey) *depgraph.ProdFetchCacheEntry {
	return e.cache[key]
}

// LookupEmbeddedArtifact checks whether the current binary has a mold, ore,
// or ingot dep embedded that matches (source, subpath). Returns (fs.FS,
// version, commit, true) when found. The returned FS is rooted at the
// artifact directory so mold.yaml / ore.yaml / ingot.yaml is at the root.
func LookupEmbeddedArtifact(source, subpath string) (fsys fs.FS, version, commit string, ok bool) {
	if !HasEmbeddedMold() {
		return nil, "", "", false
	}
	embFS, err := OpenEmbeddedMold()
	if err != nil {
		return nil, "", "", false
	}
	data, err := fs.ReadFile(embFS, "deps/manifest.json")
	if err != nil {
		return nil, "", "", false
	}
	var manifest DepManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", "", false
	}
	return lookupArtifactInFS(embFS, &manifest, source, subpath)
}

// lookupArtifactInFS searches embFS for a mold/ore/ingot matching (source,
// subpath). Molds are checked before ores so that a mold dep in the consumer's
// dependency list is served from the embedded tree rather than falling through
// to a network fetch.
func lookupArtifactInFS(embFS fs.FS, manifest *DepManifest, source, subpath string) (fsys fs.FS, version, commit string, ok bool) {
	lookup := func(entries []DepEntry, kind string) (fs.FS, string, string, bool) {
		for _, e := range entries {
			if e.Source == source && e.Subpath == subpath {
				p := depFSPath(kind, source, subpath)
				sub, serr := fs.Sub(embFS, p)
				if serr != nil {
					return nil, "", "", false
				}
				return sub, e.Version, e.Commit, true
			}
		}
		return nil, "", "", false
	}

	if f, v, c, found := lookup(manifest.Molds, "molds"); found {
		return f, v, c, true
	}
	if f, v, c, found := lookup(manifest.Ores, "ores"); found {
		return f, v, c, true
	}
	return lookup(manifest.Ingots, "ingots")
}
