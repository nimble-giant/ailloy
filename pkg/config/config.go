package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ModelOption represents a single option within a model (e.g., "P0" in Priority)
type ModelOption struct {
	Label string `yaml:"label"`        // User-facing label (or their GitHub field option name)
	ID    string `yaml:"id,omitempty"` // Resolved GraphQL option ID (populated by discovery layer)
}

// ModelConfig represents a single semantic model (Status, Priority, or Iteration)
type ModelConfig struct {
	Enabled      bool                   `yaml:"enabled"`
	FieldMapping string                 `yaml:"field_mapping,omitempty"` // User's GitHub Project field name
	FieldID      string                 `yaml:"field_id,omitempty"`      // Resolved GraphQL field ID
	Options      map[string]ModelOption `yaml:"options,omitempty"`       // concept name → option
}

// Models holds all semantic model configurations
type Models struct {
	Status    ModelConfig `yaml:"status"`
	Priority  ModelConfig `yaml:"priority"`
	Iteration ModelConfig `yaml:"iteration"`
}

// Config represents the Ailloy configuration structure
type Config struct {
	Project   ProjectConfig   `yaml:"project"`
	Templates TemplateConfig  `yaml:"templates"`
	Workflows WorkflowConfig  `yaml:"workflows"`
	User      UserConfig      `yaml:"user"`
	Providers ProvidersConfig `yaml:"providers"`
	Models    Models          `yaml:"models"`
}

// ProjectConfig holds project-specific settings
type ProjectConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Providers   []string `yaml:"ai_providers"`
	TemplDirs   []string `yaml:"template_directories"`
}

// TemplateConfig holds template-related settings
type TemplateConfig struct {
	DefaultProvider string            `yaml:"default_provider"`
	AutoUpdate      bool              `yaml:"auto_update"`
	Repositories    []string          `yaml:"repositories"`
	Variables       map[string]string `yaml:"variables"`
}

// WorkflowConfig holds workflow definitions
type WorkflowConfig struct {
	IssueCreation WorkflowStep `yaml:"issue_creation"`
}

// WorkflowStep defines a single workflow step
type WorkflowStep struct {
	Template string `yaml:"template"`
	Provider string `yaml:"provider"`
}

// UserConfig holds user preferences
type UserConfig struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

// ProvidersConfig holds AI provider configurations
type ProvidersConfig struct {
	Claude ClaudeConfig `yaml:"claude"`
	GPT    GPTConfig    `yaml:"gpt"`
}

// ClaudeConfig holds Claude-specific configuration
type ClaudeConfig struct {
	Enabled   bool   `yaml:"enabled"`
	APIKeyEnv string `yaml:"api_key_env"`
}

// GPTConfig holds GPT/OpenAI-specific configuration
type GPTConfig struct {
	Enabled   bool   `yaml:"enabled"`
	APIKeyEnv string `yaml:"api_key_env"`
}

// GetConfigDir returns the appropriate config directory
func GetConfigDir(global bool) (string, error) {
	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".ailloy"), nil
	}

	// Local project config
	return ".ailloy", nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath(global bool) (string, error) {
	configDir, err := GetConfigDir(global)
	if err != nil {
		return "", err
	}

	configFile := "ailloy.yaml"
	if global {
		configFile = "ailloy.yaml"
	}

	return filepath.Join(configDir, configFile), nil
}

// LoadConfig loads configuration from the specified file
func LoadConfig(global bool) (*Config, error) {
	configPath, err := GetConfigPath(global)
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaults := DefaultModels()
		return &Config{
			Templates: TemplateConfig{
				Variables: make(map[string]string),
			},
			Models: defaults,
		}, nil
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- CLI tool reads user config files
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize Variables map if nil
	if config.Templates.Variables == nil {
		config.Templates.Variables = make(map[string]string)
	}

	// Initialize models with defaults for any missing options
	initModels(&config.Models)

	return &config, nil
}

// SaveConfig saves configuration to the specified file
func SaveConfig(config *Config, global bool) error {
	configPath, err := GetConfigPath(global)
	if err != nil {
		return err
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil { // #nosec G301 -- Config directory needs group read access
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil { // Config file - restricted permissions
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultModels returns the default model definitions with ailloy-native concepts
func DefaultModels() Models {
	return Models{
		Status: ModelConfig{
			Enabled: false,
			Options: map[string]ModelOption{
				"ready":       {Label: "Ready"},
				"in_progress": {Label: "In Progress"},
				"in_review":   {Label: "In Review"},
				"done":        {Label: "Done"},
			},
		},
		Priority: ModelConfig{
			Enabled: false,
			Options: map[string]ModelOption{
				"p0": {Label: "P0"},
				"p1": {Label: "P1"},
				"p2": {Label: "P2"},
				"p3": {Label: "P3"},
			},
		},
		Iteration: ModelConfig{
			Enabled: false,
		},
	}
}

// initModels fills in missing default options without overwriting user-customized values
func initModels(models *Models) {
	defaults := DefaultModels()

	initModelConfig(&models.Status, &defaults.Status)
	initModelConfig(&models.Priority, &defaults.Priority)
	// Iteration has no predefined options, just ensure the struct exists
}

// initModelConfig fills in missing options for a single model
func initModelConfig(model *ModelConfig, defaults *ModelConfig) {
	if model.Options == nil && defaults.Options != nil {
		model.Options = make(map[string]ModelOption, len(defaults.Options))
		for key, opt := range defaults.Options {
			model.Options[key] = opt
		}
		return
	}

	// Fill in any missing default options
	if defaults.Options != nil {
		if model.Options == nil {
			model.Options = make(map[string]ModelOption)
		}
		for key, opt := range defaults.Options {
			if _, exists := model.Options[key]; !exists {
				model.Options[key] = opt
			}
		}
	}
}

// ModelsToVariables converts model state into the flat variable format that existing templates expect
func ModelsToVariables(models Models) map[string]string {
	vars := make(map[string]string)

	if models.Status.Enabled {
		if opt, ok := models.Status.Options["ready"]; ok {
			vars["default_status"] = opt.Label
		}
		if models.Status.FieldID != "" {
			vars["status_field_id"] = models.Status.FieldID
		}
	}

	if models.Priority.Enabled {
		if opt, ok := models.Priority.Options["p1"]; ok {
			vars["default_priority"] = opt.Label
		}
		if models.Priority.FieldID != "" {
			vars["priority_field_id"] = models.Priority.FieldID
		}
	}

	if models.Iteration.Enabled {
		if models.Iteration.FieldID != "" {
			vars["iteration_field_id"] = models.Iteration.FieldID
		}
	}

	return vars
}

// MergeModelVariables merges model-derived variables into the config's template variables.
// Existing flat variables take precedence over model-derived values.
func MergeModelVariables(cfg *Config) {
	modelVars := ModelsToVariables(cfg.Models)
	if cfg.Templates.Variables == nil {
		cfg.Templates.Variables = make(map[string]string)
	}
	for key, value := range modelVars {
		if _, exists := cfg.Templates.Variables[key]; !exists {
			cfg.Templates.Variables[key] = value
		}
	}
}
