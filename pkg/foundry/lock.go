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
type LockEntry struct {
	Name      string    `yaml:"name"`
	Source    string    `yaml:"source"`
	Version   string    `yaml:"version"`
	Commit    string    `yaml:"commit"`
	Subpath   string    `yaml:"subpath,omitempty"`
	Timestamp time.Time `yaml:"timestamp"`
}

// LockFile is the on-disk lock file format.
type LockFile struct {
	APIVersion string      `yaml:"apiVersion"`
	Molds      []LockEntry `yaml:"molds"`
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

// FindEntry looks up a lock entry by source (cache key).
func (lf *LockFile) FindEntry(source string) *LockEntry {
	if lf == nil {
		return nil
	}
	for i := range lf.Molds {
		if lf.Molds[i].Source == source {
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

// UpsertEntry adds or updates a lock entry by source.
func (lf *LockFile) UpsertEntry(entry LockEntry) {
	for i := range lf.Molds {
		if lf.Molds[i].Source == entry.Source {
			lf.Molds[i] = entry
			return
		}
	}
	lf.Molds = append(lf.Molds, entry)
}
