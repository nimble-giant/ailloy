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

func TestMergeFile_ReadError_NonENOENT(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(dest, []byte(`{"a":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Make the file unreadable to trigger a non-ENOENT error from os.ReadFile.
	if err := os.Chmod(dest, 0); err != nil {
		t.Fatalf("chmod 0: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dest, 0644) })

	err := MergeFile(dest, []byte(`{"b":2}`), Options{})
	if err == nil {
		t.Fatal("expected error reading unreadable file, got nil")
	}
	// Must NOT be a *ParseError — it's a read error, not a parse error.
	var pe *ParseError
	if errors.As(err, &pe) {
		t.Errorf("expected non-ParseError, got *ParseError: %v", err)
	}
	if !strings.Contains(err.Error(), "read existing") {
		t.Errorf("expected error message to mention 'read existing', got: %v", err)
	}
}

func TestMergeFile_MalformedNewContent_NotParseError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "opencode.json")
	// Existing file is valid.
	if err := os.WriteFile(dest, []byte(`{"a":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	// New content is malformed — would be a template/programmer bug in production.
	err := MergeFile(dest, []byte("not json{"), Options{})
	if err == nil {
		t.Fatal("expected error for malformed new content, got nil")
	}
	// Must NOT be a *ParseError — that type is reserved for the on-disk file.
	var pe *ParseError
	if errors.As(err, &pe) {
		t.Errorf("expected plain error for malformed new content, got *ParseError: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot parse new") {
		t.Errorf("expected message about new content; got: %v", err)
	}
}
