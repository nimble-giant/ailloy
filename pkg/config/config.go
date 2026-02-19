package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// OreOption represents a single option within an ore config (e.g., "P0" in Priority)
type OreOption struct {
	Label string `yaml:"label"`        // User-facing label (or their GitHub field option name)
	ID    string `yaml:"id,omitempty"` // Resolved GraphQL option ID (populated by discovery layer)
}

// OreConfig represents a single semantic ore model (Status, Priority, or Iteration)
type OreConfig struct {
	Enabled      bool                 `yaml:"enabled"`
	FieldMapping string               `yaml:"field_mapping,omitempty"` // User's GitHub Project field name
	FieldID      string               `yaml:"field_id,omitempty"`      // Resolved GraphQL field ID
	Options      map[string]OreOption `yaml:"options,omitempty"`       // concept name → option
}

// Ore holds all semantic ore configurations
type Ore struct {
	Status    OreConfig `yaml:"status"`
	Priority  OreConfig `yaml:"priority"`
	Iteration OreConfig `yaml:"iteration"`
}

// Config represents the Ailloy configuration structure
type Config struct {
	Project   ProjectConfig   `yaml:"project"`
	Templates TemplateConfig  `yaml:"templates"`
	Workflows WorkflowConfig  `yaml:"workflows"`
	User      UserConfig      `yaml:"user"`
	Providers ProvidersConfig `yaml:"providers"`
	Ore       Ore             `yaml:"ore"`
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
	Flux            map[string]string `yaml:"flux"`
	Ignore          []string          `yaml:"ignore"` // Patterns to exclude from template list
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

// projectConfigPaths returns candidate config file paths for project-level config, in priority order.
func projectConfigPaths() []string {
	return []string{
		".ailloyrc",
		"ailloy.yaml",
		filepath.Join(".ailloy", "ailloy.yaml"),
	}
}

// globalConfigPaths returns candidate config file paths for global-level config, in priority order.
func globalConfigPaths() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(homeDir, ".ailloy", ".ailloyrc"),
		filepath.Join(homeDir, ".ailloy", "ailloy.yaml"),
	}, nil
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

// findConfigFile searches candidate paths and returns the first that exists.
func findConfigFile(global bool) (string, error) {
	if global {
		paths, err := globalConfigPaths()
		if err != nil {
			return "", err
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
		return "", nil // none found
	}

	for _, p := range projectConfigPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil // none found
}

// LoadConfig loads configuration from the specified scope (project or global).
// It searches multiple candidate file locations and returns the first found.
func LoadConfig(global bool) (*Config, error) {
	configPath, err := findConfigFile(global)
	if err != nil {
		return nil, err
	}

	// If no config file found, return default config
	if configPath == "" {
		defaults := DefaultOre()
		return &Config{
			Templates: TemplateConfig{
				Flux: make(map[string]string),
			},
			Ore: defaults,
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

	// Initialize Flux map if nil
	if config.Templates.Flux == nil {
		config.Templates.Flux = make(map[string]string)
	}

	// Initialize ore with defaults for any missing options
	initOre(&config.Ore)

	return &config, nil
}

// LoadLayeredConfig loads configuration with full layering:
// mold defaults < project config < global config < set overrides.
func LoadLayeredConfig(setFlags []string) (*Config, error) {
	// Layer 1: Start with defaults (includes mold flux defaults)
	cfg := &Config{
		Templates: TemplateConfig{
			Flux: make(map[string]string),
		},
		Ore: DefaultOre(),
	}

	// Layer 2: Load project config
	projectCfg, err := LoadConfig(false)
	if err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}
	mergeConfig(cfg, projectCfg)

	// Layer 3: Load global config
	globalCfg, err := LoadConfig(true)
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}
	mergeConfig(cfg, globalCfg)

	// Layer 4: Apply --set flag overrides
	for _, flag := range setFlags {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set format: %q (expected key=value)", flag)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("--set key cannot be empty")
		}
		cfg.Templates.Flux[key] = value
	}

	return cfg, nil
}

