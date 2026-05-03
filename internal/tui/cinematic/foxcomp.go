// Package cinematic holds shared rendering primitives for the splash and
// evolution cinematics: the AilloyFox layout, sparkle particle field, etc.
// Anything specific to a single scene lives in that scene's package.
package cinematic

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// PadFoxLines returns the AilloyFox lines, leading/trailing newlines stripped,
// every line padded with trailing spaces to a uniform width. Both splash and
// evolution call this so the fox renders without sideways skew under any
// centering layer (lipgloss.Place, lipgloss.JoinVertical with Center) — those
// would otherwise re-center each row independently and warp the silhouette.
func PadFoxLines() (lines []string, cols int) {
	art := strings.TrimLeft(styles.AilloyFox, "\n")
	art = strings.TrimRight(art, "\n")
	lines = strings.Split(art, "\n")
	for _, l := range lines {
		if w := lipgloss.Width(l); w > cols {
			cols = w
		}
	}
	for i, l := range lines {
		if pad := cols - lipgloss.Width(l); pad > 0 {
			lines[i] = l + strings.Repeat(" ", pad)
		}
	}
	return lines, cols
}

// RenderFox composes the padded fox by calling colorFor(rowIndex) for each
// row's foreground. Returns one multi-line string ready to feed to
// lipgloss.Place. Caller decides the centering wrapper.
func RenderFox(lines []string, colorFor func(row int) lipgloss.Color) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = lipgloss.NewStyle().Foreground(colorFor(i)).Render(l)
	}
	return strings.Join(out, "\n")
}

// RenderFoxBlock composes the padded fox where every row uses the same
// foreground color. Convenience over RenderFox for the common case.
func RenderFoxBlock(lines []string, color lipgloss.Color) string {
	return RenderFox(lines, func(int) lipgloss.Color { return color })
}
