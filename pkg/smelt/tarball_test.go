package smelt

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// writeMoldFixture creates a minimal valid mold directory structure in dir.
func writeMoldFixture(t *testing.T, dir string) {
	t.Helper()

	moldYAML := `apiVersion: v1
kind: mold
name: test-mold
version: 1.2.3
description: "A test mold for packaging"
flux:
  - name: org
    type: string
    required: false
    default: "acme"
commands:
  - hello.md
skills:
  - helper.md
workflows:
  - ci.yml
`
	dirs := []string{
		filepath.Join(dir, ".claude", "commands"),
		filepath.Join(dir, ".claude", "skills"),
		filepath.Join(dir, ".github", "workflows"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"mold.yaml":                 moldYAML,
		".claude/commands/hello.md": "# Hello\nCommand template.\n",
		".claude/skills/helper.md":  "# Helper\nSkill template.\n",
		".github/workflows/ci.yml":  "name: CI\non: push\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// listTarEntries returns the paths of all entries in a .tar.gz file.
func listTarEntries(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path) // #nosec G304
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("closing tar file: %v", err)
		}
	})

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := gr.Close(); err != nil {
			t.Errorf("closing gzip reader: %v", err)
		}
	})

	tr := tar.NewReader(gr)
	var entries []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, hdr.Name)
	}
	return entries
}

// readTarEntry reads the content of a specific entry from a .tar.gz file.
func readTarEntry(t *testing.T, tarPath, entryName string) string {
	t.Helper()

	f, err := os.Open(tarPath) // #nosec G304
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == entryName {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("reading entry %s: %v", entryName, err)
			}
			return string(data)
		}
	}
	t.Fatalf("entry %q not found in tarball", entryName)
	return ""
}

func TestPackageTarball_ValidMold(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()

	outputPath, size, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check output filename convention
	expectedName := "test-mold-1.2.3.tar.gz"
	if filepath.Base(outputPath) != expectedName {
		t.Errorf("expected filename %s, got %s", expectedName, filepath.Base(outputPath))
	}

	// Check file exists and has content
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

	// Verify tarball contents
	entries := listTarEntries(t, outputPath)
	prefix := "test-mold-1.2.3"
	expected := []string{
		prefix + "/mold.yaml",
		prefix + "/.claude/commands/hello.md",
		prefix + "/.claude/skills/helper.md",
		prefix + "/.github/workflows/ci.yml",
		prefix + "/flux.yaml",
	}
	entrySet := make(map[string]bool)
	for _, e := range entries {
		entrySet[e] = true
	}
	for _, exp := range expected {
		if !entrySet[exp] {
			t.Errorf("expected entry %q not found in tarball; got entries: %v", exp, entries)
		}
	}
}

