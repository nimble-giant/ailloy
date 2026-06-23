package smelt

import (
	"encoding/json"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
)

// buildEmbeddedFetcher constructs an EmbeddedDepFetcher from an in-memory FS
// that mimics the embedded binary's dep subtree, without touching the real
// executable. The fallback is a stub that always fails so tests can verify
// embedded-path behaviour in isolation.
func buildEmbeddedFetcher(t *testing.T, embFS fs.FS) *EmbeddedDepFetcher {
	t.Helper()
	var manifest DepManifest
	if data, err := fs.ReadFile(embFS, "deps/manifest.json"); err == nil {
		if jerr := json.Unmarshal(data, &manifest); jerr != nil {
			t.Fatalf("parse manifest: %v", jerr)
		}
	}
	return &EmbeddedDepFetcher{
		embFS:    embFS,
		manifest: &manifest,
		fallback: depgraph.NewProdFetcher(),
		cache:    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
}

// buildEmbeddedFetcherWithFallback is like buildEmbeddedFetcher but accepts a
// custom fallback so tests can observe fallback invocations.
func buildEmbeddedFetcherWithFallback(t *testing.T, embFS fs.FS, fb *depgraph.ProdFetcher) *EmbeddedDepFetcher {
	t.Helper()
	var manifest DepManifest
	if data, err := fs.ReadFile(embFS, "deps/manifest.json"); err == nil {
		if jerr := json.Unmarshal(data, &manifest); jerr != nil {
			t.Fatalf("parse manifest: %v", jerr)
		}
	}
	return &EmbeddedDepFetcher{
		embFS:    embFS,
		manifest: &manifest,
		fallback: fb,
		cache:    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
}

// makeManifestJSON serialises a DepManifest for use in fstest.MapFS.
func makeManifestJSON(t *testing.T, m DepManifest) []byte {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return data
}

// TestEmbeddedDepFetcher_Fetch_fromEmbedded verifies that Fetch returns mold
// data from the embedded FS when the manifest lists the dep.
func TestEmbeddedDepFetcher_Fetch_fromEmbedded(t *testing.T) {
	source := "github.com/acme/dep-a"
	embFS := fstest.MapFS{
		"deps/manifest.json": &fstest.MapFile{
			Data: makeManifestJSON(t, DepManifest{
				Molds: []DepEntry{{Source: source, Version: "v1.2.3", Commit: "abc123"}},
			}),
		},
		"deps/molds/" + source + "/mold.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: mold\nname: dep-a\nversion: 1.2.3\n"),
		},
	}

	fetcher := buildEmbeddedFetcher(t, embFS)

	ref := &foundry.Reference{Host: "github.com", Owner: "acme", Repo: "dep-a"}
	result, err := fetcher.Fetch(ref)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if result.Version != "v1.2.3" {
		t.Errorf("Version = %q, want v1.2.3", result.Version)
	}
	if result.Commit != "abc123" {
		t.Errorf("Commit = %q, want abc123", result.Commit)
	}
	if result.Mold == nil {
		t.Fatal("Mold is nil")
	}

	// CacheEntry must be populated after Fetch.
	key := depgraph.NodeKey{Source: ref.CacheKey()}
	ce := fetcher.CacheEntry(key)
	if ce == nil {
		t.Fatal("CacheEntry is nil after Fetch")
	}
	if ce.FS == nil {
		t.Error("CacheEntry.FS is nil")
	}
	if ce.Resolved.Tag != "v1.2.3" {
		t.Errorf("CacheEntry.Resolved.Tag = %q, want v1.2.3", ce.Resolved.Tag)
	}
}

// TestEmbeddedDepFetcher_Fetch_subpath verifies subpath molds are resolved from
// the correct embedded path.
func TestEmbeddedDepFetcher_Fetch_subpath(t *testing.T) {
	source := "github.com/acme/mono"
	subpath := "sub/mold"
	embFS := fstest.MapFS{
		"deps/manifest.json": &fstest.MapFile{
			Data: makeManifestJSON(t, DepManifest{
				Molds: []DepEntry{{Source: source, Subpath: subpath, Version: "v2.0.0"}},
			}),
		},
		"deps/molds/" + source + "/" + subpath + "/mold.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: mold\nname: sub-mold\nversion: 2.0.0\n"),
		},
	}

	fetcher := buildEmbeddedFetcher(t, embFS)

	ref := &foundry.Reference{Host: "github.com", Owner: "acme", Repo: "mono", Subpath: subpath}
	result, err := fetcher.Fetch(ref)
	if err != nil {
		t.Fatalf("Fetch subpath: %v", err)
	}
	if result.Version != "v2.0.0" {
		t.Errorf("Version = %q, want v2.0.0", result.Version)
	}
}

