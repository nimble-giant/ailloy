package templates

import (
	"strings"
	"testing"
	"testing/fstest"
)

// testReader creates a MoldReader backed by an fstest.MapFS with dotted paths.
func testReader() *MoldReader {
	fsys := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte(`apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
commands:
  - create-issue.md
  - open-pr.md
skills:
  - brainstorm.md
workflows:
  - ci.yml
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`board: Engineering
org: test-org
`)},
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`- name: org
  type: string
  required: true
`)},
		".claude/commands/create-issue.md": &fstest.MapFile{Data: []byte("# create-issue\nCreate a GitHub issue.")},
		".claude/commands/open-pr.md":      &fstest.MapFile{Data: []byte("# open-pr\nOpen a pull request.")},
		".claude/skills/brainstorm.md":     &fstest.MapFile{Data: []byte("# brainstorm\nStructured brainstorming.")},
		".github/workflows/ci.yml":         &fstest.MapFile{Data: []byte("name: CI\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest")},
	}
	return NewMoldReader(fsys)
}

func TestListTemplates(t *testing.T) {
	reader := testReader()
	templates, err := reader.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(templates) == 0 {
		t.Fatal("expected at least one template, got none")
	}

	for _, tmpl := range templates {
		if !strings.HasSuffix(tmpl, ".md") {
			t.Errorf("expected template name ending in .md, got %s", tmpl)
		}
	}
}

func TestListTemplates_ExpectedTemplates(t *testing.T) {
	reader := testReader()
	templates, err := reader.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"create-issue.md", "open-pr.md"}
	templateSet := make(map[string]bool)
	for _, tmpl := range templates {
		templateSet[tmpl] = true
	}
	for _, exp := range expected {
		if !templateSet[exp] {
			t.Errorf("expected template %s not found in list", exp)
		}
	}
}

func TestGetTemplate_ValidTemplate(t *testing.T) {
	reader := testReader()
	content, err := reader.GetTemplate("create-issue.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty template content")
	}
	if content[0] != '#' {
		t.Errorf("expected template to start with '#', got %q", string(content[0]))
	}
}

func TestGetTemplate_NonExistentTemplate(t *testing.T) {
	reader := testReader()
	_, err := reader.GetTemplate("nonexistent-template.md")
	if err == nil {
		t.Error("expected error for non-existent template, got nil")
	}
}

func TestGetTemplate_AllTemplatesReadable(t *testing.T) {
	reader := testReader()
	templates, err := reader.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error listing templates: %v", err)
	}
	for _, tmpl := range templates {
		content, err := reader.GetTemplate(tmpl)
		if err != nil {
			t.Errorf("failed to read template %s: %v", tmpl, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("template %s has empty content", tmpl)
		}
	}
}

func TestListSkills(t *testing.T) {
	reader := testReader()
	skills, err := reader.ListSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("expected at least one skill, got none")
	}
	for _, skill := range skills {
		if !strings.HasSuffix(skill, ".md") {
			t.Errorf("expected skill name ending in .md, got %s", skill)
		}
	}
}

func TestGetSkill_ValidSkill(t *testing.T) {
	reader := testReader()
	content, err := reader.GetSkill("brainstorm.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty skill content")
	}
	if content[0] != '#' {
		t.Errorf("expected skill to start with '#', got %q", string(content[0]))
	}
}

func TestGetSkill_NonExistentSkill(t *testing.T) {
	reader := testReader()
	_, err := reader.GetSkill("nonexistent-skill.md")
	if err == nil {
		t.Error("expected error for non-existent skill, got nil")
	}
}

func TestListWorkflowTemplates(t *testing.T) {
	reader := testReader()
	workflows, err := reader.ListWorkflowTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workflows) == 0 {
		t.Fatal("expected at least one workflow template, got none")
	}
	for _, wf := range workflows {
		if !strings.HasSuffix(wf, ".yml") {
			t.Errorf("expected workflow name ending in .yml, got %s", wf)
		}
	}
}

func TestGetWorkflowTemplate_ValidTemplate(t *testing.T) {
	reader := testReader()
	content, err := reader.GetWorkflowTemplate("ci.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty workflow template content")
	}
	if !strings.Contains(string(content), "name:") {
		t.Error("expected workflow template to contain 'name:' field")
	}
}

func TestGetWorkflowTemplate_NonExistentTemplate(t *testing.T) {
	reader := testReader()
	_, err := reader.GetWorkflowTemplate("nonexistent-workflow.yml")
	if err == nil {
		t.Error("expected error for non-existent workflow template, got nil")
	}
}

func TestLoadManifest(t *testing.T) {
	reader := testReader()
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifest.Name != "test-mold" {
		t.Errorf("expected name=test-mold, got %q", manifest.Name)
	}
	if len(manifest.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(manifest.Commands))
	}
	if len(manifest.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(manifest.Skills))
	}
}

func TestLoadFluxDefaults(t *testing.T) {
	reader := testReader()
	vals, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) == 0 {
		t.Fatal("expected non-empty flux defaults")
	}
	if vals["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %q", vals["board"])
	}
}

func TestLoadFluxSchema(t *testing.T) {
	reader := testReader()
	schema, err := reader.LoadFluxSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
	if len(schema) != 1 {
		t.Fatalf("expected 1 schema entry, got %d", len(schema))
	}
	if schema[0].Name != "org" {
		t.Errorf("expected schema[0].name=org, got %q", schema[0].Name)
	}
	if !schema[0].Required {
		t.Error("expected org to be required")
	}
}

func TestNewMoldReaderFromPath(t *testing.T) {
	// Test with a non-existent directory
	_, err := NewMoldReaderFromPath("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}

	// Test with a valid temp directory
	tmpDir := t.TempDir()
	reader, err := NewMoldReaderFromPath(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Error("expected non-nil reader")
	}
	if reader.FS() == nil {
		t.Error("expected non-nil FS")
	}
}
