package depgraph

import (
	"fmt"
	"io/fs"
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
	// Offline disables all network operations; tag listing and bare-clone
	// fetches are served from the local cache. Set by --offline on cast.
	Offline bool

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
	var opts []foundry.ResolveOption
	if p.LockPath != "" {
		opts = append(opts, foundry.WithLockPath(p.LockPath))
	}
	if p.Offline {
		opts = append(opts, foundry.WithOffline())
	}
	// Resolve from the *Reference directly so an explicitly-set Type (e.g. an
	// exact pin to a monorepo-prefixed tag during constraint re-fetch) is not
	// lost to a raw-string round-trip.
	fsys, result, err := foundry.ResolveReferenceWithMetadata(ref, opts...)
	if err != nil {
		return FetchResult{}, fmt.Errorf("resolve %s: %w", refToRaw(ref), err)
	}

	// foundry.ResolveWithMetadata already returns an fs.FS rooted at the
	// mold directory (subpath navigation applied), so the manifest is at the
	// fs root regardless of any subpath on the reference.
	m, err := mold.LoadMoldFromFS(fsys, "mold.yaml")
	if err != nil {
		return FetchResult{}, fmt.Errorf("loading mold.yaml for %s: %w", refToRaw(ref), err)
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

// Tags lists the semver tags for the given source+subpath. When p.Offline is
// true the listing is served from the local bare-clone cache instead of
// git ls-remote. For monorepo subpath molds it also reads each tag's mold.yaml
// version so the constraint solver ranks by the mold's own version rather than
// the shared release-train version baked into the tag name. Tags whose mold
// manifest is absent are dropped.
func (p *ProdFetcher) Tags(source, subpath string) (map[string]TagInfo, error) {
	git, err := p.effectiveGitRunner()
	if err != nil {
		return nil, err
	}
	url := "https://" + source + ".git"
	tags, err := foundry.RemoteTags(url, subpath, git)
	if err != nil {
		return nil, err
	}

	var reader foundry.MoldVersionReader
	if strings.Trim(subpath, "/") != "" {
		if r, rerr := p.moldVersionReader(source, subpath); rerr == nil {
			reader = r
		}
	}

	out := make(map[string]TagInfo, len(tags))
	for tag, sha := range tags {
		info := TagInfo{SHA: sha}
		if reader != nil {
			mv, found := reader(tag)
			if !found {
				continue // mold manifest absent at this tag
			}
			info.MoldVersion = mv
		}
		out[tag] = info
	}
	return out, nil
}

// moldVersionReader builds a foundry.MoldVersionReader for the given
// source+subpath, backed by a bare clone.
func (p *ProdFetcher) moldVersionReader(source, subpath string) (foundry.MoldVersionReader, error) {
	ref, err := foundry.ParseReference(source)
	if err != nil {
		return nil, err
	}
	ref.Subpath = subpath
	git, err := p.effectiveGitRunner()
	if err != nil {
		return nil, err
	}
	fetcher, err := foundry.NewFetcher(git)
	if err != nil {
		return nil, err
	}
	return fetcher.MoldVersionReaderFor(ref)
}

// effectiveGitRunner returns the GitRunner to use for this fetch. When
// p.Offline is true it wraps the real runner with the offline interceptor so
// that network-requiring commands are served from (or blocked by) the local
// cache.
func (p *ProdFetcher) effectiveGitRunner() (foundry.GitRunner, error) {
	if !p.Offline {
		return p.GitRunner, nil
	}
	cacheDir, err := foundry.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("offline mode: %w", err)
	}
	return foundry.NewOfflineGitRunner(p.GitRunner, cacheDir), nil
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
