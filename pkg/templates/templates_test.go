package templates

import (
	"strings"
	"testing"
)

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(templates) == 0 {
		t.Fatal("expected at least one template, got none")
	}

	// All templates should end in .md
	for _, tmpl := range templates {
		if !strings.HasSuffix(tmpl, ".md") {
			t.Errorf("expected template name ending in .md, got %s", tmpl)
		}
	}
}

func TestListTemplates_ExpectedTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"brainstorm.md",
		"create-issue.md",
		"open-pr.md",
		"pr-comments.md",
		"pr-description.md",
		"pr-review.md",
		"preflight.md",
		"start-issue.md",
		"update-pr.md",
	}

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
	content, err := GetTemplate("create-issue.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty template content")
	}

	// Should start with a markdown header
	if content[0] != '#' {
		t.Errorf("expected template to start with '#', got %q", string(content[0]))
	}
}

func TestGetTemplate_NonExistentTemplate(t *testing.T) {
	_, err := GetTemplate("nonexistent-template.md")
	if err == nil {
		t.Error("expected error for non-existent template, got nil")
	}
}

func TestGetTemplate_AllTemplatesReadable(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error listing templates: %v", err)
	}

	for _, tmpl := range templates {
		content, err := GetTemplate(tmpl)
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
	skills, err := ListSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) == 0 {
		t.Fatal("expected at least one skill, got none")
	}

	// All skills should end in .md
	for _, skill := range skills {
		if !strings.HasSuffix(skill, ".md") {
			t.Errorf("expected skill name ending in .md, got %s", skill)
		}
	}
}

func TestListSkills_ExpectedSkills(t *testing.T) {
	skills, err := ListSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"brainstorm.md",
	}

	skillSet := make(map[string]bool)
	for _, skill := range skills {
		skillSet[skill] = true
	}

	for _, exp := range expected {
		if !skillSet[exp] {
			t.Errorf("expected skill %s not found in list", exp)
		}
	}
}

func TestGetSkill_ValidSkill(t *testing.T) {
	content, err := GetSkill("brainstorm.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty skill content")
	}

	// Should start with a markdown header
	if content[0] != '#' {
		t.Errorf("expected skill to start with '#', got %q", string(content[0]))
	}
}

func TestGetSkill_NonExistentSkill(t *testing.T) {
	_, err := GetSkill("nonexistent-skill.md")
	if err == nil {
		t.Error("expected error for non-existent skill, got nil")
	}
}

func TestGetSkill_AllSkillsReadable(t *testing.T) {
	skills, err := ListSkills()
	if err != nil {
		t.Fatalf("unexpected error listing skills: %v", err)
	}

	for _, skill := range skills {
		content, err := GetSkill(skill)
		if err != nil {
			t.Errorf("failed to read skill %s: %v", skill, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("skill %s has empty content", skill)
		}
	}
}

func TestGetTemplate_ContentContainsMarkdown(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, tmpl := range templates {
		content, err := GetTemplate(tmpl)
		if err != nil {
			t.Errorf("failed to read template %s: %v", tmpl, err)
			continue
		}

		contentStr := string(content)
		// Each template should have at least one markdown heading
		if !strings.Contains(contentStr, "#") {
			t.Errorf("template %s does not appear to contain markdown headings", tmpl)
		}
	}
}

func TestListWorkflowTemplates(t *testing.T) {
	workflows, err := ListWorkflowTemplates()
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

func TestListWorkflowTemplates_ExpectedWorkflows(t *testing.T) {
	workflows, err := ListWorkflowTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"claude-code.yml",
	}

	workflowSet := make(map[string]bool)
	for _, wf := range workflows {
		workflowSet[wf] = true
	}

	for _, exp := range expected {
		if !workflowSet[exp] {
			t.Errorf("expected workflow %s not found in list", exp)
		}
	}
}

func TestGetWorkflowTemplate_ValidTemplate(t *testing.T) {
	content, err := GetWorkflowTemplate("claude-code.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty workflow template content")
	}

	// Should contain YAML workflow name
	if !strings.Contains(string(content), "name:") {
		t.Error("expected workflow template to contain 'name:' field")
	}
}

func TestGetWorkflowTemplate_NonExistentTemplate(t *testing.T) {
	_, err := GetWorkflowTemplate("nonexistent-workflow.yml")
	if err == nil {
		t.Error("expected error for non-existent workflow template, got nil")
	}
}

