package merge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("oops")
	pe := &ParseError{Path: "x.json", Format: "json", Err: inner}
	if got := pe.Error(); got == "" {
		t.Fatal("ParseError.Error returned empty string")
	}
	if !errors.Is(pe, inner) {
		t.Fatal("errors.Is should match wrapped inner")
	}
}

func TestMergeFile_NonexistentDest_WritesVerbatim(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "sub", "opencode.json")
	content := []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`)
	if err := MergeFile(dest, content, Options{}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("nonexistent dest should write verbatim.\nwant: %s\ngot:  %s", content, got)
	}
}

func TestMergeFile_UnknownExt_WritesVerbatim(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "README.md")
	if err := os.WriteFile(dest, []byte("# old"), 0644); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("# new")
	if err := MergeFile(dest, newContent, Options{}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "# new" {
		t.Errorf("unknown ext should replace.\ngot: %s", got)
	}
}

func TestMergeFile_JSON_Merges(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(dest, []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	overlay := []byte(`{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)
	if err := MergeFile(dest, overlay, Options{}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	if !strings.Contains(gs, "outline") || !strings.Contains(gs, "replicated-docs") {
		t.Errorf("expected both servers, got: %s", gs)
	}
}

func TestMergeFile_YAML_Merges(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(dest, []byte("a: 1\nb: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	overlay := []byte("c: 3\n")
	if err := MergeFile(dest, overlay, Options{}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	if !strings.Contains(gs, "a:") || !strings.Contains(gs, "b:") || !strings.Contains(gs, "c:") {
		t.Errorf("merged YAML missing keys, got: %s", gs)
	}
}

func TestMergeFile_Malformed_ReturnsParseError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(dest, []byte("not json{"), 0644); err != nil {
		t.Fatal(err)
	}
	err := MergeFile(dest, []byte(`{"a":1}`), Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pe.Format != "json" {
		t.Errorf("Format: want json, got %s", pe.Format)
	}
}

func TestMergeFile_Malformed_ForceReplaceClobbers(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(dest, []byte("not json{"), 0644); err != nil {
		t.Fatal(err)
	}
	newContent := []byte(`{"a":1}`)
	if err := MergeFile(dest, newContent, Options{ForceReplaceOnParseError: true}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != string(newContent) {
		t.Errorf("force-replace should clobber.\nwant: %s\ngot:  %s", newContent, got)
	}
}
