package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// writeCachedIndex writes a foundry index YAML to the cache location for entry.
func writeCachedIndex(t *testing.T, cacheDir string, entry index.FoundryEntry, body string) {
	t.Helper()
	path := index.CachedIndexPath(cacheDir, &entry)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadCatalog_FlattensNestedFoundries verifies that the Discover catalog
// includes molds from foundries reached via the registered root's
// `foundries:` field, and tags them with their resolution chain.
func TestLoadCatalog_FlattensNestedFoundries(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", cacheDir)

	// Root foundry references one nested foundry by source URL.
	root := index.FoundryEntry{Name: "root", URL: "https://github.com/example/root", Type: "git"}
	writeCachedIndex(t, cacheDir, root, `apiVersion: v1
kind: foundry-index
name: root
molds:
  - name: alpha
    source: github.com/example/alpha
foundries:
  - name: child
    source: https://github.com/example/child
`)

	// Nested foundry — must be cached separately under its own URL hash.
	child := index.FoundryEntry{URL: "https://github.com/example/child", Type: "git"}
	writeCachedIndex(t, cacheDir, child, `apiVersion: v1
kind: foundry-index
name: child
molds:
  - name: beta
    source: github.com/example/beta
`)

	cfg := &index.Config{Foundries: []index.FoundryEntry{root}}
	catalog, err := LoadCatalog(cfg)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("len(catalog) = %d want 2; got %+v", len(catalog), catalog)
	}

	byName := map[string]CatalogEntry{}
	for _, e := range catalog {
		byName[e.Name] = e
	}

	alpha, ok := byName["alpha"]
	if !ok {
		t.Fatal("expected alpha (root mold) in catalog")
	}
	if alpha.IsNested() {
		t.Errorf("root mold reported nested: chain=%v", alpha.OwnerChain)
	}
	if alpha.FoundryName != "root" {
		t.Errorf("alpha.FoundryName = %q want root", alpha.FoundryName)
	}

	beta, ok := byName["beta"]
	if !ok {
		t.Fatal("expected beta (nested mold) in catalog")
	}
	if !beta.IsNested() {
		t.Fatalf("nested mold reported as root; chain=%v", beta.OwnerChain)
	}
	if beta.FoundryName != "root" {
		t.Errorf("beta.FoundryName = %q want root (registered root foundry)", beta.FoundryName)
	}
	wantChain := []string{"root", "child"}
	if len(beta.OwnerChain) != 2 || beta.OwnerChain[0] != wantChain[0] || beta.OwnerChain[1] != wantChain[1] {
		t.Errorf("beta.OwnerChain = %v want %v", beta.OwnerChain, wantChain)
	}
}

// TestLoadCatalog_UncachedNestedFoundryIsSkipped verifies that an uncached
// child foundry doesn't break the render — the root and its direct molds
// still surface; the missing child is silently skipped (Resolver records a
// warning we drop).
func TestLoadCatalog_UncachedNestedFoundryIsSkipped(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", cacheDir)

	root := index.FoundryEntry{Name: "root", URL: "https://github.com/example/root", Type: "git"}
	writeCachedIndex(t, cacheDir, root, `apiVersion: v1
kind: foundry-index
name: root
molds:
  - name: alpha
    source: github.com/example/alpha
foundries:
  - name: missing
    source: https://github.com/example/not-cached
`)

	cfg := &index.Config{Foundries: []index.FoundryEntry{root}}
	catalog, err := LoadCatalog(cfg)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(catalog) != 1 || catalog[0].Name != "alpha" {
		t.Fatalf("expected only alpha in catalog, got %+v", catalog)
	}
}

// TestLoadCatalog_DedupesNestedMoldSharedAcrossRoots verifies that when two
// registered root foundries both transitively pull in the same nested mold
// (by source URL), it appears only once in the catalog rather than as a
// duplicate row.
func TestLoadCatalog_DedupesNestedMoldSharedAcrossRoots(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", cacheDir)

	// Two registered roots both nest the same child foundry.
	rootA := index.FoundryEntry{Name: "rootA", URL: "https://github.com/example/rootA", Type: "git"}
	rootB := index.FoundryEntry{Name: "rootB", URL: "https://github.com/example/rootB", Type: "git"}
	writeCachedIndex(t, cacheDir, rootA, `apiVersion: v1
kind: foundry-index
name: rootA
foundries:
  - name: shared
    source: https://github.com/example/shared
`)
	writeCachedIndex(t, cacheDir, rootB, `apiVersion: v1
kind: foundry-index
name: rootB
foundries:
  - name: shared
    source: https://github.com/example/shared
`)
	shared := index.FoundryEntry{URL: "https://github.com/example/shared", Type: "git"}
	writeCachedIndex(t, cacheDir, shared, `apiVersion: v1
kind: foundry-index
name: shared
molds:
  - name: gamma
    source: github.com/example/gamma
`)

	cfg := &index.Config{Foundries: []index.FoundryEntry{rootA, rootB}}
	catalog, err := LoadCatalog(cfg)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	count := 0
	for _, e := range catalog {
		if e.Name == "gamma" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected gamma to appear once, got %d times", count)
	}
}
