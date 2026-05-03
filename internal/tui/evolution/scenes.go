package evolution

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/cinematic"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Palette
var (
	preColor   = styles.Primary1           // pre-evolve fox color
	postColor  = styles.Primary2           // post-evolve fox color (the alloy)
	emberWarm  = lipgloss.Color("#ff9800") // charge halo & rolling digits
	whiteHot   = lipgloss.Color("#ffffff") // flash & crack
	dimColor   = lipgloss.Color("#2a2540") // background pre-bloom
	headlineFg = lipgloss.Color("#fff5cc") // warm cream for the title
	gold       = lipgloss.Color("#ffd28a") // fanfare accent
)

// renderHeadline types out "What? AILLOY is evolving!" character-by-character
// over the full beat, with a subtle warm underglow that brightens with t.
func (m Model) renderHeadline(t float64) string {
	full := "What? AILLOY is evolving!"
	runes := []rune(full)
	cut := min(int(float64(len(runes))*styles.EaseOutQuad(t)), len(runes))
	visible := string(runes[:cut])

	headlineStyle := lipgloss.NewStyle().
		Foreground(headlineFg).
		Bold(true)
	cursor := ""
	if cut < len(runes) && (m.frame/8)%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(emberWarm).Render("▊")
	}

	body := headlineStyle.Render(visible) + cursor

	// Underglow: a thin rule under the text whose color brightens with t.
	rule := lipgloss.NewStyle().
		Foreground(styles.LerpHex(dimColor, emberWarm, styles.EaseOutCubic(t))).
		Render(strings.Repeat("─", lipgloss.Width(full)))

	scene := lipgloss.JoinVertical(
		lipgloss.Center,
		body,
		rule,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderHold shows the fox in pre-evolve color with the current version chip
// underneath. A quiet beat — the calm before the change.
func (m Model) renderHold(_ float64) string {
	fox := cinematic.RenderFoxBlock(m.foxLines, preColor)
	chip := versionChip(m.from, preColor)
	scene := lipgloss.JoinVertical(lipgloss.Center, fox, "", chip)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderCharge tints the fox with a slow oscillating "warming" — alternating
// between preColor and a warmer ember as t advances. The version chip starts
// to flicker.
func (m Model) renderCharge(t float64) string {
	// Per-row oscillation: each row has a small phase offset so the warming
	// looks like it's sweeping through the body. Amplitude grows with t.
	amp := styles.EaseOutCubic(t)
	colorFor := func(row int) lipgloss.Color {
		// Triangle wave per row, 0..1.
		phase := (float64(row)/float64(m.foxRows)*2.0 + t*4.0)
		tri := 1 - absF(((phase-0.5)*2-1)-0.5)*2 // simple ripple
		if tri < 0 {
			tri = 0
		}
		mix := tri * amp
		return styles.LerpHex(preColor, emberWarm, mix)
	}
	fox := cinematic.RenderFox(m.foxLines, colorFor)

	// Flickering chip: the version glyphs alternate between dim and bright
	// based on the frame counter so it looks unstable.
	chipColor := preColor
	if (m.frame/3)%2 == 0 {
		chipColor = emberWarm
	}
	chip := versionChip(m.from, chipColor)

	scene := lipgloss.JoinVertical(lipgloss.Center, fox, "", chip)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderFlash is the accelerating retro RPG-style strobe: bright→dim→bright→
// dim→silhouette over the beat. The silhouette frame replaces non-space
// characters with full block runes for a unified white shape — the squash.
func (m Model) renderFlash(t float64) string {
	// Five sub-beats; the silhouette is the last one and holds longest.
	sub := min(int(t*5), 4)
	var fox string
	switch sub {
	case 0:
		fox = cinematic.RenderFoxBlock(m.foxLines, whiteHot)
	case 1:
		fox = cinematic.RenderFoxBlock(m.foxLines, preColor)
	case 2:
		fox = cinematic.RenderFoxBlock(m.foxLines, whiteHot)
	case 3:
		fox = cinematic.RenderFoxBlock(m.foxLines, preColor)
	default:
		fox = renderSilhouette(m.foxLines)
	}

	// Chip transitions through whitewash on the brightest frames.
	chipColor := preColor
	if sub%2 == 0 || sub == 4 {
		chipColor = whiteHot
	}
	chip := versionChip(m.from, chipColor)

	scene := lipgloss.JoinVertical(lipgloss.Center, fox, "", chip)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderCrack is the white-out: the fox area is rendered as solid white
// blocks fading to postColor over a quarter-second, sweeping from the center
// outward.
func (m Model) renderCrack(t float64) string {
	colorFor := func(row int) lipgloss.Color {
		// Distance from vertical center, normalized.
		mid := float64(m.foxRows) / 2
		dist := absF(float64(row)-mid) / mid
		// Sweep edge accelerates: rows further from center fade later.
		localT := (t - dist*0.5) / 0.5
		if localT < 0 {
			return whiteHot
		}
		if localT > 1 {
			return postColor
		}
		return styles.LerpHex(whiteHot, postColor, styles.EaseInOut(localT))
	}
	// Solid block silhouette — the shape we transition through.
	fox := cinematic.RenderFox(silhouetteLines(m.foxLines), colorFor)
	scene := lipgloss.JoinVertical(lipgloss.Center, fox, "", versionChip(m.from, whiteHot))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderReveal lands the new fox in postColor and runs the digit wheel
// underneath, cycling from m.from → m.to.
func (m Model) renderReveal(t float64) string {
	// Fox fades from white-hot edge into full postColor over the first
	// 30% of the beat, then holds.
	colorFor := func(int) lipgloss.Color {
		if t < 0.3 {
			return styles.LerpHex(whiteHot, postColor, styles.EaseOutCubic(t/0.3))
		}
		return postColor
	}
	fox := cinematic.RenderFox(m.foxLines, colorFor)

	wheel := RenderDigitWheel(m.from, m.to, t, m.frame)
	chip := chipBox(wheel, postColor)

	scene := lipgloss.JoinVertical(lipgloss.Center, fox, "", chip)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderFanfare is the celebration: stable post-evolve fox, sparkles drawn
// in a band around it, "Congratulations!" headline, target version chip.
func (m Model) renderFanfare(t float64) string {
	fox := cinematic.RenderFoxBlock(m.foxLines, postColor)
	sparkles := cinematic.RenderSparkles(m.sparkles, t, m.foxRows, m.foxCols)

	headline := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("✨  Congratulations!  ✨")

	subtitle := lipgloss.NewStyle().
		Foreground(styles.LerpHex(dimColor, headlineFg, styles.EaseOutCubic(t))).
		Render("ailloy " + m.from + " → " + m.to)

	chip := versionChip(m.to, postColor)

	scene := lipgloss.JoinVertical(
		lipgloss.Center,
		overlay(fox, sparkles),
		"",
		chip,
		"",
		headline,
		subtitle,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// versionChip renders a [vX.Y.Z] pill in the given color.
func versionChip(version string, color lipgloss.Color) string {
	return chipBox(lipgloss.NewStyle().Foreground(color).Bold(true).Render(version), color)
}

func chipBox(content string, borderColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2).
		Render(content)
}

// silhouetteLines returns the fox lines with every non-space rune replaced
// by '█' so the fox becomes a unified solid shape — used by the silhouette
// flash frame and the crack white-out.
func silhouetteLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		var b strings.Builder
		for _, r := range line {
			if r == ' ' {
				b.WriteRune(' ')
			} else {
				b.WriteRune('█')
			}
		}
		out[i] = b.String()
	}
	return out
}

func renderSilhouette(lines []string) string {
	return cinematic.RenderFoxBlock(silhouetteLines(lines), whiteHot)
}

// overlay composites top onto bottom by replacing bottom's cells with top's
// non-space cells. Both inputs must have matching line counts and widths;
// caller is responsible (we use the fox dimensions).
func overlay(bottom, top string) string {
	bLines := strings.Split(bottom, "\n")
	tLines := strings.Split(top, "\n")
	if len(tLines) > len(bLines) {
		tLines = tLines[:len(bLines)]
	}
	out := make([]string, len(bLines))
	copy(out, bLines)
	for i, t := range tLines {
		out[i] = compositeLine(bLines[i], t)
	}
	return strings.Join(out, "\n")
}

// compositeLine returns top's non-space rune slots overlaid on bottom. Both
// strings may carry ANSI escapes; we operate on rendered cells via
// lipgloss.Width and rune iteration. For visual correctness in the alt
// screen this is good enough — non-space top cells win, including their
// ANSI styling.
func compositeLine(bottom, top string) string {
	// Strip ANSI from top to find non-space rune positions; then walk the
	// top runes and copy them (with their preceding ANSI escapes) into
	// the result. Bottom is left untouched at space positions.
	bRunes := splitANSI(bottom)
	tRunes := splitANSI(top)
	if len(tRunes) == 0 {
		return bottom
	}
	out := make([]string, max(len(bRunes), len(tRunes)))
	copy(out, bRunes)
	for i, tr := range tRunes {
		if visibleRune(tr) == ' ' || visibleRune(tr) == 0 {
			continue
		}
		out[i] = tr
	}
	return strings.Join(out, "")
}

// splitANSI splits a string into per-cell substrings, each containing the
// rune at that cell plus any preceding ANSI escape sequence(s). Cells
// without escapes are just the rune as a string.
func splitANSI(s string) []string {
	var out []string
	var pending strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b { // ESC
			pending.WriteByte(s[i])
			i++
			for i < len(s) && s[i] != 'm' {
				pending.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				pending.WriteByte(s[i]) // 'm'
				i++
			}
			continue
		}
		// Decode one rune.
		r, sz := decodeRune(s[i:])
		var cell strings.Builder
		cell.WriteString(pending.String())
		pending.Reset()
		cell.WriteString(string(r))
		out = append(out, cell.String())
		i += sz
	}
	if pending.Len() > 0 && len(out) > 0 {
		out[len(out)-1] += pending.String()
	}
	return out
}

func decodeRune(s string) (rune, int) {
	for _, r := range s {
		// Return the first rune and its byte length.
		return r, len(string(r))
	}
	return 0, 0
}

// visibleRune returns the actual character carried by a cell string emitted
// by splitANSI (skipping any leading ANSI escapes).
func visibleRune(cell string) rune {
	i := 0
	for i < len(cell) {
		if cell[i] == 0x1b {
			for i < len(cell) && cell[i] != 'm' {
				i++
			}
			if i < len(cell) {
				i++
			}
			continue
		}
		for _, r := range cell[i:] {
			return r
		}
	}
	return 0
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
