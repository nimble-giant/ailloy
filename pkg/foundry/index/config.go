package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// Config represents the ~/.ailloy/config.yaml structure.
type Config struct {
	Foundries []FoundryEntry `yaml:"foundries,omitempty"`
}

// FoundryEntry tracks a registered foundry with metadata.
type FoundryEntry struct {
	Name        string    `yaml:"name"`
	URL         string    `yaml:"url"`
	Type        string    `yaml:"type"` // "git" or "url"
	LastUpdated time.Time `yaml:"lastUpdated,omitempty"`
	Status      string    `yaml:"status,omitempty"` // "ok", "error", "pending"
}

// legacyConfig represents the old config format with plain string URLs.
type legacyConfig struct {
	Foundries []string `yaml:"foundries,omitempty"`
}

// ConfigPath returns the path to ~/.ailloy/config.yaml.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ailloy", "config.yaml"), nil
}

// LoadConfig reads and parses ~/.ailloy/config.yaml.
// It auto-migrates the old string-list format to the new FoundryEntry format.
func LoadConfig() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadConfigFrom(configPath)
}

// LoadConfigFrom reads and parses config from a specific path.
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- reading user config file
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Try the new format first.
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err == nil && len(cfg.Foundries) > 0 {
		// Check if we actually got FoundryEntry structs (URL field populated).
		if cfg.Foundries[0].URL != "" {
			return &cfg, nil
		}
	}

	// Fall back to the legacy string-list format.
	var legacy legacyConfig
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	migrated := &Config{}
	for _, url := range legacy.Foundries {
		migrated.Foundries = append(migrated.Foundries, FoundryEntry{
			Name:   nameFromURL(url),
			URL:    url,
			Type:   DetectType(url),
			Status: "pending",
		})
	}
	return migrated, nil
}

// SaveConfig writes the config to ~/.ailloy/config.yaml.
func SaveConfig(cfg *Config) error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveConfigTo(cfg, configPath)
}

// SaveConfigTo writes the config to a specific path.
func SaveConfigTo(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil { // #nosec G306 -- user config file
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// AddFoundry adds a foundry entry, deduplicating by URL.
// Returns true if the entry was added (not a duplicate).
func (c *Config) AddFoundry(entry FoundryEntry) bool {
	for _, existing := range c.Foundries {
		if strings.EqualFold(existing.URL, entry.URL) {
			return false
		}
	}
	c.Foundries = append(c.Foundries, entry)
	return true
}

// RemoveFoundry removes a foundry by name or URL. Returns true if found.
func (c *Config) RemoveFoundry(nameOrURL string) bool {
	for i, entry := range c.Foundries {
		if strings.EqualFold(entry.Name, nameOrURL) || strings.EqualFold(entry.URL, nameOrURL) {
			c.Foundries = append(c.Foundries[:i], c.Foundries[i+1:]...)
			return true
		}
	}
	return false
}

// FindFoundry looks up a foundry by name or URL.
func (c *Config) FindFoundry(nameOrURL string) *FoundryEntry {
	for i, entry := range c.Foundries {
		if strings.EqualFold(entry.Name, nameOrURL) || strings.EqualFold(entry.URL, nameOrURL) {
			return &c.Foundries[i]
		}
	}
	return nil
}

// DetectType determines whether a URL is a git repo or a raw YAML file.
func DetectType(url string) string {
	lower := strings.ToLower(url)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return "url"
	}
	return "git"
}

// nameFromURL derives a short name from a foundry URL.
func nameFromURL(url string) string {
	// Strip common prefixes.
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Use the last path segment as the name.
	parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, ".yaml")
		name = strings.TrimSuffix(name, ".yml")
		if name != "" {
			return name
		}
	}
	return "foundry"
}
