package evolution

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// RenderDigitWheel returns a string where each rune position blends from
// `from` to `to` based on per-position eased t. Non-digit characters (the
// leading 'v', dots, dashes) are held constant from `to` so the layout never
// shifts. Each digit position has its own staggered start so they appear to
// "land" left-to-right like a flip-board / slot machine.
//
// During the rolling window, each position cycles through 0..9 driven by a
// frame-derived hash for visual energy, then locks to the target value once
// the position's lock threshold is crossed. Non-digit positions in `from` and
// `to` must match — otherwise we fall back to the static `to`.
//
// `frame` is an integer ticking ~60Hz; it provides the variation during the
// rolling window. `t` is normalized [0, 1] phase progress.
func RenderDigitWheel(from, to string, t float64, frame int) string {
	fromRunes := []rune(from)
	toRunes := []rune(to)
	if len(fromRunes) != len(toRunes) {
		return renderStatic(to)
	}

	// Validate non-digit positions match. If they don't, the version
	// strings have different shapes (e.g. "v0.6.21" vs "v0.7") and we
	// can't sensibly cycle digit-by-digit.
	for i := range toRunes {
		if !unicode.IsDigit(toRunes[i]) && fromRunes[i] != toRunes[i] {
			return renderStatic(to)
		}
	}

	// Per-position lock threshold: leftmost locks first, rightmost last.
	// Maps to a curve that gives the "left to right" reel-stop feel.
	locks := perPositionLocks(len(toRunes))

	var b strings.Builder
	for i := range toRunes {
		ch := toRunes[i]
		if !unicode.IsDigit(ch) {
			b.WriteString(staticChar(ch))
			continue
		}
		if t >= locks[i] {
			b.WriteString(lockedDigit(ch))
			continue
		}
		// Cycle: pseudo-random digit derived from (position, frame).
		digit := byte('0' + ((i*17 + frame*3) % 10))
		b.WriteString(rollingDigit(rune(digit)))
	}
	return b.String()
}

// perPositionLocks returns one lock threshold per position, scaled across
// roughly the first 80% of the phase so the last digit lands by t=0.8.
func perPositionLocks(n int) []float64 {
	locks := make([]float64, n)
	if n == 0 {
		return locks
	}
	for i := range n {
		// Linearly distributed 0.15 → 0.80.
		locks[i] = 0.15 + 0.65*float64(i)/float64(max(n-1, 1))
	}
	return locks
}

func renderStatic(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteString(lockedDigit(r))
		} else {
			b.WriteString(staticChar(r))
		}
	}
	return b.String()
}

func staticChar(r rune) string {
	return lipgloss.NewStyle().Foreground(styles.Primary1).Render(string(r))
}

func lockedDigit(r rune) string {
	return lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true).Render(string(r))
}

func rollingDigit(r rune) string {
	// Rolling digits are warm/ember-colored to suggest motion energy.
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9800")).Bold(true).Render(string(r))
}
