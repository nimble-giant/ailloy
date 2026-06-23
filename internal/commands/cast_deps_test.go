package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/depgraph"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestHasMoldDeps(t *testing.T) {
	t.Run("nil mold", func(t *testing.T) {
		if hasMoldDeps(nil) {
			t.Error("nil mold must return false")
		}
	})
	t.Run("no deps", func(t *testing.T) {
		m := &mold.Mold{Dependencies: nil}
		if hasMoldDeps(m) {
			t.Error("no-deps mold must return false")
		}
	})
	t.Run("only ingot/ore deps", func(t *testing.T) {
		m := &mold.Mold{Dependencies: []mold.Dependency{
			{Ingot: "x", Version: "^1.0.0"},
			{Ore: "y", Version: "^1.0.0"},
		}}
		if hasMoldDeps(m) {
			t.Error("ingot/ore-only mold must return false")
		}
	})
	t.Run("with mold dep", func(t *testing.T) {
		m := &mold.Mold{Dependencies: []mold.Dependency{
			{Ingot: "x", Version: "^1.0.0"},
			{Mold: "github.com/x/y", Version: "^1.0.0"},
		}}
		if !hasMoldDeps(m) {
			t.Error("mold with kind=mold dep must return true")
		}
	})
}

func TestDepAlias(t *testing.T) {
	t.Run("from parent As", func(t *testing.T) {
		node := &depgraph.Node{
			Parents: []depgraph.ParentEdge{{As: "parent-alias"}},
		}
		manifest := &mold.Mold{Name: "fallback"}
		if got := depAlias(node, manifest); got != "parent-alias" {
			t.Errorf("got %q; want parent-alias", got)
		}
	})
	t.Run("falls back to mold name", func(t *testing.T) {
		node := &depgraph.Node{Parents: []depgraph.ParentEdge{{As: ""}}}
		manifest := &mold.Mold{Name: "leaf"}
		if got := depAlias(node, manifest); got != "leaf" {
			t.Errorf("got %q; want leaf", got)
		}
	})
}

// fakeDepFetcher implements depFetcher backed by an in-memory fs.FS per node.
// It mirrors depgraph.ProdFetcher's contract: Fetch + Tags + CacheEntry.
type fakeDepFetcher struct {
	molds map[string]map[string]*moldFixture // sourceKey -> version -> fixture
	tags  map[string]map[string]string       // sourceKey -> tag -> sha
	cache map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry
}

type moldFixture struct {
	mold *mold.Mold
	fs   fs.FS
	root string // on-disk root; empty for in-memory
}

func newFakeDepFetcher() *fakeDepFetcher {
	return &fakeDepFetcher{
		molds: map[string]map[string]*moldFixture{},
		tags:  map[string]map[string]string{},
		cache: map[depgraph.NodeKey]*depgraph.ProdFetchCacheEntry{},
	}
}

func (f *fakeDepFetcher) addMold(source, version string, fixture *moldFixture) {
	if f.molds[source] == nil {
		f.molds[source] = map[string]*moldFixture{}
	}
	f.molds[source][version] = fixture
	if f.tags[source] == nil {
		f.tags[source] = map[string]string{}
	}
	f.tags[source]["v"+version] = "sha-" + source + "-" + version
}

func (f *fakeDepFetcher) Fetch(ref *foundry.Reference) (depgraph.FetchResult, error) {
	key := ref.CacheKey()
	if ref.Subpath != "" {
		key = key + "//" + ref.Subpath
	}
	versions := f.molds[key]
	if versions == nil {
		return depgraph.FetchResult{}, &fakeNotFoundErr{key}
	}
	bare := strings.TrimPrefix(ref.Version, "v")
	v, ok := versions[bare]
	if !ok {
		// Constraint fallback: pick any single version (test fixtures register one each).
		for _, candidate := range versions {
			v = candidate
			ok = true
			break
		}
	}
	if !ok {
		return depgraph.FetchResult{}, &fakeNotFoundErr{key + "@" + ref.Version}
	}
	resolved := foundry.ResolvedVersion{Tag: "v" + bare, Commit: "sha-" + key + "-" + bare}
	if bare == "" {
		resolved = foundry.ResolvedVersion{Tag: "v1.0.0", Commit: "sha-" + key + "-1.0.0"}
	}
	cached := &depgraph.ProdFetchCacheEntry{
		FS:        v.fs,
		Root:      v.root,
		Mold:      v.mold,
		Resolved:  resolved,
		Reference: ref,
	}
	nodeKey := depgraph.NodeKey{Source: ref.CacheKey(), Subpath: ref.Subpath}
	f.cache[nodeKey] = cached
	return depgraph.FetchResult{
		Mold:    v.mold,
		Version: resolved.Tag,
		Commit:  resolved.Commit,
	}, nil
}

