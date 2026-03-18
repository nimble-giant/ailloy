package assay

import (
	"os"
	"path/filepath"
	"sort"

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

// AddAllowedFrontmatterFields merges fields into command-frontmatter.options.extra-allowed-fields
// in the project's .ailloyrc.yaml (creating the file with starter defaults if absent).
// Returns the (possibly unchanged) set of all allowed fields after the update and a flag
// indicating whether the file was modified.
func AddAllowedFrontmatterFields(rootDir string, fields []string) (added []string, err error) {
	cfgPath := filepath.Join(rootDir, ".ailloyrc.yaml")

	// Load or bootstrap config
	cfg, loadErr := LoadConfig(rootDir)
	if loadErr != nil {
		return nil, loadErr
	}
	if cfg.Rules == nil {
		cfg.Rules = make(map[string]RuleConfig)
	}

	rc := cfg.Rules["command-frontmatter"]
	if rc.Options == nil {
		rc.Options = make(map[string]any)
	}

	// Merge existing + new fields, deduplicating
	existing := make(map[string]bool)
	if cur, ok := rc.Options["extra-allowed-fields"].([]any); ok {
		for _, v := range cur {
			if s, ok := v.(string); ok {
				existing[s] = true
			}
		}
	}
	for _, f := range fields {
		if !existing[f] {
			existing[f] = true
			added = append(added, f)
		}
	}
	if len(added) == 0 {
		return nil, nil // nothing new
	}

	// Rebuild sorted slice
	merged := make([]string, 0, len(existing))
	for k := range existing {
		merged = append(merged, k)
	}
	sort.Strings(merged)

	// Convert to []any for YAML marshal
	mergedAny := make([]any, len(merged))
	for i, v := range merged {
		mergedAny[i] = v
	}
	rc.Options["extra-allowed-fields"] = mergedAny
	cfg.Rules["command-frontmatter"] = rc

	data, err := yaml.Marshal(ailloyRC{Assay: *cfg})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(cfgPath, data, 0644); err != nil { //#nosec G306
		return nil, err
	}
	return added, nil
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
      # options:
      #   extra-allowed-fields: [topic, source, created, updated, tags]  # suppress warnings for custom metadata fields
    settings-schema:
      enabled: true
    plugin-manifest:
      enabled: true
    plugin-hooks:
      enabled: true
  ignore: []
    # - "vendor/**"
    # - ".claude/rules/generated-*.md"
  # platforms:             # uncomment to limit to specific platforms
  #   - claude
  #   - cursor
`
}
