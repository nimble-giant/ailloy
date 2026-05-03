package mold

import (
	"testing"
	"testing/fstest"
)

func TestResolveFiles_StringOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"skills/helper.md":  &fstest.MapFile{Data: []byte("helper")},
	}

	resolved, err := ResolveFiles(".claude", moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	expected := map[string]string{
		"commands/hello.md": ".claude/commands/hello.md",
		"skills/helper.md":  ".claude/skills/helper.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
		if !rf.Process {
			t.Errorf("src %s: expected Process=true", rf.SrcPath)
		}
	}
}

func TestResolveFiles_MapOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"skills/helper.md":  &fstest.MapFile{Data: []byte("helper")},
	}

	output := map[string]any{
		"commands": ".claude/commands",
		"skills":   ".claude/skills",
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	expected := map[string]string{
		"commands/hello.md": ".claude/commands/hello.md",
		"skills/helper.md":  ".claude/skills/helper.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
	}
}

func TestResolveFiles_ExpandedOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"workflows/ci.yml":  &fstest.MapFile{Data: []byte("name: CI")},
	}

	output := map[string]any{
		"commands": ".claude/commands",
		"workflows": map[string]any{
			"dest":    ".github/workflows",
			"process": false,
		},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	for _, rf := range resolved {
		switch rf.SrcPath {
		case "commands/hello.md":
			if rf.DestPath != ".claude/commands/hello.md" {
				t.Errorf("commands: expected dest .claude/commands/hello.md, got %s", rf.DestPath)
			}
			if !rf.Process {
				t.Error("commands: expected Process=true")
			}
		case "workflows/ci.yml":
			if rf.DestPath != ".github/workflows/ci.yml" {
				t.Errorf("workflows: expected dest .github/workflows/ci.yml, got %s", rf.DestPath)
			}
			if rf.Process {
				t.Error("workflows: expected Process=false")
			}
		default:
			t.Errorf("unexpected src path: %s", rf.SrcPath)
		}
	}
}

func TestResolveFiles_FileLevelOverride(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md":   &fstest.MapFile{Data: []byte("hello")},
		"commands/special.md": &fstest.MapFile{Data: []byte("special")},
	}

	output := map[string]any{
		"commands":            ".claude/commands",
		"commands/special.md": "custom/special.md",
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	for _, rf := range resolved {
		switch rf.SrcPath {
		case "commands/hello.md":
			if rf.DestPath != ".claude/commands/hello.md" {
				t.Errorf("hello: expected dest .claude/commands/hello.md, got %s", rf.DestPath)
			}
		case "commands/special.md":
			if rf.DestPath != "custom/special.md" {
				t.Errorf("special: expected dest custom/special.md, got %s", rf.DestPath)
			}
		default:
			t.Errorf("unexpected src path: %s", rf.SrcPath)
		}
	}
}

func TestResolveFiles_RecursiveWalk(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/sub/nested.md": &fstest.MapFile{Data: []byte("nested")},
	}

	output := map[string]any{
		"commands": "dest",
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file, got %d", len(resolved))
	}

	if resolved[0].SrcPath != "commands/sub/nested.md" {
		t.Errorf("expected src commands/sub/nested.md, got %s", resolved[0].SrcPath)
	}
	if resolved[0].DestPath != "dest/sub/nested.md" {
		t.Errorf("expected dest dest/sub/nested.md, got %s", resolved[0].DestPath)
	}
}

func TestResolveFiles_NoOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"ingots/foo.md":     &fstest.MapFile{Data: []byte("ingot")},
	}

	resolved, err := ResolveFiles(nil, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file (ingots excluded), got %d", len(resolved))
	}

	rf := resolved[0]
	if rf.SrcPath != "commands/hello.md" {
		t.Errorf("expected src commands/hello.md, got %s", rf.SrcPath)
	}
	if rf.DestPath != "commands/hello.md" {
		t.Errorf("expected identity mapping (dest=src), got dest %s", rf.DestPath)
	}
	if !rf.Process {
		t.Error("expected Process=true for identity mapping")
	}
}

func TestResolveFiles_MissingSourceDir(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
	}

	output := map[string]any{
		"nonexistent": ".claude/nonexistent",
	}

	// "nonexistent" is not a directory in the FS, so parseMapOutput treats it
	// as a file mapping. The remaining-files loop tries to stat it and returns
	// an error since the file doesn't exist.
	_, err := ResolveFiles(output, moldFS)
	if err == nil {
		t.Fatal("expected error for nonexistent file mapping")
	}
}

func TestResolveFiles_RootFiles_Identity(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Agent instructions")},
	}

	resolved, err := ResolveFiles(nil, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	expected := map[string]string{
		"AGENTS.md":         "AGENTS.md",
		"commands/hello.md": "commands/hello.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
		if !rf.Process {
			t.Errorf("src %s: expected Process=true", rf.SrcPath)
		}
	}
}

