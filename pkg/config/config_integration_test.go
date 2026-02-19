package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Integration tests for template customization features

func TestIntegration_VariableSubstitutionWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Step 1: Create a config with flux variables
	cfg := &Config{
		Project: ProjectConfig{
			Name: "test-project",
		},
		Templates: TemplateConfig{
			Flux: map[string]string{
				"organization":   "myteam",
				"default_board":  "Engineering",
				"default_status": "Ready",
			},
		},
	}

	// Step 2: Save the config
	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Step 3: Load the config and verify flux
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Templates.Flux["organization"] != "myteam" {
		t.Errorf("expected organization 'myteam', got '%s'", loaded.Templates.Flux["organization"])
	}

	// Step 4: Process a template with these flux variables
	tmpl := `# Create Issue for {{organization}}

## Board: {{default_board}}
## Status: {{default_status}}
`

	result, err := ProcessTemplate(tmpl, loaded.Templates.Flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == tmpl {
		t.Error("expected template to be processed")
	}
	if !contains(result, "myteam") {
		t.Error("expected 'myteam' in processed template")
	}
	if !contains(result, "Engineering") {
		t.Error("expected 'Engineering' in processed template")
	}
	if !contains(result, "Ready") {
		t.Error("expected 'Ready' in processed template")
	}
}

func TestIntegration_ConfigMerging_GlobalAndProject(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create project config with specific flux
	projectCfg := &Config{
		Project: ProjectConfig{
			Name: "my-project",
		},
		Templates: TemplateConfig{
			Flux: map[string]string{
				"organization":  "project-org",
				"default_board": "ProjectBoard",
			},
		},
	}

	err := SaveConfig(projectCfg, false)
	if err != nil {
		t.Fatalf("failed to save project config: %v", err)
	}

	// Load project config
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load project config: %v", err)
	}

	// Simulate global config merge (as done in copyTemplateFiles)
	globalFlux := map[string]string{
		"organization":    "global-org",
		"default_board":   "GlobalBoard",
		"default_status":  "Backlog",
		"global_only_var": "global-value",
	}

	// Project flux takes precedence
	mergedFlux := make(map[string]string)
	for k, v := range globalFlux {
		mergedFlux[k] = v
	}
	for k, v := range loaded.Templates.Flux {
		mergedFlux[k] = v
	}

	// Verify precedence
	if mergedFlux["organization"] != "project-org" {
		t.Errorf("expected project org to take precedence, got '%s'", mergedFlux["organization"])
	}
	if mergedFlux["default_board"] != "ProjectBoard" {
		t.Errorf("expected project board to take precedence, got '%s'", mergedFlux["default_board"])
	}
	// Global-only flux should be present
	if mergedFlux["default_status"] != "Backlog" {
		t.Errorf("expected global default_status, got '%s'", mergedFlux["default_status"])
	}
	if mergedFlux["global_only_var"] != "global-value" {
		t.Errorf("expected global_only_var, got '%s'", mergedFlux["global_only_var"])
	}
}

func TestIntegration_SetAndDeleteVariables(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Start with empty config
	cfg := &Config{
		Templates: TemplateConfig{
			Flux: map[string]string{},
		},
	}

	// Set flux variables
	cfg.Templates.Flux["key1"] = "value1"
	cfg.Templates.Flux["key2"] = "value2"
	cfg.Templates.Flux["key3"] = "value3"

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if len(loaded.Templates.Flux) != 3 {
		t.Errorf("expected 3 flux variables, got %d", len(loaded.Templates.Flux))
	}

	// Delete a variable
	delete(loaded.Templates.Flux, "key2")

	err = SaveConfig(loaded, false)
	if err != nil {
		t.Fatalf("failed to save after delete: %v", err)
	}

	// Load again and verify deletion
	reloaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	if len(reloaded.Templates.Flux) != 2 {
		t.Errorf("expected 2 flux variables after deletion, got %d", len(reloaded.Templates.Flux))
	}
	if _, exists := reloaded.Templates.Flux["key2"]; exists {
		t.Error("expected key2 to be deleted")
	}
	if reloaded.Templates.Flux["key1"] != "value1" {
		t.Error("expected key1 to still exist")
	}
	if reloaded.Templates.Flux["key3"] != "value3" {
		t.Error("expected key3 to still exist")
	}
}

