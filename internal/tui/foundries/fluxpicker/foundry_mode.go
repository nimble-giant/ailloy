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
