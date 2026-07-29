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

func TestDetectFS_CodexSkillsAndAgents(t *testing.T) {
	fsys := fstest.MapFS{
		".agents/skills/reviewing/SKILL.md":                          &fstest.MapFile{Data: []byte("---\nname: reviewing\ndescription: Reviews code.\n---\n")},
		".agents/skills/reviewing/references/checks.md":              &fstest.MapFile{Data: []byte("# Checks")},
		".agents/skills/reviewing/scripts/review.sh":                 &fstest.MapFile{Data: []byte("#!/bin/sh")},
		".codex/agents/reviewer.toml":                                &fstest.MapFile{Data: []byte(`name = "reviewer"`)},
		"services/api/.agents/skills/testing/SKILL.md":               &fstest.MapFile{Data: []byte("---\nname: testing\ndescription: Tests APIs.\n---\n")},
		"services/api/.agents/skills/testing/references/fixtures.md": &fstest.MapFile{Data: []byte("# Fixtures")},
		"services/api/.agents/skills/testing/scripts/test.sh":        &fstest.MapFile{Data: []byte("#!/bin/sh")},
		"services/api/AGENTS.md":                                     &fstest.MapFile{Data: []byte("# API instructions")},
	}

	files, err := detectFS(fsys, "", []Platform{PlatformCodex})
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]Platform, len(files))
	for _, f := range files {
		got[f.Path] = f.Platform
	}
	for _, path := range []string{
		".agents/skills/reviewing/SKILL.md",
		".agents/skills/reviewing/references/checks.md",
		".codex/agents/reviewer.toml",
		"services/api/.agents/skills/testing/SKILL.md",
		"services/api/.agents/skills/testing/references/fixtures.md",
		"services/api/AGENTS.md",
	} {
		if got[path] != PlatformCodex {
			t.Errorf("%s: platform = %q, want %q", path, got[path], PlatformCodex)
		}
	}
	if _, ok := got[".agents/skills/reviewing/scripts/review.sh"]; ok {
		t.Error("non-markdown skill support files should not be linted")
	}
	if _, ok := got["services/api/.agents/skills/testing/scripts/test.sh"]; ok {
		t.Error("non-markdown scoped skill support files should not be linted")
	}
}

func TestDetect_RejectsSymlinksOutsideProjectRoot(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()

	outsideAgents := filepath.Join(outside, "AGENTS.md")
	if err := os.WriteFile(outsideAgents, []byte("# Outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	outsideSkill := filepath.Join(outside, "SKILL.md")
	if err := os.WriteFile(outsideSkill, []byte("---\nname: outside\ndescription: Outside.\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outsideSkillDir := filepath.Join(outside, "skill-dir")
	if err := os.MkdirAll(outsideSkillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideSkillDir, "SKILL.md"), []byte("---\nname: outside-dir\ndescription: Outside directory.\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	skillDir := filepath.Join(project, ".agents", "skills", "outside")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	for target, link := range map[string]string{
		outsideAgents: filepath.Join(project, "AGENTS.md"),
		outsideSkill:  filepath.Join(skillDir, "SKILL.md"),
		outsideSkillDir: filepath.Join(
			project, ".agents", "skills", "outside-dir",
		),
	} {
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}

	files, err := Detect(project, []Platform{PlatformCodex})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("detected files through symlinks outside project root: %v", files)
	}
}

func TestDetect_FollowsSymlinkedSkillDirectoriesWithinProject(t *testing.T) {
	project := t.TempDir()
	actualSkill := filepath.Join(project, "shared", "reviewing")
	if err := os.MkdirAll(filepath.Join(actualSkill, "references"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(actualSkill, "SKILL.md"),
		[]byte("---\nname: reviewing\ndescription: Reviews code.\n---\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(actualSkill, "references", "checks.md"), []byte("# Checks\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	skillsDir := filepath.Join(project, ".agents", "skills")
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		filepath.Join("..", "..", "shared", "reviewing"),
		filepath.Join(skillsDir, "reviewing"),
	); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	files, err := Detect(project, []Platform{PlatformCodex})
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool, len(files))
	for _, file := range files {
		got[filepath.ToSlash(file.Path)] = true
	}
	for _, path := range []string{
		".agents/skills/reviewing/SKILL.md",
		".agents/skills/reviewing/references/checks.md",
	} {
		if !got[path] {
			t.Errorf("expected symlinked skill file %s to be detected; got %v", path, got)
		}
	}
}

func TestDetect_DoesNotRecurseThroughSymlinkCycles(t *testing.T) {
	project := t.TempDir()
	skillsDir := filepath.Join(project, ".agents", "skills")
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		filepath.Join("..", ".."),
		filepath.Join(skillsDir, "project-root"),
	); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	files, err := Detect(project, []Platform{PlatformCodex})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected cyclic skill symlink to be ignored; got %v", files)
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

func TestFindProjectRoot_CodexMarkers(t *testing.T) {
	for _, marker := range []string{".agents", ".codex"} {
		t.Run(marker, func(t *testing.T) {
			tmp := t.TempDir()
			project := filepath.Join(tmp, "project")
			nested := filepath.Join(project, "src", "pkg")
			if err := os.MkdirAll(filepath.Join(project, marker), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(nested, 0755); err != nil {
				t.Fatal(err)
			}

			root, err := FindProjectRoot(nested)
			if err != nil {
				t.Fatal(err)
			}
			if root != project {
				t.Errorf("root = %q, want %q", root, project)
			}
		})
	}
}

func TestFindProjectRoot_PrefersEnclosingGitRoot(t *testing.T) {
	for _, marker := range []string{".claude", ".agents", ".codex"} {
		t.Run(marker, func(t *testing.T) {
			tmp := t.TempDir()
			project := filepath.Join(tmp, "project")
			scoped := filepath.Join(project, "services", "api")
			nested := filepath.Join(scoped, "src")
			if err := os.MkdirAll(filepath.Join(project, ".git"), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(scoped, marker), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(nested, 0o750); err != nil {
				t.Fatal(err)
			}

			root, err := FindProjectRoot(nested)
			if err != nil {
				t.Fatal(err)
			}
			if root != project {
				t.Errorf("root = %q, want enclosing git root %q", root, project)
			}
		})
	}
}

func TestFindProjectRoot_DoesNotPromoteHomeToolDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	resolvedHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(resolvedHome) != filepath.Clean(home) {
		t.Skipf("os.UserHomeDir does not use HOME on this platform: got %q, set %q", resolvedHome, home)
	}

	for _, marker := range []string{".claude", ".agents", ".codex"} {
		if err := os.MkdirAll(filepath.Join(home, marker), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	start := filepath.Join(home, "work", "unmarked-project", "src")
	if err := os.MkdirAll(start, 0o750); err != nil {
		t.Fatal(err)
	}

	root, err := FindProjectRoot(start)
	if err != nil {
		t.Fatal(err)
	}
	if root != start {
		t.Errorf("root = %q, want original start directory %q", root, start)
	}
}