func TestGetWorkflowTemplate_AllWorkflowsReadable(t *testing.T) {
	workflows, err := ListWorkflowTemplates()
	if err != nil {
		t.Fatalf("unexpected error listing workflows: %v", err)
	}

	for _, wf := range workflows {
		content, err := GetWorkflowTemplate(wf)
		if err != nil {
			t.Errorf("failed to read workflow %s: %v", wf, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("workflow %s has empty content", wf)
		}
	}
}

// --- LoadFluxDefaults tests ---

func TestLoadFluxDefaults_ReturnsValues(t *testing.T) {
	vals, err := LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) == 0 {
		t.Fatal("expected non-empty flux defaults")
	}
}

func TestLoadFluxDefaults_ContainsExpectedKeys(t *testing.T) {
	vals, err := LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedKeys := []string{
		"default_board",
		"scm_provider",
		"scm_cli",
		"scm_base_url",
		"issue_view_cmd",
		"pr_create_cmd",
		"api_cmd",
		"auth_check_cmd",
	}
	for _, key := range expectedKeys {
		if _, exists := vals[key]; !exists {
			t.Errorf("expected flux defaults to contain key %q", key)
		}
	}
}

func TestLoadFluxDefaults_ValuesMatchExpected(t *testing.T) {
	vals, err := LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]string{
		"default_board": "Engineering",
		"scm_provider":  "GitHub",
		"scm_cli":       "gh",
		"scm_base_url":  "https://github.com",
	}
	for key, expected := range checks {
		if vals[key] != expected {
			t.Errorf("expected %s=%q, got %q", key, expected, vals[key])
		}
	}
}

func TestLoadFluxDefaults_MultilineValues(t *testing.T) {
	vals, err := LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// api_reply_to_review_comment_cmd has a multiline default
	val, exists := vals["api_reply_to_review_comment_cmd"]
	if !exists {
		t.Fatal("expected api_reply_to_review_comment_cmd to exist")
	}
	if !strings.Contains(val, "--method POST") {
		t.Errorf("expected multiline value to contain '--method POST', got %q", val)
	}
	if !strings.Contains(val, "gh api") {
		t.Errorf("expected multiline value to start with 'gh api', got %q", val)
	}
}

// --- LoadFluxSchema tests ---

func TestLoadFluxSchema_ReturnsSchema(t *testing.T) {
	schema, err := LoadFluxSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema (flux.schema.yaml exists)")
	}
}

func TestLoadFluxSchema_ContainsOrganizationRequired(t *testing.T) {
	schema, err := LoadFluxSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, entry := range schema {
		if entry.Name == "organization" {
			found = true
			if !entry.Required {
				t.Error("expected 'organization' to be required in schema")
			}
			if entry.Type != "string" {
				t.Errorf("expected 'organization' type=string, got %q", entry.Type)
			}
		}
	}
	if !found {
		t.Error("expected schema to contain 'organization' entry")
	}
}

func TestLoadFluxSchema_IsMinimal(t *testing.T) {
	schema, err := LoadFluxSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults, err := LoadFluxDefaults()
	if err != nil {
		t.Fatalf("unexpected error loading defaults: %v", err)
	}

	// Schema should be much smaller than flux defaults — only what needs validation
	if len(schema) >= len(defaults) {
		t.Errorf("expected schema (%d entries) to be smaller than flux defaults (%d keys)",
			len(schema), len(defaults))
	}
}

// --- Lean mold.yaml tests ---

func TestLoadManifest_NoInlineFluxSection(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Flux) > 0 {
		t.Errorf("expected mold.yaml to have no inline flux declarations (Helm-style), got %d entries", len(manifest.Flux))
	}
}

func TestLoadManifest_HasMetadata(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Name != "ailloy-defaults" {
		t.Errorf("expected name=ailloy-defaults, got %q", manifest.Name)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %q", manifest.Version)
	}
	if len(manifest.Commands) == 0 {
		t.Error("expected mold to have commands")
	}
	if len(manifest.Skills) == 0 {
		t.Error("expected mold to have skills")
	}
	if len(manifest.Workflows) == 0 {
		t.Error("expected mold to have workflows")
	}
}
