package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestBuildIngotResolver_PrependsMoldRoot is a regression test for issue #140:
// when a mold is cast from a path or remote source, its bundled ingots/ directory
// (located at the mold's source root) must be searched, not just the destination CWD.
func TestBuildIngotResolver_PrependsMoldRoot(t *testing.T) {
	moldDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(moldDir, "ingots"), 0750); err != nil {
		t.Fatalf("creating ingots dir: %v", err)
	}
	const ingotContent = "Hello from bundled ingot"
	if err := os.WriteFile(filepath.Join(moldDir, "ingots", "greeting.md"),
		[]byte(ingotContent), 0644); err != nil {
		t.Fatalf("writing ingot: %v", err)
	}

	// Run from a different CWD — the destination directory in a real cast.
	destDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(destDir); err != nil {
		t.Fatalf("chdir to dest: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	resolver := buildIngotResolver(map[string]any{}, moldDir)

	got, err := resolver.Resolve("greeting")
	if err != nil {
		t.Fatalf("Resolve(\"greeting\"): %v", err)
	}
	if !strings.Contains(got, ingotContent) {
		t.Errorf("resolved content = %q, want it to contain %q", got, ingotContent)
	}
}

// TestBuildIngotResolver_NoMoldRootFallsBackToCWD ensures the existing behavior
// (search "." when no mold root is known) still works for the embedded-mold case.
func TestBuildIngotResolver_NoMoldRootFallsBackToCWD(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "ingots"), 0750); err != nil {
		t.Fatalf("creating ingots dir: %v", err)
	}
	const ingotContent = "Local ingot content"
	if err := os.WriteFile(filepath.Join(dir, "ingots", "local.md"),
		[]byte(ingotContent), 0644); err != nil {
		t.Fatalf("writing ingot: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	resolver := buildIngotResolver(map[string]any{}, "")

	got, err := resolver.Resolve("local")
	if err != nil {
		t.Fatalf("Resolve(\"local\"): %v", err)
	}
	if !strings.Contains(got, ingotContent) {
		t.Errorf("resolved content = %q, want it to contain %q", got, ingotContent)
	}
}

// TestCastFromPath_ResolvesBundledIngot is a higher-level regression test for
// issue #140: casting a path-based mold from a different CWD must find the
// mold's bundled ingots, not silently fail with "ingot not found".
func TestCastFromPath_ResolvesBundledIngot(t *testing.T) {
	moldDir := t.TempDir()

	mustWrite(t, filepath.Join(moldDir, "mold.yaml"), `apiVersion: v1
kind: mold
name: test-mold
version: 0.1.0
`)
	mustWrite(t, filepath.Join(moldDir, "flux.yaml"), `output:
  agents: agents
`)
	if err := os.MkdirAll(filepath.Join(moldDir, "ingots"), 0750); err != nil {
		t.Fatalf("creating ingots dir: %v", err)
	}
	const ingotContent = "Bundled ingot content"
	mustWrite(t, filepath.Join(moldDir, "ingots", "greeting.md"), ingotContent)

	if err := os.MkdirAll(filepath.Join(moldDir, "agents"), 0750); err != nil {
		t.Fatalf("creating agents dir: %v", err)
	}
	mustWrite(t, filepath.Join(moldDir, "agents", "test.md"), `{{ingot "greeting"}}`+"\n")

	// chdir to a different destination directory.
	destDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(destDir); err != nil {
		t.Fatalf("chdir to dest: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	reader, err := blanks.NewMoldReaderFromPath(moldDir)
	if err != nil {
		t.Fatalf("NewMoldReaderFromPath: %v", err)
	}
	if reader.Root() != moldDir {
		t.Fatalf("reader.Root() = %q, want %q", reader.Root(), moldDir)
	}

	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("LoadFluxDefaults: %v", err)
	}

	resolver := buildIngotResolver(flux, reader.Root())
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("ResolveFiles: %v", err)
	}
	if len(resolved) == 0 {
		t.Fatal("expected at least one resolved file")
	}

	for _, rf := range resolved {
		content, err := os.ReadFile(filepath.Join(moldDir, rf.SrcPath))
		if err != nil {
			t.Fatalf("reading %s: %v", rf.SrcPath, err)
		}
		rendered, err := mold.ProcessTemplate(string(content), flux, opts...)
		if err != nil {
			t.Fatalf("rendering %s: %v", rf.SrcPath, err)
		}
		if !strings.Contains(rendered, ingotContent) {
			t.Errorf("rendered %s = %q, want it to contain %q", rf.SrcPath, rendered, ingotContent)
		}
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
