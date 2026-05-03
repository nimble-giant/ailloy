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
	FoundryName string   // root foundry the user registered (catalog grouping)
	FoundryURL  string   // root foundry URL
	OwnerChain  []string // resolution path from root to the foundry that owns the mold; nil/empty for root molds
	Verified    bool
	IndexedAt   time.Time // root foundry's LastUpdated, used for the "Recent" section
}

// IsNested reports whether the entry came from a transitively-resolved
// foundry rather than directly from a registered root foundry.
func (e CatalogEntry) IsNested() bool { return len(e.OwnerChain) > 0 }

// LoadCatalog walks every effective foundry's cached index and flattens it
// into a single slice, resolving any nested foundries declared by each
// registered root via their `foundries:` field. Lookups are cache-only —
// uncached foundries (root or nested) are skipped silently and surface once
// the user runs `foundry update` or hits `r` in the TUI.
func LoadCatalog(cfg *index.Config) ([]CatalogEntry, error) {
	cacheDir, err := index.IndexCacheDir()
	if err != nil {
		return nil, err
	}
	lookup := cacheOnlyLookup(cacheDir)

	seen := map[string]bool{} // dedupe nested molds shared across roots, keyed by source
	var out []CatalogEntry
	for _, entry := range cfg.EffectiveFoundries() {
		r := index.NewResolver(lookup)
		root, molds, err := r.Resolve(entry.URL)
		if err != nil {
			// Root not cached (or invalid) — skip, mirroring the prior behavior.
			continue
		}
		verified := index.IsOfficialFoundry(entry.URL)
		for _, m := range molds {
			if seen[m.Entry.Source] {
				continue
			}
			seen[m.Entry.Source] = true
			var chain []string
			if m.Foundry != root {
				// Parents is rooted at the registered root, plus the owning
				// foundry's own name for the final hop.
				chain = append(append([]string(nil), m.Foundry.Parents...), m.Foundry.Index.Name)
			}
			out = append(out, CatalogEntry{
				Name:        m.Entry.Name,
				Source:      m.Entry.Source,
				Description: m.Entry.Description,
				Tags:        m.Entry.Tags,
				FoundryName: entry.Name,
				FoundryURL:  entry.URL,
				OwnerChain:  chain,
				Verified:    verified && m.Foundry == root,
				IndexedAt:   entry.LastUpdated,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// cacheOnlyLookup returns an IndexLookup that reads from the index cache and
// never falls back to the network. The TUI's existing UX is "skip indexes
// that aren't cached yet" — `r` refreshes the active tab. Honor that for
// nested foundries too: an uncached child surfaces as a Resolver warning we
// drop, not a silent network fetch from the render loop.
func cacheOnlyLookup(cacheDir string) index.IndexLookup {
	return func(source string) (*index.Index, error) {
		url := index.NormalizeFoundryURL(source)
		entry := &index.FoundryEntry{URL: url, Type: index.DetectType(url)}
		return index.LoadCachedIndex(cacheDir, entry)
	}
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
