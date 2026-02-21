package mold

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/safepath"
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

// DiscoverSpec declares how to dynamically discover options for a flux variable.
// Discovery commands are executed lazily during `ailloy anneal` when the user
// reaches the relevant wizard section.
type DiscoverSpec struct {
	Command  string         `yaml:"command"`             // Shell command to run
	Parse    string         `yaml:"parse,omitempty"`     // Go template to extract label|value pairs from JSON output
	Prompt   string         `yaml:"prompt,omitempty"`    // "select" for dropdown, "input" for freeform (default)
	AlsoSets map[string]int `yaml:"also_sets,omitempty"` // Maps flux var names to extra pipe-segment indices (0=label, 1=value, 2+)
}

// SelectOption declares a static option for a select-type flux variable.
type SelectOption struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

// FluxVar declares a template variable with type information.
type FluxVar struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Type        string         `yaml:"type"`
	Required    bool           `yaml:"required"`
	Default     string         `yaml:"default,omitempty"`
	Options     []SelectOption `yaml:"options,omitempty"`  // Static options for select type
	Discover    *DiscoverSpec  `yaml:"discover,omitempty"` // Dynamic discovery specification
}

// Dependency declares a dependency on an ingot.
type Dependency struct {
	Ingot   string `yaml:"ingot"`
	Version string `yaml:"version"`
}

// OutputTarget represents a single output directory or file mapping.
// It supports two YAML forms:
//   - Simple string: "dest/path" (process defaults to true)
//   - Expanded map: {dest: "dest/path", process: false}
type OutputTarget struct {
	Dest    string `yaml:"dest"`
	Process *bool  `yaml:"process,omitempty"` // nil = true (default)
}

// ShouldProcess returns whether files under this target should be template-processed.
func (o OutputTarget) ShouldProcess() bool {
	return o.Process == nil || *o.Process
}

// ResolvedFile represents a single file resolved from the output mapping.
type ResolvedFile struct {
	SrcPath  string // path within the mold fs (e.g., "commands/hello.md")
	DestPath string // output path (e.g., ".claude/commands/hello.md")
	Process  bool   // whether to apply template processing
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
