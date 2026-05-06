package fluxpicker

import (
	"sort"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// AggregateSchemas merges per-mold flux schemas into a single deduped slice.
// Keys whose Type or Default differs across declaring molds are returned in
// `conflicts`, keyed by var name → sorted list of declaring mold names.
//
// The unified slice contains exactly one entry per unique name. For
// conflicting keys the first observed declaration wins for the unified row;
// callers (the foundry-mode picker) use the conflicts map to expand the row
// into per-mold sub-rows so the user can set each independently.
//
// Output order is alphabetical by var name for deterministic rendering.
func AggregateSchemas(perMold map[string][]mold.FluxVar) ([]mold.FluxVar, map[string][]string) {
	if len(perMold) == 0 {
		return nil, nil
	}

	// Iterate molds in sorted order so "first observed" is deterministic.
	moldNames := make([]string, 0, len(perMold))
	for name := range perMold {
		moldNames = append(moldNames, name)
	}
	sort.Strings(moldNames)

	type seen struct {
		first   mold.FluxVar
		molds   []string // names that DECLARE this key
		differs bool     // true once we've seen a Type/Default mismatch
	}
	byName := map[string]*seen{}

	for _, m := range moldNames {
		for _, v := range perMold[m] {
			s, ok := byName[v.Name]
			if !ok {
				byName[v.Name] = &seen{first: v, molds: []string{m}}
				continue
			}
			s.molds = append(s.molds, m)
			if v.Type != s.first.Type || v.Default != s.first.Default {
				s.differs = true
			}
		}
	}

	unified := make([]mold.FluxVar, 0, len(byName))
	conflicts := map[string][]string{}
	for name, s := range byName {
		unified = append(unified, s.first)
		if s.differs {
			cm := append([]string(nil), s.molds...)
			sort.Strings(cm)
			conflicts[name] = cm
		}
	}
	sort.Slice(unified, func(i, j int) bool { return unified[i].Name < unified[j].Name })
	return unified, conflicts
}
