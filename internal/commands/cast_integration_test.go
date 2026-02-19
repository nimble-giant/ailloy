package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/config"
	embeddedtemplates "github.com/nimble-giant/ailloy/pkg/templates"
)

func TestIntegration_CopyTemplateFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create required directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	err := copyTemplateFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = copySkillFiles()
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	// Verify all expected templates were created
	expectedTemplates := []string{
		"brainstorm.md",
		"pr-description.md",
		"create-issue.md",
		"start-issue.md",
		"update-pr.md",
		"open-pr.md",
		"preflight.md",
		"pr-comments.md",
		"pr-review.md",
	}

	for _, tmpl := range expectedTemplates {
		path := filepath.Join(".claude", "commands", tmpl)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected template %s to be created: %v", tmpl, err)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", tmpl, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("template %s is empty", tmpl)
		}
	}

	// Verify all expected skills were created
	expectedSkills := []string{
		"brainstorm.md",
	}

	for _, skill := range expectedSkills {
		path := filepath.Join(".claude", "skills", skill)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected skill %s to be created: %v", skill, err)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read skill %s: %v", skill, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("skill %s is empty", skill)
		}
	}
}

func TestIntegration_CopyTemplateFiles_WithVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create config with variables
	cfg := &config.Config{
		Templates: config.TemplateConfig{
			Variables: map[string]string{
				"organization":  "test-org",
				"default_board": "TestBoard",
			},
		},
	}
	if err := config.SaveConfig(cfg, false); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Run copyTemplateFiles
	err := copyTemplateFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that templates exist and are non-empty
	entries, err := os.ReadDir(".claude/commands")
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected templates to be copied")
	}

	// Verify files are readable
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(".claude", "commands", entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
		}
		if len(content) == 0 {
			t.Errorf("%s is empty", entry.Name())
		}
	}
}

func TestIntegration_TemplateFilesMatchEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	// Copy templates (no variable substitution since no config)
	err := copyTemplateFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Copy skills
	err = copySkillFiles()
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	// List embedded templates
	embeddedList, err := embeddedtemplates.ListTemplates()
	if err != nil {
		t.Fatalf("failed to list embedded templates: %v", err)
	}

	// Verify each embedded template has a corresponding file
	// Note: templates are processed through the Go template engine,
	// so we compare against the processed embedded content (not raw source)
	for _, tmplName := range embeddedList {
		embeddedContent, err := embeddedtemplates.GetTemplate(tmplName)
		if err != nil {
			t.Errorf("failed to get embedded template %s: %v", tmplName, err)
			continue
		}

		// Process embedded content through the same template engine
		// (with default models and no variables, matching what copyTemplateFiles does)
		defaultModels := config.DefaultModels()
		expectedContent, err := config.ProcessTemplate(
			string(embeddedContent),
			make(map[string]string),
			&defaultModels,
		)
		if err != nil {
			t.Errorf("failed to process embedded template %s: %v", tmplName, err)
			continue
		}

		filePath := filepath.Join(".claude", "commands", tmplName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied template %s: %v", tmplName, err)
			continue
		}

		if expectedContent != string(fileContent) {
			t.Errorf("template %s content mismatch between processed embedded and copied version", tmplName)
		}
	}

	// List embedded skills
	embeddedSkills, err := embeddedtemplates.ListSkills()
	if err != nil {
		t.Fatalf("failed to list embedded skills: %v", err)
	}

	// Verify each embedded skill has a corresponding file
	for _, skillName := range embeddedSkills {
		embeddedContent, err := embeddedtemplates.GetSkill(skillName)
		if err != nil {
			t.Errorf("failed to get embedded skill %s: %v", skillName, err)
			continue
		}

		// Process embedded content through the same template engine
		defaultModels := config.DefaultModels()
		expectedContent, err := config.ProcessTemplate(
			string(embeddedContent),
			make(map[string]string),
			&defaultModels,
		)
		if err != nil {
			t.Errorf("failed to process embedded skill %s: %v", skillName, err)
			continue
		}

		filePath := filepath.Join(".claude", "skills", skillName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied skill %s: %v", skillName, err)
			continue
		}

		if expectedContent != string(fileContent) {
			t.Errorf("skill %s content mismatch between processed embedded and copied version", skillName)
		}
	}
}

