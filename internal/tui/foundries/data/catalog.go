package data

import (
	"sort"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// CatalogEntry is one mold visible to the Discover tab.
type CatalogEntry struct {
	Name        string
	Source      string
	Description string
	Tags        []string
	FoundryName string
	FoundryURL  string
	Verified    bool
	IndexedAt   time.Time // foundry's LastUpdated, used for the "Recent" section
}

// LoadCatalog walks every effective foundry's cached index and flattens it
// into a single slice. Foundries without a cached index are skipped silently
// (they'll appear once the user runs `foundry update` or hits `r` in the TUI).
func LoadCatalog(cfg *index.Config) ([]CatalogEntry, error) {
	cacheDir, err := index.IndexCacheDir()
	if err != nil {
		return nil, err
	}
	var out []CatalogEntry
	for _, entry := range cfg.EffectiveFoundries() {
		idx, err := index.LoadCachedIndex(cacheDir, &entry)
		if err != nil {
			continue
		}
		verified := index.IsOfficialFoundry(entry.URL)
		for _, m := range idx.Molds {
			out = append(out, CatalogEntry{
				Name:        m.Name,
				Source:      m.Source,
				Description: m.Description,
				Tags:        m.Tags,
				FoundryName: entry.Name,
				FoundryURL:  entry.URL,
				Verified:    verified,
				IndexedAt:   entry.LastUpdated,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Recent returns up to n catalog entries indexed within the lookback window,
// sorted newest first. Used by the Discover tab to show a "what's new" list.
func Recent(catalog []CatalogEntry, lookback time.Duration, n int) []CatalogEntry {
	cutoff := time.Now().Add(-lookback)
	var recent []CatalogEntry
	for _, e := range catalog {
		if e.IndexedAt.After(cutoff) {
			recent = append(recent, e)
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].IndexedAt.After(recent[j].IndexedAt) })
	if len(recent) > n {
		recent = recent[:n]
	}
	return recent
}
