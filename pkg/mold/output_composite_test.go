package mold

import (
	"io/fs"
	"sort"
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