func TestIntegration_CastProject_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Simulate the directory creation part of castProject (default, no workflows)
	dirs := []string{
		".claude",
		".claude/commands",
		".claude/skills",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Verify directories were created
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestIntegration_CastGlobal_DirectoryCreation(t *testing.T) {
	// Create a temp "home" directory to avoid modifying real home
	tmpHome := t.TempDir()

	globalDir := filepath.Join(tmpHome, ".ailloy")
	dirs := []string{
		globalDir,
		filepath.Join(globalDir, "templates"),
		filepath.Join(globalDir, "providers"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Verify directories
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	// Create global config file
	configPath := filepath.Join(globalDir, "ailloy.yaml")
	configContent := `user:
  name: "Test User"
  email: "test@example.com"
providers:
  claude:
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	// Verify config file
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected config permissions 0600, got %o", info.Mode().Perm())
	}

	// Verify config content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "claude") {
		t.Error("expected config to mention claude provider")
	}
}

func TestIntegration_TemplateFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	err := copyTemplateFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = copySkillFiles()
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	// Check permissions of created template files
	checkPermissions := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir %s: %v", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("failed to stat %s: %v", entry.Name(), err)
				continue
			}
			perm := info.Mode().Perm()
			if perm != 0644 {
				t.Errorf("expected permissions 0644 for %s, got %o", path, perm)
			}
		}
	}

	checkPermissions(".claude/commands")
	checkPermissions(".claude/skills")
}

func TestIntegration_CastProject_DefaultSkipsWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		withWorkflows = false
		_ = os.Chdir(origDir)
	}()

	// Ensure withWorkflows is false (default)
	withWorkflows = false

	err := castProject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// .claude/commands should exist
	if _, err := os.Stat(".claude/commands"); err != nil {
		t.Error("expected .claude/commands to be created")
	}

	// .github/workflows should NOT exist
	if _, err := os.Stat(".github/workflows"); err == nil {
		t.Error("expected .github/workflows to NOT be created by default")
	}

	// No workflow files should exist
	if _, err := os.Stat(filepath.Join(".github", "workflows", "claude-code.yml")); err == nil {
		t.Error("expected claude-code.yml to NOT be created by default")
	}
}

func TestIntegration_CastProject_WithWorkflowsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		withWorkflows = false
		_ = os.Chdir(origDir)
	}()

	// Enable workflows
	withWorkflows = true

	err := castProject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// .claude/commands should exist
	if _, err := os.Stat(".claude/commands"); err != nil {
		t.Error("expected .claude/commands to be created")
	}

	// .github/workflows should exist
	if _, err := os.Stat(".github/workflows"); err != nil {
		t.Error("expected .github/workflows to be created with --with-workflows")
	}

	// Workflow file should exist and be non-empty
	wfPath := filepath.Join(".github", "workflows", "claude-code.yml")
	content, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("expected claude-code.yml to be created: %v", err)
	}
	if len(content) == 0 {
		t.Error("workflow file is empty")
	}
	if !strings.Contains(string(content), "claude-code-action") {
		t.Error("workflow file should reference claude-code-action")
	}
}

func TestIntegration_CopyWorkflowTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(".github/workflows", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	err := copyWorkflowTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedWorkflows := []string{
		"claude-code.yml",
	}

	for _, wf := range expectedWorkflows {
		path := filepath.Join(".github", "workflows", wf)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected workflow %s to be created: %v", wf, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("workflow %s is empty", wf)
		}
		if !strings.Contains(string(content), "name:") {
			t.Errorf("workflow %s does not contain YAML name field", wf)
		}
	}
}

func TestIntegration_WorkflowFilesMatchEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(".github/workflows", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	err := copyWorkflowTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	embeddedList, err := embeddedtemplates.ListWorkflowTemplates()
	if err != nil {
		t.Fatalf("failed to list embedded workflows: %v", err)
	}

	for _, wfName := range embeddedList {
		embeddedContent, err := embeddedtemplates.GetWorkflowTemplate(wfName)
		if err != nil {
			t.Errorf("failed to get embedded workflow %s: %v", wfName, err)
			continue
		}

		filePath := filepath.Join(".github", "workflows", wfName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied workflow %s: %v", wfName, err)
			continue
		}

		if string(embeddedContent) != string(fileContent) {
			t.Errorf("workflow %s content mismatch between embedded and copied version", wfName)
		}
	}
}
