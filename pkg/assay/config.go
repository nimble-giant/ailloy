package assay

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// Config holds assay configuration loaded from .ailloyrc.yaml.
type Config struct {
	Rules     map[string]RuleConfig `yaml:"rules"`
	Ignore    []string              `yaml:"ignore"`
	Platforms []string              `yaml:"platforms"`
}

// RuleConfig holds per-rule configuration.
type RuleConfig struct {
	Enabled *bool          `yaml:"enabled"`
	Options map[string]any `yaml:"options,omitempty"`
}

// ailloyRC represents the top-level .ailloyrc.yaml structure.
type ailloyRC struct {
	Assay Config `yaml:"assay"`
}

// IsRuleEnabled returns whether a rule is enabled in the config.
// If the rule is not configured, it returns true (enabled by default).
func (c *Config) IsRuleEnabled(ruleName string) bool {
	if c == nil || c.Rules == nil {
		return true
	}
	rc, ok := c.Rules[ruleName]
	if !ok {
		return true
	}
	if rc.Enabled == nil {
		return true
	}
	return *rc.Enabled
}

// RuleOption returns a rule-specific option value, or the fallback if not set.
func (c *Config) RuleOption(ruleName, optionName string, fallback any) any {
	if c == nil || c.Rules == nil {
		return fallback
	}
	rc, ok := c.Rules[ruleName]
	if !ok {
		return fallback
	}
	if rc.Options == nil {
		return fallback
	}
	v, ok := rc.Options[optionName]
	if !ok {
		return fallback
	}
	return v
}

// LoadConfig loads the assay config from .ailloyrc.yaml in the given directory.
// Returns a default (all-enabled) config if the file does not exist.
func LoadConfig(rootDir string) (*Config, error) {
	path := filepath.Join(rootDir, ".ailloyrc.yaml")
	data, err := os.ReadFile(path) //#nosec G304 -- user-controlled project config
	if err != nil {
		if os.IsNotExist(err) {
			// Try .yml extension
			path = filepath.Join(rootDir, ".ailloyrc.yml")
			data, err = os.ReadFile(path) //#nosec G304
			if err != nil {
				if os.IsNotExist(err) {
					return DefaultConfig(), nil
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var rc ailloyRC
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, err
	}

	return &rc.Assay, nil
}

// DefaultConfig returns a config with all rules enabled at default thresholds.
func DefaultConfig() *Config {
	return &Config{}
}

// GenerateStarterConfig returns a starter .ailloyrc.yaml content.
func GenerateStarterConfig() string {
	return `# Ailloy configuration file
# See: https://github.com/nimble-giant/ailloy

assay:
  rules:
    line-count:
      enabled: true
      options:
        max-lines: 150       # warn if instruction file exceeds this many lines
    structure:
      enabled: true
    agents-md-presence:
      enabled: true
    cross-reference:
      enabled: true
    import-validation:
      enabled: true
    empty-file:
      enabled: true
    duplicate-topics:
      enabled: true
    agent-frontmatter:
      enabled: true
    command-frontmatter:
      enabled: true
    settings-schema:
      enabled: true
  ignore: []
    # - "vendor/**"
    # - ".claude/rules/generated-*.md"
  # platforms:             # uncomment to limit to specific platforms
  #   - claude
  #   - cursor
`
}
