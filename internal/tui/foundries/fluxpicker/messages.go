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

// SchemaFetchedMsg is dispatched when async schema fetch completes. The App
// fires this in response to OpenFor and the picker stitches the result into
// its Model when MoldRef matches the current target.
type SchemaFetchedMsg struct {
	MoldRef  string
	Schema   []mold.FluxVar
	Defaults map[string]any
	Err      error
}
