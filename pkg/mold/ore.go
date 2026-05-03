package mold

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/safepath"
)

// Ore represents an ore.yaml manifest. An ore is a packaged flux-schema
// fragment that consumers install with `ailloy ore add`.
type Ore struct {
	APIVersion  string   `yaml:"apiVersion"`
	Kind        string   `yaml:"kind"`
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Author      Author   `yaml:"author,omitempty"`
	Requires    Requires `yaml:"requires,omitempty"`
}

// LoadOre reads and parses an ore.yaml file from the given path.
func LoadOre(path string) (*Ore, error) {
	cleanPath, err := safepath.Clean(path)
	if err != nil {
		return nil, fmt.Errorf("reading ore manifest: %w", err)
	}
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path sanitized by safepath.Clean
	if err != nil {
		return nil, fmt.Errorf("reading ore manifest: %w", err)
	}
	return ParseOre(data)
}

// LoadOreFromFS reads and parses an ore.yaml file from an fs.FS.
func LoadOreFromFS(fsys fs.FS, path string) (*Ore, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("reading ore manifest from fs: %w", err)
	}
	return ParseOre(data)
}

// ParseOre parses raw YAML bytes into an Ore struct.
func ParseOre(data []byte) (*Ore, error) {
	var o Ore
	if err := yaml.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("parsing ore manifest: %w", err)
	}
	return &o, nil
}
