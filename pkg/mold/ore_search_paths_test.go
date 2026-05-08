package mold_test

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestBuildDefaultOreSearchPaths_OrderAndContent verifies the helper returns
// the documented precedence: mold-local first, then project (if cwd
// resolvable), then global (if home resolvable), and uses the caller-provided
// moldFS for the mold-local entry.
func TestBuildDefaultOreSearchPaths_OrderAndContent(t *testing.T) {
	moldFS := fstest.MapFS{
		"ores/example/ore.yaml": &fstest.MapFile{Data: []byte("name: example\n")},
	}

	paths := mold.BuildDefaultOreSearchPaths(moldFS, false)

	if len(paths) == 0 {
		t.Fatalf("expected at least the mold-local entry, got none")
	}
	if paths[0].Name != "mold-local" {
		t.Errorf("paths[0].Name = %q, want mold-local", paths[0].Name)
	}
	if paths[0].Root != "ores" {
		t.Errorf("paths[0].Root = %q, want ores", paths[0].Root)
	}
	// Cannot compare maps with !=; verify identity by reading a known file.
	if _, err := paths[0].FS.Open("ores/example/ore.yaml"); err != nil {
		t.Errorf("paths[0].FS missing caller-provided file: %v", err)
	}

	// project + global are best-effort — only assert that, when present,
	// they appear after mold-local and use the right Root.
	gotNames := make([]string, 0, len(paths))
	for _, p := range paths {
		gotNames = append(gotNames, p.Name)
		switch p.Name {
		case "project", "global":
			if p.Root != ".ailloy/ores" {
				t.Errorf("%s path Root = %q, want .ailloy/ores", p.Name, p.Root)
			}
		}
	}
	// Ensure mold-local is strictly first.
	for i, n := range gotNames {
		if n == "mold-local" && i != 0 {
			t.Errorf("mold-local appears at index %d, want 0", i)
		}
	}
}

// TestBuildDefaultOreSearchPaths_GlobalFlagDoesNotChangeOrder pins the current
// behavior: the global flag is reserved for future use and must not alter
// the returned path list. If this assumption changes, update the doc comment
// on BuildDefaultOreSearchPaths and this test together.
func TestBuildDefaultOreSearchPaths_GlobalFlagDoesNotChangeOrder(t *testing.T) {
	moldFS := fstest.MapFS{}

	a := mold.BuildDefaultOreSearchPaths(moldFS, false)
	b := mold.BuildDefaultOreSearchPaths(moldFS, true)

	if len(a) != len(b) {
		t.Fatalf("len differs: false=%d true=%d", len(a), len(b))
	}
	aNames := make([]string, len(a))
	bNames := make([]string, len(b))
	for i := range a {
		aNames[i] = a[i].Name
		bNames[i] = b[i].Name
	}
	if !reflect.DeepEqual(aNames, bNames) {
		t.Errorf("path Name order differs: false=%v true=%v", aNames, bNames)
	}
}
