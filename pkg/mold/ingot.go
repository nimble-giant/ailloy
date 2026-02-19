package mold

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/nimble-giant/ailloy/pkg/safepath"
	"gopkg.in/yaml.v3"
)

// Ingot represents an ingot.yaml manifest.
type Ingot struct {
	APIVersion  string   `yaml:"apiVersion"`
	Kind        string   `yaml:"kind"`
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Files       []string `yaml:"files,omitempty"`
	Requires    Requires `yaml:"requires,omitempty"`
}

// LoadIngot reads and parses an ingot.yaml file from the given path.
func LoadIngot(path string) (*Ingot, error) {
	cleanPath, err := safepath.Clean(path)
	if err != nil {
		return nil, fmt.Errorf("reading ingot manifest: %w", err)
	}
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path sanitized by safepath.Clean
	if err != nil {
		return nil, fmt.Errorf("reading ingot manifest: %w", err)
	}
	return ParseIngot(data)
}

// LoadIngotFromFS reads and parses an ingot.yaml file from an fs.FS.
func LoadIngotFromFS(fsys fs.FS, path string) (*Ingot, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("reading ingot manifest from fs: %w", err)
	}
	return ParseIngot(data)
}

// ParseIngot parses raw YAML bytes into an Ingot struct.
func ParseIngot(data []byte) (*Ingot, error) {
	var i Ingot
	if err := yaml.Unmarshal(data, &i); err != nil {
		return nil, fmt.Errorf("parsing ingot manifest: %w", err)
	}
	return &i, nil
}
