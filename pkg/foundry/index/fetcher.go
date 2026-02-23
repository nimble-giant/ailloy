package index

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GitRunner executes a git command and returns its combined output.
type GitRunner func(args ...string) ([]byte, error)

// Fetcher retrieves foundry index files from remote sources.
type Fetcher struct {
	git      GitRunner
	cacheDir string
}

// NewFetcher creates a Fetcher that caches into the default index cache directory.
func NewFetcher(git GitRunner) (*Fetcher, error) {
	dir, err := IndexCacheDir()
	if err != nil {
		return nil, err
	}
	return &Fetcher{git: git, cacheDir: dir}, nil
}

// NewFetcherWithCacheDir creates a Fetcher with a specific cache directory (for testing).
func NewFetcherWithCacheDir(git GitRunner, cacheDir string) *Fetcher {
	return &Fetcher{git: git, cacheDir: cacheDir}
}

// FetchIndex retrieves the foundry.yaml from the given entry, caches it,
// and returns the parsed Index. The entry's LastUpdated and Status are updated.
func (f *Fetcher) FetchIndex(entry *FoundryEntry) (*Index, error) {
	var idx *Index
	var err error

	switch entry.Type {
	case "url":
		idx, err = f.fetchURLIndex(entry)
	default: // "git" or unspecified
		idx, err = f.fetchGitIndex(entry)
	}

	if err != nil {
		entry.Status = "error"
		return nil, err
	}

	entry.LastUpdated = time.Now().UTC()
	entry.Status = "ok"
	return idx, nil
}

// fetchGitIndex clones or updates a bare repo and reads foundry.yaml from HEAD.
func (f *Fetcher) fetchGitIndex(entry *FoundryEntry) (*Index, error) {
	indexDir := CachedIndexDir(f.cacheDir, entry)
	bareDir := filepath.Join(indexDir, "git")

	// Clone or fetch.
	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); err == nil {
		// Bare clone exists — fetch updates.
		out, err := f.git("-C", bareDir, "fetch", "--all")
		if err != nil {
			return nil, fmt.Errorf("git fetch: %w\n%s", err, out)
		}
	} else {
		// Create bare clone.
		if err := os.MkdirAll(filepath.Dir(bareDir), 0750); err != nil {
			return nil, fmt.Errorf("creating cache directory: %w", err)
		}
		out, err := f.git("clone", "--bare", entry.URL, bareDir)
		if err != nil {
			return nil, fmt.Errorf("git clone: %w\n%s", err, out)
		}
	}

	// Read foundry.yaml from HEAD using git show.
	out, err := f.git("-C", bareDir, "show", "HEAD:"+indexFileName)
	if err != nil {
		return nil, fmt.Errorf("reading %s from HEAD: %w\n%s", indexFileName, err, out)
	}

	idx, err := ParseIndex(out)
	if err != nil {
		return nil, err
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}

	// Cache the raw YAML for offline use.
	cachePath := CachedIndexPath(f.cacheDir, entry)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(cachePath, out, 0644); err != nil { // #nosec G306 -- cache file
		return nil, fmt.Errorf("writing cache: %w", err)
	}

	// Populate name from index if not set.
	if entry.Name == "" || entry.Name == nameFromURL(entry.URL) {
		entry.Name = idx.Name
	}

	return idx, nil
}

// fetchURLIndex downloads a raw YAML file via HTTP GET.
func (f *Fetcher) fetchURLIndex(entry *FoundryEntry) (*Index, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(entry.URL) // #nosec G107 -- user-provided foundry URL
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching index: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB limit
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	idx, err := ParseIndex(data)
	if err != nil {
		return nil, err
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}

	// Cache the raw YAML.
	cachePath := CachedIndexPath(f.cacheDir, entry)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil { // #nosec G306 -- cache file
		return nil, fmt.Errorf("writing cache: %w", err)
	}

	// Populate name from index if not set.
	if entry.Name == "" || entry.Name == nameFromURL(entry.URL) {
		entry.Name = idx.Name
	}

	return idx, nil
}
