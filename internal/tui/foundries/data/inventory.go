package data

import (
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// Scope identifies which lockfile a casted mold lives in.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// InventoryItem represents one casted mold seen in a lockfile.
type InventoryItem struct {
	Scope    Scope
	LockPath string
	Entry    foundry.LockEntry
	Verified bool
	Foundry  string // foundry name, "" if unknown
}

// LoadInventory reads the project lockfile (./ailloy.lock) and the global
// lockfile (~/ailloy.lock) and merges their entries into a single slice.
// Verified status is determined by looking up each entry's source against
// cfg's effective foundries.
func LoadInventory(cfg *index.Config) ([]InventoryItem, error) {
	var out []InventoryItem

	add := func(scope Scope, lockPath string) {
		lock, err := foundry.ReadLockFile(lockPath)
		if err != nil || lock == nil {
			return
		}
		for _, e := range lock.Molds {
			item := InventoryItem{Scope: scope, LockPath: lockPath, Entry: e}
			if f := cfg.FoundryForSource(e.Source); f != nil {
				item.Foundry = f.Name
				item.Verified = index.IsOfficialFoundry(f.URL)
			}
			out = append(out, item)
		}
	}

	add(ScopeProject, foundry.LockFileName)
	if home, err := os.UserHomeDir(); err == nil {
		add(ScopeGlobal, filepath.Join(home, foundry.LockFileName))
	}
	return out, nil
}
