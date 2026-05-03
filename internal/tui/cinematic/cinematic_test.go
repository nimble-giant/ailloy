package cinematic

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPadFoxLines_UniformWidth(t *testing.T) {
	lines, cols := PadFoxLines()
	if len(lines) == 0 {
		t.Fatal("expected non-empty fox lines")
	}
	if cols == 0 {
		t.Fatal("expected non-zero fox width")
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != cols {
			t.Errorf("line %d width = %d, want uniform %d", i, w, cols)
		}
	}
}

func TestPadFoxLines_NoLeadingTrailingNewlines(t *testing.T) {
	lines, _ := PadFoxLines()
	if strings.TrimSpace(lines[0]) == "" {
		// The padded first line is whitespace-only? That's a sign the
		// pre-pad strip didn't remove a leading blank. Allow it only
		// if the original art genuinely starts with a blank-content
		// line — fox top is decorative and may include blank rows
		// inside, but not as the first row.
		t.Errorf("first padded line is blank, expected art content")
	}
}

func TestNewSparkleField_Deterministic(t *testing.T) {
	a := NewSparkleField(10, 30, 70, 42)
	b := NewSparkleField(10, 30, 70, 42)
	if len(a) != 10 || len(b) != 10 {
		t.Fatalf("expected 10 sparkles each, got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("sparkle %d differs between runs with same seed: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestNewSparkleField_AvoidsCenterBand(t *testing.T) {
	rows := 30
	cols := 70
	field := NewSparkleField(60, rows, cols, 7)
	avoidLo := int(float64(rows) * 0.30)
	avoidHi := int(float64(rows) * 0.70)
	for i, s := range field {
		if s.Row > avoidLo && s.Row < avoidHi {
			t.Errorf("sparkle %d at row %d is inside the avoid band (%d, %d)", i, s.Row, avoidLo, avoidHi)
		}
		if s.Col < 0 || s.Col >= cols {
			t.Errorf("sparkle %d col %d out of bounds [0, %d)", i, s.Col, cols)
		}
	}
}

func TestRenderFoxBlock_Layout(t *testing.T) {
	lines, cols := PadFoxLines()
	out := RenderFoxBlock(lines, lipgloss.Color("#ffffff"))
	rendered := strings.Split(out, "\n")
	if len(rendered) != len(lines) {
		t.Fatalf("RenderFoxBlock changed line count: got %d, want %d", len(rendered), len(lines))
	}
	for i, line := range rendered {
		if w := lipgloss.Width(line); w != cols {
			t.Errorf("line %d rendered width = %d, want %d", i, w, cols)
		}
	}
}
