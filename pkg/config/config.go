package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the Ailloy configuration structure
type Config struct {
	Project   ProjectConfig   `yaml:"project"`
	Templates TemplateConfig  `yaml:"templates"`
	Workflows WorkflowConfig  `yaml:"workflows"`
	User      UserConfig      `yaml:"user"`
	Providers ProvidersConfig `yaml:"providers"`
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
		return &Config{
			Templates: TemplateConfig{
				Variables: make(map[string]string),
			},
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

// ProcessTemplate processes a template string by replacing {{variable}} placeholders
func ProcessTemplate(content string, variables map[string]string) string {
	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
