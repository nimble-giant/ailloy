package cinematic

import (
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Sparkle is one twinkle: a position relative to the fox bounding box,
// a glyph, and a per-particle phase offset that controls when it peaks.
type Sparkle struct {
	Row      int
	Col      int
	Glyph    string
	PhaseOff float64 // 0..1
}

// NewSparkleField generates n deterministic sparkles around a fox-sized box.
// Sparkles avoid the central-mass region (rows ~30-70% of the box) so the
// fox's face stays clean. The seed is deterministic — this is a movie, not
// RNG, and the same upgrade should look the same on every replay.
func NewSparkleField(n, foxRows, foxCols int, seed int64) []Sparkle {
	r := rand.New(rand.NewSource(seed)) // #nosec G404 -- visual variety, not security
	glyphs := []string{"✦", "✧", "·", "*", "⋆", "✺"}
	out := make([]Sparkle, 0, n)
	avoidLo := int(float64(foxRows) * 0.30)
	avoidHi := int(float64(foxRows) * 0.70)
	for range n {
		var row, col int
		// Try a few times to land outside the avoid band; if we can't,
		// fall back to a forced top/bottom band placement so we never
		// loop forever.
		for range 8 {
			row = r.Intn(foxRows)
			col = r.Intn(foxCols)
			if row < avoidLo || row > avoidHi {
				break
			}
		}
		if row >= avoidLo && row <= avoidHi {
			if r.Intn(2) == 0 {
				row = r.Intn(avoidLo + 1)
			} else {
				row = avoidHi + r.Intn(foxRows-avoidHi)
			}
		}
		out = append(out, Sparkle{
			Row:      row,
			Col:      col,
			Glyph:    glyphs[r.Intn(len(glyphs))],
			PhaseOff: r.Float64(),
		})
	}
	return out
}

// RenderSparkles returns a multi-line overlay (foxRows tall, foxCols wide)
// with each sparkle drawn at intensity = ease(t - PhaseOff). Empty cells are
// spaces so the overlay can be composited via per-line max with the fox
// underneath, but in practice we render this as a separate panel below the
// fox in the fanfare beat — keeping the fox unobstructed.
func RenderSparkles(sparkles []Sparkle, t float64, rows, cols int) string {
	grid := make([][]rune, rows)
	intensities := make([][]float64, rows)
	for i := range grid {
		grid[i] = make([]rune, cols)
		intensities[i] = make([]float64, cols)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	for _, s := range sparkles {
		// Each sparkle has a 0.35-wide eased window centered on PhaseOff.
		// Outside that window: invisible. Inside: brightness peaks at the
		// midpoint and falls off symmetrically.
		const window = 0.35
		dt := t - s.PhaseOff
		if dt < 0 || dt > window {
			continue
		}
		pos := dt / window
		intensity := 1 - absF(pos-0.5)*2
		if intensity <= 0 {
			continue
		}
		if s.Row < 0 || s.Row >= rows || s.Col < 0 || s.Col >= cols {
			continue
		}
		runes := []rune(s.Glyph)
		if len(runes) == 0 {
			continue
		}
		if intensity > intensities[s.Row][s.Col] {
			grid[s.Row][s.Col] = runes[0]
			intensities[s.Row][s.Col] = intensity
		}
	}

	// Render line-by-line, per-glyph color picked by its intensity.
	var b strings.Builder
	for i := range rows {
		for j := range cols {
			ch := grid[i][j]
			if ch == ' ' {
				b.WriteByte(' ')
				continue
			}
			c := styles.LerpHex(lipgloss.Color("#9aa0a6"), lipgloss.Color("#fff5cc"), intensities[i][j])
			b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(ch)))
		}
		if i < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
