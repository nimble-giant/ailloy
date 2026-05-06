package fluxpicker

import (
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// OpenForFoundry opens the picker scoped to every mold in a foundry.
// `perMoldSchemas` carries each mold's flux schema (used for aggregation).
// `perMoldSourceRefs` carries each mold's source ref string (used at save
// time to derive the per-mold flux file slug — must match the slug used by
// the cast pipeline so subsequent casts pick up these values). Pass nil for
// `perMoldSourceRefs` when the picker is opened before refs are known; it
// can be filled in later via WithFoundrySchemas (added in Task 10).
//
// The picker aggregates schemas: keys with consistent type AND default
// render as one unified row whose value applies to every declaring mold;
// keys whose type or default differs across molds are flagged in
// SchemaConflicts and rendered as expandable per-mold sub-rows by the View.
func (m Model) OpenForFoundry(
	foundryName string,
	scope data.Scope,
	perMoldSchemas map[string][]mold.FluxVar,
	perMoldSourceRefs map[string]string,
) Model {
	unified, conflicts := AggregateSchemas(perMoldSchemas)
	m.open = true
	m.moldRef = "" // foundry-scope, no single mold ref
	m.scope = scope
	m.schema = unified
	m.defaults = map[string]any{} // no aggregated defaults; per-mold defaults shown in conflict expansions
	m.overrides = map[string]any{}
	m.cursor = 0
	m.focus = focusFilter
	m.filter.SetValue("")
	m.filter.Focus()
	m.err = nil
	m.fetching = false
	m.foundryName = foundryName
	m.perMoldSchemas = perMoldSchemas
	m.perMoldSourceRefs = perMoldSourceRefs
	m.schemaConflicts = conflicts
	return m
}

// IsFoundryMode reports whether the picker is scoped to a foundry rather than
// a single mold.
func (m Model) IsFoundryMode() bool { return m.foundryName != "" }

// FoundryName returns the foundry name when in foundry mode, "" otherwise.
func (m Model) FoundryName() string { return m.foundryName }

// PerMoldSchemas returns the per-mold schemas captured at open time. Used by
// persistence to decide which mold flux files to write at fan-out.
func (m Model) PerMoldSchemas() map[string][]mold.FluxVar { return m.perMoldSchemas }

// PerMoldSourceRefs returns the per-mold source refs captured at open time.
// Used at save time to derive each mold's flux file slug.
func (m Model) PerMoldSourceRefs() map[string]string { return m.perMoldSourceRefs }

// SchemaConflicts returns conflict info: var name → sorted list of mold names
// that disagree on Type or Default for that var.
func (m Model) SchemaConflicts() map[string][]string { return m.schemaConflicts }

// SetMoldOverride records a per-mold value for a conflict-expanded key. Only
// meaningful in foundry mode; no-op otherwise.
func (m Model) SetMoldOverride(moldName, key string, value any) Model {
	if !m.IsFoundryMode() {
		return m
	}
	if m.moldOverrides == nil {
		m.moldOverrides = map[string]any{}
	}
	m.moldOverrides[moldOverrideKey(moldName, key)] = value
	return m
}

// MoldOverride returns the per-mold value for a conflict-expanded key, or
// nil if unset.
func (m Model) MoldOverride(moldName, key string) any {
	if m.moldOverrides == nil {
		return nil
	}
	return m.moldOverrides[moldOverrideKey(moldName, key)]
}

// MoldOverrides returns the full per-mold override map. Result keyed by
// mold name → flat dotted-key map (e.g. "alpha" → {"theme": "midnight"}).
func (m Model) MoldOverrides() map[string]map[string]any {
	if m.moldOverrides == nil {
		return nil
	}
	out := map[string]map[string]any{}
	for k, v := range m.moldOverrides {
		moldName, key, ok := splitMoldOverrideKey(k)
		if !ok {
			continue
		}
		if out[moldName] == nil {
			out[moldName] = map[string]any{}
		}
		out[moldName][key] = v
	}
	return out
}

func moldOverrideKey(moldName, key string) string {
	return moldName + "\x00" + key
}

func splitMoldOverrideKey(k string) (moldName, key string, ok bool) {
	for i := 0; i < len(k); i++ {
		if k[i] == 0 {
			return k[:i], k[i+1:], true
		}
	}
	return "", "", false
}

// WithFoundrySchemas folds asynchronously-fetched per-mold schemas into the
// picker, runs aggregation, and clears the fetching flag. No-op when the
// picker is not currently scoped to the named foundry.
func (m Model) WithFoundrySchemas(foundryName string, schemas map[string][]mold.FluxVar, refs map[string]string) Model {
	if m.foundryName != foundryName {
		return m
	}
	unified, conflicts := AggregateSchemas(schemas)
	m.schema = unified
	m.schemaConflicts = conflicts
	m.perMoldSchemas = schemas
	m.perMoldSourceRefs = refs
	m.fetching = false
	return m
}
