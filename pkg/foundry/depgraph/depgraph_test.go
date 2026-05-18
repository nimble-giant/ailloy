package depgraph

import (
	"sort"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// fakeFetcher is an in-memory Fetcher used by tests. It lets us assemble a
// foundry of fake molds with declared tag lists without touching git.
type fakeFetcher struct {
	molds       map[string]map[string]*mold.Mold // sourceKey -> version -> mold
	tags        map[string]map[string]string     // sourceKey -> tag -> sha
	moldVersion map[string]map[string]string     // sourceKey -> tag -> mold.yaml version
}

func newFakeFetcher() *fakeFetcher {
	return &fakeFetcher{
		molds:       map[string]map[string]*mold.Mold{},
		tags:        map[string]map[string]string{},
		moldVersion: map[string]map[string]string{},
	}
}

func (f *fakeFetcher) addMold(source, version string, m *mold.Mold) {
	if f.molds[source] == nil {
		f.molds[source] = map[string]*mold.Mold{}
	}
	f.molds[source][version] = m
	if f.tags[source] == nil {
		f.tags[source] = map[string]string{}
	}
	f.tags[source]["v"+version] = "sha-" + source + "-" + version
}

func (f *fakeFetcher) Fetch(ref *foundry.Reference) (FetchResult, error) {
	key := refKey(ref)
	versions := f.molds[key]
	if versions == nil {
		return FetchResult{}, errNotFound(key)
	}
	// Exact / 'v'-stripped match wins.
	if v, ok := versions[ref.Version]; ok {
		return FetchResult{Mold: v, Version: ref.Version, Commit: "sha-" + key + "-" + ref.Version}, nil
	}
	if v, ok := versions[strings.TrimPrefix(ref.Version, "v")]; ok {
		return FetchResult{Mold: v, Version: strings.TrimPrefix(ref.Version, "v"), Commit: "sha-" + key + "-" + ref.Version}, nil
	}
	// Otherwise treat as constraint and pick the highest matching version.
	tags := f.tags[key]
	tag, _, ok := highestSatisfyingFromRaw(tags, ref.Version)
	if !ok {
		return FetchResult{}, errNotFound(key + "@" + ref.Version)
	}
	bare := strings.TrimPrefix(tag, "v")
	v, ok := versions[bare]
	if !ok {
		return FetchResult{}, errNotFound(key + "@" + tag)
	}
	return FetchResult{Mold: v, Version: tag, Commit: "sha-" + key + "-" + bare}, nil
}

// highestSatisfyingFromRaw picks the highest tag in `tags` that satisfies
// the raw constraint `raw`. Used by the fake fetcher to mimic real resolution.
func highestSatisfyingFromRaw(tags map[string]string, raw string) (string, string, bool) {
	var c *semver.Constraints
	if raw != "" && raw != "latest" {
		var err error
		c, err = semver.NewConstraint(raw)
		if err != nil {
			return "", "", false
		}
	}
	type entry struct {
		tag string
		sha string
		ver *semver.Version
	}
	var candidates []entry
	for tag, sha := range tags {
		v, err := semver.NewVersion(strings.TrimPrefix(tag, "v"))
		if err != nil {
			continue
		}
		if c != nil && !c.Check(v) {
			continue
		}
		candidates = append(candidates, entry{tag, sha, v})
	}
	if len(candidates) == 0 {
		return "", "", false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ver.LessThan(candidates[j].ver)
	})
	best := candidates[len(candidates)-1]
	return best.tag, best.sha, true
}

func (f *fakeFetcher) Tags(source, subpath string) (map[string]TagInfo, error) {
	key := source
	if subpath != "" {
		key = source + "//" + subpath
	}
	tags := f.tags[key]
	if tags == nil {
		return nil, errNotFound(key)
	}
	out := make(map[string]TagInfo, len(tags))
	for k, v := range tags {
		out[k] = TagInfo{SHA: v, MoldVersion: f.moldVersion[key][k]}
	}
	return out, nil
}

func errNotFound(s string) error {
	return &fakeNotFound{s}
}

type fakeNotFound struct{ key string }

func (e *fakeNotFound) Error() string { return "fake fetcher: not found: " + e.key }

