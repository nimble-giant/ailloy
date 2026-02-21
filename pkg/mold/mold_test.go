package mold

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseMold_Valid(t *testing.T) {
	data := []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
description: "A test mold"
author:
  name: Test Author
  url: https://example.com
requires:
  ailloy: ">=0.2.0"
flux:
  - name: org
    description: "Organization name"
    type: string
    required: true
  - name: board
    description: "Board name"
    type: string
    required: false
    default: "Engineering"
output:
  commands: .claude/commands
  skills: .claude/skills
  workflows:
    dest: .github/workflows
    process: false
dependencies:
  - ingot: pr-format
    version: "^1.0.0"
`)

	m, err := ParseMold(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.APIVersion != "v1" {
		t.Errorf("expected apiVersion v1, got %s", m.APIVersion)
	}
	if m.Kind != "mold" {
		t.Errorf("expected kind mold, got %s", m.Kind)
	}
	if m.Name != "test-mold" {
		t.Errorf("expected name test-mold, got %s", m.Name)
	}
	if m.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.Version)
	}
	if m.Description != "A test mold" {
		t.Errorf("expected description 'A test mold', got %s", m.Description)
	}
	if m.Author.Name != "Test Author" {
		t.Errorf("expected author name Test Author, got %s", m.Author.Name)
	}
	if m.Requires.Ailloy != ">=0.2.0" {
		t.Errorf("expected requires.ailloy >=0.2.0, got %s", m.Requires.Ailloy)
	}
	if len(m.Flux) != 2 {
		t.Fatalf("expected 2 flux vars, got %d", len(m.Flux))
	}
	if m.Flux[0].Name != "org" {
		t.Errorf("expected flux[0].name org, got %s", m.Flux[0].Name)
	}
	if !m.Flux[0].Required {
		t.Error("expected flux[0].required to be true")
	}
	if m.Flux[1].Default != "Engineering" {
		t.Errorf("expected flux[1].default Engineering, got %s", m.Flux[1].Default)
	}
	if m.Output == nil {
		t.Error("expected output to be set")
	}
	if len(m.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(m.Dependencies))
	}
	if m.Dependencies[0].Ingot != "pr-format" {
		t.Errorf("expected dependency ingot pr-format, got %s", m.Dependencies[0].Ingot)
	}
}

func TestParseMold_InvalidYAML(t *testing.T) {
	data := []byte(`{{{invalid yaml`)
	_, err := ParseMold(data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseMold_EmptyDocument(t *testing.T) {
	data := []byte(``)
	m, err := ParseMold(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty YAML produces zero-value struct
	if m.Name != "" {
		t.Errorf("expected empty name, got %s", m.Name)
	}
}

func TestValidateMold_Valid(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
	}
	if err := ValidateMold(m); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateMold_MissingRequiredFields(t *testing.T) {
	m := &Mold{}
	err := ValidateMold(m)
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

func TestValidateMold_InvalidKind(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test",
		Version:    "1.0.0",
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for wrong kind")
	}
	if !strings.Contains(err.Error(), "must be \"mold\"") {
		t.Errorf("expected kind error, got: %v", err)
	}
}

func TestValidateMold_InvalidSemver(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "not-a-version",
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for invalid semver")
	}
	if !strings.Contains(err.Error(), "not valid semver") {
		t.Errorf("expected semver error, got: %v", err)
	}
}

func TestValidateMold_InvalidFluxType(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
		Flux: []FluxVar{
			{Name: "var1", Type: "float"},
		},
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for invalid flux type")
	}
	if !strings.Contains(err.Error(), "not valid") {
		t.Errorf("expected flux type error, got: %v", err)
	}
}

func TestValidateMold_FluxMissingName(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
		Flux: []FluxVar{
			{Type: "string"},
		},
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for missing flux name")
	}
	if !strings.Contains(err.Error(), "flux[0].name") {
		t.Errorf("expected flux name error, got: %v", err)
	}
}

func TestValidateMold_InvalidRequiresConstraint(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
		Requires:   Requires{Ailloy: "latest"},
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for invalid requires constraint")
	}
	if !strings.Contains(err.Error(), "requires.ailloy") {
		t.Errorf("expected requires error, got: %v", err)
	}
}

func TestValidateMold_InvalidDependencyVersion(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test",
		Version:    "1.0.0",
		Dependencies: []Dependency{
			{Ingot: "foo", Version: "invalid"},
		},
	}
	err := ValidateMold(m)
	if err == nil {
		t.Fatal("expected validation error for invalid dependency version")
	}
	if !strings.Contains(err.Error(), "dependencies[0].version") {
		t.Errorf("expected dependency version error, got: %v", err)
	}
}

func TestValidateMold_ValidFluxTypes(t *testing.T) {
	for _, typ := range []string{"string", "bool", "int"} {
		m := &Mold{
			APIVersion: "v1",
			Kind:       "mold",
			Name:       "test",
			Version:    "1.0.0",
			Flux:       []FluxVar{{Name: "v", Type: typ}},
		}
		if err := ValidateMold(m); err != nil {
			t.Errorf("expected flux type %q to be valid, got: %v", typ, err)
		}
	}
}

func TestValidateOutputSources(t *testing.T) {
	m := &Mold{
		Output: map[string]any{
			"commands": ".claude/commands",
		},
	}
	fsys := fstest.MapFS{
		"commands/create-issue.md": &fstest.MapFile{Data: []byte("# test")},
	}
	if err := ValidateOutputSources(m, fsys); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateOutputSources_NilOutput(t *testing.T) {
	m := &Mold{Output: nil}
	fsys := fstest.MapFS{}
	if err := ValidateOutputSources(m, fsys); err != nil {
		t.Errorf("expected no error for nil output, got: %v", err)
	}
}

func TestLoadMoldFromFS(t *testing.T) {
	yaml := `apiVersion: v1
kind: mold
name: fs-test
version: 2.0.0
`
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(yaml)},
	}

	m, err := LoadMoldFromFS(fsys, "mold.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "fs-test" {
		t.Errorf("expected name fs-test, got %s", m.Name)
	}
	if m.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", m.Version)
	}
}

func TestLoadMoldFromFS_NotFound(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := LoadMoldFromFS(fsys, "mold.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
