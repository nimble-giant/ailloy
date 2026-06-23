package smelt

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// fakeSmeltFetcher is a test double for smeltDepFetcher.
// Fetch populates an in-memory cache keyed on (source, subpath); CacheEntry
// reads it back so collectDepsWith can retrieve the FS after Build.
//
// Subdir deps (source//subpath) are registered via addSubpathDep and take
// priority over source-only entries registered via addDep.
type fakeSmeltFetcher struct {
	molds map[string]*mold.Mold // CacheKey -> mold (no subpath)
	fss   map[string]fs.FS      // CacheKey -> mold FS (no subpath)

	// subDeps supports (source, subpath) keyed deps for monorepo / subdir molds.
	// Key format: "source//subpath"
	subDeps map[string]*mold.Mold
	subFss  map[string]fs.FS
	// subTags holds the tag→TagInfo map for each (source, subpath).
	// When set, Tags returns exactly these entries instead of the default v1.0.0.
	subTags map[string]map[string]depgraph.TagInfo

	cache map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry
}

func newFakeSmeltFetcher() *fakeSmeltFetcher {
	return &fakeSmeltFetcher{
		molds:   map[string]*mold.Mold{},
		fss:     map[string]fs.FS{},
		subDeps: map[string]*mold.Mold{},
		subFss:  map[string]fs.FS{},
		subTags: map[string]map[string]depgraph.TagInfo{},
		cache:   map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
}

func (f *fakeSmeltFetcher) addDep(source string, m *mold.Mold, fsys fs.FS) {
	f.molds[source] = m
	f.fss[source] = fsys
}

// addSubpathDep registers a (source, subpath) mold with its own tag map.
// tags maps git tag names to TagInfo (SHA + MoldVersion). The Fetch call for
// this dep will return the highest tag from the map as the resolved version.
func (f *fakeSmeltFetcher) addSubpathDep(source, subpath string, m *mold.Mold, fsys fs.FS, tags map[string]depgraph.TagInfo) {
	k := source + "//" + subpath
	f.subDeps[k] = m
	f.subFss[k] = fsys
	f.subTags[k] = tags
}

func (f *fakeSmeltFetcher) Fetch(ref *foundry.Reference) (depgraph.FetchResult, error) {
	cacheKey := ref.CacheKey()
	nk := depgraph.NodeKey{Source: cacheKey, Subpath: ref.Subpath}

	// Prefer (source, subpath) specific entry.
	if ref.Subpath != "" {
		subKey := cacheKey + "//" + ref.Subpath
		if m, ok := f.subDeps[subKey]; ok {
			// Pick the first (and usually only) tag from the registered map.
			version, sha := "v0.1.0", "abc123"
			for t, info := range f.subTags[subKey] {
				version = t
				sha = info.SHA
				break
			}
			f.cache[nk] = &depgraph.ProdFetchCacheEntry{
				FS:   f.subFss[subKey],
				Mold: m,
			}
			return depgraph.FetchResult{Mold: m, Version: version, Commit: sha}, nil
		}
	}

	m, ok := f.molds[cacheKey]
	if !ok {
		return depgraph.FetchResult{}, fmt.Errorf("fake: no mold for %s", cacheKey)
	}
	f.cache[nk] = &depgraph.ProdFetchCacheEntry{
		FS:   f.fss[cacheKey],
		Mold: m,
	}
	return depgraph.FetchResult{Mold: m, Version: "v1.0.0", Commit: "abc123"}, nil
}

func (f *fakeSmeltFetcher) Tags(source, subpath string) (map[string]depgraph.TagInfo, error) {
	// Prefer (source, subpath) specific tags.
	if subpath != "" {
		subKey := source + "//" + subpath
		if tags, ok := f.subTags[subKey]; ok {
			return tags, nil
		}
	}
	if _, ok := f.molds[source]; ok {
		return map[string]depgraph.TagInfo{"v1.0.0": {SHA: "abc123"}}, nil
	}
	return map[string]depgraph.TagInfo{}, nil
}

func (f *fakeSmeltFetcher) CacheEntry(key depgraph.NodeKey) *depgraph.ProdFetchCacheEntry {
	return f.cache[key]
}

// noopArtifactResolver never resolves anything (used for molds with no ore/ingot deps).
func noopArtifactResolver(_ string, _ ...foundry.ResolveOption) (fs.FS, *foundry.ResolveResult, error) {
	return nil, nil, fmt.Errorf("unexpected artifact resolution in test")
}

// fakeDepMold builds a minimal mold.Mold with mold-kind deps.
func fakeDepMold(t *testing.T, name string, moldDeps ...string) *mold.Mold {
	t.Helper()
	m := &mold.Mold{Name: name, Version: "1.0.0", Kind: "mold", APIVersion: "v1"}
	for _, dep := range moldDeps {
		m.Dependencies = append(m.Dependencies, mold.Dependency{Mold: dep})
	}
	return m
}

// fakeDepFS builds a minimal in-memory mold FS with just a mold.yaml.
func fakeDepFS(name string) fs.FS {
	return fstest.MapFS{
		"mold.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: mold\nname: " + name + "\nversion: 1.0.0\n"),
		},
		"commands/foo.md": &fstest.MapFile{Data: []byte("# " + name)},
	}
}

