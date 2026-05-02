package index

import (
	"fmt"
	"strings"
	"testing"
)

func TestCanonicalizeSource(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://GitHub.com/Owner/Repo", "github.com/owner/repo"},
		{"http://example.com/foundry.yaml", "example.com/foundry.yaml"},
		{"github.com/owner/repo/", "github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := canonicalizeSource(tt.in)
			if got != tt.want {
				t.Errorf("canonicalizeSource(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolverFlatFoundry(t *testing.T) {
	root := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "root",
		Molds: []MoldEntry{
			{Name: "alpha", Source: "github.com/x/alpha"},
			{Name: "beta", Source: "github.com/x/beta"},
		},
	}
	lookup := func(source string) (*Index, error) {
		if canonicalizeSource(source) == "github.com/x/root" {
			return root, nil
		}
		t.Fatalf("unexpected lookup: %q", source)
		return nil, nil
	}

	r := NewResolver(lookup)
	rf, molds, err := r.Resolve("github.com/x/root")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if rf.Index != root {
		t.Errorf("rf.Index = %v, want root", rf.Index)
	}
	if rf.Source != "github.com/x/root" {
		t.Errorf("rf.Source = %q, want github.com/x/root", rf.Source)
	}
	if len(rf.Parents) != 0 {
		t.Errorf("rf.Parents = %v, want empty", rf.Parents)
	}
	if len(rf.Children) != 0 {
		t.Errorf("rf.Children = %v, want empty", rf.Children)
	}
	if len(molds) != 2 {
		t.Fatalf("len(molds) = %d, want 2", len(molds))
	}
	for i, m := range molds {
		if m.Foundry != rf {
			t.Errorf("molds[%d].Foundry = %v, want root", i, m.Foundry)
		}
	}
}

// fakeLookup builds an IndexLookup from a static map of canonical source → index.
func fakeLookup(t *testing.T, m map[string]*Index) (IndexLookup, *int) {
	t.Helper()
	calls := 0
	return func(source string) (*Index, error) {
		calls++
		key := canonicalizeSource(source)
		idx, ok := m[key]
		if !ok {
			t.Fatalf("unexpected lookup for %q", source)
		}
		return idx, nil
	}, &calls
}

func TestResolverChildRecursion(t *testing.T) {
	root := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "root",
		Molds: []MoldEntry{{Name: "alpha", Source: "github.com/x/alpha"}},
		Foundries: []FoundryRef{
			{Name: "child", Source: "github.com/x/child"},
		},
	}
	child := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "child",
		Molds: []MoldEntry{{Name: "gamma", Source: "github.com/x/gamma"}},
	}
	lookup, calls := fakeLookup(t, map[string]*Index{
		"github.com/x/root":  root,
		"github.com/x/child": child,
	})

	rf, molds, err := NewResolver(lookup).Resolve("github.com/x/root")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if *calls != 2 {
		t.Errorf("lookup calls = %d, want 2", *calls)
	}
	if len(rf.Children) != 1 {
		t.Fatalf("len(Children) = %d, want 1", len(rf.Children))
	}
	if rf.Children[0].Index.Name != "child" {
		t.Errorf("Children[0].Index.Name = %q, want child", rf.Children[0].Index.Name)
	}
	if got := rf.Children[0].Parents; len(got) != 1 || got[0] != "root" {
		t.Errorf("Children[0].Parents = %v, want [root]", got)
	}
	if len(molds) != 2 {
		t.Fatalf("len(molds) = %d, want 2 (parent first, child second)", len(molds))
	}
	if molds[0].Entry.Name != "alpha" {
		t.Errorf("molds[0] = %q, want alpha (parent first)", molds[0].Entry.Name)
	}
	if molds[1].Entry.Name != "gamma" {
		t.Errorf("molds[1] = %q, want gamma (child second)", molds[1].Entry.Name)
	}
	if molds[1].Foundry.Index.Name != "child" {
		t.Errorf("molds[1].Foundry = %q, want child", molds[1].Foundry.Index.Name)
	}
}

func TestResolverCycle(t *testing.T) {
	a := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "a",
		Foundries: []FoundryRef{{Name: "b", Source: "github.com/x/b"}},
	}
	b := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "b",
		Foundries: []FoundryRef{{Name: "a", Source: "github.com/x/a"}},
	}
	lookup, calls := fakeLookup(t, map[string]*Index{
		"github.com/x/a": a,
		"github.com/x/b": b,
	})

	rf, _, err := NewResolver(lookup).Resolve("github.com/x/a")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if *calls != 2 {
		t.Errorf("lookup calls = %d, want 2 (each fetched once)", *calls)
	}
	if len(rf.Children) != 1 || rf.Children[0].Index.Name != "b" {
		t.Fatalf("rf.Children layout unexpected: %+v", rf.Children)
	}
	// b's child "a" should resolve back to the root, not a fresh node.
	if len(rf.Children[0].Children) != 1 || rf.Children[0].Children[0] != rf {
		t.Errorf("expected b.Children[0] to be the root a (cycle short-circuit)")
	}
}

