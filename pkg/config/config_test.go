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
	if cfg.Templates.Flux == nil {
		t.Error("expected Flux map to be initialized")
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
  flux:
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
	if cfg.Templates.Flux["org"] != "myorg" {
		t.Errorf("expected flux org=myorg, got '%s'", cfg.Templates.Flux["org"])
	}
	if cfg.Templates.Flux["board"] != "Engineering" {
		t.Errorf("expected flux board=Engineering, got '%s'", cfg.Templates.Flux["board"])
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

func TestLoadConfig_NilFluxInitialized(t *testing.T) {
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

	// YAML without flux section
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
	if cfg.Templates.Flux == nil {
		t.Error("expected Flux map to be initialized even when not in YAML")
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
			Flux: map[string]string{
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
	if loaded.Templates.Flux["key1"] != "value1" {
		t.Errorf("expected flux key1=value1, got '%s'", loaded.Templates.Flux["key1"])
	}
	if loaded.Templates.Flux["key2"] != "value2" {
		t.Errorf("expected flux key2=value2, got '%s'", loaded.Templates.Flux["key2"])
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
			Flux: map[string]string{},
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
	flux := map[string]string{
		"name":    "Alice",
		"project": "Ailloy",
	}

	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Hello Alice, welcome to Ailloy!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_NoVariables(t *testing.T) {
	content := "Hello world, no placeholders here!"
	flux := map[string]string{}

	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected content to remain unchanged, got %q", result)
	}
}

func TestProcessTemplate_EmptyContent(t *testing.T) {
	result, err := ProcessTemplate("", map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestProcessTemplate_MultipleOccurrences(t *testing.T) {
	content := "{{name}} said: Hello {{name}}!"
	flux := map[string]string{"name": "Bob"}

	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Bob said: Hello Bob!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_PartialMatch(t *testing.T) {
	content := "Use {{board}} for issues on {{board_id}}"
	flux := map[string]string{
		"board":    "Engineering",
		"board_id": "12345",
	}

	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Use Engineering for issues on 12345"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_NilVariables(t *testing.T) {
	content := "Hello world!"
	result, err := ProcessTemplate(content, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected content to remain unchanged with nil variables, got %q", result)
	}
}

func TestProcessTemplate_SpecialCharacters(t *testing.T) {
	content := "URL: {{url}}"
	flux := map[string]string{
		"url": "https://example.com/path?query=1&other=2",
	}
	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "URL: https://example.com/path?query=1&other=2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Ore Layer Tests ---

func TestDefaultOre(t *testing.T) {
	ore := DefaultOre()

	// Status ore
	if ore.Status.Enabled {
		t.Error("expected status ore to be disabled by default")
	}
	if len(ore.Status.Options) != 4 {
		t.Errorf("expected 4 status options, got %d", len(ore.Status.Options))
	}
	if ore.Status.Options["ready"].Label != "Ready" {
		t.Errorf("expected ready label 'Ready', got %q", ore.Status.Options["ready"].Label)
	}
	if ore.Status.Options["in_progress"].Label != "In Progress" {
		t.Errorf("expected in_progress label 'In Progress', got %q", ore.Status.Options["in_progress"].Label)
	}
	if ore.Status.Options["in_review"].Label != "In Review" {
		t.Errorf("expected in_review label 'In Review', got %q", ore.Status.Options["in_review"].Label)
	}
	if ore.Status.Options["done"].Label != "Done" {
		t.Errorf("expected done label 'Done', got %q", ore.Status.Options["done"].Label)
	}

	// Priority ore
	if ore.Priority.Enabled {
		t.Error("expected priority ore to be disabled by default")
	}
	if len(ore.Priority.Options) != 4 {
		t.Errorf("expected 4 priority options, got %d", len(ore.Priority.Options))
	}
	for _, key := range []string{"p0", "p1", "p2", "p3"} {
		if _, ok := ore.Priority.Options[key]; !ok {
			t.Errorf("expected priority option %q to exist", key)
		}
	}

	// Iteration ore
	if ore.Iteration.Enabled {
		t.Error("expected iteration ore to be disabled by default")
	}
	if ore.Iteration.Options != nil {
		t.Error("expected iteration ore to have no predefined options")
	}
}

func TestLoadConfig_WithOre(t *testing.T) {
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
ore:
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

	// Status ore
	if !cfg.Ore.Status.Enabled {
		t.Error("expected status ore to be enabled")
	}
	if cfg.Ore.Status.FieldMapping != "Vibes" {
		t.Errorf("expected field mapping 'Vibes', got %q", cfg.Ore.Status.FieldMapping)
	}
	if cfg.Ore.Status.FieldID != "PVTSSF_abc123" {
		t.Errorf("expected field ID 'PVTSSF_abc123', got %q", cfg.Ore.Status.FieldID)
	}
	// User-customized labels should be preserved
	if cfg.Ore.Status.Options["ready"].Label != "Not Started" {
		t.Errorf("expected ready label 'Not Started', got %q", cfg.Ore.Status.Options["ready"].Label)
	}
	if cfg.Ore.Status.Options["in_progress"].Label != "Doing" {
		t.Errorf("expected in_progress label 'Doing', got %q", cfg.Ore.Status.Options["in_progress"].Label)
	}
	if cfg.Ore.Status.Options["in_progress"].ID != "opt_123" {
		t.Errorf("expected in_progress ID 'opt_123', got %q", cfg.Ore.Status.Options["in_progress"].ID)
	}
	// Default options should be filled in for missing keys
	if _, ok := cfg.Ore.Status.Options["in_review"]; !ok {
		t.Error("expected in_review option to be filled in from defaults")
	}
	if _, ok := cfg.Ore.Status.Options["done"]; !ok {
		t.Error("expected done option to be filled in from defaults")
	}

	// Priority ore - user defined only p0 and p1, p2 and p3 should be filled in
	if !cfg.Ore.Priority.Enabled {
		t.Error("expected priority ore to be enabled")
	}
	if cfg.Ore.Priority.Options["p0"].Label != "Critical" {
		t.Errorf("expected p0 label 'Critical', got %q", cfg.Ore.Priority.Options["p0"].Label)
	}
	if _, ok := cfg.Ore.Priority.Options["p2"]; !ok {
		t.Error("expected p2 option to be filled in from defaults")
	}
	if _, ok := cfg.Ore.Priority.Options["p3"]; !ok {
		t.Error("expected p3 option to be filled in from defaults")
	}

	// Iteration should remain disabled
	if cfg.Ore.Iteration.Enabled {
		t.Error("expected iteration ore to be disabled")
	}
}

func TestLoadConfig_WithoutOre_GetsDefaults(t *testing.T) {
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

	// YAML without ore section
	yamlContent := `project:
  name: legacy-project
templates:
  flux:
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

	// Ore should be initialized with defaults
	if len(cfg.Ore.Status.Options) != 4 {
		t.Errorf("expected 4 default status options, got %d", len(cfg.Ore.Status.Options))
	}
	if len(cfg.Ore.Priority.Options) != 4 {
		t.Errorf("expected 4 default priority options, got %d", len(cfg.Ore.Priority.Options))
	}
	// Existing flat flux should be untouched
	if cfg.Templates.Flux["default_status"] != "Ready" {
		t.Errorf("expected flat flux default_status='Ready', got %q", cfg.Templates.Flux["default_status"])
	}
}

func TestSaveConfig_WithOre(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &Config{
		Project: ProjectConfig{Name: "ore-project"},
		Templates: TemplateConfig{
			Flux: map[string]string{"org": "myorg"},
		},
		Ore: Ore{
			Status: OreConfig{
				Enabled:      true,
				FieldMapping: "Status",
				FieldID:      "PVTSSF_xyz",
				Options: map[string]OreOption{
					"ready":       {Label: "Ready", ID: "opt_1"},
					"in_progress": {Label: "In Progress", ID: "opt_2"},
				},
			},
			Priority: OreConfig{
				Enabled: false,
			},
		},
	}

	err := SaveConfig(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	// Load it back and verify ore round-trip
	loaded, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if !loaded.Ore.Status.Enabled {
		t.Error("expected status ore to be enabled after round-trip")
	}
	if loaded.Ore.Status.FieldMapping != "Status" {
		t.Errorf("expected field mapping 'Status', got %q", loaded.Ore.Status.FieldMapping)
	}
	if loaded.Ore.Status.FieldID != "PVTSSF_xyz" {
		t.Errorf("expected field ID 'PVTSSF_xyz', got %q", loaded.Ore.Status.FieldID)
	}
	if loaded.Ore.Status.Options["ready"].ID != "opt_1" {
		t.Errorf("expected ready ID 'opt_1', got %q", loaded.Ore.Status.Options["ready"].ID)
	}
}

func TestOreToFlux(t *testing.T) {
	ore := Ore{
		Status: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_status",
			Options: map[string]OreOption{
				"ready":       {Label: "Not Started"},
				"in_progress": {Label: "Doing"},
			},
		},
		Priority: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_priority",
			Options: map[string]OreOption{
				"p1": {Label: "High"},
			},
		},
		Iteration: OreConfig{
			Enabled: true,
			FieldID: "PVTIF_iteration",
		},
	}

	vars := OreToFlux(ore)

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

func TestOreToFlux_DisabledOreSkipped(t *testing.T) {
	ore := Ore{
		Status: OreConfig{
			Enabled: false,
			FieldID: "PVTSSF_status",
			Options: map[string]OreOption{
				"ready": {Label: "Ready"},
			},
		},
		Priority: OreConfig{
			Enabled: false,
		},
	}

	vars := OreToFlux(ore)

	if _, exists := vars["default_status"]; exists {
		t.Error("expected disabled status ore to not generate default_status variable")
	}
	if _, exists := vars["status_field_id"]; exists {
		t.Error("expected disabled status ore to not generate status_field_id variable")
	}
	if _, exists := vars["default_priority"]; exists {
		t.Error("expected disabled priority ore to not generate default_priority variable")
	}
}

func TestMergeOreFlux_DoesNotOverwriteExisting(t *testing.T) {
	cfg := &Config{
		Templates: TemplateConfig{
			Flux: map[string]string{
				"default_status":   "Custom Status",
				"default_priority": "Custom Priority",
			},
		},
		Ore: Ore{
			Status: OreConfig{
				Enabled: true,
				FieldID: "PVTSSF_status",
				Options: map[string]OreOption{
					"ready": {Label: "Ore Status"},
				},
			},
			Priority: OreConfig{
				Enabled: true,
				FieldID: "PVTSSF_priority",
				Options: map[string]OreOption{
					"p1": {Label: "Ore Priority"},
				},
			},
		},
	}

	MergeOreFlux(cfg)

	// Existing flat flux should NOT be overwritten
	if cfg.Templates.Flux["default_status"] != "Custom Status" {
		t.Errorf("expected existing default_status to be preserved, got %q", cfg.Templates.Flux["default_status"])
	}
	if cfg.Templates.Flux["default_priority"] != "Custom Priority" {
		t.Errorf("expected existing default_priority to be preserved, got %q", cfg.Templates.Flux["default_priority"])
	}

	// But field IDs that weren't in flat flux should be added
	if cfg.Templates.Flux["status_field_id"] != "PVTSSF_status" {
		t.Errorf("expected status_field_id to be added from ore, got %q", cfg.Templates.Flux["status_field_id"])
	}
	if cfg.Templates.Flux["priority_field_id"] != "PVTSSF_priority" {
		t.Errorf("expected priority_field_id to be added from ore, got %q", cfg.Templates.Flux["priority_field_id"])
	}
}

// --- New Config File Location Tests ---

func TestLoadConfig_FromAilloyrc(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	yamlContent := `project:
  name: from-ailloyrc
templates:
  flux:
    source: ailloyrc
`
	if err := os.WriteFile(".ailloyrc", []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write .ailloyrc: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project.Name != "from-ailloyrc" {
		t.Errorf("expected project name 'from-ailloyrc', got '%s'", cfg.Project.Name)
	}
	if cfg.Templates.Flux["source"] != "ailloyrc" {
		t.Errorf("expected flux source=ailloyrc, got '%s'", cfg.Templates.Flux["source"])
	}
}

func TestLoadConfig_FromRootAilloyYaml(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	yamlContent := `project:
  name: from-root-yaml
templates:
  flux:
    source: root-yaml
`
	if err := os.WriteFile("ailloy.yaml", []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write ailloy.yaml: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project.Name != "from-root-yaml" {
		t.Errorf("expected project name 'from-root-yaml', got '%s'", cfg.Project.Name)
	}
}

func TestLoadConfig_AilloyrcTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create both files - .ailloyrc should be loaded first
	if err := os.WriteFile(".ailloyrc", []byte(`project:
  name: from-ailloyrc
`), 0600); err != nil {
		t.Fatalf("failed to write .ailloyrc: %v", err)
	}
	if err := os.WriteFile("ailloy.yaml", []byte(`project:
  name: from-root-yaml
`), 0600); err != nil {
		t.Fatalf("failed to write ailloy.yaml: %v", err)
	}

	cfg, err := LoadConfig(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project.Name != "from-ailloyrc" {
		t.Errorf("expected .ailloyrc to take precedence, got '%s'", cfg.Project.Name)
	}
}

// --- Layered Config Tests ---

func TestLoadLayeredConfig_SetFlagOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create project config
	configDir := filepath.Join(tmpDir, ".ailloy")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	yamlContent := `templates:
  flux:
    org: project-org
    board: Engineering
`
	if err := os.WriteFile(filepath.Join(configDir, "ailloy.yaml"), []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadLayeredConfig([]string{"org=override-org", "new_var=new_value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// --set should override project config
	if cfg.Templates.Flux["org"] != "override-org" {
		t.Errorf("expected org='override-org', got %q", cfg.Templates.Flux["org"])
	}
	// --set should add new vars
	if cfg.Templates.Flux["new_var"] != "new_value" {
		t.Errorf("expected new_var='new_value', got %q", cfg.Templates.Flux["new_var"])
	}
	// Existing vars not overridden should remain
	if cfg.Templates.Flux["board"] != "Engineering" {
		t.Errorf("expected board='Engineering', got %q", cfg.Templates.Flux["board"])
	}
}

func TestLoadLayeredConfig_InvalidSetFormat(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err := LoadLayeredConfig([]string{"invalid-no-equals"})
	if err == nil {
		t.Error("expected error for invalid --set format")
	}
}
