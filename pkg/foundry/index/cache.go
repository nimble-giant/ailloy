package index

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const indexFileName = "foundry.yaml"

// IndexCacheDir returns the root cache directory for foundry indexes (~/.ailloy/cache/indexes).
func IndexCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ailloy", "cache", "indexes"), nil
}

// CachedIndexDir returns the cache directory for a specific foundry entry.
// For git-type entries, the path mirrors the URL structure: <host>/<owner>/<repo>.
// For url-type entries, a SHA-256 hash of the URL is used.
func CachedIndexDir(cacheDir string, entry *FoundryEntry) string {
	if entry.Type == "git" {
		// Strip scheme and trailing slashes for a clean path.
		cleaned := entry.URL
		cleaned = strings.TrimPrefix(cleaned, "https://")
		cleaned = strings.TrimPrefix(cleaned, "http://")
		cleaned = strings.TrimSuffix(cleaned, "/")
		return filepath.Join(cacheDir, cleaned)
	}
	// URL type: use a hash-based directory.
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(entry.URL)))
	return filepath.Join(cacheDir, hash[:16])
}

// CachedIndexPath returns the path to the cached foundry.yaml for an entry.
func CachedIndexPath(cacheDir string, entry *FoundryEntry) string {
	return filepath.Join(CachedIndexDir(cacheDir, entry), indexFileName)
}

// LoadCachedIndex reads a previously cached index file without network access.
func LoadCachedIndex(cacheDir string, entry *FoundryEntry) (*Index, error) {
	path := CachedIndexPath(cacheDir, entry)
	data, err := os.ReadFile(path) // #nosec G304 -- reading cached index file
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no cached index for %q; run 'ailloy foundry update'", entry.Name)
		}
		return nil, fmt.Errorf("reading cached index: %w", err)
	}
	return ParseIndex(data)
}

// CleanIndexCache removes the cached data for a specific foundry entry.
func CleanIndexCache(cacheDir string, entry *FoundryEntry) error {
	dir := CachedIndexDir(cacheDir, entry)

	// Safety: ensure the target is under the cache directory.
	absCache, err := filepath.Abs(cacheDir)
	if err != nil {
		return fmt.Errorf("resolving cache path: %w", err)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving index cache path: %w", err)
	}
	if !strings.HasPrefix(absDir, absCache+string(filepath.Separator)) {
		return fmt.Errorf("index cache path %q escapes cache root %q", absDir, absCache)
	}

	return os.RemoveAll(dir)
}