func (f *fakeDepFetcher) Tags(source, subpath string) (map[string]depgraph.TagInfo, error) {
	key := source
	if subpath != "" {
		key = source + "//" + subpath
	}
	tags := f.tags[key]
	if tags == nil {
		return nil, &fakeNotFoundErr{key}
	}
	out := make(map[string]depgraph.TagInfo, len(tags))
	for k, v := range tags {
		out[k] = depgraph.TagInfo{SHA: v}
	}
	return out, nil
}

func (f *fakeDepFetcher) CacheEntry(k depgraph.NodeKey) *depgraph.ProdFetchCacheEntry {
	return f.cache[k]
}

type fakeNotFoundErr struct{ key string }

func (e *fakeNotFoundErr) Error() string { return "fake fetch: not found: " + e.key }

// TestCastTransitiveDeps_NoOpWhenNoMoldDeps verifies the function is a true
// no-op for molds with no kind=mold dependencies.
func TestCastTransitiveDeps_NoOpWhenNoMoldDeps(t *testing.T) {
	root := &mold.Mold{
		APIVersion: "v1", Kind: "mold", Name: "root", Version: "1.0.0",
	}
	if err := castTransitiveDepsWith(newFakeDepFetcher(), nil, root, nil, ""); err != nil {
		t.Fatalf("expected nil for no-mold-deps mold, got %v", err)
	}
}

