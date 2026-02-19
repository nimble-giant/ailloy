package mold

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseIngot_Valid(t *testing.T) {
	data := []byte(`
apiVersion: v1
kind: ingot
name: pr-format
version: 1.0.0
description: "PR formatting utilities"
files:
  - templates/pr-body.md
  - templates/pr-title.md
requires:
  ailloy: ">=0.2.0"
`)

	i, err := ParseIngot(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if i.APIVersion != "v1" {
		t.Errorf("expected apiVersion v1, got %s", i.APIVersion)
	}
	if i.Kind != "ingot" {
		t.Errorf("expected kind ingot, got %s", i.Kind)
	}
	if i.Name != "pr-format" {
		t.Errorf("expected name pr-format, got %s", i.Name)
	}
	if i.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", i.Version)
	}
	if i.Description != "PR formatting utilities" {
		t.Errorf("expected description 'PR formatting utilities', got %s", i.Description)
	}
	if len(i.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(i.Files))
	}
	if i.Files[0] != "templates/pr-body.md" {
		t.Errorf("expected files[0] templates/pr-body.md, got %s", i.Files[0])
	}
	if i.Requires.Ailloy != ">=0.2.0" {
		t.Errorf("expected requires.ailloy >=0.2.0, got %s", i.Requires.Ailloy)
	}
}

func TestParseIngot_InvalidYAML(t *testing.T) {
	data := []byte(`{{{invalid yaml`)
	_, err := ParseIngot(data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateIngot_Valid(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test",
		Version:    "1.0.0",
	}
	if err := ValidateIngot(i); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateIngot_MissingRequiredFields(t *testing.T) {
	i := &Ingot{}
	err := ValidateIngot(i)
	if err == nil {
		t.Fatal("expected validation error")
	}
	errMsg := err.Error()
	for _, field := range []string{"apiVersion", "kind", "name", "version"} {
		if !strings.Contains(errMsg, field) {
			t.Errorf("expected error to mention %q, got: %s", field, errMsg)
		}
	}
}

func TestValidateIngot_InvalidKind(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
	}
	err := ValidateIngot(i)
	if err == nil {
		t.Fatal("expected validation error for wrong kind")
	}
	if !strings.Contains(err.Error(), "must be \"ingot\"") {
		t.Errorf("expected kind error, got: %v", err)
	}
}

func TestValidateIngot_InvalidSemver(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test",
		Version:    "abc",
	}
	err := ValidateIngot(i)
	if err == nil {
		t.Fatal("expected validation error for invalid semver")
	}
	if !strings.Contains(err.Error(), "not valid semver") {
		t.Errorf("expected semver error, got: %v", err)
	}
}

func TestValidateIngot_InvalidRequiresConstraint(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test",
		Version:    "1.0.0",
		Requires:   Requires{Ailloy: "latest"},
	}
	err := ValidateIngot(i)
	if err == nil {
		t.Fatal("expected validation error for invalid requires constraint")
	}
}

func TestValidateIngotFiles(t *testing.T) {
	i := &Ingot{
		Files: []string{"a.md", "b.md"},
	}
	fsys := fstest.MapFS{
		"root/a.md": &fstest.MapFile{Data: []byte("a")},
		"root/b.md": &fstest.MapFile{Data: []byte("b")},
	}

	if err := ValidateIngotFiles(i, fsys, "root"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateIngotFiles_MissingFiles(t *testing.T) {
	i := &Ingot{
		Files: []string{"missing.md"},
	}
	fsys := fstest.MapFS{}

	err := ValidateIngotFiles(i, fsys, "root")
	if err == nil {
		t.Fatal("expected error for missing files")
	}
	if !strings.Contains(err.Error(), "missing.md") {
		t.Errorf("expected error to mention missing.md, got: %v", err)
	}
}

func TestLoadIngotFromFS(t *testing.T) {
	yaml := `apiVersion: v1
kind: ingot
name: fs-test
version: 3.0.0
`
	fsys := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte(yaml)},
	}

	i, err := LoadIngotFromFS(fsys, "ingot.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.Name != "fs-test" {
		t.Errorf("expected name fs-test, got %s", i.Name)
	}
	if i.Version != "3.0.0" {
		t.Errorf("expected version 3.0.0, got %s", i.Version)
	}
}

func TestLoadIngotFromFS_NotFound(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := LoadIngotFromFS(fsys, "ingot.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
