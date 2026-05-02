package data

import (
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// Scope identifies which manifest a casted mold lives in.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// InventoryItem represents one casted mold seen in an installed manifest.
type InventoryItem struct {
	Scope        Scope
	ManifestPath string // path to .ailloy/installed.yaml that backs this entry
	Entry        foundry.InstalledEntry
	Verified     bool
	Foundry      string // foundry name, "" if unknown
}

// LoadInventory reads the project installed manifest (./.ailloy/installed.yaml)
// and the global manifest (~/.ailloy/installed.yaml) and merges their entries
// into a single slice. Verified status is determined by looking up each entry's
// source against cfg's effective foundries.
func LoadInventory(cfg *index.Config) ([]InventoryItem, error) {
	var out []InventoryItem

	add := func(scope Scope, manifestPath string) {
		m, err := foundry.ReadInstalledManifest(manifestPath)
		if err != nil || m == nil {
			return
		}
		for _, e := range m.Molds {
			item := InventoryItem{Scope: scope, ManifestPath: manifestPath, Entry: e}
			if f := cfg.FoundryForSource(e.Source); f != nil {
				item.Foundry = f.Name
				item.Verified = index.IsOfficialFoundry(f.URL)
			}
			out = append(out, item)
		}
	}

	add(ScopeProject, foundry.InstalledManifestPath)
	if home, err := os.UserHomeDir(); err == nil {
		add(ScopeGlobal, filepath.Join(home, foundry.InstalledManifestPath))
	}
	return out, nil
}
