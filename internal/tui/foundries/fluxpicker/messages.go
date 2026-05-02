package fluxpicker

import (
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// SaveTarget identifies where the picker writes overrides on save.
type SaveTarget int

const (
	SaveTargetSession SaveTarget = iota
	SaveTargetProject
	SaveTargetGlobal
)

// FluxOverridesMsg is emitted by the picker on close. The active tab folds
// these into its next cast call (via CastOptions.SetOverrides) or notes that
// they have already been written to a file (Target != SaveTargetSession).
type FluxOverridesMsg struct {
	MoldRef   string
	Scope     data.Scope
	Overrides map[string]any
	Target    SaveTarget
}

// OpenPickerMsg requests that the picker open scoped to a particular mold.
type OpenPickerMsg struct {
	MoldRef string
	Scope   data.Scope
}

// schemaFetchedMsg is dispatched when async schema fetch completes.
// (Promoted to exported SchemaFetchedMsg in Task 13.)
type schemaFetchedMsg struct {
	moldRef  string
	schema   []mold.FluxVar
	defaults map[string]any
	err      error
}