func TestResolveFiles_RootFiles_StringOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Agent instructions")},
	}

	resolved, err := ResolveFiles(".claude", moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	// Root files go to project root, not under .claude/
	expected := map[string]string{
		"AGENTS.md":         "AGENTS.md",
		"commands/hello.md": ".claude/commands/hello.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
		if !rf.Process {
			t.Errorf("src %s: expected Process=true", rf.SrcPath)
		}
	}
}

func TestResolveFiles_RootFiles_ExcludesMetadata(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Agent instructions")},
		"mold.yaml":         &fstest.MapFile{Data: []byte("name: test")},
		"flux.yaml":         &fstest.MapFile{Data: []byte("org: acme")},
		"flux.schema.yaml":  &fstest.MapFile{Data: []byte("schema")},
		"ingot.yaml":        &fstest.MapFile{Data: []byte("ingot")},
		"README.md":         &fstest.MapFile{Data: []byte("# readme")},
		"PLUGIN_SUMMARY.md": &fstest.MapFile{Data: []byte("# plugin")},
		"LICENSE":           &fstest.MapFile{Data: []byte("MIT")},
		".hidden":           &fstest.MapFile{Data: []byte("hidden")},
	}

	resolved, err := ResolveFiles(nil, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only commands/hello.md and AGENTS.md should be resolved
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d: %v", len(resolved), resolved)
	}

	found := make(map[string]bool)
	for _, rf := range resolved {
		found[rf.SrcPath] = true
	}

	if !found["AGENTS.md"] {
		t.Error("expected AGENTS.md to be discovered")
	}
	if !found["commands/hello.md"] {
		t.Error("expected commands/hello.md to be discovered")
	}
}

func TestResolveFiles_RootFiles_MapOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Agent instructions")},
	}

	// Map form: explicit file mapping
	output := map[string]any{
		"commands":  ".claude/commands",
		"AGENTS.md": "AGENTS.md",
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	expected := map[string]string{
		"AGENTS.md":         "AGENTS.md",
		"commands/hello.md": ".claude/commands/hello.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
	}
}

func TestResolveFiles_RootFiles_MapOutput_AutoDiscover(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Agent instructions")},
	}

	// Map form with only directories — AGENTS.md is NOT in the map
	// but should be auto-discovered from the mold root.
	output := map[string]any{
		"commands": ".claude/commands",
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	expected := map[string]string{
		"AGENTS.md":         "AGENTS.md",
		"commands/hello.md": ".claude/commands/hello.md",
	}

	for _, rf := range resolved {
		wantDest, ok := expected[rf.SrcPath]
		if !ok {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
			continue
		}
		if rf.DestPath != wantDest {
			t.Errorf("src %s: expected dest %s, got %s", rf.SrcPath, wantDest, rf.DestPath)
		}
	}
}

func TestResolveFiles_ListOutput_Strings(t *testing.T) {
	moldFS := fstest.MapFS{
		"agents/coding.md": &fstest.MapFile{Data: []byte("agent")},
	}

	output := map[string]any{
		"agents": []any{".claude/agents", ".opencode/agents"},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files (one per dest), got %d", len(resolved))
	}

	dests := map[string]bool{}
	for _, rf := range resolved {
		if rf.SrcPath != "agents/coding.md" {
			t.Errorf("unexpected src path: %s", rf.SrcPath)
		}
		if !rf.Process {
			t.Errorf("expected Process=true for %s", rf.DestPath)
		}
		dests[rf.DestPath] = true
	}

	if !dests[".claude/agents/coding.md"] {
		t.Error("missing .claude/agents/coding.md")
	}
	if !dests[".opencode/agents/coding.md"] {
		t.Error("missing .opencode/agents/coding.md")
	}
}