func TestHasMoldDeps(t *testing.T) {
	tests := []struct {
		name string
		m    *mold.Mold
		want bool
	}{
		{"nil mold", nil, false},
		{"no deps", &mold.Mold{}, false},
		{"ore only", &mold.Mold{Dependencies: []mold.Dependency{{Ore: "github.com/a/b"}}}, false},
		{"ingot only", &mold.Mold{Dependencies: []mold.Dependency{{Ingot: "github.com/a/b"}}}, false},
		{"mold dep", &mold.Mold{Dependencies: []mold.Dependency{{Mold: "github.com/a/b"}}}, true},
		{"mixed", &mold.Mold{Dependencies: []mold.Dependency{
			{Ore: "github.com/a/ore"},
			{Mold: "github.com/a/b"},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasMoldDeps(tt.m); got != tt.want {
				t.Errorf("hasMoldDeps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDepFSPath(t *testing.T) {
	tests := []struct {
		kind, source, subpath, want string
	}{
		{"molds", "github.com/a/b", "", "deps/molds/github.com/a/b"},
		{"molds", "github.com/a/b", "sub/path", "deps/molds/github.com/a/b/sub/path"},
		{"molds", "github.com/a/b", "/sub/", "deps/molds/github.com/a/b/sub"},
		{"ores", "github.com/a/ore", "", "deps/ores/github.com/a/ore"},
		{"ingots", "github.com/a/ing", "pkg", "deps/ingots/github.com/a/ing/pkg"},
	}
	for _, tt := range tests {
		t.Run(tt.kind+"/"+tt.source+"/"+tt.subpath, func(t *testing.T) {
			got := depFSPath(tt.kind, tt.source, tt.subpath)
			if got != tt.want {
				t.Errorf("depFSPath(%q, %q, %q) = %q, want %q", tt.kind, tt.source, tt.subpath, got, tt.want)
			}
		})
	}
}

func TestWalkIntoFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("content1")},
		"commands/hello.md": &fstest.MapFile{Data: []byte("content2")},
	}

	files, err := walkIntoFiles(fsys, "deps/molds/github.com/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	pathSet := map[string]string{}
	for _, f := range files {
		pathSet[f.path] = string(f.data)
	}

	if v, ok := pathSet["deps/molds/github.com/a/b/mold.yaml"]; !ok || v != "content1" {
		t.Errorf("missing or wrong mold.yaml: %v", pathSet)
	}
	if v, ok := pathSet["deps/molds/github.com/a/b/commands/hello.md"]; !ok || v != "content2" {
		t.Errorf("missing or wrong commands/hello.md: %v", pathSet)
	}
}

func TestMarshalDepManifest(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		if marshalDepManifest(nil) != nil {
			t.Error("expected nil for nil manifest")
		}
	})

	t.Run("empty manifest", func(t *testing.T) {
		m := &DepManifest{Molds: []DepEntry{}, Ores: []DepEntry{}, Ingots: []DepEntry{}}
		if marshalDepManifest(m) != nil {
			t.Error("expected nil for empty manifest")
		}
	})

	t.Run("non-empty manifest", func(t *testing.T) {
		m := &DepManifest{
			Molds: []DepEntry{{Source: "github.com/a/b", Version: "v1.0.0", Commit: "abc"}},
		}
		data := marshalDepManifest(m)
		if data == nil {
			t.Fatal("expected non-nil JSON")
		}
		if !strings.Contains(string(data), "github.com/a/b") {
			t.Errorf("expected source in JSON: %s", data)
		}
	})
}

func TestCollectDepsWith_LeafMold(t *testing.T) {
	// A mold with no deps should produce empty files + empty manifest.
	m := &mold.Mold{Name: "leaf", Version: "1.0.0", Kind: "mold", APIVersion: "v1"}

	fetcher := newFakeSmeltFetcher()
	files, manifest, err := collectDepsWith(m, nil, fetcher, noopArtifactResolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files for leaf mold, got %d: %v", len(files), files)
	}
	if len(manifest.Molds) != 0 || len(manifest.Ores) != 0 || len(manifest.Ingots) != 0 {
		t.Errorf("expected empty manifest for leaf mold, got %+v", manifest)
	}
}

func TestCollectDepsWith_MoldDep(t *testing.T) {
	// Root mold declares one mold dep. The transitive node's FS should be
	// embedded under deps/molds/<source>/.
	depSource := "github.com/acme/dep-a"
	depMold := fakeDepMold(t, "dep-a")
	depFS := fakeDepFS("dep-a")

	fetcher := newFakeSmeltFetcher()
	fetcher.addDep(depSource, depMold, depFS)

	root := fakeDepMold(t, "root", depSource)

	files, manifest, err := collectDepsWith(root, nil, fetcher, noopArtifactResolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Molds) != 1 {
		t.Fatalf("expected 1 mold in manifest, got %d", len(manifest.Molds))
	}
	if manifest.Molds[0].Source != depSource {
		t.Errorf("manifest source = %q, want %q", manifest.Molds[0].Source, depSource)
	}

	// Verify dep files are prefixed under deps/molds/<source>/.
	wantPrefix := "deps/molds/" + depSource + "/"
	var found []string
	for _, f := range files {
		if strings.HasPrefix(f.path, wantPrefix) {
			found = append(found, f.path)
		}
	}
	if len(found) == 0 {
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.path
		}
		t.Errorf("no files with prefix %q; got %v", wantPrefix, paths)
	}
}

func TestCollectDepsWith_OreAndIngotDeps(t *testing.T) {
	// Root mold declares ore + ingot deps but no mold deps.
	// The artifact resolver is called for each remote dep.
	oreSource := "github.com/acme/my-ore"
	ingotSource := "github.com/acme/my-ingot"

	oreFS := fstest.MapFS{
		"ore.yaml": &fstest.MapFile{Data: []byte("kind: ore\nname: my-ore\n")},
	}
	ingotFS := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte("kind: ingot\nname: my-ingot\n")},
	}

	resolver := func(rawRef string, _ ...foundry.ResolveOption) (fs.FS, *foundry.ResolveResult, error) {
		ref, _ := foundry.ParseReference(rawRef)
		var fsys fs.FS
		switch ref.CacheKey() {
		case oreSource:
			fsys = oreFS
		case ingotSource:
			fsys = ingotFS
		default:
			return nil, nil, fmt.Errorf("unexpected ref %s", rawRef)
		}
		return fsys, &foundry.ResolveResult{
			Ref:      ref,
			Resolved: foundry.ResolvedVersion{Tag: "v0.1.0", Commit: "sha1"},
		}, nil
	}

	root := &mold.Mold{
		Name:       "root",
		Version:    "1.0.0",
		Kind:       "mold",
		APIVersion: "v1",
		Dependencies: []mold.Dependency{
			{Ore: oreSource},
			{Ingot: ingotSource},
		},
	}

	fetcher := newFakeSmeltFetcher()
	files, manifest, err := collectDepsWith(root, nil, fetcher, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Ores) != 1 || manifest.Ores[0].Source != oreSource {
		t.Errorf("ores manifest: %+v", manifest.Ores)
	}
	if len(manifest.Ingots) != 1 || manifest.Ingots[0].Source != ingotSource {
		t.Errorf("ingots manifest: %+v", manifest.Ingots)
	}

	// Both ore and ingot files should appear under deps/ores/ and deps/ingots/.
	var oreFiles, ingotFiles []string
	for _, f := range files {
		switch {
		case strings.HasPrefix(f.path, "deps/ores/"):
			oreFiles = append(oreFiles, f.path)
		case strings.HasPrefix(f.path, "deps/ingots/"):
			ingotFiles = append(ingotFiles, f.path)
		}
	}
	if len(oreFiles) == 0 {
		t.Errorf("no ore files embedded; files: %v", fileNames(files))
	}
	if len(ingotFiles) == 0 {
		t.Errorf("no ingot files embedded; files: %v", fileNames(files))
	}
}

