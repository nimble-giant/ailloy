package mold

import (
	"testing"
	"testing/fstest"
)

func TestParseIgnoreFile(t *testing.T) {
	input := `# Mold documentation - not for target projects
docs/
examples/

# Specific files
CONTRIBUTING.md
*.example
`

	patterns := parseIgnoreFile([]byte(input))

	expected := []string{
		"docs/",
		"examples/",
		"CONTRIBUTING.md",
		"*.example",
	}

	if len(patterns) != len(expected) {
		t.Fatalf("expected %d patterns, got %d: %v", len(expected), len(patterns), patterns)
	}

	for i, p := range patterns {
		if p != expected[i] {
			t.Errorf("pattern[%d]: expected %q, got %q", i, expected[i], p)
		}
	}
}

func TestParseIgnoreFile_Empty(t *testing.T) {
	patterns := parseIgnoreFile([]byte(""))
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns for empty file, got %d", len(patterns))
	}
}

func TestParseIgnoreFile_OnlyComments(t *testing.T) {
	input := "# comment 1\n# comment 2\n"
	patterns := parseIgnoreFile([]byte(input))
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns for comments-only file, got %d", len(patterns))
	}
}

func TestMatchIgnorePattern(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		// Directory patterns with trailing slash
		{"dir slash matches subfile", "docs/guide.md", "docs/", true},
		{"dir slash matches nested", "docs/sub/deep.md", "docs/", true},
		{"dir slash no match other", "commands/hello.md", "docs/", false},
		{"dir slash no match prefix", "mydocs/hello.md", "docs/", false},

		// Directory patterns with **
		{"dir glob matches subfile", "docs/guide.md", "docs/**", true},
		{"dir glob matches nested", "docs/sub/deep.md", "docs/**", true},
		{"dir glob no match other", "commands/hello.md", "docs/**", false},

		// Exact file match
		{"exact match root file", "CONTRIBUTING.md", "CONTRIBUTING.md", true},
		{"exact match no match", "README.md", "CONTRIBUTING.md", false},

		// Glob pattern against basename
		{"glob basename match", "examples/test.example", "*.example", true},
		{"glob basename match root", "config.example", "*.example", true},
		{"glob basename no match", "config.md", "*.example", false},

		// Glob pattern against full path
		{"glob full path", "docs/config.md", "docs/*.md", true},
		{"glob full path no match", "commands/hello.md", "docs/*.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIgnorePattern(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchIgnorePattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	patterns := []string{"docs/", "*.example", "CONTRIBUTING.md"}

	tests := []struct {
		path string
		want bool
	}{
		{"docs/guide.md", true},
		{"docs/sub/deep.md", true},
		{"config.example", true},
		{"CONTRIBUTING.md", true},
		{"commands/hello.md", false},
		{"AGENTS.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := shouldIgnore(tt.path, patterns)
			if got != tt.want {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFilterIgnored(t *testing.T) {
	files := []ResolvedFile{
		{SrcPath: "commands/hello.md", DestPath: "commands/hello.md", Process: true},
		{SrcPath: "docs/guide.md", DestPath: "docs/guide.md", Process: true},
		{SrcPath: "CONTRIBUTING.md", DestPath: "CONTRIBUTING.md", Process: true},
		{SrcPath: "AGENTS.md", DestPath: "AGENTS.md", Process: true},
	}

	patterns := []string{"docs/", "CONTRIBUTING.md"}
	result := filterIgnored(files, patterns)

	if len(result) != 2 {
		t.Fatalf("expected 2 files after filtering, got %d", len(result))
	}

	expected := map[string]bool{
		"commands/hello.md": true,
		"AGENTS.md":         true,
	}

	for _, f := range result {
		if !expected[f.SrcPath] {
			t.Errorf("unexpected file after filtering: %s", f.SrcPath)
		}
	}
}

func TestLoadIgnorePatterns_BothSources(t *testing.T) {
	moldFS := fstest.MapFS{
		".ailloyignore": &fstest.MapFile{Data: []byte("docs/\n*.example\n")},
	}

	manifest := &Mold{
		Ignore: []string{"examples/", "CONTRIBUTING.md"},
	}

	patterns := LoadIgnorePatterns(moldFS, manifest)

	if len(patterns) != 4 {
		t.Fatalf("expected 4 patterns, got %d: %v", len(patterns), patterns)
	}

	expected := map[string]bool{
		"docs/":           true,
		"*.example":       true,
		"examples/":       true,
		"CONTRIBUTING.md": true,
	}

	for _, p := range patterns {
		if !expected[p] {
			t.Errorf("unexpected pattern: %q", p)
		}
	}
}

func TestLoadIgnorePatterns_OnlyAilloyignore(t *testing.T) {
	moldFS := fstest.MapFS{
		".ailloyignore": &fstest.MapFile{Data: []byte("docs/\n")},
	}

	patterns := LoadIgnorePatterns(moldFS, nil)

	if len(patterns) != 1 || patterns[0] != "docs/" {
		t.Errorf("expected [docs/], got %v", patterns)
	}
}

func TestLoadIgnorePatterns_OnlyManifest(t *testing.T) {
	moldFS := fstest.MapFS{}

	manifest := &Mold{Ignore: []string{"examples/"}}
	patterns := LoadIgnorePatterns(moldFS, manifest)

	if len(patterns) != 1 || patterns[0] != "examples/" {
		t.Errorf("expected [examples/], got %v", patterns)
	}
}

func TestLoadIgnorePatterns_NeitherSource(t *testing.T) {
	moldFS := fstest.MapFS{}

	patterns := LoadIgnorePatterns(moldFS, nil)

	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(patterns))
	}
}

func TestResolveFiles_WithIgnorePatterns_Identity(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"docs/guide.md":     &fstest.MapFile{Data: []byte("guide")},
		"CONTRIBUTING.md":   &fstest.MapFile{Data: []byte("contrib")},
		"AGENTS.md":         &fstest.MapFile{Data: []byte("agents")},
	}

	patterns := []string{"docs/", "CONTRIBUTING.md"}
	resolved, err := ResolveFiles(nil, moldFS, WithIgnorePatterns(patterns))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// commands/hello.md and AGENTS.md should remain; docs/ and CONTRIBUTING.md ignored
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d: %v", len(resolved), resolved)
	}

	found := make(map[string]bool)
	for _, rf := range resolved {
		found[rf.SrcPath] = true
	}

	if !found["commands/hello.md"] {
		t.Error("expected commands/hello.md to be resolved")
	}
	if !found["AGENTS.md"] {
		t.Error("expected AGENTS.md to be resolved")
	}
}

func TestResolveFiles_WithIgnorePatterns_StringOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"docs/guide.md":     &fstest.MapFile{Data: []byte("guide")},
	}

	patterns := []string{"docs/"}
	resolved, err := ResolveFiles(".claude", moldFS, WithIgnorePatterns(patterns))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file, got %d", len(resolved))
	}

	if resolved[0].SrcPath != "commands/hello.md" {
		t.Errorf("expected commands/hello.md, got %s", resolved[0].SrcPath)
	}
}

