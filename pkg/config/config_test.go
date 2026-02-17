package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir_Project(t *testing.T) {
	dir, err := GetConfigDir(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != ".ailloy" {
		t.Errorf("expected .ailloy, got %s", dir)
	}
}

func TestGetConfigDir_Global(t *testing.T) {
	dir, err := GetConfigDir(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".ailloy")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetConfigPath_Project(t *testing.T) {
	path, err := GetConfigPath(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(".ailloy", "ailloy.yaml")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestGetConfigPath_Global(t *testing.T) {
	path, err := GetConfigPath(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".ailloy", "ailloy.yaml")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Use a temp dir so the config file doesn't exist
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Templates.Variables == nil {
		t.Error("expected Variables map to be initialized")
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	yamlContent := `project:
  name: test-project
  description: A test project
templates:
  default_provider: claude
  variables:
    org: myorg
    board: Engineering
user:
  name: Test User
  email: test@example.com
providers:
  claude:
    enabled: true
    api_key_env: ANTHROPIC_API_KEY
`
	configPath := filepath.Join(configDir, "ailloy.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", cfg.Project.Name)
	}
	if cfg.Project.Description != "A test project" {
		t.Errorf("expected description 'A test project', got '%s'", cfg.Project.Description)
	}
	if cfg.Templates.DefaultProvider != "claude" {
		t.Errorf("expected default provider 'claude', got '%s'", cfg.Templates.DefaultProvider)
	}
	if cfg.Templates.Variables["org"] != "myorg" {
		t.Errorf("expected variable org=myorg, got '%s'", cfg.Templates.Variables["org"])
	}
	if cfg.Templates.Variables["board"] != "Engineering" {
		t.Errorf("expected variable board=Engineering, got '%s'", cfg.Templates.Variables["board"])
	}
	if cfg.User.Name != "Test User" {
		t.Errorf("expected user name 'Test User', got '%s'", cfg.User.Name)
	}
	if cfg.User.Email != "test@example.com" {
		t.Errorf("expected user email 'test@example.com', got '%s'", cfg.User.Email)
	}
	if !cfg.Providers.Claude.Enabled {
		t.Error("expected claude provider to be enabled")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	invalidYAML := `{invalid: yaml: [broken`
	configPath := filepath.Join(configDir, "ailloy.yaml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadConfig(false)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfig_NilVariablesInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// YAML without variables section
	yamlContent := `project:
  name: test
`
	configPath := filepath.Join(configDir, "ailloy.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Templates.Variables == nil {
		t.Error("expected Variables map to be initialized even when not in YAML")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Project: ProjectConfig{
			Name:        "saved-project",
			Description: "A saved project",
		},
		Templates: TemplateConfig{
			DefaultProvider: "claude",
			Variables: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		User: UserConfig{
			Name:  "Saver",
			Email: "saver@example.com",
		},
	}

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	// Verify the file was created
	configPath := filepath.Join(".ailloy", "ailloy.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back and verify
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.Project.Name != "saved-project" {
		t.Errorf("expected project name 'saved-project', got '%s'", loaded.Project.Name)
	}
	if loaded.Templates.Variables["key1"] != "value1" {
		t.Errorf("expected variable key1=value1, got '%s'", loaded.Templates.Variables["key1"])
	}
	if loaded.Templates.Variables["key2"] != "value2" {
		t.Errorf("expected variable key2=value2, got '%s'", loaded.Templates.Variables["key2"])
	}
}

func TestSaveConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Templates: TemplateConfig{
			Variables: map[string]string{},
		},
	}

	// Directory shouldn't exist yet
	if _, err := os.Stat(".ailloy"); !os.IsNotExist(err) {
		t.Fatal("expected .ailloy directory to not exist initially")
	}

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now directory should exist
	info, err := os.Stat(".ailloy")
	if err != nil {
		t.Fatalf("expected .ailloy directory to be created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .ailloy to be a directory")
	}
}

func TestProcessTemplate_BasicSubstitution(t *testing.T) {
	content := "Hello {{name}}, welcome to {{project}}!"
	variables := map[string]string{
		"name":    "Alice",
		"project": "Ailloy",
	}

	result := ProcessTemplate(content, variables)
	expected := "Hello Alice, welcome to Ailloy!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_NoVariables(t *testing.T) {
	content := "Hello {{name}}, welcome to {{project}}!"
	variables := map[string]string{}

	result := ProcessTemplate(content, variables)
	// Should remain unchanged
	if result != content {
		t.Errorf("expected content to remain unchanged, got %q", result)
	}
}

func TestProcessTemplate_EmptyContent(t *testing.T) {
	result := ProcessTemplate("", map[string]string{"key": "value"})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestProcessTemplate_MultipleOccurrences(t *testing.T) {
	content := "{{name}} said: Hello {{name}}!"
	variables := map[string]string{"name": "Bob"}

	result := ProcessTemplate(content, variables)
	expected := "Bob said: Hello Bob!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_PartialMatch(t *testing.T) {
	content := "Use {{board}} for issues on {{board_id}}"
	variables := map[string]string{
		"board":    "Engineering",
		"board_id": "12345",
	}

	result := ProcessTemplate(content, variables)
	expected := "Use Engineering for issues on 12345"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_NilVariables(t *testing.T) {
	content := "Hello {{name}}!"
	result := ProcessTemplate(content, nil)
	if result != content {
		t.Errorf("expected content to remain unchanged with nil variables, got %q", result)
	}
}

func TestProcessTemplate_SpecialCharacters(t *testing.T) {
	content := "URL: {{url}}"
	variables := map[string]string{
		"url": "https://example.com/path?query=1&other=2",
	}
	result := ProcessTemplate(content, variables)
	expected := "URL: https://example.com/path?query=1&other=2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Model Layer Tests ---

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	// Status model
	if models.Status.Enabled {
		t.Error("expected status model to be disabled by default")
	}
	if len(models.Status.Options) != 4 {
		t.Errorf("expected 4 status options, got %d", len(models.Status.Options))
	}
	if models.Status.Options["ready"].Label != "Ready" {
		t.Errorf("expected ready label 'Ready', got %q", models.Status.Options["ready"].Label)
	}
	if models.Status.Options["in_progress"].Label != "In Progress" {
		t.Errorf("expected in_progress label 'In Progress', got %q", models.Status.Options["in_progress"].Label)
	}
	if models.Status.Options["in_review"].Label != "In Review" {
		t.Errorf("expected in_review label 'In Review', got %q", models.Status.Options["in_review"].Label)
	}
	if models.Status.Options["done"].Label != "Done" {
		t.Errorf("expected done label 'Done', got %q", models.Status.Options["done"].Label)
	}

	// Priority model
	if models.Priority.Enabled {
		t.Error("expected priority model to be disabled by default")
	}
	if len(models.Priority.Options) != 4 {
		t.Errorf("expected 4 priority options, got %d", len(models.Priority.Options))
	}
	for _, key := range []string{"p0", "p1", "p2", "p3"} {
		if _, ok := models.Priority.Options[key]; !ok {
			t.Errorf("expected priority option %q to exist", key)
		}
	}

	// Iteration model
	if models.Iteration.Enabled {
		t.Error("expected iteration model to be disabled by default")
	}
	if models.Iteration.Options != nil {
		t.Error("expected iteration model to have no predefined options")
	}
}

func TestLoadConfig_WithModels(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	yamlContent := `project:
  name: test-project
models:
  status:
    enabled: true
    field_mapping: "Vibes"
    field_id: "PVTSSF_abc123"
    options:
      ready:
        label: "Not Started"
      in_progress:
        label: "Doing"
        id: "opt_123"
  priority:
    enabled: true
    field_mapping: "Priority"
    options:
      p0:
        label: "Critical"
      p1:
        label: "High"
  iteration:
    enabled: false
`
	configPath := filepath.Join(configDir, "ailloy.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Status model
	if !cfg.Models.Status.Enabled {
		t.Error("expected status model to be enabled")
	}
	if cfg.Models.Status.FieldMapping != "Vibes" {
		t.Errorf("expected field mapping 'Vibes', got %q", cfg.Models.Status.FieldMapping)
	}
	if cfg.Models.Status.FieldID != "PVTSSF_abc123" {
		t.Errorf("expected field ID 'PVTSSF_abc123', got %q", cfg.Models.Status.FieldID)
	}
	// User-customized labels should be preserved
	if cfg.Models.Status.Options["ready"].Label != "Not Started" {
		t.Errorf("expected ready label 'Not Started', got %q", cfg.Models.Status.Options["ready"].Label)
	}
	if cfg.Models.Status.Options["in_progress"].Label != "Doing" {
		t.Errorf("expected in_progress label 'Doing', got %q", cfg.Models.Status.Options["in_progress"].Label)
	}
	if cfg.Models.Status.Options["in_progress"].ID != "opt_123" {
		t.Errorf("expected in_progress ID 'opt_123', got %q", cfg.Models.Status.Options["in_progress"].ID)
	}
	// Default options should be filled in for missing keys
	if _, ok := cfg.Models.Status.Options["in_review"]; !ok {
		t.Error("expected in_review option to be filled in from defaults")
	}
	if _, ok := cfg.Models.Status.Options["done"]; !ok {
		t.Error("expected done option to be filled in from defaults")
	}

	// Priority model - user defined only p0 and p1, p2 and p3 should be filled in
	if !cfg.Models.Priority.Enabled {
		t.Error("expected priority model to be enabled")
	}
	if cfg.Models.Priority.Options["p0"].Label != "Critical" {
		t.Errorf("expected p0 label 'Critical', got %q", cfg.Models.Priority.Options["p0"].Label)
	}
	if _, ok := cfg.Models.Priority.Options["p2"]; !ok {
		t.Error("expected p2 option to be filled in from defaults")
	}
	if _, ok := cfg.Models.Priority.Options["p3"]; !ok {
		t.Error("expected p3 option to be filled in from defaults")
	}

	// Iteration should remain disabled
	if cfg.Models.Iteration.Enabled {
		t.Error("expected iteration model to be disabled")
	}
}

func TestLoadConfig_WithoutModels_GetsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// YAML without models section
	yamlContent := `project:
  name: legacy-project
templates:
  variables:
    default_status: "Ready"
    default_priority: "P1"
`
	configPath := filepath.Join(configDir, "ailloy.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Models should be initialized with defaults
	if len(cfg.Models.Status.Options) != 4 {
		t.Errorf("expected 4 default status options, got %d", len(cfg.Models.Status.Options))
	}
	if len(cfg.Models.Priority.Options) != 4 {
		t.Errorf("expected 4 default priority options, got %d", len(cfg.Models.Priority.Options))
	}
	// Existing flat variables should be untouched
	if cfg.Templates.Variables["default_status"] != "Ready" {
		t.Errorf("expected flat variable default_status='Ready', got %q", cfg.Templates.Variables["default_status"])
	}
}

func TestSaveConfig_WithModels(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Project: ProjectConfig{Name: "model-project"},
		Templates: TemplateConfig{
			Variables: map[string]string{"org": "myorg"},
		},
		Models: Models{
			Status: ModelConfig{
				Enabled:      true,
				FieldMapping: "Status",
				FieldID:      "PVTSSF_xyz",
				Options: map[string]ModelOption{
					"ready":       {Label: "Ready", ID: "opt_1"},
					"in_progress": {Label: "In Progress", ID: "opt_2"},
				},
			},
			Priority: ModelConfig{
				Enabled: false,
			},
		},
	}

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	// Load it back and verify models round-trip
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if !loaded.Models.Status.Enabled {
		t.Error("expected status model to be enabled after round-trip")
	}
	if loaded.Models.Status.FieldMapping != "Status" {
		t.Errorf("expected field mapping 'Status', got %q", loaded.Models.Status.FieldMapping)
	}
	if loaded.Models.Status.FieldID != "PVTSSF_xyz" {
		t.Errorf("expected field ID 'PVTSSF_xyz', got %q", loaded.Models.Status.FieldID)
	}
	if loaded.Models.Status.Options["ready"].ID != "opt_1" {
		t.Errorf("expected ready ID 'opt_1', got %q", loaded.Models.Status.Options["ready"].ID)
	}
}

func TestModelsToVariables(t *testing.T) {
	models := Models{
		Status: ModelConfig{
			Enabled: true,
			FieldID: "PVTSSF_status",
			Options: map[string]ModelOption{
				"ready":       {Label: "Not Started"},
				"in_progress": {Label: "Doing"},
			},
		},
		Priority: ModelConfig{
			Enabled: true,
			FieldID: "PVTSSF_priority",
			Options: map[string]ModelOption{
				"p1": {Label: "High"},
			},
		},
		Iteration: ModelConfig{
			Enabled: true,
			FieldID: "PVTIF_iteration",
		},
	}

	vars := ModelsToVariables(models)

	if vars["default_status"] != "Not Started" {
		t.Errorf("expected default_status='Not Started', got %q", vars["default_status"])
	}
	if vars["status_field_id"] != "PVTSSF_status" {
		t.Errorf("expected status_field_id='PVTSSF_status', got %q", vars["status_field_id"])
	}
	if vars["default_priority"] != "High" {
		t.Errorf("expected default_priority='High', got %q", vars["default_priority"])
	}
	if vars["priority_field_id"] != "PVTSSF_priority" {
		t.Errorf("expected priority_field_id='PVTSSF_priority', got %q", vars["priority_field_id"])
	}
	if vars["iteration_field_id"] != "PVTIF_iteration" {
		t.Errorf("expected iteration_field_id='PVTIF_iteration', got %q", vars["iteration_field_id"])
	}
}

func TestModelsToVariables_DisabledModelsSkipped(t *testing.T) {
	models := Models{
		Status: ModelConfig{
			Enabled: false,
			FieldID: "PVTSSF_status",
			Options: map[string]ModelOption{
				"ready": {Label: "Ready"},
			},
		},
		Priority: ModelConfig{
			Enabled: false,
		},
	}

	vars := ModelsToVariables(models)

	if _, exists := vars["default_status"]; exists {
		t.Error("expected disabled status model to not generate default_status variable")
	}
	if _, exists := vars["status_field_id"]; exists {
		t.Error("expected disabled status model to not generate status_field_id variable")
	}
	if _, exists := vars["default_priority"]; exists {
		t.Error("expected disabled priority model to not generate default_priority variable")
	}
}

func TestMergeModelVariables_DoesNotOverwriteExisting(t *testing.T) {
	cfg := &Config{
		Templates: TemplateConfig{
			Variables: map[string]string{
				"default_status":   "Custom Status",
				"default_priority": "Custom Priority",
			},
		},
		Models: Models{
			Status: ModelConfig{
				Enabled: true,
				FieldID: "PVTSSF_status",
				Options: map[string]ModelOption{
					"ready": {Label: "Model Status"},
				},
			},
			Priority: ModelConfig{
				Enabled: true,
				FieldID: "PVTSSF_priority",
				Options: map[string]ModelOption{
					"p1": {Label: "Model Priority"},
				},
			},
		},
	}

	MergeModelVariables(cfg)

	// Existing flat variables should NOT be overwritten
	if cfg.Templates.Variables["default_status"] != "Custom Status" {
		t.Errorf("expected existing default_status to be preserved, got %q", cfg.Templates.Variables["default_status"])
	}
	if cfg.Templates.Variables["default_priority"] != "Custom Priority" {
		t.Errorf("expected existing default_priority to be preserved, got %q", cfg.Templates.Variables["default_priority"])
	}

	// But field IDs that weren't in flat variables should be added
	if cfg.Templates.Variables["status_field_id"] != "PVTSSF_status" {
		t.Errorf("expected status_field_id to be added from model, got %q", cfg.Templates.Variables["status_field_id"])
	}
	if cfg.Templates.Variables["priority_field_id"] != "PVTSSF_priority" {
		t.Errorf("expected priority_field_id to be added from model, got %q", cfg.Templates.Variables["priority_field_id"])
	}
}
