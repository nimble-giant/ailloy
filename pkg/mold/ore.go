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
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	// Namespace, if set, is the publisher's declared canonical flux namespace
	// for this ore. When empty, the namespace falls back to Name. Consumer
	// overrides (mold.yaml `as:` and `ailloy ore add --as <alias>`) still take
	// precedence — the namespace ranking, highest wins, is:
	//
	//   1. mold.yaml dependency entry's `as:` (per-cast consumer override)
	//   2. `--as <alias>` recorded in installed.yaml as Alias (per-install)
	//   3. `namespace:` in ore.yaml (publisher-declared)
	//   4. `name:` in ore.yaml (fallback)
	//
	// Layers (1)–(2) are reflected in the on-disk install-dir name; the
	// resolver layers (3)–(4) on top via EffectiveNamespace.
	Namespace   string   `yaml:"namespace,omitempty"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Author      Author   `yaml:"author,omitempty"`
	Requires    Requires `yaml:"requires,omitempty"`
}

// EffectiveNamespace returns the publisher-declared namespace: Namespace if
// non-empty, else Name. Consumer-side overrides (alias) live outside the
// manifest and are layered on top of this value by callers; see Ore.Namespace
// for the full precedence chain.
func (o *Ore) EffectiveNamespace() string {
	if o == nil {
		return ""
	}
	if o.Namespace != "" {
		return o.Namespace
	}
	return o.Name
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
