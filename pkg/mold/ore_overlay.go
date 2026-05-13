package mold

import (
	"fmt"
	"sort"
)

// OreOutputOverlay is one ore's `output:` block extracted from its flux.yaml.
// Entries match the same YAML shape consumers use in their own `output:` map
// — string, map, or list values per source-path key.
type OreOutputOverlay struct {
	Source  string         // e.g. "ore:agent_targets"
	Entries map[string]any // source-path → destination(s)
}

// MergeFluxOutput combines a consumer mold's `output:` map (base, possibly
// nil) with N ore overlays. Rules mirror MergeFluxSchema:
//
//   - If a source-path key is in base, base wins; the overlay copy is
//     recorded in report.ShadowedOutput.
//   - If a key is in two overlays (and not in base), return a fatal error
//     naming both sources.
//   - Net-new overlay entries are appended to the merged map; deterministic
//     iteration is not preserved (Go map), but overlay sort by Source makes
//     conflict messages stable.
//
// The merged map is a freshly allocated copy — neither base nor overlay
// inputs are mutated. The report.OutputSources map records the origin of
// every key in the result.
func MergeFluxOutput(base map[string]any, overlays []OreOutputOverlay) (map[string]any, OreLoadReport, error) {
	report := OreLoadReport{OutputSources: map[string]string{}}

	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
		report.OutputSources[k] = ""
	}

	sortedOverlays := make([]OreOutputOverlay, len(overlays))
	copy(sortedOverlays, overlays)
	sort.Slice(sortedOverlays, func(i, j int) bool { return sortedOverlays[i].Source < sortedOverlays[j].Source })

	overlayOwner := make(map[string]string)
	for _, ov := range sortedOverlays {
		for k, v := range ov.Entries {
			if _, inBase := base[k]; inBase {
				report.ShadowedOutput = append(report.ShadowedOutput, ShadowedOutputEntry{Key: k, Source: ov.Source})
				continue
			}
			if prev, dup := overlayOwner[k]; dup {
				return nil, report, fmt.Errorf(
					"ore output key conflict: %q defined by both %s and %s; resolve by overriding in the consumer mold's output or use 'as:' on the dependency",
					k, prev, ov.Source,
				)
			}
			overlayOwner[k] = ov.Source
			out[k] = v
			report.OutputSources[k] = ov.Source
		}
	}
	return out, report, nil
}

// ExtractOreOutput pulls the top-level `output:` key out of an ore's
// flux.yaml defaults map and returns it, leaving the remaining defaults
// (the schema-mergeable ones) in the input map's place. Returns nil if no
// `output:` key is set. Mutates the input map.
func ExtractOreOutput(oreFluxDefaults map[string]any) map[string]any {
	if oreFluxDefaults == nil {
		return nil
	}
	raw, ok := oreFluxDefaults[OreOutputKey]
	if !ok {
		return nil
	}
	delete(oreFluxDefaults, OreOutputKey)
	out, ok := raw.(map[string]any)
	if !ok {
		// Schema/temper rules will reject this; return nil so callers don't
		// crash on a malformed overlay.
		return nil
	}
	return out
}
