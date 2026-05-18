package foundry

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
)

// Fetcher clones and checks out mold versions from git repositories.
type Fetcher struct {
	git      GitRunner
	cacheDir string
}

// NewFetcher creates a Fetcher that caches into the default cache directory.
func NewFetcher(git GitRunner) (*Fetcher, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	return &Fetcher{git: git, cacheDir: dir}, nil
}

// NewFetcherWithCacheDir creates a Fetcher with a specific cache directory
// (useful for testing).
func NewFetcherWithCacheDir(git GitRunner, cacheDir string) *Fetcher {
	return &Fetcher{git: git, cacheDir: cacheDir}
}

// Fetch resolves and extracts a mold version, returning an fs.FS rooted at
// the (possibly subpath-navigated) mold directory along with its on-disk root.
func (f *Fetcher) Fetch(ref *Reference, resolved *ResolvedVersion) (fs.FS, string, error) {
	if err := f.ensureBareClone(ref); err != nil {
		return nil, "", fmt.Errorf("ensuring bare clone: %w", err)
	}

	vDir := VersionDir(f.cacheDir, ref, resolved.Tag)
	if !hasMoldManifestInDir(vDir, ref.Subpath) {
		if err := f.checkoutVersion(ref, resolved); err != nil {
			return nil, "", fmt.Errorf("checking out version: %w", err)
		}
	}

	return f.navigateSubpath(ref, resolved)
}

// MoldVersionReaderFor returns a MoldVersionReader that reads the `version:`
// field of the reference's mold.yaml at any given git tag. It ensures the
// bare clone exists once, then serves each lookup with `git show <tag>:<path>`
// against the local clone — cheap and offline. Results are memoised per tag.
//
// The reader reports found=false when no mold manifest exists at a tag (the
// caller excludes that candidate), and found=true with an empty version when
// the manifest exists but declares no version (the caller falls back to the
// tag-embedded semver).
func (f *Fetcher) MoldVersionReaderFor(ref *Reference) (MoldVersionReader, error) {
	if err := f.ensureBareClone(ref); err != nil {
		return nil, fmt.Errorf("ensuring bare clone: %w", err)
	}
	bareDir := BareCloneDir(f.cacheDir, ref)

	manifestPath := "mold.yaml"
	if sp := strings.Trim(ref.Subpath, "/"); sp != "" {
		manifestPath = sp + "/mold.yaml" // git object paths are always forward-slash
	}

	type result struct {
		version string
		found   bool
	}
	var mu sync.Mutex
	cache := map[string]result{}

	return func(tag string) (string, bool) {
		mu.Lock()
		defer mu.Unlock()
		if r, ok := cache[tag]; ok {
			return r.version, r.found
		}
		out, err := f.git("-C", bareDir, "show", tag+":"+manifestPath)
		r := result{found: err == nil}
		if r.found {
			var m struct {
				Version string `yaml:"version"`
			}
			if yaml.Unmarshal(out, &m) == nil {
				r.version = m.Version
			}
		}
		cache[tag] = r
		return r.version, r.found
	}, nil
}

// ensureBareClone creates or updates the bare clone for the reference.
func (f *Fetcher) ensureBareClone(ref *Reference) error {
	bareDir := BareCloneDir(f.cacheDir, ref)

	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); err == nil {
		// Bare clone exists — fetch updates including new tags.
		// --tags is required so tags published after the initial clone
		// (e.g. v0.4.0 of nimble-mold) become resolvable. Without it,
		// `git archive <new-tag>` fails with "not a valid object name".
		out, err := f.git("-C", bareDir, "fetch", "--all", "--tags", "--force")
		if err != nil {
			return fmt.Errorf("git fetch --all --tags: %w\n%s", err, out)
		}
		return nil
	}

	// Create parent directories.
	if err := os.MkdirAll(filepath.Dir(bareDir), 0750); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	out, err := f.git("clone", "--bare", ref.CloneURL(), bareDir)
	if err != nil {
		return fmt.Errorf("git clone --bare %s: %w\n%s", ref.CloneURL(), err, out)
	}
	return nil
}

// checkoutVersion extracts a specific version from the bare clone into a
// version directory using git archive.
func (f *Fetcher) checkoutVersion(ref *Reference, resolved *ResolvedVersion) error {
	bareDir := BareCloneDir(f.cacheDir, ref)
	vDir := VersionDir(f.cacheDir, ref, resolved.Tag)

	if err := os.MkdirAll(vDir, 0750); err != nil {
		return fmt.Errorf("creating version directory: %w", err)
	}

	// Use git archive to extract files without a working tree.
	out, err := f.git("-C", bareDir, "archive", "--format=tar", resolved.Tag)
	if err != nil {
		return fmt.Errorf("git archive %s: %w\n%s", resolved.Tag, err, out)
	}

	// Extract tar into version directory.
	if err := extractTar(out, vDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}

// navigateSubpath applies the //subpath and validates the mold manifest exists.
// It returns the resulting fs.FS and the on-disk root path.
func (f *Fetcher) navigateSubpath(ref *Reference, resolved *ResolvedVersion) (fs.FS, string, error) {
	vDir := VersionDir(f.cacheDir, ref, resolved.Tag)
	root := vDir

	if ref.Subpath != "" {
		// Safety: ensure subpath doesn't escape the version directory.
		absVersion, err := filepath.Abs(vDir)
		if err != nil {
			return nil, "", fmt.Errorf("resolving version path: %w", err)
		}
		absTarget, err := filepath.Abs(filepath.Join(vDir, ref.Subpath))
		if err != nil {
			return nil, "", fmt.Errorf("resolving subpath: %w", err)
		}
		if !strings.HasPrefix(absTarget, absVersion+string(filepath.Separator)) {
			return nil, "", fmt.Errorf("subpath %q escapes version directory", ref.Subpath)
		}
		root = absTarget
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("subpath %q does not exist in %s@%s", ref.Subpath, ref.CacheKey(), resolved.Tag)
	}

	if !hasMoldManifest(root) {
		return nil, "", fmt.Errorf("no mold.yaml or ingot.yaml found at %s@%s//%s", ref.CacheKey(), resolved.Tag, ref.Subpath)
	}

	return os.DirFS(root), root, nil
}

// hasMoldManifestInDir checks the version dir (with optional subpath) for a manifest.
func hasMoldManifestInDir(vDir, subpath string) bool {
	root := vDir
	if subpath != "" {
		root = filepath.Join(vDir, subpath)
	}
	return hasMoldManifest(root)
}
