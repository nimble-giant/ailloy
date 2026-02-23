package index

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// Index represents a parsed foundry.yaml index manifest.
type Index struct {
	APIVersion  string      `yaml:"apiVersion"`
	Kind        string      `yaml:"kind"`
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Author      Author      `yaml:"author,omitempty"`
	Molds       []MoldEntry `yaml:"molds"`
}

// Author represents the author of a foundry index.
type Author struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// MoldEntry is a single mold listing within a foundry index.
type MoldEntry struct {
	Name        string   `yaml:"name"`
	Source      string   `yaml:"source"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Version     string   `yaml:"version,omitempty"`
}

// ParseIndex parses raw YAML bytes into an Index.
func ParseIndex(data []byte) (*Index, error) {
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing foundry index: %w", err)
	}
	return &idx, nil
}

// Validate checks that required fields are present and correct.
func (idx *Index) Validate() error {
	var errs []string

	if idx.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if idx.Kind != "foundry-index" {
		errs = append(errs, fmt.Sprintf("kind must be \"foundry-index\", got %q", idx.Kind))
	}
	if idx.Name == "" {
		errs = append(errs, "name is required")
	}

	for i, m := range idx.Molds {
		if m.Name == "" {
			errs = append(errs, fmt.Sprintf("molds[%d].name is required", i))
		}
		if m.Source == "" {
			errs = append(errs, fmt.Sprintf("molds[%d].source is required", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("foundry index validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
