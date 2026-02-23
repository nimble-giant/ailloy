package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchGitIndex(t *testing.T) {
	cacheDir := t.TempDir()

	validIndex := `apiVersion: v1
kind: foundry-index
name: test-foundry
molds:
  - name: mold1
    source: github.com/test/mold1
    description: "Test mold"
`
	// Mock git to handle clone and show commands.
	git := func(args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		if strings.Contains(key, "clone --bare") {
			// Create a fake bare clone directory with HEAD.
			for _, arg := range args {
				if strings.HasPrefix(arg, cacheDir) {
					if err := os.MkdirAll(arg, 0750); err != nil {
						return nil, err
					}
					if err := os.WriteFile(filepath.Join(arg, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
						return nil, err
					}
				}
			}
			return []byte("Cloning..."), nil
		}
		if strings.Contains(key, "show") && strings.Contains(key, "foundry.yaml") {
			return []byte(validIndex), nil
		}
		return nil, fmt.Errorf("unexpected: %s", key)
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	entry := &FoundryEntry{
		Name: "test",
		URL:  "https://github.com/test/foundry-index",
		Type: "git",
	}

	idx, err := fetcher.FetchIndex(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Name != "test-foundry" {
		t.Errorf("Name = %q, want test-foundry", idx.Name)
	}
	if len(idx.Molds) != 1 {
		t.Errorf("len(Molds) = %d, want 1", len(idx.Molds))
	}
	if entry.Status != "ok" {
		t.Errorf("Status = %q, want ok", entry.Status)
	}
	// Name was explicitly set to "test", so it should not be overridden.
	if entry.Name != "test" {
		t.Errorf("entry.Name = %q, want test (should preserve explicit name)", entry.Name)
	}

	// Verify the index was cached.
	cachePath := CachedIndexPath(cacheDir, entry)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("expected cached index file to exist")
	}
}

func TestFetchGitIndex_FetchExisting(t *testing.T) {
	cacheDir := t.TempDir()

	validIndex := `apiVersion: v1
kind: foundry-index
name: updated-foundry
molds: []
`
	entry := &FoundryEntry{
		Name: "test",
		URL:  "https://github.com/test/foundry-index",
		Type: "git",
	}

	// Pre-create the bare clone directory so fetch is used instead of clone.
	bareDir := filepath.Join(CachedIndexDir(cacheDir, entry), "git")
	if err := os.MkdirAll(bareDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bareDir, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatal(err)
	}

	fetchCalled := false
	git := func(args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		if strings.Contains(key, "fetch --all") {
			fetchCalled = true
			return []byte(""), nil
		}
		if strings.Contains(key, "show") && strings.Contains(key, "foundry.yaml") {
			return []byte(validIndex), nil
		}
		return nil, fmt.Errorf("unexpected: %s", key)
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	idx, err := fetcher.FetchIndex(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fetchCalled {
		t.Error("expected git fetch to be called for existing bare clone")
	}
	if idx.Name != "updated-foundry" {
		t.Errorf("Name = %q, want updated-foundry", idx.Name)
	}
}

func TestFetchIndex_ErrorSetsStatus(t *testing.T) {
	cacheDir := t.TempDir()

	git := func(args ...string) ([]byte, error) {
		return nil, fmt.Errorf("network error")
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	entry := &FoundryEntry{
		Name: "test",
		URL:  "https://github.com/test/broken",
		Type: "git",
	}

	_, err := fetcher.FetchIndex(entry)
	if err == nil {
		t.Fatal("expected error")
	}
	if entry.Status != "error" {
		t.Errorf("Status = %q, want error", entry.Status)
	}
}