func TestResolverSelfCycle(t *testing.T) {
	a := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "a",
		Foundries: []FoundryRef{{Name: "a", Source: "github.com/x/a"}},
	}
	lookup, calls := fakeLookup(t, map[string]*Index{"github.com/x/a": a})
	rf, _, err := NewResolver(lookup).Resolve("github.com/x/a")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if *calls != 1 {
		t.Errorf("lookup calls = %d, want 1", *calls)
	}
	if len(rf.Children) != 1 || rf.Children[0] != rf {
		t.Errorf("self-cycle not short-circuited: %+v", rf.Children)
	}
}

func TestResolverNameCollision(t *testing.T) {
	root := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "root",
		Foundries: []FoundryRef{
			{Name: "x", Source: "github.com/a/x"},
			{Name: "x", Source: "github.com/b/x"},
		},
	}
	a := &Index{APIVersion: "v1", Kind: "foundry-index", Name: "dup"}
	b := &Index{APIVersion: "v1", Kind: "foundry-index", Name: "dup"}
	lookup, _ := fakeLookup(t, map[string]*Index{
		"github.com/x/root": root,
		"github.com/a/x":    a,
		"github.com/b/x":    b,
	})

	_, _, err := NewResolver(lookup).Resolve("github.com/x/root")
	if err == nil {
		t.Fatal("expected name-collision error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"dup"`) {
		t.Errorf("error %q should name the conflicting foundry name", msg)
	}
	if !strings.Contains(msg, "github.com/a/x") || !strings.Contains(msg, "github.com/b/x") {
		t.Errorf("error %q should name both source URLs", msg)
	}
}

func TestResolverChildFetchFailure(t *testing.T) {
	root := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "root",
		Molds: []MoldEntry{{Name: "ok-mold", Source: "github.com/x/ok"}},
		Foundries: []FoundryRef{
			{Name: "broken", Source: "github.com/x/broken"},
			{Name: "good", Source: "github.com/x/good"},
		},
	}
	good := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "good",
		Molds: []MoldEntry{{Name: "good-mold", Source: "github.com/x/good-mold"}},
	}
	lookup := func(source string) (*Index, error) {
		switch canonicalizeSource(source) {
		case "github.com/x/root":
			return root, nil
		case "github.com/x/broken":
			return nil, fmt.Errorf("network down")
		case "github.com/x/good":
			return good, nil
		}
		t.Fatalf("unexpected lookup: %q", source)
		return nil, nil
	}

	r := NewResolver(lookup)
	rf, molds, err := r.Resolve("github.com/x/root")
	if err != nil {
		t.Fatalf("Resolve: %v (broken child should be a warning, not an error)", err)
	}
	// Only the good child should appear in rf.Children.
	if len(rf.Children) != 1 || rf.Children[0].Index.Name != "good" {
		t.Errorf("rf.Children = %+v, want exactly [good]", rf.Children)
	}
	// ok-mold + good-mold should both be present.
	var names []string
	for _, m := range molds {
		names = append(names, m.Entry.Name)
	}
	if len(names) != 2 {
		t.Errorf("got molds %v, want 2 (ok-mold + good-mold)", names)
	}
	// Warning should record the broken source.
	warnings := r.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %+v, want 1", warnings)
	}
	if !strings.Contains(canonicalizeSource(warnings[0].Source), "github.com/x/broken") {
		t.Errorf("warning source = %q, want broken", warnings[0].Source)
	}
}

func TestResolverDiamond(t *testing.T) {
	p := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "p",
		Foundries: []FoundryRef{
			{Name: "b", Source: "github.com/x/b"},
			{Name: "c", Source: "github.com/x/c"},
		},
	}
	b := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "b",
		Foundries: []FoundryRef{{Name: "d", Source: "github.com/x/d"}},
	}
	c := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "c",
		Foundries: []FoundryRef{{Name: "d", Source: "github.com/x/d"}},
	}
	d := &Index{
		APIVersion: "v1", Kind: "foundry-index", Name: "d",
		Molds: []MoldEntry{{Name: "delta", Source: "github.com/x/delta"}},
	}
	lookup, calls := fakeLookup(t, map[string]*Index{
		"github.com/x/p": p, "github.com/x/b": b, "github.com/x/c": c, "github.com/x/d": d,
	})

	rf, molds, err := NewResolver(lookup).Resolve("github.com/x/p")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if *calls != 4 {
		t.Errorf("lookup calls = %d, want 4 (d fetched once)", *calls)
	}
	// d appears as a child of both b and c, but is the same node.
	if rf.Children[0].Children[0] != rf.Children[1].Children[0] {
		t.Errorf("expected b's d and c's d to be the same ResolvedFoundry instance")
	}
	// Mold "delta" should appear exactly once in the flat list.
	deltas := 0
	for _, m := range molds {
		if m.Entry.Name == "delta" {
			deltas++
		}
	}
	if deltas != 1 {
		t.Errorf("delta appeared %d times in flat molds, want 1", deltas)
	}
}