func TestIntegration_TemplateVariableSubstitution_AllVariables(t *testing.T) {
	// Test with all commonly used template variables
	flux := map[string]string{
		"default_board":      "Engineering",
		"default_priority":   "P1",
		"default_status":     "Ready",
		"organization":       "nimble-giant",
		"project_id":         "PVT_abc123",
		"status_field_id":    "PVTSSF_def456",
		"priority_field_id":  "PVTSSF_ghi789",
		"iteration_field_id": "PVTIF_jkl012",
	}

	template := `Create issue on {{default_board}} board
Priority: {{default_priority}}
Status: {{default_status}}
Organization: {{organization}}
Project: {{project_id}}
Status Field: {{status_field_id}}
Priority Field: {{priority_field_id}}
Iteration Field: {{iteration_field_id}}`

	result, err := ProcessTemplate(template, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for key, value := range flux {
		if contains(result, "{{"+key+"}}") {
			t.Errorf("variable {{%s}} was not substituted", key)
		}
		if !contains(result, value) {
			t.Errorf("expected value '%s' for key '%s' in result", value, key)
		}
	}
}

func TestIntegration_ConfigPersistence_FullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Project: ProjectConfig{
			Name:        "full-project",
			Description: "A fully configured project",
			Providers:   []string{"claude", "gpt"},
			TemplDirs:   []string{"templates/custom"},
		},
		Templates: TemplateConfig{
			DefaultProvider: "claude",
			AutoUpdate:      true,
			Repositories:    []string{"https://github.com/repo1", "https://github.com/repo2"},
			Flux: map[string]string{
				"var1": "val1",
				"var2": "val2",
			},
		},
		Workflows: WorkflowConfig{
			IssueCreation: WorkflowStep{
				Template: "create-issue",
				Provider: "claude",
			},
		},
		User: UserConfig{
			Name:  "Test User",
			Email: "test@example.com",
		},
		Providers: ProvidersConfig{
			Claude: ClaudeConfig{
				Enabled:   true,
				APIKeyEnv: "ANTHROPIC_API_KEY",
			},
			GPT: GPTConfig{
				Enabled:   false,
				APIKeyEnv: "OPENAI_API_KEY",
			},
		},
	}

	// Save
	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load and verify all fields
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.Project.Name != "full-project" {
		t.Errorf("project name: expected 'full-project', got '%s'", loaded.Project.Name)
	}
	if loaded.Templates.DefaultProvider != "claude" {
		t.Errorf("default provider: expected 'claude', got '%s'", loaded.Templates.DefaultProvider)
	}
	if !loaded.Templates.AutoUpdate {
		t.Error("expected auto_update to be true")
	}
	if len(loaded.Templates.Repositories) != 2 {
		t.Errorf("expected 2 repositories, got %d", len(loaded.Templates.Repositories))
	}
	if loaded.Workflows.IssueCreation.Template != "create-issue" {
		t.Errorf("workflow template: expected 'create-issue', got '%s'", loaded.Workflows.IssueCreation.Template)
	}
	if !loaded.Providers.Claude.Enabled {
		t.Error("expected claude to be enabled")
	}
	if loaded.Providers.GPT.Enabled {
		t.Error("expected gpt to be disabled")
	}
}

func TestIntegration_ConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Templates: TemplateConfig{
			Flux: map[string]string{},
		},
	}

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Check config file permissions
	configPath := filepath.Join(".ailloy", "ailloy.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config: %v", err)
	}

	// Should be 0600 (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected file permissions 0600, got %o", perm)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
