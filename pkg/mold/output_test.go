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

	m := &Mold{Output: ".claude"}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{
		Output: map[string]any{
			"commands": ".claude/commands",
			"skills":   ".claude/skills",
		},
	}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{
		Output: map[string]any{
			"commands": ".claude/commands",
			"workflows": map[string]any{
				"dest":    ".github/workflows",
				"process": false,
			},
		},
	}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{
		Output: map[string]any{
			"commands":            ".claude/commands",
			"commands/special.md": "custom/special.md",
		},
	}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{
		Output: map[string]any{
			"commands": "dest",
		},
	}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{Output: nil}

	resolved, err := ResolveFiles(m, moldFS)
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

	m := &Mold{
		Output: map[string]any{
			"nonexistent": ".claude/nonexistent",
		},
	}

	// "nonexistent" is not a directory in the FS, so parseMapOutput treats it
	// as a file mapping. The remaining-files loop tries to stat it and returns
	// an error since the file doesn't exist.
	_, err := ResolveFiles(m, moldFS)
	if err == nil {
		t.Fatal("expected error for nonexistent file mapping")
	}
}

func TestResolveFiles_ExcludesReservedDirs(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"ingots/foo.md":     &fstest.MapFile{Data: []byte("ingot")},
		".hidden/bar.md":    &fstest.MapFile{Data: []byte("hidden")},
	}

	m := &Mold{Output: ".claude"}

	resolved, err := ResolveFiles(m, moldFS)
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
