package fluxpicker

import (
	"sort"

	"github.com/sahilm/fuzzy"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// filterKeys returns schema vars filtered (and ranked) against query. Empty
// query returns all vars sorted by name.
func filterKeys(schema []mold.FluxVar, query string) []mold.FluxVar {
	if query == "" {
		out := make([]mold.FluxVar, len(schema))
		copy(out, schema)
		sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	names := make([]string, len(schema))
	for i, v := range schema {
		names[i] = v.Name
	}
	matches := fuzzy.Find(query, names)
	out := make([]mold.FluxVar, 0, len(matches))
	for _, m := range matches {
		out = append(out, schema[m.Index])
	}
	return out
}