func refKey(r *foundry.Reference) string {
	k := r.Host + "/" + r.Owner + "/" + r.Repo
	if r.Subpath != "" {
		k += "//" + r.Subpath
	}
	return k
}

func mustRef(t *testing.T, raw string) *foundry.Reference {
	t.Helper()
	r, err := foundry.ParseReference(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return r
}

func makeMold(name string, deps ...mold.Dependency) *mold.Mold {
	return &mold.Mold{
		APIVersion:   "v1",
		Kind:         "mold",
		Name:         name,
		Version:      "1.0.0",
		Dependencies: deps,
	}
}

// TestBuild_SingleNoDeps: a root with no mold deps yields just the root node.
func TestBuild_SingleNoDeps(t *testing.T) {
	f := newFakeFetcher()
	root := makeMold("root")
	f.addMold("github.com/x/root", "1.0.0", root)
	rootRef := mustRef(t, "github.com/x/root@1.0.0")

	b := New(f)
	graph, err := b.Build(root, rootRef)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d: %+v", len(graph.Nodes), graph.Nodes)
	}
	if graph.Nodes[0].Mold.Name != "root" {
		t.Errorf("root.Name = %q", graph.Nodes[0].Mold.Name)
	}
	if len(graph.Nodes[0].Parents) != 0 {
		t.Errorf("root must have no parents, got %v", graph.Nodes[0].Parents)
	}
}

