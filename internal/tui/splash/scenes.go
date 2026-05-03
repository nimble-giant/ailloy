package splash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/cinematic"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Palette
var (
	coldDim    = lipgloss.Color("#2a2540") // pre-bloom row tint
	emberHot   = lipgloss.Color("#ffd28a") // bright ember/forge core
	emberWarm  = lipgloss.Color("#ff9800") // outer bloom
	whiteHot   = lipgloss.Color("#ffffff") // cooling sweep peak
	finalColor = styles.Primary1           // settled fox color
	titleBg    = styles.Primary1
	titleFg    = styles.White
)

// renderIgnite paints a centered spark with secondary particle accents that
// twinkle in. t ∈ [0, 1] eased. The visual: a single growing ember in the
// dead center, two flanking dots, and a soft halo glow as t approaches 1.
func (m Model) renderIgnite(t float64) string {
	// Map t → ember intensity & "size" character.
	emberChar := "·"
	switch {
	case t > 0.85:
		emberChar = "✸"
	case t > 0.6:
		emberChar = "✦"
	case t > 0.3:
		emberChar = "•"
	}
	emberColor := styles.LerpHex(coldDim, emberHot, t)
	ember := lipgloss.NewStyle().Foreground(emberColor).Bold(true).Render(emberChar)

	// Two flanking accent particles on a small offset.
	leftP := ""
	rightP := ""
	if t > 0.45 {
		pt := (t - 0.45) / 0.55
		c := styles.LerpHex(coldDim, emberWarm, styles.EaseOutCubic(pt))
		leftP = lipgloss.NewStyle().Foreground(c).Render("·")
		rightP = lipgloss.NewStyle().Foreground(c).Render("·")
	}

	// Compose a small horizontal cluster: "left   ember   right"
	cluster := joinHoriz(leftP, "    ", ember, "    ", rightP)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, cluster)
}

// renderBloom expands a warm gradient outward and progressively reveals the
// fox top-to-bottom. Each row of the fox has an "appear time" based on its
// vertical position. When t passes that row's appear time, the row begins
// fading from coldDim → finalColor over its own per-row eased window.
func (m Model) renderBloom(t float64) string {
	rows := m.foxRows
	out := make([]string, rows)

	// Bloom halo sits behind the fox: a thin band of warm color whose
	// vertical position tracks the wavefront. Implemented as a per-row
	// background tint that shifts down. Cheap and effective.
	wavefront := t // 0..1 normalized, top → bottom

	for i, line := range m.foxLines {
		appearAt := float64(i) / float64(rows)
		if wavefront < appearAt {
			// Row not yet revealed — render an empty slug so layout
			// is stable and the fox doesn't "jump" into place.
			out[i] = blank(lipgloss.Width(line))
			continue
		}
		// Per-row eased reveal window: 0.18 of normalized phase time.
		const revealWindow = 0.18
		localT := (wavefront - appearAt) / revealWindow
		if localT > 1 {
			localT = 1
		}
		eased := styles.EaseOutCubic(localT)

		// Color blends from cold dim → warm ember at mid → final purple.
		// Three-point ramp: coldDim --(0..0.5)--> emberWarm --(0.5..1)--> finalColor.
		var c lipgloss.Color
		if eased < 0.5 {
			c = styles.LerpHex(coldDim, emberWarm, eased*2)
		} else {
			c = styles.LerpHex(emberWarm, finalColor, (eased-0.5)*2)
		}
		out[i] = lipgloss.NewStyle().Foreground(c).Render(line)
	}

	scene := strings.Join(out, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderCool runs a vertical "white-hot" sweep top → bottom across the fully
// revealed fox. Rows currently within the sweep band flash White, rows behind
// the band have settled to finalColor, rows ahead are still finalColor (the
// fox is fully revealed by this beat, just receiving the cool flash).
func (m Model) renderCool(t float64) string {
	rows := m.foxRows
	out := make([]string, rows)

	// Sweep band has a half-width of 3 rows — soft falloff via cosine-ish
	// triangle. Center moves from row -3 to row rows+3 so the band fully
	// enters and exits the fox.
	bandHalf := 3.0
	bandCenter := -bandHalf + (float64(rows)+2*bandHalf)*styles.EaseInOut(t)

	for i, line := range m.foxLines {
		dist := absF(float64(i) - bandCenter)
		intensity := 0.0
		if dist < bandHalf {
			intensity = 1 - (dist / bandHalf)
		}
		c := styles.LerpHex(finalColor, whiteHot, intensity)
		out[i] = lipgloss.NewStyle().Foreground(c).Render(line)
	}

	scene := strings.Join(out, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderTitle holds the fox in its final color and slams the AILLOY badge in
// underneath with a single squash-and-stretch frame, then settles. The
// subtitle and version fade in dimly during the back half of the beat.
func (m Model) renderTitle(t float64) string {
	fox := m.staticFox()

	// Squash-and-stretch: t=0..0.15 the badge is missing (the slam hasn't
	// started); 0.15..0.3 a single oversized frame; 0.3..0.55 settle to
	// final padding; 0.55..1.0 subtitle + version fade in.
	var badge string
	switch {
	case t < 0.15:
		badge = ""
	case t < 0.3:
		badge = lipgloss.NewStyle().
			Background(titleBg).
			Foreground(titleFg).
			Bold(true).
			Padding(2, 5).
			Render("🧠 AILLOY")
	default:
		badge = lipgloss.NewStyle().
			Background(titleBg).
			Foreground(titleFg).
			Bold(true).
			Padding(1, 3).
			Render("🧠 AILLOY")
	}

	subtitle := ""
	version := ""
	if t > 0.55 {
		ft := (t - 0.55) / 0.45
		dim := styles.LerpHex(coldDim, finalColor, styles.EaseOutCubic(ft))
		subtitle = lipgloss.NewStyle().Foreground(dim).Render("AI-powered development workflows")
		version = lipgloss.NewStyle().Foreground(styles.LerpHex(coldDim, lipgloss.Color("#9aa0a6"), styles.EaseOutCubic(ft))).Render("forging…")
	}

	scene := lipgloss.JoinVertical(
		lipgloss.Center,
		fox,
		"",
		badge,
		subtitle,
		"",
		version,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

// renderSettle holds the hero composition. The description and quick-start
// blocks are intentionally NOT shown during the splash — they live in
// normal scrollback after exit. The splash's job is to make the fox + badge
// land; the help text is the receipt.
func (m Model) renderSettle(t float64) string {
	fox := m.staticFox()

	badge := lipgloss.NewStyle().
		Background(titleBg).
		Foreground(titleFg).
		Bold(true).
		Padding(1, 3).
		Render("🧠 AILLOY")

	subtitle := lipgloss.NewStyle().Foreground(finalColor).Render("AI-powered development workflows")

	// The "ready" tag fades in over the settle beat to signal hand-off.
	readyColor := styles.LerpHex(coldDim, lipgloss.Color("#9aa0a6"), t)
	ready := lipgloss.NewStyle().Foreground(readyColor).Italic(true).Render("ready.")

	scene := lipgloss.JoinVertical(
		lipgloss.Center,
		fox,
		"",
		badge,
		subtitle,
		"",
		ready,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, scene)
}

func (m Model) staticFox() string {
	return cinematic.RenderFoxBlock(m.foxLines, finalColor)
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func blank(w int) string {
	if w <= 0 {
		return ""
	}
	return strings.Repeat(" ", w)
}

func joinHoriz(parts ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