func TestCollectDepsWith_DuplicateOreDeduped(t *testing.T) {
	// Two mold deps both declare the same ore; it should be embedded only once.
	oreSource := "github.com/acme/shared-ore"

	callCount := 0
	resolver := func(rawRef string, _ ...foundry.ResolveOption) (fs.FS, *foundry.ResolveResult, error) {
		callCount++
		ref, _ := foundry.ParseReference(rawRef)
		fsys := fstest.MapFS{"ore.yaml": &fstest.MapFile{Data: []byte("kind: ore\n")}}
		return fsys, &foundry.ResolveResult{
			Ref:      ref,
			Resolved: foundry.ResolvedVersion{Tag: "v1.0.0"},
		}, nil
	}

	depMoldA := &mold.Mold{
		Name: "dep-a", Version: "1.0.0", Kind: "mold", APIVersion: "v1",
		Dependencies: []mold.Dependency{{Ore: oreSource}},
	}
	depMoldB := &mold.Mold{
		Name: "dep-b", Version: "1.0.0", Kind: "mold", APIVersion: "v1",
		Dependencies: []mold.Dependency{{Ore: oreSource}},
	}

	fetcher := newFakeSmeltFetcher()
	fetcher.addDep("github.com/acme/dep-a", depMoldA, fakeDepFS("dep-a"))
	fetcher.addDep("github.com/acme/dep-b", depMoldB, fakeDepFS("dep-b"))

	root := fakeDepMold(t, "root", "github.com/acme/dep-a", "github.com/acme/dep-b")

	_, manifest, err := collectDepsWith(root, nil, fetcher, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("ore resolved %d times, want 1 (should be deduped)", callCount)
	}
	if len(manifest.Ores) != 1 {
		t.Errorf("expected 1 ore entry in manifest, got %d", len(manifest.Ores))
	}
}

