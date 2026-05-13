package mold

import (
	"io/fs"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

func TestResolveFilesWithOreSources_OreDeclaredOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: c\nversion: 0.1.0\n")},
	}
	oreFS := fstest.MapFS{
		"ore.yaml":             &fstest.MapFile{Data: []byte("kind: ore\nname: agent_targets\n")},
		"blanks/AGENTS.md":     &fstest.MapFile{Data: []byte("# agents\n")},
		"blanks/agents/foo.md": &fstest.MapFile{Data: []byte("# foo\n")},
	}

	output := map[string]any{
		"blanks/AGENTS.md": "AGENTS.md",
		"blanks/agents":    ".claude/agents",
	}
	sources := []OreSource{{Namespace: "agent_targets", FS: oreFS, Output: output}}

	resolved, err := ResolveFilesWithOreSources(nil, moldFS, sources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := map[string]string{}
	for _, r := range resolved {
		got[r.DestPath] = r.SrcPath
		if r.SrcFS == nil {
			t.Errorf("ore-supplied entry lost SrcFS: %+v", r)
		}
		if r.Origin != "agent_targets" {
			t.Errorf("Origin = %q, want %q", r.Origin, "agent_targets")
		}
	}
	want := map[string]string{
		"AGENTS.md":             "blanks/AGENTS.md",
		".claude/agents/foo.md": "blanks/agents/foo.md",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("dest %q: got src %q, want %q", k, got[k], v)
		}
	}
}

func TestResolveFilesWithOreSources_ConsumerFromSelector(t *testing.T) {
	moldFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("name: c\n")},
	}
	oreFS := fstest.MapFS{
		"blanks/AGENTS.md": &fstest.MapFile{Data: []byte("# from ore\n")},
	}

	output := map[string]any{
		"AGENTS.md": map[string]any{
			"from": "ore/agent_targets/blanks/AGENTS.md",
			"dest": "AGENTS.md",
		},
	}
	sources := []OreSource{{Namespace: "agent_targets", FS: oreFS}}

	resolved, err := ResolveFilesWithOreSources(output, moldFS, sources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file, got %d: %+v", len(resolved), resolved)
	}
	r := resolved[0]
	if r.SrcPath != "blanks/AGENTS.md" || r.DestPath != "AGENTS.md" {
		t.Errorf("wrong paths: src=%q dest=%q", r.SrcPath, r.DestPath)
	}
	if r.Origin != "agent_targets" || r.SrcFS == nil {
		t.Errorf("origin/fs wrong: %+v", r)
	}
}

func TestResolveFilesWithOreSources_BackwardCompat_MoldOnly(t *testing.T) {
	moldFS := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("name: c\n")},
		"commands/hello.md": &fstest.MapFile{Data: []byte("hi\n")},
	}
	resolved, err := ResolveFilesWithOreSources(nil, moldFS, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 || resolved[0].SrcPath != "commands/hello.md" {
		t.Fatalf("wrong resolution: %+v", resolved)
	}
	if resolved[0].Origin != "" {
		t.Errorf("Origin should be empty for mold-origin entries, got %q", resolved[0].Origin)
	}
}

func TestResolveFilesWithOreSources_DeterministicOrder(t *testing.T) {
	moldFS := fstest.MapFS{"mold.yaml": &fstest.MapFile{Data: []byte("name: c\n")}}
	oreFS := fstest.MapFS{
		"blanks/a.md": &fstest.MapFile{Data: []byte("a")},
		"blanks/b.md": &fstest.MapFile{Data: []byte("b")},
	}
	output := map[string]any{
		"blanks/a.md": "a.md",
		"blanks/b.md": "b.md",
	}
	sources := []OreSource{{Namespace: "n", FS: oreFS, Output: output}}
	resolved, err := ResolveFilesWithOreSources(nil, moldFS, sources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	paths := make([]string, len(resolved))
	for i, r := range resolved {
		paths[i] = r.DestPath
	}
	if !sort.StringsAreSorted(paths) {
		t.Errorf("not sorted: %v", paths)
	}
}

var _ fs.FS = (fstest.MapFS)(nil)

// Regression: a consumer whose `output:` map consists ENTIRELY of `from:`
// selectors used to fall through to ResolveFiles(nil, moldFS), which
// triggered identity auto-discovery and resolved every non-reserved file
// in the mold. The fix returns an empty map (not nil) so ResolveFiles
// takes the explicit-map path instead.
func TestResolveFilesWithOreSources_AllFromConsumerDoesNotAutoDiscoverMold(t *testing.T) {
	moldFS := fstest.MapFS{
		"mold.yaml":           &fstest.MapFile{Data: []byte("name: c\n")},
		"commands/hello.md":   &fstest.MapFile{Data: []byte("hi\n")},
		"unrelated/extra.txt": &fstest.MapFile{Data: []byte("extra\n")},
	}
	oreFS := fstest.MapFS{
		"blanks/AGENTS.md": &fstest.MapFile{Data: []byte("# from ore\n")},
	}
	output := map[string]any{
		"AGENTS.md": map[string]any{
			"from": "ore/agent_targets/blanks/AGENTS.md",
			"dest": "AGENTS.md",
		},
	}
	sources := []OreSource{{Namespace: "agent_targets", FS: oreFS}}

	resolved, err := ResolveFilesWithOreSources(output, moldFS, sources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected exactly 1 resolved file (from: AGENTS.md), got %d: %+v", len(resolved), resolved)
	}
	if resolved[0].DestPath != "AGENTS.md" || resolved[0].Origin != "agent_targets" {
		t.Errorf("wrong resolved entry: %+v", resolved[0])
	}
	// Specifically: the mold's commands/hello.md and unrelated/extra.txt
	// must NOT appear in the resolved set.
	for _, r := range resolved {
		if r.SrcPath == "commands/hello.md" || r.SrcPath == "unrelated/extra.txt" {
			t.Errorf("auto-discovery leaked mold file into resolved: %+v", r)
		}
	}
}

// Regression: a `from:` selector pointing at a directory used to pass
// fs.Stat (directories exist) and then fail later with an obscure
// "is a directory" error when cast/forge tried to ReadFile. The fix
// rejects directory targets at resolve time with a clear message.
func TestResolveFilesWithOreSources_FromSelectorRejectsDirectoryTarget(t *testing.T) {
	moldFS := fstest.MapFS{"mold.yaml": &fstest.MapFile{Data: []byte("name: c\n")}}
	oreFS := fstest.MapFS{
		"blanks/agents/inner.md": &fstest.MapFile{Data: []byte("# inner\n")},
	}
	output := map[string]any{
		"agents": map[string]any{
			"from": "ore/at/blanks/agents",
			"dest": "agents",
		},
	}
	sources := []OreSource{{Namespace: "at", FS: oreFS}}

	_, err := ResolveFilesWithOreSources(output, moldFS, sources)
	if err == nil {
		t.Fatal("expected error for directory `from:` target, got nil")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention directory, got: %v", err)
	}
}
