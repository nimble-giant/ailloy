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
	PlatformCodex:   {"AGENTS.md", "codex.md"},
	PlatformCopilot: {".github/copilot-instructions.md"},
	PlatformGeneric: {"AGENTS.md"},
}

// Detect discovers AI instruction files under rootDir.
// If platforms is non-empty, only files for those platforms are returned.
func Detect(rootDir string, platforms []Platform) ([]DetectedFile, error) {
	return detectFS(os.DirFS(rootDir), rootDir, platforms)
}

// detectFS performs file detection against an fs.FS for testability.
// realRoot is used for reading file content via os when fsys is an os.DirFS.
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
				seen[match] = true

				// Validate symlinks stay within project root
				if realRoot != "" {
					fullPath := filepath.Join(realRoot, match)
					resolved, err := filepath.EvalSymlinks(fullPath)
					if err != nil {
						continue
					}
					absRoot, err := filepath.Abs(realRoot)
					if err != nil {
						continue
					}
					// Resolve symlinks in the root path too (e.g. /var -> /private/var on macOS)
					absRoot, err = filepath.EvalSymlinks(absRoot)
					if err != nil {
						continue
					}
					if !strings.HasPrefix(resolved, absRoot+string(filepath.Separator)) && resolved != absRoot {
						continue
					}
				}

				content, err := fs.ReadFile(fsys, match)
				if err != nil {
					continue
				}

				files = append(files, DetectedFile{
					Path:     match,
					Platform: plat,
					Content:  content,
				})
			}
		}
	}

	// Search for nested AGENTS.md files (per AGENTS.md spec)
	if containsPlatform(targetPlatforms, PlatformGeneric) {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			// Skip common non-project directories
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "vendor" || name == ".git" {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() == "AGENTS.md" && path != "AGENTS.md" && !seen[path] {
				seen[path] = true
				content, err := fs.ReadFile(fsys, path)
				if err != nil {
					return nil
				}
				files = append(files, DetectedFile{
					Path:     path,
					Platform: PlatformGeneric,
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

// FindProjectRoot walks up from startDir looking for .git or .claude markers.
// Recognizes both .git directories (normal repos) and .git files (worktrees).
// Returns startDir if no marker is found.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		// .git can be a directory (normal repo) or a file (worktree with
		// "gitdir: ..." content). Both indicate a project root.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		// .claude directory is also a valid project root marker.
		if info, err := os.Stat(filepath.Join(dir, ".claude")); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, use the original startDir
			return filepath.Abs(startDir)
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