func TestPackageBinary_LeafMoldNoDepsSubtree(t *testing.T) {
	// A leaf mold (no deps) must NOT produce a /deps/ subtree or manifest.json.
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fsys, err := UnstuffFS(outputPath)
	if err != nil {
		t.Fatalf("unstuffing: %v", err)
	}

	// deps/manifest.json must not exist for a leaf mold.
	if _, err := fs.Stat(fsys, "deps/manifest.json"); err == nil {
		t.Error("leaf mold binary should not contain deps/manifest.json")
	}
}

func TestDepManifestJSON(t *testing.T) {
	// Verify the JSON structure of a DepManifest round-trips correctly.
	m := &DepManifest{
		Molds:  []DepEntry{{Source: "github.com/a/b", Version: "v1.2.3", Commit: "abc"}},
		Ores:   []DepEntry{{Source: "github.com/a/ore", Version: "v0.5.0"}},
		Ingots: []DepEntry{{Source: "github.com/a/ing", Subpath: "pkg", Version: "v2.0.0"}},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got DepManifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Molds) != 1 || got.Molds[0].Source != "github.com/a/b" {
		t.Errorf("molds round-trip failed: %+v", got.Molds)
	}
	if len(got.Ores) != 1 || got.Ores[0].Source != "github.com/a/ore" {
		t.Errorf("ores round-trip failed: %+v", got.Ores)
	}
	if len(got.Ingots) != 1 || got.Ingots[0].Subpath != "pkg" {
		t.Errorf("ingots round-trip failed: %+v", got.Ingots)
	}
}

