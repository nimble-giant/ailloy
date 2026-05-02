package fluxpicker

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	fmt.Fprintf(&b, "%s — %s\n\n",
		headerStyle.Render("Flux: "+m.moldRef),
		strings.ToLower(string(m.scope)))

	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

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
		if i == m.cursor && m.focus == focusFilter {
			line = highlight.Render("▸ " + line)
		} else {
			line = "  " + row.Render(line)
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
		b.WriteString("\nsave: [p] project  [g] global  [o] this cast only  [esc] cancel\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("error: "+m.err.Error()) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(footer.Render("tab: commit filter   enter: save key   d: clear   R: reset all   s: save & close   esc: cancel"))

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
