package smelt

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
)

// createStuffedFixture creates a stuffed binary from a mold fixture and
// returns a StuffFS backed by its contents.
func createStuffedFixture(t *testing.T) *StuffFS {
	t.Helper()

	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("creating fixture binary: %v", err)
	}

	sfs, err := stuffbin.UnStuff(outputPath)
	if err != nil {
		t.Fatalf("unstuffing fixture: %v", err)
	}

	return NewStuffFS(sfs)
}

func TestStuffFS_Open(t *testing.T) {
	sfs := createStuffedFixture(t)

	f, err := sfs.Open("mold.yaml")
	if err != nil {
		t.Fatalf("Open mold.yaml: %v", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if info.Name() != "mold.yaml" {
		t.Errorf("expected name mold.yaml, got %s", info.Name())
	}
	if info.IsDir() {
		t.Error("expected regular file, got directory")
	}
	if info.Size() <= 0 {
		t.Error("expected positive size")
	}
}

func TestStuffFS_Open_NestedFile(t *testing.T) {
	sfs := createStuffedFixture(t)

	f, err := sfs.Open("commands/hello.md")
	if err != nil {
		t.Fatalf("Open nested file: %v", err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 128)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(string(buf[:n]), "# Hello") {
		t.Errorf("expected content '# Hello', got %q", string(buf[:n]))
	}
}

func TestStuffFS_ReadFile(t *testing.T) {
	sfs := createStuffedFixture(t)

	data, err := sfs.ReadFile("mold.yaml")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "apiVersion: v1") {
		t.Errorf("expected mold.yaml content, got %q", string(data))
	}
}

func TestStuffFS_ReadFile_Nested(t *testing.T) {
	sfs := createStuffedFixture(t)

	data, err := sfs.ReadFile("skills/helper.md")
	if err != nil {
		t.Fatalf("ReadFile nested: %v", err)
	}
	if !strings.Contains(string(data), "# Helper") {
		t.Errorf("expected skill content, got %q", string(data))
	}
}

func TestStuffFS_NotFound(t *testing.T) {
	sfs := createStuffedFixture(t)

	_, err := sfs.Open("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected *fs.PathError, got %T", err)
	}

	_, err = sfs.ReadFile("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestStuffFS_InvalidPath(t *testing.T) {
	sfs := createStuffedFixture(t)

	_, err := sfs.Open("/absolute/path")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}

	_, err = sfs.ReadFile("../escape")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestStuffFS_ReadDir(t *testing.T) {
	sfs := createStuffedFixture(t)

	f, err := sfs.Open("commands")
	if err != nil {
		t.Fatalf("Open directory: %v", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Fatal("expected ReadDirFile interface")
	}

	entries, err := rdf.ReadDir(-1)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	found := false
	for _, e := range entries {
		if e.Name() == "hello.md" {
			found = true
			if e.IsDir() {
				t.Error("hello.md should not be a directory")
			}
		}
	}
	if !found {
		t.Error("expected hello.md in command directory listing")
	}
}

func TestStuffFS_ReadDir_Root(t *testing.T) {
	sfs := createStuffedFixture(t)

	f, err := sfs.Open(".")
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	defer func() { _ = f.Close() }()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Fatal("expected ReadDirFile interface")
	}

	entries, err := rdf.ReadDir(-1)
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["mold.yaml"] {
		t.Error("expected mold.yaml in root listing")
	}
	if !names["commands"] {
		t.Error("expected commands directory in root listing")
	}
}

func TestStuffFS_Stat(t *testing.T) {
	sfs := createStuffedFixture(t)

	// File stat.
	info, err := sfs.Stat("mold.yaml")
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if info.IsDir() {
		t.Error("mold.yaml should not be a directory")
	}
	if info.Size() <= 0 {
		t.Error("expected positive size")
	}

	// Directory stat.
	info, err = sfs.Stat("commands")
	if err != nil {
		t.Fatalf("Stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("commands should be a directory")
	}

	// Nonexistent.
	_, err = sfs.Stat("nope.txt")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestStuffFS_FsReadFile(t *testing.T) {
	sfs := createStuffedFixture(t)

	// Verify the adapter works with the standard library fs.ReadFile.
	data, err := fs.ReadFile(sfs, "workflows/ci.yml")
	if err != nil {
		t.Fatalf("fs.ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "name: CI") {
		t.Errorf("expected workflow content, got %q", string(data))
	}
}

func TestStuffFS_FsReadDir(t *testing.T) {
	sfs := createStuffedFixture(t)

	// Verify the adapter works with the standard library fs.ReadDir.
	entries, err := fs.ReadDir(sfs, "skills")
	if err != nil {
		t.Fatalf("fs.ReadDir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected entries in skills directory")
	}

	found := false
	for _, e := range entries {
		if e.Name() == "helper.md" {
			found = true
		}
	}
	if !found {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected helper.md in skills, got: %v", names)
	}
}

func TestStuffFS_MoldReaderIntegration(t *testing.T) {
	// End-to-end: Package a mold, unstuff it, and use it with the
	// standard fs.FS APIs that MoldReader relies on.
	moldDir := t.TempDir()
	writeMoldFixtureHelmStyle(t, moldDir, true)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("packaging: %v", err)
	}

	fsys, err := UnstuffFS(outputPath)
	if err != nil {
		t.Fatalf("unstuffing: %v", err)
	}

	// Simulate MoldReader operations.
	// LoadManifest reads mold.yaml.
	moldData, err := fs.ReadFile(fsys, "mold.yaml")
	if err != nil {
		t.Fatalf("reading mold.yaml: %v", err)
	}
	if !strings.Contains(string(moldData), "helm-style") {
		t.Error("expected mold.yaml to contain mold name")
	}

	// LoadFluxDefaults reads flux.yaml.
	fluxData, err := fs.ReadFile(fsys, "flux.yaml")
	if err != nil {
		t.Fatalf("reading flux.yaml: %v", err)
	}
	if !strings.Contains(string(fluxData), "acme-corp") {
		t.Error("expected flux.yaml to contain default org")
	}

	// LoadFluxSchema reads flux.schema.yaml.
	schemaData, err := fs.ReadFile(fsys, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("reading flux.schema.yaml: %v", err)
	}
	if !strings.Contains(string(schemaData), "org") {
		t.Error("expected schema to reference org")
	}

	// Read command blank via source path.
	blankData, err := fs.ReadFile(fsys, "commands/hello.md")
	if err != nil {
		t.Fatalf("reading command blank: %v", err)
	}
	if !strings.Contains(string(blankData), "# Hello") {
		t.Error("expected command blank content")
	}
}
