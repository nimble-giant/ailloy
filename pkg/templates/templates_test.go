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
