package assay

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// platformPatterns maps each platform to the file glob patterns that identify it.
var platformPatterns = map[Platform][]string{
	PlatformClaude:  {"CLAUDE.md", ".claude/CLAUDE.md", ".claude/rules/*.md", "CLAUDE.local.md"},
	PlatformCursor:  {".cursor/rules/*.md", ".cursorrules"},
	PlatformCodex:   {"AGENTS.md", "codex.md", ".codex/agents/*.toml"},
	PlatformCopilot: {".github/copilot-instructions.md"},
	PlatformGeneric: {"AGENTS.md"},
}

// Detect discovers AI instruction files under rootDir.
// If platforms is non-empty, only files for those platforms are returned.
func Detect(rootDir string, platforms []Platform) ([]DetectedFile, error) {
	return detectFS(os.DirFS(rootDir), rootDir, platforms)
}

// detectFS performs file detection against an fs.FS for testability.
// realRoot enforces symlink containment when fsys is backed by an os.DirFS.
func detectFS(fsys fs.FS, realRoot string, platforms []Platform) ([]DetectedFile, error) {
	targetPlatforms := platforms
	if len(targetPlatforms) == 0 {
		targetPlatforms = AllPlatforms()
	}

	seen := make(map[string]bool)
	var files []DetectedFile

	for _, plat := range targetPlatforms {
		patterns, ok := platformPatterns[plat]
		if !ok {
			continue
		}
		for _, pattern := range patterns {
			matches, err := fs.Glob(fsys, pattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if seen[match] {
					continue
				}
				if !pathWithinRoot(realRoot, match) {
					continue
				}

				content, err := fs.ReadFile(fsys, match)
				if err != nil {
					continue
				}

				seen[match] = true
				files = append(files, DetectedFile{
					Path:     match,
					Platform: plat,
					Content:  content,
				})
			}
		}
	}

	// Codex discovers Agent Skills from .agents/skills directories at every
	// level between the working directory and repository root. Walk the whole
	// repository so scoped skills and their markdown references are linted.
	if containsPlatform(targetPlatforms, PlatformCodex) {
		activeSkillRoots := make(map[string]bool)
		if realRoot != "" {
			if resolvedRoot, ok := resolvedPathWithinRoot(realRoot, "."); ok {
				activeSkillRoots[resolvedRoot] = true
			}
		}
		var walkCodexSkills func(string) error
		walkCodexSkills = func(root string) error {
			return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if shouldSkipDir(d.Name()) {
						return fs.SkipDir
					}
					return nil
				}
				// WalkDir does not follow symlinks it encounters below its root.
				// Codex does follow symlinked skill folders, so recurse when the
				// logical skill root resolves to a contained directory.
				if d.Type()&fs.ModeSymlink != 0 && isCodexSkillDirectoryPath(path) {
					if resolved, ok := resolvedSymlinkDirectoryWithinRoot(realRoot, path); ok {
						if activeSkillRoots[resolved] {
							return nil
						}
						activeSkillRoots[resolved] = true
						walkErr := walkCodexSkills(path)
						delete(activeSkillRoots, resolved)
						return walkErr
					}
					return nil
				}
				if !isCodexSkillMarkdownPath(path) || seen[path] || !pathWithinRoot(realRoot, path) {
					return nil
				}
				content, err := fs.ReadFile(fsys, path)
				if err != nil {
					return nil
				}
				seen[path] = true
				files = append(files, DetectedFile{
					Path:     path,
					Platform: PlatformCodex,
					Content:  content,
				})
				return nil
			})
		}
		err := walkCodexSkills(".")
		if err != nil {
			return files, err
		}
	}

	// Search for nested AGENTS.md files (per AGENTS.md spec)
	agentsPlatform := PlatformGeneric
	searchNestedAgents := containsPlatform(targetPlatforms, PlatformGeneric)
	if !searchNestedAgents && containsPlatform(targetPlatforms, PlatformCodex) {
		agentsPlatform = PlatformCodex
		searchNestedAgents = true
	}
	if searchNestedAgents {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			// Skip common non-project directories
			if d.IsDir() {
				if shouldSkipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() == "AGENTS.md" && path != "AGENTS.md" && !seen[path] && pathWithinRoot(realRoot, path) {
				seen[path] = true
				content, err := fs.ReadFile(fsys, path)
				if err != nil {
					return nil
				}
				files = append(files, DetectedFile{
					Path:     path,
					Platform: agentsPlatform,
					Content:  content,
				})
			}
			return nil
		})
		if err != nil {
			return files, err
		}
	}

	return files, nil
}

// FindProjectRoot walks up from startDir looking for an enclosing .git root,
// falling back to the nearest supported tool-specific instruction directory
// when the path is not inside a git checkout.
// Recognizes both .git directories (normal repos) and .git files (worktrees).
// Returns startDir if no marker is found.
func FindProjectRoot(startDir string) (string, error) {
	start, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		home, _ = filepath.Abs(home)
	}

	dir := start
	var toolRoot string
	for {
		// .git can be a directory (normal repo) or a file (worktree with
		// "gitdir: ..." content). Both indicate a project root.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		// Remember the nearest tool-specific marker while continuing upward in
		// case this is a nested Codex scope inside a git repository. User-level
		// ~/.claude, ~/.agents, and ~/.codex directories are global config, not
		// evidence that the entire home directory is one project.
		if toolRoot == "" && (home == "" || filepath.Clean(dir) != filepath.Clean(home)) {
			for _, marker := range []string{".claude", ".agents", ".codex"} {
				if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
					toolRoot = dir
					break
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			if toolRoot != "" {
				return toolRoot, nil
			}
			return start, nil
		}
		dir = parent
	}
}

func containsPlatform(platforms []Platform, target Platform) bool {
	for _, p := range platforms {
		if p == target {
			return true
		}
	}
	return false
}

func isCodexSkillMarkdownPath(filePath string) bool {
	if filepath.Ext(filePath) != ".md" {
		return false
	}

	parts := strings.Split(filepath.ToSlash(filePath), "/")
	for i := 0; i+3 < len(parts); i++ {
		if parts[i] == ".agents" && parts[i+1] == "skills" {
			return true
		}
	}
	return false
}

func isCodexSkillDirectoryPath(dirPath string) bool {
	parts := strings.Split(filepath.ToSlash(dirPath), "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == ".agents" && parts[i+1] == "skills" && i+3 == len(parts) {
			return true
		}
	}
	return false
}

func shouldSkipDir(name string) bool {
	return name == "node_modules" || name == "vendor" || name == ".git"
}

func resolvedSymlinkDirectoryWithinRoot(realRoot, relativePath string) (string, bool) {
	if realRoot == "" {
		return "", false
	}
	resolved, ok := resolvedPathWithinRoot(realRoot, relativePath)
	if !ok {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return resolved, true
}

// pathWithinRoot rejects files whose symlinks escape the project root.
// An empty realRoot denotes an abstract fs.FS, where paths are already scoped.
func pathWithinRoot(realRoot, relativePath string) bool {
	if realRoot == "" {
		return true
	}
	_, ok := resolvedPathWithinRoot(realRoot, relativePath)
	return ok
}

func resolvedPathWithinRoot(realRoot, relativePath string) (string, bool) {
	absRoot, err := filepath.Abs(realRoot)
	if err != nil {
		return "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", false
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, relativePath))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return resolvedPath, true
}
