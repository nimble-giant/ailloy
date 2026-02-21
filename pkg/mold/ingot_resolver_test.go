package mold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngotResolver_BareFile(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "footer.md"), []byte("-- footer --"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil)
	result, err := r.Resolve("footer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "-- footer --" {
		t.Errorf("expected '-- footer --', got %q", result)
	}
}

func TestIngotResolver_ManifestBased(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots", "pr-format")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifest := `apiVersion: v1
kind: ingot
name: pr-format
version: 1.0.0
files:
  - header.md
  - body.md
`
	if err := os.WriteFile(filepath.Join(ingotDir, "ingot.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "header.md"), []byte("# Header\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "body.md"), []byte("Body content"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil)
	result, err := r.Resolve("pr-format")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# Header") {
		t.Error("expected result to contain header")
	}
	if !strings.Contains(result, "Body content") {
		t.Error("expected result to contain body")
	}
}

func TestIngotResolver_ManifestTakesPrecedenceOverBareFile(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots", "snippet")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifest := `apiVersion: v1
kind: ingot
name: snippet
version: 1.0.0
files:
  - content.md
`
	if err := os.WriteFile(filepath.Join(ingotDir, "ingot.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "content.md"), []byte("from manifest"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ingots", "snippet.md"), []byte("from bare file"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil)
	result, err := r.Resolve("snippet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "from manifest" {
		t.Errorf("expected manifest content, got %q", result)
	}
}

func TestIngotResolver_SearchPathOrder(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	ingotDir2 := filepath.Join(dir2, "ingots")
	if err := os.MkdirAll(ingotDir2, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir2, "shared.md"), []byte("from second path"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir1, dir2}, nil)
	result, err := r.Resolve("shared")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "from second path" {
		t.Errorf("expected 'from second path', got %q", result)
	}
}

func TestIngotResolver_FirstPathWins(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	for _, dir := range []string{dir1, dir2} {
		ingotDir := filepath.Join(dir, "ingots")
		if err := os.MkdirAll(ingotDir, 0750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir1, "ingots", "item.md"), []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "ingots", "item.md"), []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir1, dir2}, nil)
	result, err := r.Resolve("item")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "first" {
		t.Errorf("expected 'first', got %q", result)
	}
}

func TestIngotResolver_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := NewIngotResolver([]string{dir}, nil)

	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing ingot")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention ingot name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestIngotResolver_CircularReference(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "self.md"), []byte(`{{ingot "self"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil)
	_, err := r.Resolve("self")
	if err == nil {
		t.Fatal("expected error for circular reference")
	}
	if !strings.Contains(err.Error(), "circular ingot reference") {
		t.Errorf("expected circular reference error, got: %v", err)
	}
}

func TestIngotResolver_FluxVariablesRendered(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "greeting.md"), []byte("Hello {{organization}}!"), 0644); err != nil {
		t.Fatal(err)
	}

	flux := map[string]any{"organization": "Acme"}
	r := NewIngotResolver([]string{dir}, flux)
	result, err := r.Resolve("greeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello Acme!" {
		t.Errorf("expected 'Hello Acme!', got %q", result)
	}
}

func TestIngotResolver_NestedIngots(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "inner.md"), []byte("inner content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "outer.md"), []byte(`before {{ingot "inner"}} after`), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil)
	result, err := r.Resolve("outer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "before inner content after" {
		t.Errorf("expected 'before inner content after', got %q", result)
	}
}

func TestIngotResolver_EmptySearchPaths(t *testing.T) {
	r := NewIngotResolver(nil, nil)
	_, err := r.Resolve("anything")
	if err == nil {
		t.Fatal("expected error with no search paths")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}