// mergeConfig merges source config values into dest.
// Non-zero source values override dest values; flux maps are merged with source taking precedence.
func mergeConfig(dest, src *Config) {
	// Merge flux: source values override dest
	for k, v := range src.Templates.Flux {
		dest.Templates.Flux[k] = v
	}

	// Merge ore: only override if source has enabled models
	if src.Ore.Status.Enabled {
		dest.Ore.Status = src.Ore.Status
	}
	if src.Ore.Priority.Enabled {
		dest.Ore.Priority = src.Ore.Priority
	}
	if src.Ore.Iteration.Enabled {
		dest.Ore.Iteration = src.Ore.Iteration
	}

	// Merge project info if set
	if src.Project.Name != "" {
		dest.Project = src.Project
	}
	if src.User.Name != "" || src.User.Email != "" {
		dest.User = src.User
	}
	if src.Providers.Claude.Enabled || src.Providers.GPT.Enabled {
		dest.Providers = src.Providers
	}
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

// DefaultOre returns the default ore definitions with ailloy-native concepts
func DefaultOre() Ore {
	return Ore{
		Status: OreConfig{
			Enabled: false,
			Options: map[string]OreOption{
				"ready":       {Label: "Ready"},
				"in_progress": {Label: "In Progress"},
				"in_review":   {Label: "In Review"},
				"done":        {Label: "Done"},
			},
		},
		Priority: OreConfig{
			Enabled: false,
			Options: map[string]OreOption{
				"p0": {Label: "P0"},
				"p1": {Label: "P1"},
				"p2": {Label: "P2"},
				"p3": {Label: "P3"},
			},
		},
		Iteration: OreConfig{
			Enabled: false,
		},
	}
}

// initOre fills in missing default options without overwriting user-customized values
func initOre(ore *Ore) {
	defaults := DefaultOre()

	initOreConfig(&ore.Status, &defaults.Status)
	initOreConfig(&ore.Priority, &defaults.Priority)
	// Iteration has no predefined options, just ensure the struct exists
}

// initOreConfig fills in missing options for a single ore config
func initOreConfig(ore *OreConfig, defaults *OreConfig) {
	if ore.Options == nil && defaults.Options != nil {
		ore.Options = make(map[string]OreOption, len(defaults.Options))
		for key, opt := range defaults.Options {
			ore.Options[key] = opt
		}
		return
	}

	// Fill in any missing default options
	if defaults.Options != nil {
		if ore.Options == nil {
			ore.Options = make(map[string]OreOption)
		}
		for key, opt := range defaults.Options {
			if _, exists := ore.Options[key]; !exists {
				ore.Options[key] = opt
			}
		}
	}
}

// OreToFlux converts ore state into the flat flux variable format that existing templates expect
func OreToFlux(ore Ore) map[string]string {
	vars := make(map[string]string)

	if ore.Status.Enabled {
		if opt, ok := ore.Status.Options["ready"]; ok {
			vars["default_status"] = opt.Label
		}
		if ore.Status.FieldID != "" {
			vars["status_field_id"] = ore.Status.FieldID
		}
	}

	if ore.Priority.Enabled {
		if opt, ok := ore.Priority.Options["p1"]; ok {
			vars["default_priority"] = opt.Label
		}
		if ore.Priority.FieldID != "" {
			vars["priority_field_id"] = ore.Priority.FieldID
		}
	}

	if ore.Iteration.Enabled {
		if ore.Iteration.FieldID != "" {
			vars["iteration_field_id"] = ore.Iteration.FieldID
		}
	}

	return vars
}

// MergeOreFlux merges ore-derived variables into the config's template flux.
// Existing flat flux values take precedence over ore-derived values.
func MergeOreFlux(cfg *Config) {
	oreVars := OreToFlux(cfg.Ore)
	if cfg.Templates.Flux == nil {
		cfg.Templates.Flux = make(map[string]string)
	}
	for key, value := range oreVars {
		if _, exists := cfg.Templates.Flux[key]; !exists {
			cfg.Templates.Flux[key] = value
		}
	}
}
