package mold

import (
	"io/fs"
	"path"
	"strings"
)

// LoadIgnorePatterns reads ignore patterns from both .ailloyignore file
// and the mold manifest's ignore field. Patterns from both sources are merged.
func LoadIgnorePatterns(moldFS fs.FS, manifest *Mold) []string {
	var patterns []string

	// Load .ailloyignore if present.
	data, err := fs.ReadFile(moldFS, ".ailloyignore")
	if err == nil {
		patterns = append(patterns, parseIgnoreFile(data)...)
	}

	// Add manifest ignore patterns.
	if manifest != nil {
		patterns = append(patterns, manifest.Ignore...)
	}

	return patterns
}

// parseIgnoreFile parses a .ailloyignore file into patterns.
// Empty lines and lines starting with # are skipped.
func parseIgnoreFile(data []byte) []string {
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// shouldIgnore returns true if the given path matches any ignore pattern.
func shouldIgnore(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchIgnorePattern(filePath, pattern) {
			return true
		}
	}
	return false
}

// matchIgnorePattern checks if a file path matches a single ignore pattern.
//
// Supported pattern forms:
//   - "docs/" or "docs/**" — matches the directory and everything under it
//   - "CONTRIBUTING.md"    — exact match against full path or basename
//   - "*.example"          — glob match against full path or basename
func matchIgnorePattern(filePath, pattern string) bool {
	// Directory pattern: "docs/" or "docs/**"
	var dir string
	switch {
	case strings.HasSuffix(pattern, "/"):
		dir = strings.TrimSuffix(pattern, "/")
	case strings.HasSuffix(pattern, "/**"):
		dir = strings.TrimSuffix(pattern, "/**")
	}

	if dir != "" {
		return filePath == dir || strings.HasPrefix(filePath, dir+"/")
	}

	// Exact match against full path.
	if matched, _ := path.Match(pattern, filePath); matched {
		return true
	}

	// Match against just the filename (for patterns like "*.example").
	if matched, _ := path.Match(pattern, path.Base(filePath)); matched {
		return true
	}

	return false
}

// filterIgnored removes resolved files that match any ignore pattern.
func filterIgnored(files []ResolvedFile, patterns []string) []ResolvedFile {
	var result []ResolvedFile
	for _, f := range files {
		if !shouldIgnore(f.SrcPath, patterns) {
			result = append(result, f)
		}
	}
	return result
}