// TestCollectDepsWith_SubdirMoldDepsWithPrefixedTags mirrors issue #255:
// a monorepo with molds under molds/<name>, each released under per-mold tag
// prefixes (e.g. vendor-ops-v1.1.6) where the tag version differs from the
// mold's internal version: field. The aggregator uses ^ constraints that must
// be resolved against the internal mold version, not the tag-embedded version.
func TestCollectDepsWith_SubdirMoldDepsWithPrefixedTags(t *testing.T) {
	source := "github.com/org/repo"

	type subDep struct {
		subpath    string
		tag        string // e.g. "vendor-ops-v1.1.6"
		moldVer    string // internal mold.yaml version: e.g. "0.1.0"
		sha        string
		constraint string // aggregator's version: constraint
	}
	deps := []subDep{
		{"molds/vendor-ops", "vendor-ops-v1.1.6", "0.1.0", "cafe0001", "^0.1.0"},
		{"molds/replicated-cli", "replicated-cli-v0.3.6", "0.1.0", "cafe0002", "^0.1.0"},
		{"molds/launch", "launch-v0.9.0", "0.2.1", "cafe0003", "^0.2.0"},
	}

	fetcher := newFakeSmeltFetcher()

	for _, d := range deps {
		name := d.subpath[len("molds/"):]
		m := &mold.Mold{Name: name, Version: d.moldVer, Kind: "mold", APIVersion: "v1"}
		fsys := fakeDepFS(name)
		tags := map[string]depgraph.TagInfo{
			d.tag: {SHA: d.sha, MoldVersion: d.moldVer},
		}
		fetcher.addSubpathDep(source, d.subpath, m, fsys, tags)
	}

	var moldDeps []mold.Dependency
	for _, d := range deps {
		moldDeps = append(moldDeps, mold.Dependency{
			Mold:    source + "//" + d.subpath,
			Version: d.constraint,
		})
	}
	root := &mold.Mold{
		Name: "aggregator", Version: "0.1.0", Kind: "mold", APIVersion: "v1",
		Dependencies: moldDeps,
	}

	files, manifest, err := collectDepsWith(root, nil, fetcher, noopArtifactResolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Molds) != 3 {
		t.Fatalf("expected 3 molds in manifest, got %d: %+v", len(manifest.Molds), manifest.Molds)
	}

	// Build a manifest index keyed on subpath for easy lookup.
	bySubpath := map[string]DepEntry{}
	for _, me := range manifest.Molds {
		bySubpath[me.Subpath] = me
	}

	for _, d := range deps {
		me, ok := bySubpath[d.subpath]
		if !ok {
			t.Errorf("manifest missing entry for subpath %q", d.subpath)
			continue
		}
		if me.Source != source {
			t.Errorf("[%s] Source = %q, want %q", d.subpath, me.Source, source)
		}
		if me.Version != d.tag {
			t.Errorf("[%s] Version = %q, want %q", d.subpath, me.Version, d.tag)
		}
		if me.Commit != d.sha {
			t.Errorf("[%s] Commit = %q, want %q", d.subpath, me.Commit, d.sha)
		}
		if me.MoldVersion != d.moldVer {
			t.Errorf("[%s] MoldVersion = %q, want %q", d.subpath, me.MoldVersion, d.moldVer)
		}

		// Files must be embedded under deps/molds/<source>/<subpath>/.
		wantPrefix := "deps/molds/" + source + "/" + d.subpath + "/"
		var found []string
		for _, f := range files {
			if strings.HasPrefix(f.path, wantPrefix) {
				found = append(found, f.path)
			}
		}
		if len(found) == 0 {
			t.Errorf("[%s] no files embedded under %q; all paths: %v",
				d.subpath, wantPrefix, fileNames(files))
		}
	}
}

// TestEmbeddedDepFetcher_Tags_MoldVersion verifies that Tags returns the
// MoldVersion from the manifest for embedded subdir deps with prefixed tags.
// This is critical for release-train monorepos where constraints like ^0.1.0
// must be checked against the mold's own version (0.1.0), not the tag-embedded
// version (1.1.6 from "vendor-ops-v1.1.6").
func TestEmbeddedDepFetcher_Tags_MoldVersion(t *testing.T) {
	source := "github.com/org/repo"
	subpath := "molds/vendor-ops"
	embFS := fstest.MapFS{
		"deps/manifest.json": &fstest.MapFile{
			Data: makeManifestJSON(t, DepManifest{
				Molds: []DepEntry{{
					Source:      source,
					Subpath:     subpath,
					Version:     "vendor-ops-v1.1.6",
					Commit:      "deadbeef",
					MoldVersion: "0.1.0",
				}},
			}),
		},
		"deps/molds/" + source + "/" + subpath + "/mold.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: mold\nname: vendor-ops\nversion: 0.1.0\n"),
		},
	}

	fetcher := buildEmbeddedFetcher(t, embFS)
	tags, err := fetcher.Tags(source, subpath)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}

	info, ok := tags["vendor-ops-v1.1.6"]
	if !ok {
		t.Fatalf("expected tag vendor-ops-v1.1.6 in %v", tags)
	}
	if info.MoldVersion != "0.1.0" {
		t.Errorf("MoldVersion = %q, want 0.1.0", info.MoldVersion)
	}
	if info.SHA != "deadbeef" {
		t.Errorf("SHA = %q, want deadbeef", info.SHA)
	}
}

// fileNames extracts just the paths from a slice of archiveFiles (test helper).
func fileNames(files []archiveFile) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.path
	}
	return names
}
