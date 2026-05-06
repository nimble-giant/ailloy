package fluxpicker

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const footerHint = "tab: commit filter   enter: edit key   ctrl+s: save & close   esc: discard / close"

var (
	pickerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)
	headerStyle = lipgloss.NewStyle().Bold(true)
	rowSet      = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	rowDefault  = lipgloss.NewStyle().Faint(true)
	rowUnset    = lipgloss.NewStyle()
	highlight   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	footer      = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// View renders the picker overlay. Returns empty string when closed.
func (m Model) View() string {
	if !m.open {
		return ""
	}
	var b strings.Builder
	headerLabel := "Flux: " + m.moldRef
	if m.IsFoundryMode() {
		headerLabel = "Foundry: " + m.foundryName
	}
	fmt.Fprintf(&b, "%s — %s\n\n",
		headerStyle.Render(headerLabel),
		strings.ToLower(string(m.scope)))

	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	if m.fetching {
		b.WriteString("fetching schema…\n\n")
	} else if len(m.schema) == 0 && m.err == nil {
		b.WriteString(footer.Render("(no flux variables defined for this mold)") + "\n\n")
	}

	rows := m.filteredKeys()
	for i, fv := range rows {
		badge := " "
		row := rowUnset
		switch m.BadgeStateFor(fv.Name) {
		case BadgeSet:
			badge = "●"
			row = rowSet
		case BadgeDefault:
			badge = "○"
			row = rowDefault
		}
		display := m.displayValueFor(fv.Name)
		line := fmt.Sprintf("%s %-22s %-8s %s", badge, fv.Name, fv.Type, display)
		// cursor starts at 0, so the top row is highlighted by default until
		// the user moves it.
		if i == m.cursor && m.focus == focusFilter {
			line = highlight.Render("▸ " + line)
		} else {
			line = "  " + row.Render(line)
		}
		if m.IsFoundryMode() {
			if _, conflict := m.schemaConflicts[fv.Name]; conflict {
				line += "  ⚠ conflicts — expand to set per mold"
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.editor.active {
		b.WriteString("\n" + strings.Repeat("─", 40) + " value editor " + strings.Repeat("─", 24) + "\n")
		if m.editor.form != nil {
			b.WriteString(m.editor.form.View())
		} else {
			b.WriteString("editing " + m.editor.key)
		}
	}

	if m.saving.active {
		fmt.Fprintf(&b,
			"\nSave %d override(s)?  [p] project (.ailloy/flux/)  [g] global (~/.ailloy/flux/)  [o] this cast only  [esc] discard & close\n",
			len(m.overrides))
	}

	if m.err != nil {
		// Developer-facing tool — raw error text is intentional.
		b.WriteString("\n" + errStyle.Render("error: "+m.err.Error()) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(footer.Render(footerHint))

	return pickerBox.Render(b.String())
}

func (m Model) displayValueFor(name string) string {
	if v, ok := m.overrides[name]; ok {
		return fmt.Sprintf("%v", v)
	}
	if hasDottedKey(m.defaults, name) {
		return fmt.Sprintf("%v (default)", lookupDottedKey(m.defaults, name))
	}
	return "—"
}

func lookupDottedKey(m map[string]any, key string) any {
	parts := strings.Split(key, ".")
	var cur any = m
	for _, p := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}
