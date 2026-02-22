package foundry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	dir, err := CacheDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty cache dir")
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".ailloy", "cache")
	if dir != want {
		t.Errorf("CacheDir() = %q, want %q", dir, want)
	}
}

func TestIsCached(t *testing.T) {
	cacheDir := t.TempDir()
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}

	// Not cached yet.
	if IsCached(cacheDir, ref, "v1.0.0") {
		t.Error("expected not cached")
	}

	// Create version dir without manifest.
	vDir := filepath.Join(cacheDir, "github.com", "owner", "repo", "v1.0.0")
	if err := os.MkdirAll(vDir, 0750); err != nil {
		t.Fatal(err)
	}
	if IsCached(cacheDir, ref, "v1.0.0") {
		t.Error("expected not cached without manifest")
	}

	// Add mold.yaml.
	if err := os.WriteFile(filepath.Join(vDir, "mold.yaml"), []byte("name: test"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsCached(cacheDir, ref, "v1.0.0") {
		t.Error("expected cached with mold.yaml")
	}
}

func TestIsCached_Subpath(t *testing.T) {
	cacheDir := t.TempDir()
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Subpath: "sub/mold"}

	// Create version dir with manifest at subpath.
	subDir := filepath.Join(cacheDir, "github.com", "owner", "repo", "v1.0.0", "sub", "mold")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "mold.yaml"), []byte("name: test"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsCached(cacheDir, ref, "v1.0.0") {
		t.Error("expected cached at subpath")
	}
}

func TestIsCached_IngotManifest(t *testing.T) {
	cacheDir := t.TempDir()
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}

	vDir := filepath.Join(cacheDir, "github.com", "owner", "repo", "v1.0.0")
	if err := os.MkdirAll(vDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vDir, "ingot.yaml"), []byte("name: test"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsCached(cacheDir, ref, "v1.0.0") {
		t.Error("expected cached with ingot.yaml")
	}
}

func TestBareCloneDir(t *testing.T) {
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}
	got := BareCloneDir("/cache", ref)
	want := filepath.Join("/cache", "github.com", "owner", "repo", "git")
	if got != want {
		t.Errorf("BareCloneDir() = %q, want %q", got, want)
	}
}

func TestVersionDir(t *testing.T) {
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}
	got := VersionDir("/cache", ref, "v1.0.0")
	want := filepath.Join("/cache", "github.com", "owner", "repo", "v1.0.0")
	if got != want {
		t.Errorf("VersionDir() = %q, want %q", got, want)
	}
}

func TestListCachedMolds(t *testing.T) {
	cacheDir := t.TempDir()

	// Empty cache.
	entries, err := ListCachedMolds(cacheDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %d", len(entries))
	}

	// Create some cached molds.
	dirs := []string{
		filepath.Join(cacheDir, "github.com", "owner", "repo1", "v1.0.0"),
		filepath.Join(cacheDir, "github.com", "owner", "repo1", "v1.1.0"),
		filepath.Join(cacheDir, "github.com", "owner", "repo1", "git"),
		filepath.Join(cacheDir, "github.com", "other", "repo2", "v2.0.0"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatal(err)
		}
	}

	entries, err = ListCachedMolds(cacheDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Find repo1 entry.
	var repo1 *CacheEntry
	for i := range entries {
		if entries[i].Repo == "repo1" {
			repo1 = &entries[i]
			break
		}
	}
	if repo1 == nil {
		t.Fatal("expected repo1 entry")
	}
	if len(repo1.Versions) != 2 {
		t.Errorf("expected 2 versions for repo1 (git excluded), got %d", len(repo1.Versions))
	}
}

func TestListCachedMolds_NonexistentDir(t *testing.T) {
	entries, err := ListCachedMolds("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil for nonexistent dir, got %v", entries)
	}
}

func TestCleanCache(t *testing.T) {
	cacheDir := t.TempDir()
	subDir := filepath.Join(cacheDir, "github.com", "owner", "repo")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatal(err)
	}

	if err := CleanCache(cacheDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("expected cache dir to be removed")
	}
}

func TestCleanMold(t *testing.T) {
	cacheDir := t.TempDir()
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}

	moldDir := filepath.Join(cacheDir, "github.com", "owner", "repo", "v1.0.0")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}

	if err := CleanMold(cacheDir, ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cacheDir, "github.com", "owner", "repo")); !os.IsNotExist(err) {
		t.Error("expected mold dir to be removed")
	}

	// Parent dirs should still exist.
	if _, err := os.Stat(filepath.Join(cacheDir, "github.com", "owner")); err != nil {
		t.Error("expected parent dir to still exist")
	}
}
