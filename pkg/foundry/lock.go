package foundry

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// LockFileName is the default lock file name.
const LockFileName = "ailloy.lock"

// LockEntry records the resolved version of a single mold dependency.
// File-level provenance (which files were rendered and their hashes) lives
// on InstalledEntry, not here — that keeps uninstall working when the lock
// is not opted into.
type LockEntry struct {
	Name      string    `yaml:"name"`
	Source    string    `yaml:"source"`
	Version   string    `yaml:"version"`
	Commit    string    `yaml:"commit"`
	Subpath   string    `yaml:"subpath,omitempty"`
	Alias     string    `yaml:"alias,omitempty"` // populated when ore was installed --as <alias>
	Timestamp time.Time `yaml:"timestamp"`
}

// LockFile is the on-disk lock file format.
type LockFile struct {
	APIVersion string      `yaml:"apiVersion"`
	Molds      []LockEntry `yaml:"molds"`
	Ingots     []LockEntry `yaml:"ingots,omitempty"`
	Ores       []LockEntry `yaml:"ores,omitempty"`
}

// ReadLockFile reads and parses the lock file at the given path.
// Returns nil, nil if the file does not exist.
func ReadLockFile(path string) (*LockFile, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- path is constructed from known working directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var lf LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}
	return &lf, nil
}

// WriteLockFile marshals and writes the lock file to the given path.
func WriteLockFile(path string, lock *LockFile) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("marshaling lock file: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil { //#nosec G306
		return fmt.Errorf("writing lock file: %w", err)
	}
	return nil
}

// FindEntry looks up a lock entry by (source, subpath).
//
// Subpath is part of the identity because a single foundry repo can host
// multiple molds at different subpaths.
func (lf *LockFile) FindEntry(source, subpath string) *LockEntry {
	if lf == nil {
		return nil
	}
	for i := range lf.Molds {
		if lf.Molds[i].Source == source && lf.Molds[i].Subpath == subpath {
			return &lf.Molds[i]
		}
	}
	return nil
}

// FindEntryByName looks up a lock entry by name.
func (lf *LockFile) FindEntryByName(name string) *LockEntry {
	if lf == nil {
		return nil
	}
	for i := range lf.Molds {
		if lf.Molds[i].Name == name {
			return &lf.Molds[i]
		}
	}
	return nil
}

// ReferenceFromEntry reconstructs a Reference from a lock entry's source field.
// The source is in the format host/owner/repo and the entry's subpath is preserved.
// The returned reference has type Latest so it resolves to the newest available version.
func ReferenceFromEntry(entry *LockEntry) (*Reference, error) {
	parts := strings.SplitN(entry.Source, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid lock entry source %q: expected host/owner/repo", entry.Source)
	}
	return &Reference{
		Host:    parts[0],
		Owner:   parts[1],
		Repo:    parts[2],
		Subpath: entry.Subpath,
		Type:    Latest,
	}, nil
}

// UpsertEntry adds or updates a lock entry by (source, subpath).
func (lf *LockFile) UpsertEntry(entry LockEntry) {
	for i := range lf.Molds {
		if lf.Molds[i].Source == entry.Source && lf.Molds[i].Subpath == entry.Subpath {
			lf.Molds[i] = entry
			return
		}
	}
	lf.Molds = append(lf.Molds, entry)
}

// UpsertArtifactLock adds or updates a lock entry by (kind, source, subpath, alias).
// Use kind="ingot" or kind="ore".
//
// Subpath is part of the identity because a single foundry can host multiple
// artifacts at different subpaths. Alias is part of the identity because two
// installs of the same source with different aliases must coexist; this keeps
// lock-side keying symmetric with InstalledManifest.UpsertArtifact so manifest
// ↔ lock round-trips do not corrupt entries.
func (lf *LockFile) UpsertArtifactLock(kind string, entry LockEntry) {
	list := lf.artifactLockList(kind)
	for i := range *list {
		if (*list)[i].Source == entry.Source && (*list)[i].Subpath == entry.Subpath && (*list)[i].Alias == entry.Alias {
			(*list)[i] = entry
			return
		}
	}
	*list = append(*list, entry)
}

// AllLockEntry is one entry from LockFile.All(), tagged with its kind.
type AllLockEntry struct {
	Kind  string // "mold", "ingot", or "ore"
	Entry *LockEntry
}

// All yields every lock entry across molds, ingots, and ores, tagged with kind.
func (lf *LockFile) All() []AllLockEntry {
	if lf == nil {
		return nil
	}
	out := make([]AllLockEntry, 0, len(lf.Molds)+len(lf.Ingots)+len(lf.Ores))
	for i := range lf.Molds {
		out = append(out, AllLockEntry{Kind: "mold", Entry: &lf.Molds[i]})
	}
	for i := range lf.Ingots {
		out = append(out, AllLockEntry{Kind: "ingot", Entry: &lf.Ingots[i]})
	}
	for i := range lf.Ores {
		out = append(out, AllLockEntry{Kind: "ore", Entry: &lf.Ores[i]})
	}
	return out
}

func (lf *LockFile) artifactLockList(kind string) *[]LockEntry {
	switch kind {
	case "ingot":
		return &lf.Ingots
	case "ore":
		return &lf.Ores
	default:
		panic("foundry: unknown artifact kind: " + kind)
	}
}
