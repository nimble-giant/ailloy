package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCachedIndexDir_Git(t *testing.T) {
	entry := &FoundryEntry{
		URL:  "https://github.com/nimble-giant/ailloy-foundry-index",
		Type: "git",
	}
	dir := CachedIndexDir("/tmp/cache", entry)
	want := filepath.Join("/tmp/cache", "github.com", "nimble-giant", "ailloy-foundry-index")
	if dir != want {
		t.Errorf("got %q, want %q", dir, want)
	}
}

func TestCachedIndexDir_URL(t *testing.T) {
	entry := &FoundryEntry{
		URL:  "https://example.com/path/to/foundry.yaml",
		Type: "url",
	}
	dir := CachedIndexDir("/tmp/cache", entry)
	// Should be hash-based, 16 char hex.
	base := filepath.Base(dir)
	if len(base) != 16 {
		t.Errorf("expected 16-char hash dir, got %q", base)
	}
}

func TestCachedIndexPath(t *testing.T) {
	entry := &FoundryEntry{
		URL:  "https://github.com/test/index",
		Type: "git",
	}
	path := CachedIndexPath("/tmp/cache", entry)
	if !strings.HasSuffix(path, "foundry.yaml") {
		t.Errorf("expected path to end with foundry.yaml, got %q", path)
	}
}

func TestLoadCachedIndex(t *testing.T) {
	dir := t.TempDir()
	entry := &FoundryEntry{
		Name: "test",
		URL:  "https://github.com/test/index",
		Type: "git",
	}

	// Create the cached file.
	cachePath := CachedIndexPath(dir, entry)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
		t.Fatal(err)
	}

	content := `apiVersion: v1
kind: foundry-index
name: test
molds:
  - name: mold1
    source: github.com/test/mold1
`
	if err := os.WriteFile(cachePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	idx, err := LoadCachedIndex(dir, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Name != "test" {
		t.Errorf("Name = %q, want test", idx.Name)
	}
	if len(idx.Molds) != 1 {
		t.Errorf("len(Molds) = %d, want 1", len(idx.Molds))
	}
}

func TestLoadCachedIndex_NotFound(t *testing.T) {
	entry := &FoundryEntry{
		Name: "test",
		URL:  "https://github.com/test/index",
		Type: "git",
	}
	_, err := LoadCachedIndex(t.TempDir(), entry)
	if err == nil {
		t.Fatal("expected error for missing cache")
	}
	if !strings.Contains(err.Error(), "no cached index") {
		t.Errorf("error %q doesn't mention missing cache", err.Error())
	}
}

func TestCleanIndexCache(t *testing.T) {
	dir := t.TempDir()
	entry := &FoundryEntry{
		URL:  "https://github.com/test/index",
		Type: "git",
	}

	// Create a cached file.
	cachePath := CachedIndexPath(dir, entry)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CleanIndexCache(dir, entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(CachedIndexDir(dir, entry)); !os.IsNotExist(err) {
		t.Error("expected cache directory to be removed")
	}
}