// TestBuild_Chain: A → B. Result is leaves-first ordered.
func TestBuild_Chain(t *testing.T) {
	f := newFakeFetcher()
	leaf := makeMold("leaf")
	parent := makeMold("parent",
		mold.Dependency{Mold: "github.com/x/leaf", Version: "^1.0.0"},
	)
	f.addMold("github.com/x/leaf", "1.0.0", leaf)
	f.addMold("github.com/x/parent", "1.0.0", parent)

	b := New(f)
	graph, err := b.Build(parent, mustRef(t, "github.com/x/parent@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph.Nodes))
	}
	if graph.Nodes[0].Mold.Name != "leaf" {
		t.Errorf("expected leaf first (leaves-first order), got %q", graph.Nodes[0].Mold.Name)
	}
	if graph.Nodes[1].Mold.Name != "parent" {
		t.Errorf("expected parent second, got %q", graph.Nodes[1].Mold.Name)
	}
}

// TestBuild_DiamondCompatible: A→B,C; B→D@^1.0.0; C→D@^1.2.0.
// D should resolve to the highest version satisfying both (>=1.2.0,<2.0.0).
func TestBuild_DiamondCompatible(t *testing.T) {
	f := newFakeFetcher()
	d10 := makeMold("d")
	d12 := makeMold("d")
	d12.Version = "1.2.5"
	f.addMold("github.com/x/d", "1.0.0", d10)
	f.addMold("github.com/x/d", "1.2.5", d12)

	b := makeMold("b", mold.Dependency{Mold: "github.com/x/d", Version: "^1.0.0"})
	c := makeMold("c", mold.Dependency{Mold: "github.com/x/d", Version: "^1.2.0"})
	f.addMold("github.com/x/b", "1.0.0", b)
	f.addMold("github.com/x/c", "1.0.0", c)

	a := makeMold("a",
		mold.Dependency{Mold: "github.com/x/b", Version: "^1.0.0"},
		mold.Dependency{Mold: "github.com/x/c", Version: "^1.0.0"},
	)
	f.addMold("github.com/x/a", "1.0.0", a)

	builder := New(f)
	graph, err := builder.Build(a, mustRef(t, "github.com/x/a@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	dNode := graph.Find("github.com/x/d", "")
	if dNode == nil {
		t.Fatal("d not in graph")
	}
	if strings.TrimPrefix(dNode.Version, "v") != "1.2.5" {
		t.Errorf("d should resolve to 1.2.5 (highest satisfying ^1.0.0 ∩ ^1.2.0), got %q", dNode.Version)
	}
	// d should appear exactly once even though reached via two paths.
	count := 0
	for _, n := range graph.Nodes {
		if n.Key.Source == "github.com/x/d" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("d appeared %d times; want exactly 1", count)
	}
}

// TestBuild_DiamondConflict: B→D@^1.0.0; C→D@^2.0.0. No version satisfies both.
func TestBuild_DiamondConflict(t *testing.T) {
	f := newFakeFetcher()
	d1 := makeMold("d")
	d2 := makeMold("d")
	d2.Version = "2.0.0"
	f.addMold("github.com/x/d", "1.0.0", d1)
	f.addMold("github.com/x/d", "2.0.0", d2)

	b := makeMold("b", mold.Dependency{Mold: "github.com/x/d", Version: "^1.0.0"})
	c := makeMold("c", mold.Dependency{Mold: "github.com/x/d", Version: "^2.0.0"})
	f.addMold("github.com/x/b", "1.0.0", b)
	f.addMold("github.com/x/c", "1.0.0", c)

	a := makeMold("a",
		mold.Dependency{Mold: "github.com/x/b", Version: "^1.0.0"},
		mold.Dependency{Mold: "github.com/x/c", Version: "^1.0.0"},
	)
	f.addMold("github.com/x/a", "1.0.0", a)

	_, err := New(f).Build(a, mustRef(t, "github.com/x/a@1.0.0"))
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "github.com/x/d") {
		t.Errorf("error should name the conflicting mold, got: %v", err)
	}
}

// TestBuild_Cycle: A→B→A. Must error with the cycle path.
func TestBuild_Cycle(t *testing.T) {
	f := newFakeFetcher()
	a := makeMold("a", mold.Dependency{Mold: "github.com/x/b", Version: "^1.0.0"})
	b := makeMold("b", mold.Dependency{Mold: "github.com/x/a", Version: "^1.0.0"})
	f.addMold("github.com/x/a", "1.0.0", a)
	f.addMold("github.com/x/b", "1.0.0", b)

	_, err := New(f).Build(a, mustRef(t, "github.com/x/a@1.0.0"))
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention 'cycle', got: %v", err)
	}
	if !strings.Contains(err.Error(), "github.com/x/a") || !strings.Contains(err.Error(), "github.com/x/b") {
		t.Errorf("cycle error should contain both members of the cycle, got: %v", err)
	}
}

// TestBuild_DedupesByCanonicalKey: alias differences don't change identity.
func TestBuild_DedupesByCanonicalKey(t *testing.T) {
	f := newFakeFetcher()
	leaf := makeMold("leaf")
	f.addMold("github.com/x/leaf", "1.0.0", leaf)

	root := makeMold("root",
		mold.Dependency{Mold: "github.com/x/leaf", Version: "^1.0.0", As: "alias-one"},
		mold.Dependency{Mold: "github.com/x/leaf", Version: "^1.0.0", As: "alias-two"},
	)
	f.addMold("github.com/x/root", "1.0.0", root)

	graph, err := New(f).Build(root, mustRef(t, "github.com/x/root@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	count := 0
	for _, n := range graph.Nodes {
		if n.Key.Source == "github.com/x/leaf" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("leaf appeared %d times; canonical key should de-dup regardless of alias", count)
	}
}

// TestBuild_NonMoldDepsIgnored: ingot/ore deps in the graph walk are skipped
// (they are resolved separately at install time via install_deps).
func TestBuild_NonMoldDepsIgnored(t *testing.T) {
	f := newFakeFetcher()
	root := makeMold("root",
		mold.Dependency{Ingot: "github.com/x/some-ingot", Version: "^1.0.0"},
		mold.Dependency{Ore: "github.com/x/some-ore", Version: "^1.0.0"},
	)
	f.addMold("github.com/x/root", "1.0.0", root)

	graph, err := New(f).Build(root, mustRef(t, "github.com/x/root@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected only root node (ingot/ore deps don't enter the graph), got %d", len(graph.Nodes))
	}
}

// TestBuild_ParentsAndWith: a transitive node records the parents that pulled
// it in, and merges any `with:` blocks from each parent edge.
func TestBuild_ParentsAndWith(t *testing.T) {
	f := newFakeFetcher()
	leaf := makeMold("leaf")
	f.addMold("github.com/x/leaf", "1.0.0", leaf)

	root := makeMold("root",
		mold.Dependency{
			Mold: "github.com/x/leaf", Version: "^1.0.0",
			With: map[string]any{"key": "from-root"},
		},
	)
	f.addMold("github.com/x/root", "1.0.0", root)

	graph, err := New(f).Build(root, mustRef(t, "github.com/x/root@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	leafNode := graph.Find("github.com/x/leaf", "")
	if leafNode == nil {
		t.Fatal("leaf missing")
	}
	if len(leafNode.Parents) != 1 || leafNode.Parents[0].Source != "github.com/x/root" {
		t.Errorf("leaf.Parents = %+v; want [github.com/x/root]", leafNode.Parents)
	}
	if leafNode.With["key"] != "from-root" {
		t.Errorf("leaf.With[key] = %v; want from-root", leafNode.With["key"])
	}
}

// TestBuild_SubpathDistinctIdentity: two molds in the same repo at different
// subpaths are distinct nodes.
func TestBuild_SubpathDistinctIdentity(t *testing.T) {
	f := newFakeFetcher()
	leafA := makeMold("leaf-a")
	leafB := makeMold("leaf-b")
	f.addMold("github.com/x/repo//leaf-a", "1.0.0", leafA)
	f.addMold("github.com/x/repo//leaf-b", "1.0.0", leafB)
	f.tags["github.com/x/repo//leaf-a"] = map[string]string{"v1.0.0": "shaA"}
	f.tags["github.com/x/repo//leaf-b"] = map[string]string{"v1.0.0": "shaB"}

	root := makeMold("root",
		mold.Dependency{Mold: "github.com/x/repo@^1.0.0//leaf-a", Version: "^1.0.0"},
		mold.Dependency{Mold: "github.com/x/repo@^1.0.0//leaf-b", Version: "^1.0.0"},
	)
	f.addMold("github.com/x/root", "1.0.0", root)

	graph, err := New(f).Build(root, mustRef(t, "github.com/x/root@1.0.0"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes (root + leaf-a + leaf-b), got %d", len(graph.Nodes))
	}
}

func mustConstraint(t *testing.T, s string) *semver.Constraints {
	t.Helper()
	c, err := semver.NewConstraint(s)
	if err != nil {
		t.Fatalf("bad constraint %q: %v", s, err)
	}
	return c
}

// TestHighestSatisfying_MoldVersion exercises the release-train case: several
// tags share one mold version, so the intersected constraints match them all
// and the newest tag wins the tie-break.
func TestHighestSatisfying_MoldVersion(t *testing.T) {
	tags := map[string]TagInfo{
		"launch-v0.6.0": {SHA: "sha60", MoldVersion: "0.2.1"},
		"launch-v0.7.0": {SHA: "sha70", MoldVersion: "0.2.1"},
		"launch-v0.7.1": {SHA: "sha71", MoldVersion: "0.2.1"},
	}
	constraints := []*semver.Constraints{
		mustConstraint(t, "^0.2.0"),
		mustConstraint(t, ">=0.2.1"),
	}
	tag, sha, ok := highestSatisfying(tags, constraints)
	if !ok {
		t.Fatal("expected a satisfying tag")
	}
	if tag != "launch-v0.7.1" {
		t.Errorf("tag = %q, want launch-v0.7.1 (newest train tag)", tag)
	}
	if sha != "sha71" {
		t.Errorf("sha = %q, want sha71", sha)
	}
}

// TestHighestSatisfying_FallbackToTag confirms tags with no mold version are
// ranked by their tag-embedded semver.
func TestHighestSatisfying_FallbackToTag(t *testing.T) {
	tags := map[string]TagInfo{
		"v1.0.0": {SHA: "a"},
		"v1.2.0": {SHA: "b"},
		"v0.9.0": {SHA: "c"},
	}
	tag, sha, ok := highestSatisfying(tags, []*semver.Constraints{mustConstraint(t, "^1.0.0")})
	if !ok || tag != "v1.2.0" || sha != "b" {
		t.Errorf("got (%q, %q, %v), want (v1.2.0, b, true)", tag, sha, ok)
	}
}
