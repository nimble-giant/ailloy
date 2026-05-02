package fluxpicker

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/huh"

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
	open     bool
	moldRef  string
	scope    data.Scope
	schema   []mold.FluxVar
	defaults map[string]any
	// overrides is keyed by the dotted FluxVar.Name (e.g. "agents.targets"),
	// not by nested maps — kept flat so badge lookup and --set encoding stay
	// trivial. defaults is YAML-shaped (nested), populated from flux.yaml.
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

// editorState holds the active key being edited.
type editorState struct {
	active   bool
	key      string
	form     *huh.Form
	rawValue *string
	boolVal  *bool
}

// saveState holds the save-target prompt's transient state.
type saveState struct {
	active bool
}

// BadgeState describes the visual badge to draw next to a key in the list.
type BadgeState int

const (
	BadgeUnset BadgeState = iota
	BadgeDefault
	BadgeSet
)

// SetOverride records a session override.
func (m Model) SetOverride(key string, value any) Model {
	if m.overrides == nil {
		m.overrides = map[string]any{}
	}
	m.overrides[key] = value
	return m
}

// ClearOverride removes a session override.
func (m Model) ClearOverride(key string) Model {
	delete(m.overrides, key)
	return m
}

// ResetOverrides clears all session overrides.
func (m Model) ResetOverrides() Model {
	m.overrides = map[string]any{}
	return m
}

// BadgeStateFor returns the badge to render next to a key. The defaults map
// uses dotted-path lookup (e.g. "agents.targets" → defaults["agents"]["targets"]).
func (m Model) BadgeStateFor(key string) BadgeState {
	if _, ok := m.overrides[key]; ok {
		return BadgeSet
	}
	if hasDottedKey(m.defaults, key) {
		return BadgeDefault
	}
	return BadgeUnset
}

// hasDottedKey reports whether a dotted key exists in a nested map.
func hasDottedKey(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	parts := strings.Split(key, ".")
	cur := any(m)
	for _, p := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		v, present := mm[p]
		if !present {
			return false
		}
		cur = v
	}
	return true
}
