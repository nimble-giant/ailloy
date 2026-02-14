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
