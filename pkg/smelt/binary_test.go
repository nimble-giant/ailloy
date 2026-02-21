package smelt

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
)

func TestPackageBinary_BasicMold(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()

	outputPath, size, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check output filename convention.
	expectedName := "test-mold-1.2.3"
	if filepath.Base(outputPath) != expectedName {
		t.Errorf("expected filename %s, got %s", expectedName, filepath.Base(outputPath))
	}

	// Check file exists and has content.
	if size <= 0 {
		t.Error("expected positive file size")
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() != size {
		t.Errorf("reported size %d != actual size %d", size, info.Size())
	}

	// Check executable permission.
	if info.Mode()&0111 == 0 {
		t.Error("expected output to be executable")
	}

	// Verify it's a valid stuffed binary.
	_, err = stuffbin.GetFileID(outputPath)
	if err != nil {
		t.Errorf("expected valid stuffbin ID, got error: %v", err)
	}
}

func TestPackageBinary_InvalidMold(t *testing.T) {
	emptyDir := t.TempDir()
	_, _, err := PackageBinary(emptyDir, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing mold.yaml")
	}
	if !strings.Contains(err.Error(), "loading mold") {
		t.Errorf("expected loading error, got: %v", err)
	}
}

func TestPackageBinary_ValidationError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte("kind: wrong\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := PackageBinary(dir, t.TempDir())
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validating mold") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestPackageBinary_GeneratesFluxDefaults(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)
	// writeMoldFixture has flux vars with defaults but no flux.yaml file.

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unstuff and verify flux.yaml was generated.
	sfs, err := stuffbin.UnStuff(outputPath)
	if err != nil {
		t.Fatalf("unstuffing: %v", err)
	}

	data, err := sfs.Read("/flux.yaml")
	if err != nil {
		t.Fatal("expected generated flux.yaml in binary")
	}
	if !strings.Contains(string(data), "acme") {
		t.Errorf("expected flux.yaml to contain default value 'acme', got: %s", data)
	}
}

func TestPackageBinary_WithOutputDir(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Dir(outputPath) != outputDir {
		t.Errorf("expected output in %s, got %s", outputDir, filepath.Dir(outputPath))
	}
}

func TestPackageBinary_ContentRoundTrip(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unstuff and verify all files are present and content matches.
	fsys, err := UnstuffFS(outputPath)
	if err != nil {
		t.Fatalf("unstuffing to fs.FS: %v", err)
	}

	wantFiles := map[string]string{
		"mold.yaml":         "apiVersion: v1",
		"commands/hello.md": "# Hello",
		"skills/helper.md":  "# Helper",
		"workflows/ci.yml":  "name: CI",
	}

	for path, wantSubstr := range wantFiles {
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		if !strings.Contains(string(data), wantSubstr) {
			t.Errorf("%s: expected content containing %q, got %q", path, wantSubstr, string(data))
		}
	}
}

func TestPackageBinary_HelmStyleMold(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixtureHelmStyle(t, moldDir, true)

	outputDir := t.TempDir()
	outputPath, _, err := PackageBinary(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sfs, err := stuffbin.UnStuff(outputPath)
	if err != nil {
		t.Fatalf("unstuffing: %v", err)
	}

	// Verify flux.yaml from source is preserved.
	data, err := sfs.Read("/flux.yaml")
	if err != nil {
		t.Fatal("expected flux.yaml in binary")
	}
	if !strings.Contains(string(data), "acme-corp") {
		t.Errorf("expected source flux.yaml content, got: %s", data)
	}

	// Verify schema is present.
	_, err = sfs.Read("/flux.schema.yaml")
	if err != nil {
		t.Fatal("expected flux.schema.yaml in binary")
	}
}

func TestPackageBinary_DefaultOutputDir(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	// Save and restore working directory.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	outDir := t.TempDir()
	if err := os.Chdir(outDir); err != nil {
		t.Fatal(err)
	}

	outputPath, _, err := PackageBinary(moldDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Dir(outputPath) != "." {
		t.Errorf("expected output in current directory, got %s", outputPath)
	}
}
