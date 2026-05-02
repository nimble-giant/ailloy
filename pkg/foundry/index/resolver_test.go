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
