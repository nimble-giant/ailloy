package blanks

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// testReader creates a MoldReader backed by an fstest.MapFS with convention-based structure.
func testReader() *MoldReader {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`output:
  commands: .claude/commands
  skills: .claude/skills
  workflows:
    dest: .github/workflows
    process: false
board: Engineering
org: test-org
`)},
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`- name: org
  type: string
  required: true
`)},
		"commands/create-issue.md": &fstest.MapFile{Data: []byte("# create-issue\nCreate a GitHub issue.")},
		"commands/open-pr.md":      &fstest.MapFile{Data: []byte("# open-pr\nOpen a pull request.")},
		"skills/brainstorm.md":     &fstest.MapFile{Data: []byte("# brainstorm\nStructured brainstorming.")},
		"workflows/ci.yml":         &fstest.MapFile{Data: []byte("name: CI\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest")},
	}
	return NewMoldReader(fsys)
}

func TestResolveFiles(t *testing.T) {
	reader := testReader()
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error loading flux: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("unexpected error resolving files: %v", err)
	}

	if len(resolved) == 0 {
		t.Fatal("expected resolved files, got none")
	}

	// Verify we can read each resolved file
	for _, rf := range resolved {
		content, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			t.Errorf("failed to read resolved file %s: %v", rf.SrcPath, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("resolved file %s is empty", rf.SrcPath)
		}
	}
}

func TestResolveFiles_OutputMapping(t *testing.T) {
	reader := testReader()
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"commands/create-issue.md": ".claude/commands/create-issue.md",
		"commands/open-pr.md":      ".claude/commands/open-pr.md",
		"skills/brainstorm.md":     ".claude/skills/brainstorm.md",
		"workflows/ci.yml":         ".github/workflows/ci.yml",
	}

	if len(resolved) != len(expected) {
		t.Fatalf("expected %d resolved files, got %d", len(expected), len(resolved))
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

func TestResolveFiles_ProcessFlag(t *testing.T) {
	reader := testReader()
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, rf := range resolved {
		if rf.SrcPath == "workflows/ci.yml" {
			if rf.Process {
				t.Error("expected workflows to have Process=false")
			}
		} else {
			if !rf.Process {
				t.Errorf("expected %s to have Process=true", rf.SrcPath)
			}
		}
	}
}

func TestLoadManifest(t *testing.T) {
	reader := testReader()
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifest.Name != "test-mold" {
		t.Errorf("expected name=test-mold, got %q", manifest.Name)
	}
}

func TestLoadFluxDefaults(t *testing.T) {
	reader := testReader()
	vals, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) == 0 {
		t.Fatal("expected non-empty flux defaults")
	}
	if vals["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %q", vals["board"])
	}
}

func TestLoadFluxSchema(t *testing.T) {
	reader := testReader()
	schema, err := reader.LoadFluxSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
	if len(schema) != 1 {
		t.Fatalf("expected 1 schema entry, got %d", len(schema))
	}
	if schema[0].Name != "org" {
		t.Errorf("expected schema[0].name=org, got %q", schema[0].Name)
	}
	if !schema[0].Required {
		t.Error("expected org to be required")
	}
}

func TestNewMoldReaderFromPath(t *testing.T) {
	// Test with a non-existent directory
	_, err := NewMoldReaderFromPath("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}

	// Test with a valid temp directory
	tmpDir := t.TempDir()
	reader, err := NewMoldReaderFromPath(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Error("expected non-nil reader")
	}
	if reader.FS() == nil {
		t.Error("expected non-nil FS")
	}
}
