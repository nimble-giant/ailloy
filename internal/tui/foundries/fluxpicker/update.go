package fluxpicker

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// Update is the Bubble Tea update function for the picker. The caller (App)
// only routes messages here when m.IsOpen() is true.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case schemaFetchedMsg:
		if msg.moldRef == m.moldRef {
			m.schema = msg.schema
			m.defaults = msg.defaults
			m.err = msg.err
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch m.focus {
	case focusEditor:
		return m.handleEditorKey(k)
	case focusSavePrompt:
		return m.handleSaveKey(k)
	default:
		return m.handleFilterKey(k)
	}
}

func (m Model) handleFilterKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		// Task 12 will add unsaved-changes confirm; for now, close.
		return m.Close(), nil
	case tea.KeyDown:
		m.cursor = clamp(m.cursor+1, 0, len(m.filteredKeys())-1)
		return m, nil
	case tea.KeyUp:
		m.cursor = clamp(m.cursor-1, 0, len(m.filteredKeys())-1)
		return m, nil
	case tea.KeyTab, tea.KeyEnter:
		filtered := m.filteredKeys()
		if len(filtered) == 0 {
			return m, nil
		}
		idx := m.cursor
		if k.Type == tea.KeyTab {
			idx = 0 // tab commits top match
		}
		if idx >= len(filtered) {
			idx = 0
		}
		m.editor = editorState{active: true, key: filtered[idx].Name}
		m.focus = focusEditor
		m.filter.Blur()
		return m, nil
	case tea.KeyRunes:
		// Single-character shortcuts only fire when the filter is blurred —
		// otherwise typing 'd'/'R'/'s' into the filter would clear overrides
		// or open the save prompt instead of editing the query.
		if len(k.Runes) == 1 && !m.filter.Focused() {
			switch k.Runes[0] {
			case 'd':
				filtered := m.filteredKeys()
				if m.cursor < len(filtered) {
					m = m.ClearOverride(filtered[m.cursor].Name)
				}
				return m, nil
			case 'R':
				return m.ResetOverrides(), nil
			case 's':
				m.saving = saveState{active: true}
				m.focus = focusSavePrompt
				return m, nil
			}
		}
	}
	// Otherwise pass through to the textinput; filter changed, so reset the
	// highlight to the top of the new result list.
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(k)
	m.cursor = 0
	return m, cmd
}

// handleEditorKey is filled in by Task 7. For now, esc returns to filter.
func (m Model) handleEditorKey(k tea.KeyMsg) (Model, tea.Cmd) {
	if k.Type == tea.KeyEsc {
		m.editor = editorState{}
		m.focus = focusFilter
		m.filter.Focus()
		return m, nil
	}
	return m, nil
}

// handleSaveKey is filled in by Task 8. For now, esc returns to filter.
func (m Model) handleSaveKey(k tea.KeyMsg) (Model, tea.Cmd) {
	if k.Type == tea.KeyEsc {
		m.saving = saveState{}
		m.focus = focusFilter
		m.filter.Focus()
		return m, nil
	}
	return m, nil
}

// filteredKeys returns the schema vars currently visible in the result list.
func (m Model) filteredKeys() []mold.FluxVar {
	return filterKeys(m.schema, m.filter.Value())
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
