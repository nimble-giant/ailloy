package foundry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CacheDir returns the root cache directory (~/.ailloy/cache).
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ailloy", "cache"), nil
}

// IsCached returns true when the given version directory exists and contains
// a mold.yaml or ingot.yaml manifest.
func IsCached(cacheDir string, ref *Reference, version string) bool {
	versionDir := filepath.Join(cacheDir, ref.CacheKey(), version)
	root := versionDir
	if ref.Subpath != "" {
		root = filepath.Join(versionDir, ref.Subpath)
	}
	if hasMoldManifest(root) {
		return true
	}
	return false
}

// hasMoldManifest checks whether a directory contains mold.yaml or ingot.yaml.
func hasMoldManifest(dir string) bool {
	for _, name := range []string{"mold.yaml", "ingot.yaml"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// BareCloneDir returns the path to the bare git clone for a reference.
func BareCloneDir(cacheDir string, ref *Reference) string {
	return filepath.Join(cacheDir, ref.CacheKey(), "git")
}

// VersionDir returns the path to a specific version snapshot.
func VersionDir(cacheDir string, ref *Reference, version string) string {
	return filepath.Join(cacheDir, ref.CacheKey(), version)
}

// CacheEntry represents a cached mold entry.
type CacheEntry struct {
	Host     string
	Owner    string
	Repo     string
	Versions []string
}

// ListCachedMolds lists all cached molds and their versions.
func ListCachedMolds(cacheDir string) ([]CacheEntry, error) {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil, nil
	}

	var entries []CacheEntry

	// Walk host/owner/repo structure
	hosts, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("reading cache directory: %w", err)
	}

	for _, host := range hosts {
		if !host.IsDir() {
			continue
		}
		owners, err := os.ReadDir(filepath.Join(cacheDir, host.Name()))
		if err != nil {
			continue
		}
		for _, owner := range owners {
			if !owner.IsDir() {
				continue
			}
			repos, err := os.ReadDir(filepath.Join(cacheDir, host.Name(), owner.Name()))
			if err != nil {
				continue
			}
			for _, repo := range repos {
				if !repo.IsDir() {
					continue
				}
				entry := CacheEntry{
					Host:  host.Name(),
					Owner: owner.Name(),
					Repo:  repo.Name(),
				}
				// Collect version directories (skip "git" bare clone dir)
				versions, err := os.ReadDir(filepath.Join(cacheDir, host.Name(), owner.Name(), repo.Name()))
				if err != nil {
					continue
				}
				for _, v := range versions {
					if v.IsDir() && v.Name() != "git" {
						entry.Versions = append(entry.Versions, v.Name())
					}
				}
				entries = append(entries, entry)
			}
		}
	}
	return entries, nil
}

// CleanCache removes the entire cache directory.
func CleanCache(cacheDir string) error {
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("cleaning cache: %w", err)
	}
	return nil
}

// CleanMold removes all cached data for a specific mold.
func CleanMold(cacheDir string, ref *Reference) error {
	moldDir := filepath.Join(cacheDir, ref.CacheKey())

	// Safety: ensure the target is under the cache directory.
	absCache, err := filepath.Abs(cacheDir)
	if err != nil {
		return fmt.Errorf("resolving cache path: %w", err)
	}
	absMold, err := filepath.Abs(moldDir)
	if err != nil {
		return fmt.Errorf("resolving mold cache path: %w", err)
	}
	if !strings.HasPrefix(absMold, absCache+string(filepath.Separator)) {
		return fmt.Errorf("mold cache path %q escapes cache root %q", absMold, absCache)
	}

	if err := os.RemoveAll(moldDir); err != nil {
		return fmt.Errorf("cleaning mold cache for %s: %w", ref.CacheKey(), err)
	}
	return nil
}
