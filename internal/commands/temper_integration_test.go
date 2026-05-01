package commands

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/assay"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestIntegration_TemperLint_RendersAndLints(t *testing.T) {
	reader := testMoldReader(t)
	flux := testFlux(t, reader)

	// Load manifest to ensure it's valid (used implicitly by ResolveFiles)
	_, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	// Build ingot resolver and resolve files
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	if len(resolved) == 0 {
		t.Fatal("expected resolved files, got none")
	}

	// Render to temp directory
	tmpDir := t.TempDir()
	err = writeRenderedFiles(resolved, reader.FS(), flux, opts, tmpDir)
	if err != nil {
		t.Fatalf("writeRenderedFiles failed: %v", err)
	}

	// Verify files were written
	for _, rf := range resolved {
		path := filepath.Join(tmpDir, rf.DestPath)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s to be created: %v", rf.DestPath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", rf.DestPath)
		}
	}

	// Run assay on the rendered output
	cfg := assay.DefaultConfig()
	result, err := assay.Assay(tmpDir, cfg)
	if err != nil {
		t.Fatalf("assay failed: %v", err)
	}

	if result.FilesScanned == 0 {
		t.Error("expected assay to scan at least one file")
	}
}

func TestIntegration_TemperLint_WriteRenderedFiles_CreatesDirectories(t *testing.T) {
	reader := testMoldReader(t)
	flux := testFlux(t, reader)

	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	tmpDir := t.TempDir()
	err = writeRenderedFiles(resolved, reader.FS(), flux, opts, tmpDir)
	if err != nil {
		t.Fatalf("writeRenderedFiles failed: %v", err)
	}

	// Verify expected directories exist
	expectedDirs := []string{
		".claude/commands",
		".claude/skills",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(tmpDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestIntegration_TemperLint_AssayFindsClaudeFiles(t *testing.T) {
	// Create a minimal rendered output structure that assay can detect
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude", "commands")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Write a command file with frontmatter that assay will scan
	content := `---
description: Test command for temper lint integration
---

# Test Command

This is a test command.
`
	if err := os.WriteFile(filepath.Join(claudeDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := assay.DefaultConfig()
	result, err := assay.Assay(tmpDir, cfg)
	if err != nil {
		t.Fatalf("assay failed: %v", err)
	}

	if result.FilesScanned == 0 {
		t.Error("expected assay to find files in rendered output")
	}
}

func TestWriteRenderedFiles_SkipsEmptyRenderedFiles(t *testing.T) {
	// Create an in-memory filesystem with a template that renders to empty
	moldFS := fstest.MapFS{
		"commands/empty.md": &fstest.MapFile{
			Data: []byte("{{- if false -}}content{{- end -}}"),
		},
		"commands/nonempty.md": &fstest.MapFile{
			Data: []byte("# Hello\nThis has content."),
		},
	}

	resolved := []mold.ResolvedFile{
		{SrcPath: "commands/empty.md", DestPath: ".claude/commands/empty.md", Process: true},
		{SrcPath: "commands/nonempty.md", DestPath: ".claude/commands/nonempty.md", Process: true},
	}

	tmpDir := t.TempDir()
	err := writeRenderedFiles(resolved, moldFS, map[string]any{}, nil, tmpDir)
	if err != nil {
		t.Fatalf("writeRenderedFiles failed: %v", err)
	}

	// The empty-rendering file should NOT exist
	emptyPath := filepath.Join(tmpDir, ".claude/commands/empty.md")
	if _, err := os.Stat(emptyPath); err == nil {
		t.Error("expected empty-rendering file to be skipped, but it was written")
	}

	// The non-empty file SHOULD exist
	nonEmptyPath := filepath.Join(tmpDir, ".claude/commands/nonempty.md")
	info, err := os.Stat(nonEmptyPath)
	if err != nil {
		t.Fatalf("expected non-empty file to be written: %v", err)
	}
	if info.Size() == 0 {
		t.Error("non-empty file should have content")
	}
}

func TestWriteRenderedFiles_SkipsWhitespaceOnlyRenderedFiles(t *testing.T) {
	moldFS := fstest.MapFS{
		"commands/whitespace.md": &fstest.MapFile{
			Data: []byte("  \n\t\n  "),
		},
	}

	resolved := []mold.ResolvedFile{
		{SrcPath: "commands/whitespace.md", DestPath: ".claude/commands/whitespace.md", Process: true},
	}

	tmpDir := t.TempDir()
	err := writeRenderedFiles(resolved, moldFS, map[string]any{}, nil, tmpDir)
	if err != nil {
		t.Fatalf("writeRenderedFiles failed: %v", err)
	}

	wsPath := filepath.Join(tmpDir, ".claude/commands/whitespace.md")
	if _, err := os.Stat(wsPath); err == nil {
		t.Error("expected whitespace-only file to be skipped, but it was written")
	}
}

func TestWriteRenderedFiles_RendersTemplates(t *testing.T) {
	reader := testMoldReader(t)
	flux := testFlux(t, reader)

	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	// Filter to processable files only
	var processable []mold.ResolvedFile
	for _, rf := range resolved {
		if rf.Process {
			processable = append(processable, rf)
		}
	}

	if len(processable) == 0 {
		t.Skip("no processable files found in mold")
	}

	tmpDir := t.TempDir()
	err = writeRenderedFiles(processable, reader.FS(), flux, opts, tmpDir)
	if err != nil {
		t.Fatalf("writeRenderedFiles failed: %v", err)
	}

	// Verify content is non-empty and was rendered
	for _, rf := range processable {
		content, err := os.ReadFile(filepath.Join(tmpDir, rf.DestPath))
		if err != nil {
			t.Errorf("failed to read %s: %v", rf.DestPath, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("rendered file %s is empty", rf.DestPath)
		}
	}
}
