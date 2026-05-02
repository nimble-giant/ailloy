package index

import "testing"

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
