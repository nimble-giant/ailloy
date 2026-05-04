package data

import "testing"

func TestMoldIdentity(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		subpath string
		want    string
	}{
		{
			name:    "bare repo, no subpath",
			source:  "github.com/nimble-giant/nimble-mold",
			subpath: "",
			want:    "github.com/nimble-giant/nimble-mold",
		},
		{
			name:    "catalog form: embedded subpath",
			source:  "github.com/kriscoleman/replicated-foundry//molds/launch",
			subpath: "",
			want:    "github.com/kriscoleman/replicated-foundry//molds/launch",
		},
		{
			name:    "installed form: separate subpath",
			source:  "github.com/kriscoleman/replicated-foundry",
			subpath: "molds/launch",
			want:    "github.com/kriscoleman/replicated-foundry//molds/launch",
		},
		{
			name:    "agreement: catalog and installed produce the same key",
			source:  "github.com/x/y//z",
			subpath: "z",
			want:    "github.com/x/y//z",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MoldIdentity(tc.source, tc.subpath); got != tc.want {
				t.Fatalf("MoldIdentity(%q, %q) = %q, want %q", tc.source, tc.subpath, got, tc.want)
			}
		})
	}
}

// TestMoldIdentity_BothFormsMatch is the load-bearing invariant: a catalog row
// and its installed-manifest counterpart must produce the same key.
func TestMoldIdentity_BothFormsMatch(t *testing.T) {
	catalog := MoldIdentity("github.com/kriscoleman/replicated-foundry//molds/launch", "")
	installed := MoldIdentity("github.com/kriscoleman/replicated-foundry", "molds/launch")
	if catalog != installed {
		t.Fatalf("identity mismatch: catalog=%q installed=%q", catalog, installed)
	}
}
