package mold

import (
	"strings"
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

func TestTemper_HasFunctionInTemplate(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"commands/has.md": &fstest.MapFile{Data: []byte(`{{- if has "claude" .agent.targets -}}yes{{- end -}}`)},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		for _, d := range result.Errors() {
			t.Errorf("unexpected error: %s: %s", d.File, d.Message)
		}
	}
}

func TestTemper_IngotFunctionInTemplate(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"commands/inc.md": &fstest.MapFile{Data: []byte(`{{ingot "header"}}`)},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		for _, d := range result.Errors() {
			t.Errorf("unexpected error: %s: %s", d.File, d.Message)
		}
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

func TestTemper_SkipsNonOutputMarkdown(t *testing.T) {
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
		"commands/good.md": &fstest.MapFile{Data: []byte("# Good\n{{if .foo}}yes{{end}}")},
		"docs/plan.md":     &fstest.MapFile{Data: []byte("Example: {{.broken")},
		"README.md":        &fstest.MapFile{Data: []byte("See {{.broken")},
	}

	result := Temper(fsys)

	for _, d := range result.Errors() {
		if d.File == "docs/plan.md" || d.File == "README.md" {
			t.Errorf("should not validate non-output file %s: %s", d.File, d.Message)
		}
	}
}

func TestTemper_SkipsTemplateValidationForProcessFalse(t *testing.T) {
	// Files in a `process: false` output mapping are copied as-is at cast time,
	// so temper must not attempt Go template parsing on them. They may contain
	// foreign template syntax (Helm, KOTS, Jinja, GitHub Actions) that would
	// otherwise produce false-positive parse errors.
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`
output:
  skills:
    dest: .claude/skills
    process: false
`)},
		"skills/helm-skill/SKILL.md": &fstest.MapFile{Data: []byte(
			"# Helm\nUse `{{ .Values.image.tag }}` for templated image refs.\n",
		)},
		"skills/kots-skill/SKILL.md": &fstest.MapFile{Data: []byte(
			"# KOTS\nUse `{{repl ConfigOption \"x\"}}` for KOTS config.\n",
		)},
	}

	result := Temper(fsys)

	for _, d := range result.Errors() {
		if d.File == "skills/helm-skill/SKILL.md" || d.File == "skills/kots-skill/SKILL.md" {
			t.Errorf("should not validate template syntax on process:false file %s: %s", d.File, d.Message)
		}
	}
}

func TestTemper_StillValidatesTemplatesForProcessTrue(t *testing.T) {
	// A sibling output with default (process: true) must still have its template
	// syntax validated, even when other outputs use process: false.
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`
output:
  skills:
    dest: .claude/skills
    process: false
  commands: .claude/commands
`)},
		"skills/helm/SKILL.md": &fstest.MapFile{Data: []byte("# Helm\n{{ .Values.foo }}")},
		"commands/broken.md":   &fstest.MapFile{Data: []byte("# Broken\n{{if .foo}")},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected template syntax error for commands/broken.md")
	}
	found := false
	for _, d := range result.Errors() {
		if d.File == "commands/broken.md" {
			found = true
		}
		if d.File == "skills/helm/SKILL.md" {
			t.Errorf("should not validate template syntax on process:false file %s: %s", d.File, d.Message)
		}
	}
	if !found {
		t.Error("expected error referencing commands/broken.md")
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

func TestTemper_DependencyBothIngotAndOre(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
dependencies:
  - ingot: a
    ore: b
    version: "1.0.0"
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors when dependency has both ingot and ore set")
	}

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != SeverityError {
			continue
		}
		if strings.Contains(d.Message, "ingot") && strings.Contains(d.Message, "ore") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SeverityError mentioning both 'ingot' and 'ore', got: %v", result.Errors())
	}
}

func TestTemper_DependencyNeitherIngotNorOre(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
dependencies:
  - version: "1.0.0"
`)},
	}

	result := Temper(fsys)

	if !result.HasErrors() {
		t.Fatal("expected errors when dependency has neither ingot nor ore set")
	}

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != SeverityError {
			continue
		}
		if strings.Contains(d.Message, "ingot") && strings.Contains(d.Message, "ore") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SeverityError mentioning both 'ingot' and 'ore', got: %v", result.Errors())
	}
}

func TestTemper_DependencyOreOnlyValid(t *testing.T) {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`
apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
dependencies:
  - ore: github.com/x/y
    version: "^1.0.0"
    as: foo
`)},
	}

	result := Temper(fsys)

	if result.HasErrors() {
		t.Errorf("expected no errors for valid ore-only dependency, got: %v", result.Errors())
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
