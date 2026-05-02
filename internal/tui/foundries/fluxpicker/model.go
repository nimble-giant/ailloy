package fluxpicker

import (
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// focusArea identifies which sub-component holds keyboard focus.
type focusArea int

const (
	focusFilter focusArea = iota
	focusEditor
	focusSavePrompt
)

// Model is the state for the flux picker overlay. It is a value type to fit
// Bubble Tea's pattern of pure Update functions.
type Model struct {
	open      bool
	moldRef   string
	scope     data.Scope
	schema    []mold.FluxVar
	defaults  map[string]any
	overrides map[string]any
	filter    textinput.Model
	cursor    int
	focus     focusArea
	editor    editorState
	saving    saveState
	err       error
	width     int
	height    int
	fetching  bool
}

// New returns a closed picker model.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "filter flux keys (e.g. agents.tar)"
	ti.Prompt = "▸ "
	return Model{
		filter:    ti,
		overrides: map[string]any{},
	}
}

// IsOpen reports whether the picker is currently visible.
func (m Model) IsOpen() bool { return m.open }

// MoldRef returns the mold reference the picker is scoped to.
func (m Model) MoldRef() string { return m.moldRef }

// Scope returns the scope (project/global) inherited from the active tab.
func (m Model) Scope() data.Scope { return m.scope }

// Overrides returns the current session overrides.
func (m Model) Overrides() map[string]any { return m.overrides }

// OpenFor opens the picker scoped to the given mold and schema.
func (m Model) OpenFor(ref string, scope data.Scope, schema []mold.FluxVar, defaults map[string]any) Model {
	m.open = true
	m.moldRef = ref
	m.scope = scope
	m.schema = schema
	m.defaults = defaults
	m.overrides = map[string]any{}
	m.cursor = 0
	m.focus = focusFilter
	m.filter.SetValue("")
	m.filter.Focus()
	m.err = nil
	return m
}

// Close hides the picker and clears transient state. Overrides are retained
// so the App can broadcast them to the active tab on close.
func (m Model) Close() Model {
	m.open = false
	m.focus = focusFilter
	m.editor = editorState{}
	m.saving = saveState{}
	return m
}

// editorState holds the active key being edited. Form fields are added in
// Task 7.
type editorState struct {
	active bool
	key    string
}

// saveState holds the save-target prompt's transient state.
type saveState struct {
	active bool
}
