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
