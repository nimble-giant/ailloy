package mold

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/nimble-giant/ailloy/pkg/safepath"
	"gopkg.in/yaml.v3"
)

// Author represents the author of a mold or ingot.
type Author struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// Requires specifies version constraints for ailloy.
type Requires struct {
	Ailloy string `yaml:"ailloy"`
}

// FluxVar declares a template variable with type information.
type FluxVar struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default,omitempty"`
}

// Dependency declares a dependency on an ingot.
type Dependency struct {
	Ingot   string `yaml:"ingot"`
	Version string `yaml:"version"`
}

// Mold represents a mold.yaml manifest.
type Mold struct {
	APIVersion   string       `yaml:"apiVersion"`
	Kind         string       `yaml:"kind"`
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Description  string       `yaml:"description,omitempty"`
	Author       Author       `yaml:"author,omitempty"`
	Requires     Requires     `yaml:"requires,omitempty"`
	Flux         []FluxVar    `yaml:"flux,omitempty"`
	Commands     []string     `yaml:"commands,omitempty"`
	Skills       []string     `yaml:"skills,omitempty"`
	Workflows    []string     `yaml:"workflows,omitempty"`
	Dependencies []Dependency `yaml:"dependencies,omitempty"`
}

// LoadMold reads and parses a mold.yaml file from the given path.
func LoadMold(path string) (*Mold, error) {
	cleanPath, err := safepath.Clean(path)
	if err != nil {
		return nil, fmt.Errorf("reading mold manifest: %w", err)
	}
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path sanitized by safepath.Clean
	if err != nil {
		return nil, fmt.Errorf("reading mold manifest: %w", err)
	}
	return ParseMold(data)
}

// LoadMoldFromFS reads and parses a mold.yaml file from an fs.FS.
func LoadMoldFromFS(fsys fs.FS, path string) (*Mold, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("reading mold manifest from fs: %w", err)
	}
	return ParseMold(data)
}

// ParseMold parses raw YAML bytes into a Mold struct.
func ParseMold(data []byte) (*Mold, error) {
	var m Mold
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing mold manifest: %w", err)
	}
	return &m, nil
}