func TestResolveFiles_WithIgnorePatterns_MapOutput(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md":   &fstest.MapFile{Data: []byte("hello")},
		"docs/guide.md":       &fstest.MapFile{Data: []byte("guide")},
		"examples/example.md": &fstest.MapFile{Data: []byte("example")},
	}

	output := map[string]any{
		"commands": ".claude/commands",
		"docs":     "project-docs",
		"examples": "project-examples",
	}

	patterns := []string{"docs/", "examples/"}
	resolved, err := ResolveFiles(output, moldFS, WithIgnorePatterns(patterns))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved file, got %d", len(resolved))
	}

	if resolved[0].SrcPath != "commands/hello.md" {
		t.Errorf("expected commands/hello.md, got %s", resolved[0].SrcPath)
	}
}

func TestResolveFiles_WithIgnorePatterns_GlobMatch(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/hello.md":     &fstest.MapFile{Data: []byte("hello")},
		"commands/test.example": &fstest.MapFile{Data: []byte("example")},
		"skills/helper.md":      &fstest.MapFile{Data: []byte("helper")},
		"skills/draft.example":  &fstest.MapFile{Data: []byte("draft")},
	}

	patterns := []string{"*.example"}
	resolved, err := ResolveFiles(nil, moldFS, WithIgnorePatterns(patterns))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(resolved))
	}

	found := make(map[string]bool)
	for _, rf := range resolved {
		found[rf.SrcPath] = true
	}

	if !found["commands/hello.md"] {
		t.Error("expected commands/hello.md")
	}
	if !found["skills/helper.md"] {
		t.Error("expected skills/helper.md")
	}
}

func TestResolveFiles_WithoutIgnorePatterns_Unchanged(t *testing.T) {
	// Verify backward compatibility: no options = no filtering.
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		"docs/guide.md":     &fstest.MapFile{Data: []byte("guide")},
	}

	resolved, err := ResolveFiles(nil, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved files (no filtering), got %d", len(resolved))
	}
}

func TestResolveFiles_AilloyignoreNotCast(t *testing.T) {
	// .ailloyignore itself should not be included in resolved files
	// because it's in reservedRootFiles.
	moldFS := fstest.MapFS{
		"commands/hello.md": &fstest.MapFile{Data: []byte("hello")},
		".ailloyignore":     &fstest.MapFile{Data: []byte("docs/\n")},
	}

	resolved, err := ResolveFiles(nil, moldFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, rf := range resolved {
		if rf.SrcPath == ".ailloyignore" {
			t.Error(".ailloyignore should not be included in resolved files")
		}
	}
}
