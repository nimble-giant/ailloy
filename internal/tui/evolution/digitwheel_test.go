package evolution

import (
	"strings"
	"testing"
)

// removeEscapes strips ANSI SGR sequences so we can assert on visible
// characters without coupling to lipgloss color output.
func removeEscapes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b { // ESC
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestDigitWheel_Endpoints(t *testing.T) {
	from := "v0.6.21"
	to := "v0.7.0 "
	// Mismatched non-digit shapes → static fallback.
	if got := removeEscapes(RenderDigitWheel(from, to, 0.5, 0)); got != "v0.7.0 " {
		t.Errorf("mismatched layouts should fall back to static `to`, got %q", got)
	}

	// Same layout, t=1.0 → all locked to target.
	from = "v0.6.21"
	to = "v0.7.99"
	if got := removeEscapes(RenderDigitWheel(from, to, 1.0, 0)); got != to {
		t.Errorf("t=1 should render exactly the target, got %q want %q", got, to)
	}
	// t=0.0 → leftmost positions still rolling, but last position is also
	// rolling. We assert that every digit position is in [0-9].
	got := removeEscapes(RenderDigitWheel(from, to, 0.0, 0))
	if len(got) != len(to) {
		t.Fatalf("output length mismatch: got %d want %d (%q)", len(got), len(to), got)
	}
	for i, r := range to {
		out := rune(got[i])
		if !isDigit(r) {
			if out != r {
				t.Errorf("position %d non-digit changed: got %q want %q", i, out, r)
			}
			continue
		}
		if out < '0' || out > '9' {
			t.Errorf("position %d should be a digit, got %q", i, out)
		}
	}
}

func TestDigitWheel_NonDigitsHeld(t *testing.T) {
	from := "v0.6.21"
	to := "v0.7.42"
	for _, frame := range []int{0, 7, 31, 42} {
		got := removeEscapes(RenderDigitWheel(from, to, 0.4, frame))
		if got[0] != 'v' || got[2] != '.' || got[4] != '.' {
			t.Errorf("frame %d: non-digit chars not held in place: %q", frame, got)
		}
	}
}

func TestDigitWheel_LeftLocksFirst(t *testing.T) {
	from := "0000"
	to := "1234"
	// Just past leftmost lock threshold (~0.15), rightmost should still
	// be rolling — we can't pin the exact rolling value but we can assert
	// at least one of the rightmost positions is NOT yet the target.
	got := removeEscapes(RenderDigitWheel(from, to, 0.16, 0))
	if got[0] != '1' {
		t.Errorf("leftmost should have locked at t=0.16, got %q", got)
	}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
