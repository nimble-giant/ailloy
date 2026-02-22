package foundry

import (
	"fmt"
	"os"
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
