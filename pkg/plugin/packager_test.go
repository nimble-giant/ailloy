package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestPackager_Package_PathTranslation(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "out")

	files := []RenderedFile{
		{CastDest: ".claude/commands/foo.md", Content: []byte("# foo")},
		{CastDest: ".claude/skills/bar/SKILL.md", Content: []byte("# bar")},
		{CastDest: ".claude/agents/baz.md", Content: []byte("# baz")},
		{CastDest: ".claude/hooks/qux.json", Content: []byte("{}")},
		{CastDest: "AGENTS.md", Content: []byte("# agents")},
	}
	manifest := ManifestInput{Name: "test", Version: "1.2.3", Description: "d", Author: mold.Author{Name: "a"}}

	p := &Packager{OutputDir: outputDir}
	if err := p.Package(files, manifest, nil); err != nil {
		t.Fatalf("Package: %v", err)
	}

	mustExist(t, filepath.Join(outputDir, "commands", "foo.md"))
	mustExist(t, filepath.Join(outputDir, "skills", "bar", "SKILL.md"))
	mustExist(t, filepath.Join(outputDir, "agents", "baz.md"))
	mustExist(t, filepath.Join(outputDir, "hooks", "qux.json"))
	mustExist(t, filepath.Join(outputDir, "AGENTS.md"))
	mustExist(t, filepath.Join(outputDir, ".claude-plugin", "plugin.json"))
}

func TestPackager_Package_DropsUnrecognized(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "out")
	files := []RenderedFile{
		{CastDest: ".github/workflows/ci.yml", Content: []byte("name: ci")},
		{CastDest: "weird/path.md", Content: []byte("x")},
		{CastDest: ".claude/commands/keep.md", Content: []byte("k")},
	}
	manifest := ManifestInput{Name: "t", Version: "0.0.1"}

	p := &Packager{OutputDir: outputDir}
	if err := p.Package(files, manifest, nil); err != nil {
		t.Fatalf("Package: %v", err)
	}

	mustExist(t, filepath.Join(outputDir, "commands", "keep.md"))
	mustNotExist(t, filepath.Join(outputDir, ".github"))
	mustNotExist(t, filepath.Join(outputDir, "weird"))
}

func TestPackager_Package_PathCollision(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "out")
	files := []RenderedFile{
		{CastDest: ".claude/commands/dup.md", Content: []byte("a")},
		{CastDest: ".claude/commands/dup.md", Content: []byte("b")},
	}
	manifest := ManifestInput{Name: "t", Version: "0.0.1"}

	p := &Packager{OutputDir: outputDir}
	err := p.Package(files, manifest, nil)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
}

func TestPackager_Manifest_DefaultsAndOmissions(t *testing.T) {
	tests := []struct {
		name        string
		input       ManifestInput
		wantVersion string
		wantAuthor  bool
		wantDesc    bool
	}{
		{
			name:        "version defaults",
			input:       ManifestInput{Name: "x"},
			wantVersion: "0.1.0",
		},
		{
			name:        "version provided",
			input:       ManifestInput{Name: "x", Version: "9.9.9"},
			wantVersion: "9.9.9",
		},
		{
			name:        "author included",
			input:       ManifestInput{Name: "x", Author: mold.Author{Name: "alice"}},
			wantVersion: "0.1.0",
			wantAuthor:  true,
		},
		{
			name:        "description included",
			input:       ManifestInput{Name: "x", Description: "hello"},
			wantVersion: "0.1.0",
			wantDesc:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := &Packager{OutputDir: dir}
			if err := p.Package(nil, tc.input, nil); err != nil {
				t.Fatalf("Package: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(dir, ".claude-plugin", "plugin.json"))
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if raw["version"] != tc.wantVersion {
				t.Errorf("version = %v, want %v", raw["version"], tc.wantVersion)
			}
			if _, ok := raw["author"]; ok != tc.wantAuthor {
				t.Errorf("author present = %v, want %v", ok, tc.wantAuthor)
			}
			if _, ok := raw["description"]; ok != tc.wantDesc {
				t.Errorf("description present = %v, want %v", ok, tc.wantDesc)
			}
		})
	}
}

func TestPackager_Wipe_PreservesSiblings(t *testing.T) {
	dir := t.TempDir()
	pluginA := filepath.Join(dir, "plugin-a")
	pluginB := filepath.Join(dir, "plugin-b")

	if err := os.MkdirAll(pluginA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginB, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginA, "stale.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginB, "untouched.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Packager{OutputDir: pluginA}
	if err := p.Package([]RenderedFile{{CastDest: ".claude/commands/new.md", Content: []byte("new")}}, ManifestInput{Name: "a"}, nil); err != nil {
		t.Fatalf("Package: %v", err)
	}

	if _, err := os.Stat(filepath.Join(pluginA, "stale.md")); !os.IsNotExist(err) {
		t.Error("expected stale.md to be removed")
	}
	mustExist(t, filepath.Join(pluginA, "commands", "new.md"))
	mustExist(t, filepath.Join(pluginB, "untouched.md"))
}

func TestPackager_TargetIsFile_Errors(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(target, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Packager{OutputDir: target}
	err := p.Package(nil, ManifestInput{Name: "x"}, nil)
	if err == nil {
		t.Fatal("expected error when target is a file")
	}
}

func TestPackager_README_FromMold(t *testing.T) {
	dir := t.TempDir()
	p := &Packager{OutputDir: dir}
	if err := p.Package(nil, ManifestInput{Name: "x"}, []byte("# my readme")); err != nil {
		t.Fatalf("Package: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read readme: %v", err)
	}
	if string(data) != "# my readme" {
		t.Errorf("readme content = %q", data)
	}
}

func TestPackager_README_OmittedWhenNil(t *testing.T) {
	dir := t.TempDir()
	p := &Packager{OutputDir: dir}
	if err := p.Package(nil, ManifestInput{Name: "x"}, nil); err != nil {
		t.Fatalf("Package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); !os.IsNotExist(err) {
		t.Error("expected README.md not to be written when nil")
	}
}

func TestPackager_ParentDirCreated(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "deep", "plugin")
	p := &Packager{OutputDir: target}
	if err := p.Package(nil, ManifestInput{Name: "x"}, nil); err != nil {
		t.Fatalf("Package: %v", err)
	}
	mustExist(t, filepath.Join(target, ".claude-plugin", "plugin.json"))
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s NOT to exist", path)
	}
}
