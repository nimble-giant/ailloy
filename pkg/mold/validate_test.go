package mold

import (
	"strings"
	"testing"
	"testing/fstest"
)

// --- ValidationResult tests ---

func TestValidationResult_AddError(t *testing.T) {
	r := &ValidationResult{}
	r.AddError("mold.yaml", "something is wrong")
	if len(r.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(r.Errors))
	}
	if r.Errors[0].File != "mold.yaml" {
		t.Errorf("expected file mold.yaml, got %q", r.Errors[0].File)
	}
	if r.Errors[0].Message != "something is wrong" {
		t.Errorf("expected message 'something is wrong', got %q", r.Errors[0].Message)
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	r := &ValidationResult{}
	r.AddWarning("mold.yaml", "minor issue")
	if len(r.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(r.Warnings))
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	r := &ValidationResult{}
	if r.HasErrors() {
		t.Error("expected no errors initially")
	}
	r.AddError("", "err")
	if !r.HasErrors() {
		t.Error("expected HasErrors to be true after AddError")
	}
}

func TestValidationResult_Merge(t *testing.T) {
	r1 := &ValidationResult{}
	r1.AddError("a", "err1")
	r1.AddWarning("a", "warn1")

	r2 := &ValidationResult{}
	r2.AddError("b", "err2")
	r2.AddWarning("b", "warn2")

	r1.Merge(r2)

	if len(r1.Errors) != 2 {
		t.Errorf("expected 2 errors after merge, got %d", len(r1.Errors))
	}
	if len(r1.Warnings) != 2 {
		t.Errorf("expected 2 warnings after merge, got %d", len(r1.Warnings))
	}
}

// --- ValidateTemplateSyntax tests ---

func TestValidateTemplateSyntax_Valid(t *testing.T) {
	content := []byte(`# Hello {{.name}}

{{if .enabled}}Feature is on{{end}}

Use {{ingot "partial"}} here.`)

	result := ValidateTemplateSyntax("test.md", content)
	if result.HasErrors() {
		t.Errorf("expected no errors for valid template, got: %v", result.Errors[0].Message)
	}
}

func TestValidateTemplateSyntax_Invalid(t *testing.T) {
	content := []byte(`# Broken {{if}}{{end`)

	result := ValidateTemplateSyntax("broken.md", content)
	if !result.HasErrors() {
		t.Fatal("expected error for invalid template syntax")
	}
	if !strings.Contains(result.Errors[0].Message, "template syntax error") {
		t.Errorf("expected 'template syntax error' in message, got: %q", result.Errors[0].Message)
	}
}

func TestValidateTemplateSyntax_EmptyTemplate(t *testing.T) {
	result := ValidateTemplateSyntax("empty.md", []byte(""))
	if result.HasErrors() {
		t.Error("expected no errors for empty template")
	}
}

func TestValidateTemplateSyntax_PlainText(t *testing.T) {
	result := ValidateTemplateSyntax("plain.md", []byte("No template syntax here, just markdown."))
	if result.HasErrors() {
		t.Error("expected no errors for plain text")
	}
}

// --- TemperMold tests ---

func TestTemperMold_Valid(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test-mold",
		Version:    "1.0.0",
		Commands:   []string{"cmd.md"},
		Skills:     []string{"skill.md"},
		Flux: []FluxVar{
			{Name: "org", Type: "string", Required: true},
		},
	}

	fsys := fstest.MapFS{
		"root/claude/commands/cmd.md": &fstest.MapFile{Data: []byte("# Command {{.org}}")},
		"root/claude/skills/skill.md": &fstest.MapFile{Data: []byte("# Skill")},
	}

	result := TemperMold(m, fsys, "root")
	if result.HasErrors() {
		t.Errorf("expected no errors for valid mold, got %d error(s):", len(result.Errors))
		for _, e := range result.Errors {
			t.Errorf("  %s: %s", e.File, e.Message)
		}
	}
}

func TestTemperMold_MissingFiles(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test-mold",
		Version:    "1.0.0",
		Commands:   []string{"missing.md"},
	}

	fsys := fstest.MapFS{}

	result := TemperMold(m, fsys, "root")
	if !result.HasErrors() {
		t.Fatal("expected errors for missing files")
	}
}

func TestTemperMold_InvalidManifest(t *testing.T) {
	m := &Mold{} // all required fields missing

	fsys := fstest.MapFS{}

	result := TemperMold(m, fsys, "root")
	if !result.HasErrors() {
		t.Fatal("expected errors for invalid manifest")
	}
}

func TestTemperMold_BadTemplateSyntax(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test-mold",
		Version:    "1.0.0",
		Commands:   []string{"bad.md"},
	}

	fsys := fstest.MapFS{
		"root/claude/commands/bad.md": &fstest.MapFile{Data: []byte("# Broken {{if}}{{end")},
	}

	result := TemperMold(m, fsys, "root")
	if !result.HasErrors() {
		t.Fatal("expected errors for bad template syntax")
	}

	foundSyntaxError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "template syntax error") {
			foundSyntaxError = true
			break
		}
	}
	if !foundSyntaxError {
		t.Error("expected a template syntax error in results")
	}
}

func TestTemperMold_InvalidFluxType(t *testing.T) {
	m := &Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "test-mold",
		Version:    "1.0.0",
		Flux: []FluxVar{
			{Name: "var1", Type: "float"},
		},
	}

	fsys := fstest.MapFS{}

	result := TemperMold(m, fsys, "root")
	// Invalid flux type is caught by ValidateMold as an error, but also by TemperMold's
	// flux schema check as a warning. Either way, the result should flag it.
	if !result.HasErrors() && len(result.Warnings) == 0 {
		t.Fatal("expected errors or warnings for invalid flux type")
	}
}

// --- TemperIngot tests ---

func TestTemperIngot_Valid(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test-ingot",
		Version:    "1.0.0",
		Files:      []string{"templates/partial.md"},
	}

	fsys := fstest.MapFS{
		"root/templates/partial.md": &fstest.MapFile{Data: []byte("# Partial")},
	}

	result := TemperIngot(i, fsys, "root")
	if result.HasErrors() {
		t.Errorf("expected no errors for valid ingot, got %d error(s):", len(result.Errors))
		for _, e := range result.Errors {
			t.Errorf("  %s: %s", e.File, e.Message)
		}
	}
}

func TestTemperIngot_InvalidManifest(t *testing.T) {
	i := &Ingot{} // all required fields missing

	fsys := fstest.MapFS{}

	result := TemperIngot(i, fsys, "root")
	if !result.HasErrors() {
		t.Fatal("expected errors for invalid ingot manifest")
	}
}

func TestTemperIngot_MissingFiles(t *testing.T) {
	i := &Ingot{
		APIVersion: "v1",
		Kind:       "ingot",
		Name:       "test-ingot",
		Version:    "1.0.0",
		Files:      []string{"missing.md"},
	}

	fsys := fstest.MapFS{}

	result := TemperIngot(i, fsys, "root")
	if !result.HasErrors() {
		t.Fatal("expected errors for missing ingot files")
	}
}
