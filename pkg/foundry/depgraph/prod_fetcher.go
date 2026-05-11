package depgraph

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// ProdFetcher implements Fetcher against the production foundry stack
// (foundry.ResolveWithMetadata + git ls-remote tag listing). It also caches
// fetched filesystems so callers can later read the rendered mold contents
// without re-fetching.
type ProdFetcher struct {
	GitRunner foundry.GitRunner
	// LockPath is forwarded to ResolveWithMetadata so transitive fetches
	// participate in the same lockfile as the root cast.
	LockPath string

	cache map[NodeKey]*ProdFetchCacheEntry
}

// ProdFetchCacheEntry captures everything callers need after Build to perform
// the actual cast pass for one node: the on-disk fs.FS, the absolute root
// path inside that FS, and the parsed mold.yaml.
type ProdFetchCacheEntry struct {
	FS        fs.FS
	Root      string
	Mold      *mold.Mold
	Resolved  foundry.ResolvedVersion
	Reference *foundry.Reference
}

// NewProdFetcher builds a fetcher backed by the default git runner.
func NewProdFetcher() *ProdFetcher {
	return &ProdFetcher{
		GitRunner: foundry.DefaultGitRunner(),
		cache:     map[NodeKey]*ProdFetchCacheEntry{},
	}
}

// CacheEntry returns the cached fetch for the given key, or nil if none.
// Callers use this after Build to obtain the fs.FS for each transitive node
// without re-fetching from the remote.
func (p *ProdFetcher) CacheEntry(key NodeKey) *ProdFetchCacheEntry {
	if p == nil || p.cache == nil {
		return nil
	}
	return p.cache[key]
}

// Fetch resolves the given reference and parses its mold.yaml. The fs.FS,
// resolved version, and parsed Mold are cached for later retrieval via
// CacheEntry().
func (p *ProdFetcher) Fetch(ref *foundry.Reference) (FetchResult, error) {
	if p.cache == nil {
		p.cache = map[NodeKey]*ProdFetchCacheEntry{}
	}
	rawRef := refToRaw(ref)
	var opts []foundry.ResolveOption
	if p.LockPath != "" {
		opts = append(opts, foundry.WithLockPath(p.LockPath))
	}
	fsys, result, err := foundry.ResolveWithMetadata(rawRef, opts...)
	if err != nil {
		return FetchResult{}, fmt.Errorf("resolve %s: %w", rawRef, err)
	}

	manifestPath := "mold.yaml"
	if ref.Subpath != "" {
		manifestPath = path.Join(strings.Trim(ref.Subpath, "/"), "mold.yaml")
	}
	m, err := mold.LoadMoldFromFS(fsys, manifestPath)
	if err != nil {
		return FetchResult{}, fmt.Errorf("loading mold.yaml at %s: %w", manifestPath, err)
	}

	key := NodeKey{Source: ref.CacheKey(), Subpath: ref.Subpath}
	p.cache[key] = &ProdFetchCacheEntry{
		FS:        fsys,
		Root:      result.Root,
		Mold:      m,
		Resolved:  result.Resolved,
		Reference: result.Ref,
	}
	return FetchResult{
		Mold:    m,
		Version: result.Resolved.Tag,
		Commit:  result.Resolved.Commit,
	}, nil
}

// Tags lists all semver tags for the given source+subpath via git ls-remote.
func (p *ProdFetcher) Tags(source, subpath string) (map[string]string, error) {
	url := "https://" + source + ".git"
	return foundry.RemoteTags(url, subpath, p.GitRunner)
}

// refToRaw renders a Reference back to a raw string suitable for
// foundry.ResolveWithMetadata. It mirrors Reference.String() but always emits
// the version + subpath when present.
func refToRaw(ref *foundry.Reference) string {
	s := ref.CacheKey()
	if ref.Version != "" {
		s += "@" + ref.Version
	}
	if ref.Subpath != "" {
		s += "//" + ref.Subpath
	}
	return s
}