func TestPackageTarball_DefaultOutputDir(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	// Save and restore working directory
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	outDir := t.TempDir()
	if err := os.Chdir(outDir); err != nil {
		t.Fatal(err)
	}

	outputPath, _, err := PackageTarball(moldDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be created in current directory
	if filepath.Dir(outputPath) != "." {
		t.Errorf("expected output in current directory, got %s", outputPath)
	}
}

func TestPackageTarball_MissingMoldYAML(t *testing.T) {
	emptyDir := t.TempDir()
	_, _, err := PackageTarball(emptyDir, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing mold.yaml")
	}
	if !strings.Contains(err.Error(), "loading mold") {
		t.Errorf("expected loading error, got: %v", err)
	}
}

func TestPackageTarball_InvalidMold(t *testing.T) {
	dir := t.TempDir()
	// Write a mold.yaml that fails validation (missing required fields)
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte("kind: wrong\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := PackageTarball(dir, t.TempDir())
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validating mold") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestPackageTarball_MissingTemplateFile(t *testing.T) {
	dir := t.TempDir()
	moldYAML := `apiVersion: v1
kind: mold
name: broken
version: 0.1.0
commands:
  - nonexistent.md
`
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := PackageTarball(dir, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing template file")
	}
	if !strings.Contains(err.Error(), "nonexistent.md") {
		t.Errorf("expected error to mention missing file, got: %v", err)
	}
}

func TestPackageTarball_WithIngots(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	// Add an ingots directory
	ingotDir := filepath.Join(moldDir, "ingots", "my-ingot")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "ingot.yaml"), []byte("apiVersion: v1\nkind: ingot\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "partial.md"), []byte("# Partial\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := listTarEntries(t, outputPath)
	prefix := "test-mold-1.2.3"
	found := false
	for _, e := range entries {
		if strings.Contains(e, "ingots/my-ingot/ingot.yaml") {
			found = true
			if !strings.HasPrefix(e, prefix+"/") {
				t.Errorf("ingot entry %q missing prefix %s/", e, prefix)
			}
		}
	}
	if !found {
		t.Errorf("expected ingot files in tarball; got entries: %v", entries)
	}
}

func TestPackageTarball_NoFluxDefaults(t *testing.T) {
	dir := t.TempDir()
	moldYAML := `apiVersion: v1
kind: mold
name: minimal
version: 0.1.0
`
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(dir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not contain flux.yaml when there are no defaults
	entries := listTarEntries(t, outputPath)
	for _, e := range entries {
		if strings.HasSuffix(e, "flux.yaml") {
			t.Errorf("did not expect flux.yaml in tarball with no flux defaults; got entries: %v", entries)
		}
	}
}

func TestPackageTarball_IncludesFluxSchema(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	// Add a flux.schema.yaml
	schemaContent := "- name: org\n  type: string\n  required: true\n"
	if err := os.WriteFile(filepath.Join(moldDir, "flux.schema.yaml"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := listTarEntries(t, outputPath)
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e, "flux.schema.yaml") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected flux.schema.yaml in tarball; got entries: %v", entries)
	}
}

func TestPackageTarball_NoFluxSchema(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixture(t, moldDir)

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := listTarEntries(t, outputPath)
	for _, e := range entries {
		if strings.HasSuffix(e, "flux.schema.yaml") {
			t.Errorf("did not expect flux.schema.yaml in tarball without schema file; got entries: %v", entries)
		}
	}
}

// writeMoldFixtureHelmStyle creates a Helm-style mold directory (no inline flux: section,
// with separate flux.yaml and optional flux.schema.yaml).
func writeMoldFixtureHelmStyle(t *testing.T, dir string, includeSchema bool) {
	t.Helper()

	moldYAML := `apiVersion: v1
kind: mold
name: helm-style
version: 2.0.0
description: "Helm-style mold with no inline flux"
commands:
  - hello.md
`
	fluxYAML := `# Default values
org: acme-corp
board: Engineering
cli: gh
`
	dirs := []string{
		filepath.Join(dir, ".claude", "commands"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"mold.yaml":                 moldYAML,
		"flux.yaml":                 fluxYAML,
		".claude/commands/hello.md": "# Hello\nCommand template.\n",
	}
	if includeSchema {
		files["flux.schema.yaml"] = "- name: org\n  type: string\n  required: true\n"
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPackageTarball_HelmStyleMold(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixtureHelmStyle(t, moldDir, false)

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := listTarEntries(t, outputPath)
	prefix := "helm-style-2.0.0"

	// Should include flux.yaml from source
	entrySet := make(map[string]bool)
	for _, e := range entries {
		entrySet[e] = true
	}

	if !entrySet[prefix+"/mold.yaml"] {
		t.Error("expected mold.yaml in tarball")
	}
	if !entrySet[prefix+"/flux.yaml"] {
		t.Error("expected flux.yaml in tarball")
	}
	if !entrySet[prefix+"/.claude/commands/hello.md"] {
		t.Error("expected command template in tarball")
	}
}

func TestPackageTarball_HelmStyleMoldWithSchema(t *testing.T) {
	moldDir := t.TempDir()
	writeMoldFixtureHelmStyle(t, moldDir, true)

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := listTarEntries(t, outputPath)
	prefix := "helm-style-2.0.0"

	entrySet := make(map[string]bool)
	for _, e := range entries {
		entrySet[e] = true
	}

	if !entrySet[prefix+"/flux.yaml"] {
		t.Error("expected flux.yaml in tarball")
	}
	if !entrySet[prefix+"/flux.schema.yaml"] {
		t.Error("expected flux.schema.yaml in tarball")
	}
}

func TestPackageTarball_SourceFluxPreservedVerbatim(t *testing.T) {
	moldDir := t.TempDir()

	// Mold with no inline flux: section
	moldYAML := `apiVersion: v1
kind: mold
name: verbatim
version: 1.0.0
commands:
  - cmd.md
`
	// flux.yaml with comments and specific formatting
	fluxYAML := "# My custom flux values\norg: my-org\n# Board setting\nboard: Product\n"

	for _, d := range []string{filepath.Join(moldDir, ".claude", "commands")} {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "flux.yaml"), []byte(fluxYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, ".claude", "commands", "cmd.md"), []byte("# Cmd\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	outputPath, _, err := PackageTarball(moldDir, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract flux.yaml content from tarball and verify it's verbatim
	content := readTarEntry(t, outputPath, "verbatim-1.0.0/flux.yaml")
	if content != fluxYAML {
		t.Errorf("expected flux.yaml to be preserved verbatim.\nExpected:\n%s\nGot:\n%s", fluxYAML, content)
	}
}

func TestGenerateFluxDefaults(t *testing.T) {
	tests := []struct {
		name     string
		vars     []mold.FluxVar
		wantNil  bool
		contains string
	}{
		{
			name:    "no vars",
			vars:    nil,
			wantNil: true,
		},
		{
			name:    "no defaults",
			vars:    []mold.FluxVar{{Name: "org", Type: "string", Required: true}},
			wantNil: true,
		},
		{
			name:     "with defaults",
			vars:     []mold.FluxVar{{Name: "board", Type: "string", Default: "Engineering"}},
			contains: "Engineering",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := generateFluxDefaults(tt.vars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil && data != nil {
				t.Errorf("expected nil data, got %q", data)
			}
			if tt.contains != "" && !strings.Contains(string(data), tt.contains) {
				t.Errorf("expected data to contain %q, got %q", tt.contains, data)
			}
		})
	}
}