// TestEmbeddedDepFetcher_Tags_embedded verifies Tags returns the single pinned
// version when the dep is in the embedded manifest.
func TestEmbeddedDepFetcher_Tags_embedded(t *testing.T) {
	source := "github.com/acme/dep-a"
	fetcher := &EmbeddedDepFetcher{
		manifest: &DepManifest{
			Molds: []DepEntry{{Source: source, Version: "v3.1.0", Commit: "deadbeef"}},
		},
		fallback: depgraph.NewProdFetcher(),
		cache:    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}

	tags, err := fetcher.Tags(source, "")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	info, ok := tags["v3.1.0"]
	if !ok {
		t.Fatalf("expected tag v3.1.0 in %v", tags)
	}
	if info.SHA != "deadbeef" {
		t.Errorf("SHA = %q, want deadbeef", info.SHA)
	}
	if len(tags) != 1 {
		t.Errorf("expected exactly 1 tag, got %d", len(tags))
	}
}

// TestEmbeddedDepFetcher_NoManifest verifies that an embedded FS without a
// deps/manifest.json (leaf mold) creates a fetcher with an empty manifest.
func TestEmbeddedDepFetcher_NoManifest(t *testing.T) {
	embFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("kind: mold\n")},
	}
	f := &EmbeddedDepFetcher{
		embFS:    embFS,
		manifest: &DepManifest{},
		fallback: depgraph.NewProdFetcher(),
		cache:    map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
	// Tags for an unlisted dep should not panic; it falls through to the
	// ProdFetcher which will fail (no network in tests), but that's a
	// separate concern. We just verify it doesn't panic on the embedded path.
	if entry := f.moldEntry("github.com/x/y", ""); entry != nil {
		t.Error("expected nil entry for unlisted dep")
	}
}

// TestLookupEmbeddedArtifact_NotSmelted verifies LookupEmbeddedArtifact
// returns ok=false when the binary has no embedded mold (the normal case for
// dev builds running without stuffbin).
func TestLookupEmbeddedArtifact_NotSmelted(t *testing.T) {
	_, _, _, ok := LookupEmbeddedArtifact("github.com/x/ore", "")
	if ok {
		t.Error("expected ok=false for non-smelted binary")
	}
}

// TestEmbeddedDepFetcher_CacheEntry_missingKey verifies CacheEntry returns nil
// for an unknown key without panicking.
func TestEmbeddedDepFetcher_CacheEntry_missingKey(t *testing.T) {
	f := &EmbeddedDepFetcher{
		cache: map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
	if ce := f.CacheEntry(depgraph.NodeKey{Source: "github.com/x/y"}); ce != nil {
		t.Errorf("expected nil, got %v", ce)
	}
}

// TestEmbeddedDepFetcher_Fetch_missingMoldYAML verifies that Fetch returns an
// error when the mold is listed in the manifest but its mold.yaml is absent
// from the embedded FS.
func TestEmbeddedDepFetcher_Fetch_missingMoldYAML(t *testing.T) {
	source := "github.com/acme/bad"
	// Manifest lists the dep but the mold.yaml is not present in the FS.
	embFS := fstest.MapFS{
		"deps/manifest.json": &fstest.MapFile{
			Data: makeManifestJSON(t, DepManifest{
				Molds: []DepEntry{{Source: source, Version: "v1.0.0"}},
			}),
		},
		// No mold.yaml under deps/molds/github.com/acme/bad/
	}

	fetcher := buildEmbeddedFetcher(t, embFS)
	ref := &foundry.Reference{Host: "github.com", Owner: "acme", Repo: "bad"}
	_, err := fetcher.Fetch(ref)
	if err == nil {
		t.Error("expected error for missing mold.yaml, got nil")
	}
}

// TestEmbeddedDepFetcher_CastableFromFS exercises the full embedded mold cast
// path in isolation: Fetch sets up the cache entry so the mold reader can load
// files from it.
func TestEmbeddedDepFetcher_CastableFromFS(t *testing.T) {
	source := "github.com/acme/castable"
	embFS := fstest.MapFS{
		"deps/manifest.json": &fstest.MapFile{
			Data: makeManifestJSON(t, DepManifest{
				Molds: []DepEntry{{Source: source, Version: "v0.1.0", Commit: "cafe"}},
			}),
		},
		"deps/molds/" + source + "/mold.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: mold\nname: castable\nversion: 0.1.0\n"),
		},
		"deps/molds/" + source + "/commands/hello.md": &fstest.MapFile{
			Data: []byte("# hello"),
		},
	}

	fetcher := buildEmbeddedFetcher(t, embFS)
	ref := &foundry.Reference{Host: "github.com", Owner: "acme", Repo: "castable"}
	if _, err := fetcher.Fetch(ref); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	key := depgraph.NodeKey{Source: ref.CacheKey()}
	ce := fetcher.CacheEntry(key)
	if ce == nil {
		t.Fatal("cache entry is nil")
	}

	// Verify the sub-FS is rooted at the mold directory (mold.yaml accessible).
	if _, err := fs.Stat(ce.FS, "mold.yaml"); err != nil {
		t.Errorf("mold.yaml not accessible from cached FS: %v", err)
	}
	if _, err := fs.Stat(ce.FS, "commands/hello.md"); err != nil {
		t.Errorf("commands/hello.md not accessible from cached FS: %v", err)
	}
	if ce.Mold == nil {
		t.Error("cached Mold is nil")
	}
}
