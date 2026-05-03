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
// Files / FileHashes are populated by RecordInstalledFiles and consumed by
// UninstallMold so the uninstall flow knows what to remove and can detect
// post-cast modifications. They are intentionally on the manifest (not the
// lock) so uninstall keeps working when ailloy.lock has not been opted into.
type InstalledEntry struct {
	Name       string            `yaml:"name"`
	Source     string            `yaml:"source"`
	Subpath    string            `yaml:"subpath,omitempty"`
	Version    string            `yaml:"version"`
	Commit     string            `yaml:"commit"`
	CastAt     time.Time         `yaml:"castAt"`
	Files      []string          `yaml:"files,omitempty"`
	FileHashes map[string]string `yaml:"fileHashes,omitempty"`
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

// UpsertEntry adds or updates an entry by (source, subpath).
//
// Subpath is part of the identity because a single foundry repo can host
// multiple molds at different subpaths; matching on Source alone caused the
// second install to overwrite the first.
func (m *InstalledManifest) UpsertEntry(entry InstalledEntry) {
	for i := range m.Molds {
		if m.Molds[i].Source == entry.Source && m.Molds[i].Subpath == entry.Subpath {
			m.Molds[i] = entry
			return
		}
	}
	m.Molds = append(m.Molds, entry)
}

// FindBySource returns the entry matching the given (source, subpath), or nil.
// Pass an empty subpath for entries installed without one.
func (m *InstalledManifest) FindBySource(source, subpath string) *InstalledEntry {
	if m == nil {
		return nil
	}
	for i := range m.Molds {
		if m.Molds[i].Source == source && m.Molds[i].Subpath == subpath {
			return &m.Molds[i]
		}
	}
	return nil
}

// FindAllBySource returns every entry matching the given source, regardless of
// subpath. Useful for CLI flows where the user passes a bare source and we
// need to disambiguate.
func (m *InstalledManifest) FindAllBySource(source string) []*InstalledEntry {
	if m == nil {
		return nil
	}
	var out []*InstalledEntry
	for i := range m.Molds {
		if m.Molds[i].Source == source {
			out = append(out, &m.Molds[i])
		}
	}
	return out
}

// FindAllByName returns every entry matching the given name. Multiple matches
// are possible because two molds at different subpaths in the same foundry
// repo may declare the same name in their mold.yaml; callers must handle the
// ambiguous case rather than silently picking the first match.
func (m *InstalledManifest) FindAllByName(name string) []*InstalledEntry {
	if m == nil {
		return nil
	}
	var out []*InstalledEntry
	for i := range m.Molds {
		if m.Molds[i].Name == name {
			out = append(out, &m.Molds[i])
		}
	}
	return out
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
