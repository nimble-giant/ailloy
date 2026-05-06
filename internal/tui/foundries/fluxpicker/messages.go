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

// SchemaFetchedMsg is dispatched when async schema fetch completes. The App
// fires this in response to OpenFor and the picker stitches the result into
// its Model when MoldRef matches the current target.
type SchemaFetchedMsg struct {
	MoldRef  string
	Schema   []mold.FluxVar
	Defaults map[string]any
	Err      error
}

// FoundrySchemasFetchedMsg is dispatched by the App after the foundry-mode
// picker is opened. The picker stitches the schemas in via WithFoundrySchemas.
type FoundrySchemasFetchedMsg struct {
	FoundryName string
	Schemas     map[string][]mold.FluxVar
	SourceRefs  map[string]string
	Err         error
}
