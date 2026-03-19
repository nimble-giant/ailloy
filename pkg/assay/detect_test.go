package assay

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDetectFS_Claude(t *testing.T) {
	fsys := fstest.MapFS{
		"CLAUDE.md": &fstest.MapFile{Data: []byte("# Instructions")},
	}
	files, err := detectFS(fsys, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected to detect CLAUDE.md")
	}
	if files[0].Platform != PlatformClaude {
		t.Errorf("expected platform claude, got %s", files[0].Platform)
	}
}

func TestDetectFS_MultiplePlatforms(t *testing.T) {
	fsys := fstest.MapFS{
		"CLAUDE.md":    &fstest.MapFile{Data: []byte("# Claude")},
		".cursorrules": &fstest.MapFile{Data: []byte("cursor rules")},
		"AGENTS.md":    &fstest.MapFile{Data: []byte("# Agents")},
	}
	files, err := detectFS(fsys, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 3 {
		t.Errorf("expected at least 3 files, got %d", len(files))
	}
}

func TestDetectFS_FilterByPlatform(t *testing.T) {
	fsys := fstest.MapFS{
		"CLAUDE.md":    &fstest.MapFile{Data: []byte("# Claude")},
		".cursorrules": &fstest.MapFile{Data: []byte("cursor rules")},
	}
	files, err := detectFS(fsys, "", []Platform{PlatformClaude})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Platform != PlatformClaude {
			t.Errorf("expected only claude files, got %s for %s", f.Platform, f.Path)
		}
	}
}

func TestDetectFS_NestedAgentsMD(t *testing.T) {
	fsys := fstest.MapFS{
		"AGENTS.md":         &fstest.MapFile{Data: []byte("# Root")},
		"src/AGENTS.md":     &fstest.MapFile{Data: []byte("# Src")},
		"src/pkg/AGENTS.md": &fstest.MapFile{Data: []byte("# Pkg")},
	}
	files, err := detectFS(fsys, "", []Platform{PlatformGeneric})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 AGENTS.md files, got %d", len(files))
	}
}

func TestDetectFS_NoneFound(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go": &fstest.MapFile{Data: []byte("package main")},
	}
	files, err := detectFS(fsys, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestDetectFS_ClaudeRules(t *testing.T) {
	fsys := fstest.MapFS{
		".claude/rules/style.md": &fstest.MapFile{Data: []byte("# Style")},
	}
	files, err := detectFS(fsys, "", []Platform{PlatformClaude})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != ".claude/rules/style.md" {
		t.Errorf("expected .claude/rules/style.md, got %s", files[0].Path)
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Test with a directory that has .git (our own repo)
	root, err := FindProjectRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestFindProjectRoot_WorktreeGitFile(t *testing.T) {
	// Simulate a git worktree where .git is a file, not a directory.
	tmp := t.TempDir()
	worktree := filepath.Join(tmp, "worktree")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a .git file (like a worktree) instead of a .git directory
	gitFile := filepath.Join(worktree, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/other/path/.git/worktrees/worktree\n"), 0644); err != nil {
		t.Fatal(err)
	}

	root, err := FindProjectRoot(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if root != worktree {
		t.Errorf("expected worktree dir %q as root, got %q", worktree, root)
	}
}

func TestFindProjectRoot_NoMarker(t *testing.T) {
	// A directory with no .git or .claude should return itself
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "empty-project")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	root, err := FindProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	// Should either return sub itself or find a parent .git — either way, not empty
	if root == "" {
		t.Error("expected non-empty root")
	}
}