func TestResolveFiles_ListOutput_WithSet(t *testing.T) {
	moldFS := fstest.MapFS{
		"agents/coding.md": &fstest.MapFile{Data: []byte("agent")},
	}

	output := map[string]any{
		"agents": []any{
			map[string]any{
				"dest": ".claude/agents",
				"set": map[string]any{
					"agent.current_target": "claude",
				},
			},
			map[string]any{
				"dest": ".opencode/agents",
				"set": map[string]any{
					"agent.current_target": "opencode",
				},
			},
		},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	byDest := map[string]ResolvedFile{}
	for _, rf := range resolved {
		byDest[rf.DestPath] = rf
	}

	claude, ok := byDest[".claude/agents/coding.md"]
	if !ok {
		t.Fatal("missing .claude/agents/coding.md")
	}
	if got := claude.Set["agent.current_target"]; got != "claude" {
		t.Errorf("claude dest: expected set agent.current_target=claude, got %v", got)
	}

	oc, ok := byDest[".opencode/agents/coding.md"]
	if !ok {
		t.Fatal("missing .opencode/agents/coding.md")
	}
	if got := oc.Set["agent.current_target"]; got != "opencode" {
		t.Errorf("opencode dest: expected set agent.current_target=opencode, got %v", got)
	}
}

func TestResolveFiles_ListOutput_MixedWithProcess(t *testing.T) {
	moldFS := fstest.MapFS{
		"workflows/ci.yml": &fstest.MapFile{Data: []byte("name: CI")},
	}

	output := map[string]any{
		"workflows": []any{
			map[string]any{
				"dest":    ".github/workflows",
				"process": false,
			},
			".other/workflows",
		},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	for _, rf := range resolved {
		switch rf.DestPath {
		case ".github/workflows/ci.yml":
			if rf.Process {
				t.Error("github workflows: expected Process=false")
			}
		case ".other/workflows/ci.yml":
			if !rf.Process {
				t.Error("other workflows: expected Process=true")
			}
		default:
			t.Errorf("unexpected dest: %s", rf.DestPath)
		}
	}
}

func TestResolveFiles_ListOutput_FileLevel(t *testing.T) {
	moldFS := fstest.MapFS{
		"AGENT.md": &fstest.MapFile{Data: []byte("body")},
	}

	output := map[string]any{
		"AGENT.md": []any{"AGENTS.md", "CLAUDE.md"},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	dests := map[string]bool{}
	for _, rf := range resolved {
		if rf.SrcPath != "AGENT.md" {
			t.Errorf("unexpected src: %s", rf.SrcPath)
		}
		dests[rf.DestPath] = true
	}
	if !dests["AGENTS.md"] || !dests["CLAUDE.md"] {
		t.Errorf("expected both AGENTS.md and CLAUDE.md, got %v", dests)
	}
}

func TestResolveFiles_ListOutput_RendersWithSet(t *testing.T) {
	moldFS := fstest.MapFS{
		"agents/coding.md": &fstest.MapFile{Data: []byte("target={{.agent.current_target}}")},
	}

	output := map[string]any{
		"agents": []any{
			map[string]any{
				"dest": ".claude/agents",
				"set":  map[string]any{"agent.current_target": "claude"},
			},
			map[string]any{
				"dest": ".opencode/agents",
				"set":  map[string]any{"agent.current_target": "opencode"},
			},
		},
	}

	resolved, err := ResolveFiles(output, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flux := map[string]any{}
	rendered := map[string]string{}
	for _, rf := range resolved {
		fluxForFile := flux
		if len(rf.Set) > 0 {
			fluxForFile = MergeSet(flux, rf.Set)
		}
		out, err := ProcessTemplate("target={{.agent.current_target}}", fluxForFile)
		if err != nil {
			t.Fatalf("ProcessTemplate %s: %v", rf.DestPath, err)
		}
		rendered[rf.DestPath] = out
	}

	if rendered[".claude/agents/coding.md"] != "target=claude" {
		t.Errorf("claude render: got %q", rendered[".claude/agents/coding.md"])
	}
	if rendered[".opencode/agents/coding.md"] != "target=opencode" {
		t.Errorf("opencode render: got %q", rendered[".opencode/agents/coding.md"])
	}
}

func TestResolveFiles_ListOutput_Empty(t *testing.T) {
	moldFS := fstest.MapFS{
		"agents/coding.md": &fstest.MapFile{Data: []byte("agent")},
	}

	output := map[string]any{
		"agents": []any{},
	}

	_, err := ResolveFiles(output, moldFS)
	if err == nil {
		t.Fatal("expected error for empty list, got nil")
	}
}

func TestResolveFiles_ListOutput_InvalidEntry(t *testing.T) {
	moldFS := fstest.MapFS{
		"agents/coding.md": &fstest.MapFile{Data: []byte("agent")},
	}

	output := map[string]any{
		"agents": []any{42},
	}

	_, err := ResolveFiles(output, moldFS)
	if err == nil {
		t.Fatal("expected error for non-string non-map list entry")
	}
}

func TestResolveFiles_ExcludesReservedDirs(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"ingots/foo.md":     &fstest.MapFile{Data: []byte("ingot")},
		".hidden/bar.md":    &fstest.MapFile{Data: []byte("hidden")},
	}

	resolved, err := ResolveFiles(".claude", moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file, got %d", len(resolved))
	}

	rf := resolved[0]
	if rf.SrcPath != "commands/hello.md" {
		t.Errorf("expected only commands/hello.md, got %s", rf.SrcPath)
	}
	if rf.DestPath != ".claude/commands/hello.md" {
		t.Errorf("expected dest .claude/commands/hello.md, got %s", rf.DestPath)
	}
}

func TestParseTargetMap_Strategy(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    string
		wantErr bool
	}{
		{name: "absent", input: map[string]any{"dest": "x"}, want: ""},
		{name: "replace", input: map[string]any{"dest": "x", "strategy": "replace"}, want: "replace"},
		{name: "merge", input: map[string]any{"dest": "x", "strategy": "merge"}, want: "merge"},
		{name: "unknown", input: map[string]any{"dest": "x", "strategy": "smush"}, wantErr: true},
		{name: "non-string", input: map[string]any{"dest": "x", "strategy": 7}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTargetMap(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (target=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Strategy != tc.want {
				t.Fatalf("Strategy: want %q, got %q", tc.want, got.Strategy)
			}
		})
	}
}
