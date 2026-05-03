package evolution

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Compact-mode renderers. Used when the terminal can't fit the giant fox
// (height < compactHeight). The beats happen on a tight horizontal band
// centered in the screen: title strip + version chip + flash + digit wheel
// + sparkle row. Same beat structure as the full layout, just no fox.

// renderHoldCompact: the title strip and current-version chip, dim and
// quiet. The breath before the strike.
func (m Model) renderHoldCompact(_ float64) string {
	return placeCompact(m,
		titleStrip(preColor),
		"",
		versionChip(m.from, preColor),
	)
}

// renderChargeCompact pulses the chip color between preColor and emberWarm.
// Pure rhythm — the dramatic warm-up.
func (m Model) renderChargeCompact(t float64) string {
	chipColor := preColor
	if (m.frame/3)%2 == 0 {
		chipColor = emberWarm
	}
	// Title gets a subtle glow that intensifies with t.
	titleColor := styles.LerpHex(headlineFg, gold, styles.EaseOutCubic(t))
	return placeCompact(m,
		titleStripColored(titleColor),
		"",
		versionChip(m.from, chipColor),
	)
}

// renderFlashCompact strobes the entire band: chip flashes, title flashes,
// finishing on a unified white silhouette of the chip.
func (m Model) renderFlashCompact(t float64) string {
	sub := min(int(t*5), 4)
	var titleColor, chipColor lipgloss.Color
	switch sub {
	case 1, 3:
		titleColor = headlineFg
		chipColor = preColor
	default: // 0, 2, 4 — bright frames; 4 is the silhouette beat
		titleColor = whiteHot
		chipColor = whiteHot
	}

	chip := versionChip(m.from, chipColor)
	if sub == 4 {
		// Render a unified-white silhouette of the chip — Pokemon's "we
		// can't see what's becoming yet" frame.
		chip = silhouetteChip(m.from, whiteHot)
	}
	return placeCompact(m,
		titleStripColored(titleColor),
		"",
		chip,
	)
}

// renderCrackCompact is the white-out: a centered band of solid blocks
// fading from white to postColor.
func (m Model) renderCrackCompact(t float64) string {
	bandColor := styles.LerpHex(whiteHot, postColor, styles.EaseInOut(t))
	band := lipgloss.NewStyle().
		Foreground(bandColor).
		Bold(true).
		Render(strings.Repeat("█", chipBandWidth(m.from, m.to)))
	return placeCompact(m,
		band,
		"",
		band,
		"",
		band,
	)
}

// renderRevealCompact lands the new color and runs the digit wheel.
func (m Model) renderRevealCompact(t float64) string {
	titleColor := postColor
	if t < 0.3 {
		titleColor = styles.LerpHex(whiteHot, postColor, styles.EaseOutCubic(t/0.3))
	}
	wheel := RenderDigitWheel(m.from, m.to, t, m.frame)
	chip := chipBox(wheel, postColor)
	return placeCompact(m,
		titleStripColored(titleColor),
		"",
		chip,
	)
}

// renderFanfareCompact is the celebration: post-color chip, sparkles in a
// thin band, "Congratulations!" headline, receipt subtitle.
func (m Model) renderFanfareCompact(t float64) string {
	headline := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("✨  Congratulations!  ✨")

	subtitle := lipgloss.NewStyle().
		Foreground(styles.LerpHex(dimColor, headlineFg, styles.EaseOutCubic(t))).
		Render("ailloy " + m.from + " → " + m.to)

	chip := versionChip(m.to, postColor)

	// One-line sparkle band above the chip — tracks the same per-particle
	// phase windows as the full layout but constrained to a single row.
	sparkleRow := compactSparkleRow(m, t)

	return placeCompact(m,
		sparkleRow,
		"",
		chip,
		"",
		headline,
		subtitle,
	)
}

// titleStrip is the "AILLOY is evolving!" headline rendered as a styled pill.
func titleStrip(fg lipgloss.Color) string {
	return titleStripColored(fg)
}

func titleStripColored(fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(fg).
		Bold(true).
		Render("⚡  AILLOY is evolving  ⚡")
}

// silhouetteChip renders the version chip with all glyphs replaced by '█'
// for the unified-white silhouette frame in the compact flash beat.
func silhouetteChip(version string, color lipgloss.Color) string {
	silhouette := strings.Repeat("█", lipgloss.Width(version))
	styled := lipgloss.NewStyle().Foreground(color).Bold(true).Render(silhouette)
	return chipBox(styled, color)
}

// chipBandWidth returns a sensible width for the white-out crack band so it
// reads as a horizontal slash across the action.
func chipBandWidth(from, to string) int {
	w := lipgloss.Width(from)
	if tw := lipgloss.Width(to); tw > w {
		w = tw
	}
	return w + 8 // a little wider than the chip itself
}

// compactSparkleRow draws a single-row sparkle band above the chip during
// fanfare, using the same per-particle phase logic as the full sparkle
// field but flattened to one row.
func compactSparkleRow(m Model, t float64) string {
	const window = 0.35
	if len(m.sparkles) == 0 {
		return ""
	}
	// Width: as wide as the chip with some breathing room.
	width := chipBandWidth(m.from, m.to)
	cells := make([]string, width)
	for i := range cells {
		cells[i] = " "
	}
	for i, s := range m.sparkles {
		dt := t - s.PhaseOff
		if dt < 0 || dt > window {
			continue
		}
		pos := dt / window
		intensity := 1 - absF(pos-0.5)*2
		if intensity <= 0 {
			continue
		}
		col := (i*7 + 3) % width
		c := styles.LerpHex(lipgloss.Color("#9aa0a6"), gold, intensity)
		cells[col] = lipgloss.NewStyle().Foreground(c).Bold(true).Render(s.Glyph)
	}
	return strings.Join(cells, "")
}

// placeCompact stacks the supplied lines vertically (centered) and centers
// the whole block within the terminal. Used by every compact-mode renderer.
func placeCompact(m Model, parts ...string) string {
	scene := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}
