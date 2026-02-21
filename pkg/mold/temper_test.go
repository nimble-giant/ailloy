package mold

import (
	"testing"
	"testing/fstest"
)

func TestTemper_ValidMold(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`
output:
  commands: .claude/commands
`)},
		"commands/hello.md": &fstest.MapFile{Data: []byte("# Hello\nThis is a test.")},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors())
	}
	if result.ManifestKind != "mold" {
		t.Errorf("expected manifest kind 'mold', got %q", result.ManifestKind)
	}
	if result.Name != "test-mold" {
		t.Errorf("expected name 'test-mold', got %q", result.Name)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", result.Version)
	}
}

func TestTemper_ValidIngot(t *testing.T) {
	fsys := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: ingot
name: test-ingot
version: 2.0.0
files:
  - templates/pr.md
`)},
		"templates/pr.md": &fstest.MapFile{Data: []byte("# PR Template")},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors())
	}
	if result.ManifestKind != "ingot" {
		t.Errorf("expected manifest kind 'ingot', got %q", result.ManifestKind)
	}
	if result.Name != "test-ingot" {
		t.Errorf("expected name 'test-ingot', got %q", result.Name)
	}
}

func TestTemper_NoManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"readme.md": &fstest.MapFile{Data: []byte("# Readme")},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for missing manifest")
	}
	errors := result.Errors()
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].Message != "no mold.yaml or ingot.yaml found" {
		t.Errorf("unexpected error message: %s", errors[0].Message)
	}
}

func TestTemper_InvalidMoldManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: ""
version: not-semver
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for invalid manifest")
	}

	errorMessages := make(map[string]bool)
	for _, d := range result.Errors() {
		errorMessages[d.Message] = true
	}

	if !errorMessages["name is required"] {
		t.Error("expected 'name is required' error")
	}
	if !errorMessages[`version "not-semver" is not valid semver`] {
		t.Errorf("expected semver error, got errors: %v", result.Errors())
	}
}

func TestTemper_InvalidIngotManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: ingot
name: ""
version: bad
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for invalid ingot manifest")
	}

	found := false
	for _, d := range result.Errors() {
		if d.Message == "name is required" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'name is required' error")
	}
}

func TestTemper_MissingIngotFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: ingot
name: test-ingot
version: 1.0.0
files:
  - missing.md
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for missing ingot files")
	}

	found := false
	for _, d := range result.Errors() {
		if d.File == "ingot.yaml" {
			found = true
		}
	}
	if !found {
		t.Error("expected error referencing ingot.yaml")
	}
}

func TestTemper_TemplateSyntaxError(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"commands/broken.md": &fstest.MapFile{Data: []byte("# Broken\n{{if .foo}")},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for broken template")
	}

	found := false
	for _, d := range result.Errors() {
		if d.File == "commands/broken.md" {
			found = true
		}
	}
	if !found {
		t.Error("expected error referencing commands/broken.md")
	}
}

func TestTemper_ValidTemplate(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"commands/good.md": &fstest.MapFile{Data: []byte("# Good\n{{if .foo}}yes{{end}}")},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors())
	}
}

func TestTemper_FluxSchemaValidation(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`
- name: org
  type: string
  required: true
- name: bad
  type: float
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for invalid flux schema type")
	}

	found := false
	for _, d := range result.Errors() {
		if d.File == "flux.schema.yaml" {
			found = true
		}
	}
	if !found {
		t.Error("expected error referencing flux.schema.yaml")
	}
}

func TestTemper_DualFluxWarning(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
flux:
  - name: org
    type: string
    required: true
`)},
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`
- name: org
  type: string
  required: true
`)},
	}

	result := Temper(fsys)

	warnings := result.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected warning about dual flux definitions")
	}

	found := false
	for _, w := range warnings {
		if w.Message == "flux variables defined in both mold.yaml and flux.schema.yaml; schema file takes precedence at runtime" {
			found = true
		}
	}
	if !found {
		t.Error("expected dual flux definition warning")
	}
}

func TestTemper_MalformedYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`{{{invalid`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for malformed YAML")
	}
	if result.Errors()[0].File != "mold.yaml" {
		t.Errorf("expected error referencing mold.yaml, got %q", result.Errors()[0].File)
	}
}

func TestTemper_MissingOutputSources(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`
output:
  nonexistent: .claude/commands
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors for missing output source directory")
	}
}

func TestTemperResult_ErrorsAndWarnings(t *testing.T) {
	r := &TemperResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityError, Message: "err1"},
			{Severity: SeverityWarning, Message: "warn1"},
			{Severity: SeverityError, Message: "err2"},
			{Severity: SeverityWarning, Message: "warn2"},
		},
	}

	if !r.HasErrors() {
		t.Error("expected HasErrors to return true")
	}
	if len(r.Errors()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(r.Errors()))
	}
	if len(r.Warnings()) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(r.Warnings()))
	}
}

func TestTemperResult_NoErrors(t *testing.T) {
	r := &TemperResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "warn1"},
		},
	}

	if r.HasErrors() {
		t.Error("expected HasErrors to return false with only warnings")
	}
}
