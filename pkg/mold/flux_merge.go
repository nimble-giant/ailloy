package mold

import (
	"fmt"
	"sort"
)

// OverlaySchema is one ore's prefixed schema entries, tagged with the source
// identifier used in error and diagnostic messages.
type OverlaySchema struct {
	Source  string
	Entries []FluxVar
}

// ShadowedEntry records a key that an overlay defined but a mold-local entry
// won. Surfaced in OreLoadReport for `forge --debug` and temper diagnostics.
type ShadowedEntry struct {
	Name   string
	Source string // "ore:status" — the overlay that lost
}

// OreLoadReport carries diagnostics produced during schema/defaults merging.
type OreLoadReport struct {
	Shadowed []ShadowedEntry
	// Sources records the source of each entry in the merged schema, keyed by
	// FluxVar.Name. "" means the mold itself.
	Sources map[string]string
}

// MergeFluxSchema combines a base schema (the mold's own flux.schema.yaml
// entries) with N overlay schemas (each is one installed ore's entries,
// already prefixed with ore.<alias|name>. by the caller).
//
// Rules:
//   - If a name is in base, base wins; the overlay copy is recorded in
//     report.Shadowed.
//   - If a name appears in two overlays, return a fatal error naming both
//     sources.
//   - Net-new overlay entries are appended after base, in deterministic
//     overlay-source order.
func MergeFluxSchema(base []FluxVar, overlays []OverlaySchema) ([]FluxVar, OreLoadReport, error) {
	report := OreLoadReport{Sources: make(map[string]string, len(base))}

	baseIdx := make(map[string]struct{}, len(base))
	out := make([]FluxVar, 0, len(base))
	for _, e := range base {
		baseIdx[e.Name] = struct{}{}
		out = append(out, e)
		report.Sources[e.Name] = ""
	}

	sortedOverlays := make([]OverlaySchema, len(overlays))
	copy(sortedOverlays, overlays)
	sort.Slice(sortedOverlays, func(i, j int) bool { return sortedOverlays[i].Source < sortedOverlays[j].Source })

	overlayOwner := make(map[string]string)

	for _, ov := range sortedOverlays {
		for _, e := range ov.Entries {
			if _, inBase := baseIdx[e.Name]; inBase {
				report.Shadowed = append(report.Shadowed, ShadowedEntry{Name: e.Name, Source: ov.Source})
				continue
			}
			if prevSource, dup := overlayOwner[e.Name]; dup {
				return nil, report, fmt.Errorf(
					"ore key conflict: %q defined by both %s and %s; resolve with 'as:' in mold.yaml or '--as' on ailloy ore add",
					e.Name, prevSource, ov.Source,
				)
			}
			overlayOwner[e.Name] = ov.Source
			out = append(out, e)
			report.Sources[e.Name] = ov.Source
		}
	}
	return out, report, nil
}

// ValidateOrphanDefaults returns dotted-path keys whose values appear in
// defaults but have no corresponding schema entry. Used as an informational
// signal at temper time.
func ValidateOrphanDefaults(schema []FluxVar, defaults map[string]any) []string {
	known := make(map[string]struct{}, len(schema))
	for _, e := range schema {
		known[e.Name] = struct{}{}
	}
	var orphans []string
	walkLeafKeys(defaults, "", func(dotted string) {
		if _, ok := known[dotted]; !ok {
			orphans = append(orphans, dotted)
		}
	})
	sort.Strings(orphans)
	return orphans
}

func walkLeafKeys(m map[string]any, prefix string, visit func(string)) {
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			walkLeafKeys(child, path, visit)
			continue
		}
		visit(path)
	}
}
