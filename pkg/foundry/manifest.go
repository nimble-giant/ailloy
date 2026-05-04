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

// ArtifactEntry records an installed ingot or ore. Mirrors InstalledEntry
// minus the file-provenance fields (which are mold-specific) and adds
// Dependents for reference-counted cascade uninstall.
type ArtifactEntry struct {
	Name        string    `yaml:"name"`
	Source      string    `yaml:"source"`
	Subpath     string    `yaml:"subpath,omitempty"`
	Version     string    `yaml:"version"`
	Commit      string    `yaml:"commit"`
	InstalledAt time.Time `yaml:"installedAt"`
	Alias       string    `yaml:"alias,omitempty"`      // populated when ore installed --as <alias>
	Dependents  []string  `yaml:"dependents,omitempty"` // mold source@subpath strings; "user" sentinel for direct installs
}

// InstalledManifest is the on-disk manifest of cast molds and installed
// artifacts (ingots, ores).
type InstalledManifest struct {
	APIVersion string           `yaml:"apiVersion"`
	Molds      []InstalledEntry `yaml:"molds"`
	Ingots     []ArtifactEntry  `yaml:"ingots,omitempty"`
	Ores       []ArtifactEntry  `yaml:"ores,omitempty"`
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

// UpsertArtifact adds or updates an entry by (kind, source, subpath, alias).
// Use kind="ingot" or kind="ore". Existing dependents are merged with the
// incoming entry's dependents (set union), preserving order.
//
// Subpath is part of the identity because a single foundry can host multiple
// artifacts at different subpaths. Alias is part of the identity because two
// installs of the same source with different aliases must coexist.
func (m *InstalledManifest) UpsertArtifact(kind string, entry ArtifactEntry) {
	list := m.artifactList(kind)
	for i := range *list {
		if (*list)[i].Source == entry.Source && (*list)[i].Subpath == entry.Subpath && (*list)[i].Alias == entry.Alias {
			merged := mergeDependents((*list)[i].Dependents, entry.Dependents)
			entry.Dependents = merged
			(*list)[i] = entry
			return
		}
	}
	*list = append(*list, entry)
}

// RemoveDependent strips moldKey from every artifact's Dependents list.
// Returns the entries whose Dependents became empty (caller GCs).
// Removes those orphan entries from the manifest as a side effect.
func (m *InstalledManifest) RemoveDependent(moldKey string) []ArtifactEntry {
	var orphans []ArtifactEntry
	for _, kind := range []string{"ingot", "ore"} {
		list := m.artifactList(kind)
		kept := (*list)[:0]
		for _, e := range *list {
			e.Dependents = stripString(e.Dependents, moldKey)
			if len(e.Dependents) == 0 {
				orphans = append(orphans, e)
				continue
			}
			kept = append(kept, e)
		}
		*list = kept
	}
	return orphans
}

// FindArtifact looks up an artifact entry by (kind, name). Name is the
// install-dir name (post-aliasing) — i.e. the one used in
// .ailloy/<kind>s/<name>/.
func (m *InstalledManifest) FindArtifact(kind, name string) *ArtifactEntry {
	if m == nil {
		return nil
	}
	list := m.artifactList(kind)
	for i := range *list {
		effective := (*list)[i].Name
		if (*list)[i].Alias != "" {
			effective = (*list)[i].Alias
		}
		if effective == name {
			return &(*list)[i]
		}
	}
	return nil
}

// AllEntry is one entry from InstalledManifest.All(), tagged with its kind.
type AllEntry struct {
	Kind     string // "mold", "ingot", or "ore"
	Mold     *InstalledEntry
	Artifact *ArtifactEntry
}

// All yields every InstalledEntry plus every ArtifactEntry, tagged with kind.
// Used by quench, uninstall, and TUI walks. Returns kind ∈ {"mold","ingot","ore"}.
func (m *InstalledManifest) All() []AllEntry {
	if m == nil {
		return nil
	}
	out := make([]AllEntry, 0, len(m.Molds)+len(m.Ingots)+len(m.Ores))
	for i := range m.Molds {
		out = append(out, AllEntry{Kind: "mold", Mold: &m.Molds[i]})
	}
	for i := range m.Ingots {
		out = append(out, AllEntry{Kind: "ingot", Artifact: &m.Ingots[i]})
	}
	for i := range m.Ores {
		out = append(out, AllEntry{Kind: "ore", Artifact: &m.Ores[i]})
	}
	return out
}

func (m *InstalledManifest) artifactList(kind string) *[]ArtifactEntry {
	switch kind {
	case "ingot":
		return &m.Ingots
	case "ore":
		return &m.Ores
	default:
		panic("foundry: unknown artifact kind: " + kind)
	}
}

func mergeDependents(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(append([]string{}, a...), b...) {
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func stripString(s []string, target string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for _, v := range s {
		if v != target {
			out = append(out, v)
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
