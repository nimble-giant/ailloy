package foundry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
)

// InstalledManifestPath is the default project manifest path.
const InstalledManifestPath = ".ailloy/installed.yaml"

// InstalledEntry records a mold that was cast into the project.
// Provenance only — does not hash rendered files (those are user-customizable).
type InstalledEntry struct {
	Name    string    `yaml:"name"`
	Source  string    `yaml:"source"`
	Subpath string    `yaml:"subpath,omitempty"`
	Version string    `yaml:"version"`
	Commit  string    `yaml:"commit"`
	CastAt  time.Time `yaml:"castAt"`
}

// InstalledManifest is the on-disk manifest of cast molds.
type InstalledManifest struct {
	APIVersion string           `yaml:"apiVersion"`
	Molds      []InstalledEntry `yaml:"molds"`
}

// ReadInstalledManifest reads and parses the manifest at the given path.
// Returns (nil, nil) if the file does not exist.
func ReadInstalledManifest(path string) (*InstalledManifest, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- path constructed by callers
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading installed manifest: %w", err)
	}
	var m InstalledManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing installed manifest: %w", err)
	}
	return &m, nil
}

// UpsertEntry adds or updates an entry by source.
func (m *InstalledManifest) UpsertEntry(entry InstalledEntry) {
	for i := range m.Molds {
		if m.Molds[i].Source == entry.Source {
			m.Molds[i] = entry
			return
		}
	}
	m.Molds = append(m.Molds, entry)
}

// FindBySource returns the entry matching the given source, or nil.
func (m *InstalledManifest) FindBySource(source string) *InstalledEntry {
	if m == nil {
		return nil
	}
	for i := range m.Molds {
		if m.Molds[i].Source == source {
			return &m.Molds[i]
		}
	}
	return nil
}

// FindByName returns the entry matching the given name, or nil.
func (m *InstalledManifest) FindByName(name string) *InstalledEntry {
	if m == nil {
		return nil
	}
	for i := range m.Molds {
		if m.Molds[i].Name == name {
			return &m.Molds[i]
		}
	}
	return nil
}

// WriteInstalledManifest marshals and writes the manifest, creating parent dirs.
func WriteInstalledManifest(path string, m *InstalledManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil { //#nosec G301
		return fmt.Errorf("creating manifest dir: %w", err)
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling installed manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil { //#nosec G306
		return fmt.Errorf("writing installed manifest: %w", err)
	}
	return nil
}