// TestCastTransitiveDeps_InstallsLeafAsTransitive runs the full helper against
// a fake fetcher where the root depends on a single leaf mold. Verifies the
// leaf's rendered file lands in the project and installed.yaml records it
// with InstalledAs=transitive and InstalledBy=[root key].
func TestCastTransitiveDeps_InstallsLeafAsTransitive(t *testing.T) {
	// Run in a temp project dir so installed.yaml lands under .ailloy/.
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	leafFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: mold\nname: leaf\nversion: 1.0.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte("output:\n  hello.md: hello-out.md\n")},
		"hello.md":  &fstest.MapFile{Data: []byte("# hi from leaf\n")},
	}
	leafMold := &mold.Mold{APIVersion: "v1", Kind: "mold", Name: "leaf", Version: "1.0.0"}

	fetcher := newFakeDepFetcher()
	fetcher.addMold("github.com/x/leaf", "1.0.0", &moldFixture{mold: leafMold, fs: leafFS})

	root := &mold.Mold{
		APIVersion: "v1", Kind: "mold", Name: "root", Version: "1.0.0",
		Dependencies: []mold.Dependency{
			{Mold: "github.com/x/leaf", Version: "^1.0.0"},
		},
	}
	rootRef, err := foundry.ParseReference("github.com/x/root@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	rootResult := &foundry.ResolveResult{
		Ref:      rootRef,
		Resolved: foundry.ResolvedVersion{Tag: "v1.0.0", Commit: "sha-root"},
		Root:     filepath.Join(tmp, "_unused"),
	}

	// Pre-create an empty installed.yaml so recordInstalled has something to
	// upsert into (mirrors what recordCastedFiles for the root would do).
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{{
			Name: "root", Source: "github.com/x/root", Version: "v1.0.0",
			Commit: "sha-root", InstalledAs: "direct",
		}},
	}
	if err := foundry.WriteInstalledManifest(filepath.Join(tmp, ".ailloy", "installed.yaml"), manifest); err != nil {
		t.Fatal(err)
	}

	if err := castTransitiveDepsWith(fetcher, rootResult, root, map[string]any{}, ""); err != nil {
		t.Fatalf("castTransitiveDepsWith: %v", err)
	}

	// Leaf's hello.md should have been rendered to hello-out.md.
	if _, err := os.Stat(filepath.Join(tmp, "hello-out.md")); err != nil {
		t.Errorf("leaf output missing: %v", err)
	}

	// installed.yaml should now have the leaf as transitive.
	got, err := foundry.ReadInstalledManifest(filepath.Join(tmp, ".ailloy", "installed.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	leafEntry := got.FindBySource("github.com/x/leaf", "")
	if leafEntry == nil {
		t.Fatalf("leaf not in installed.yaml; entries: %+v", got.Molds)
	}
	if leafEntry.InstalledAs != "transitive" {
		t.Errorf("InstalledAs = %q; want transitive", leafEntry.InstalledAs)
	}
	if len(leafEntry.InstalledBy) != 1 || leafEntry.InstalledBy[0] != "github.com/x/root" {
		t.Errorf("InstalledBy = %v; want [github.com/x/root]", leafEntry.InstalledBy)
	}

	// Root entry should still be marked as direct (untouched by transitive pass).
	rootEntry := got.FindBySource("github.com/x/root", "")
	if rootEntry == nil || rootEntry.InstalledAs != "direct" {
		t.Errorf("root entry: %+v", rootEntry)
	}
}

// TestCastTransitiveDeps_LocalDirSentinel verifies that a local-dir (or
// embedded) cast with mold-kind dependencies resolves them correctly when
// rootResult is synthesised from the local mold name (the behaviour added by
// the fix for GH#238). The sentinel reference uses host="local" so it can never
// collide with a real remote dep key.
func TestCastTransitiveDeps_LocalDirSentinel(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	leafFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: mold\nname: leaf\nversion: 1.0.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte("output:\n  readme.md: README.md\n")},
		"readme.md": &fstest.MapFile{Data: []byte("# from leaf\n")},
	}
	leafMold := &mold.Mold{APIVersion: "v1", Kind: "mold", Name: "leaf", Version: "1.0.0"}

	fetcher := newFakeDepFetcher()
	fetcher.addMold("github.com/x/leaf", "1.0.0", &moldFixture{mold: leafMold, fs: leafFS})

	// Aggregator: local mold that only declares a dependency, no own output.
	root := &mold.Mold{
		APIVersion: "v1", Kind: "mold", Name: "aggregator", Version: "0.1.0",
		Dependencies: []mold.Dependency{
			{Mold: "github.com/x/leaf", Version: "^1.0.0"},
		},
	}

	// Simulate what castTransitiveDeps now does for a local-dir cast
	// (rootResult nil → synthesise sentinel).
	localRef := &foundry.Reference{
		Host:  "local",
		Owner: "dir",
		Repo:  root.Name,
	}
	rootResult := &foundry.ResolveResult{
		Ref:  localRef,
		Root: filepath.Join(tmp, "_local"),
	}

	// Pre-create installed.yaml so recordInstalled has somewhere to upsert.
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds:      []foundry.InstalledEntry{},
	}
	if err := foundry.WriteInstalledManifest(filepath.Join(tmp, ".ailloy", "installed.yaml"), manifest); err != nil {
		t.Fatal(err)
	}

	if err := castTransitiveDepsWith(fetcher, rootResult, root, map[string]any{}, ""); err != nil {
		t.Fatalf("castTransitiveDepsWith: %v", err)
	}

	// Leaf's readme.md should have been rendered to README.md.
	if _, err := os.Stat(filepath.Join(tmp, "README.md")); err != nil {
		t.Errorf("leaf output README.md missing: %v", err)
	}

	// installed.yaml should record the leaf as transitive.
	got, err := foundry.ReadInstalledManifest(filepath.Join(tmp, ".ailloy", "installed.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	leafEntry := got.FindBySource("github.com/x/leaf", "")
	if leafEntry == nil {
		t.Fatalf("leaf not in installed.yaml; entries: %+v", got.Molds)
	}
	if leafEntry.InstalledAs != "transitive" {
		t.Errorf("InstalledAs = %q; want transitive", leafEntry.InstalledAs)
	}
}
