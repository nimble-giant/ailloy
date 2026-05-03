package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLerpHex_Endpoints(t *testing.T) {
	a := lipgloss.Color("#000000")
	b := lipgloss.Color("#ffffff")
	if got := string(LerpHex(a, b, 0)); got != "#000000" {
		t.Fatalf("t=0: got %s, want #000000", got)
	}
	if got := string(LerpHex(a, b, 1)); got != "#ffffff" {
		t.Fatalf("t=1: got %s, want #ffffff", got)
	}
}

func TestLerpHex_Midpoint(t *testing.T) {
	a := lipgloss.Color("#000000")
	b := lipgloss.Color("#ffffff")
	got := string(LerpHex(a, b, 0.5))
	// Midpoint of 0 and 255 is 127.5, rounded to 128 = 0x80.
	if got != "#808080" {
		t.Fatalf("midpoint: got %s, want #808080", got)
	}
}

func TestLerpHex_ClampedAndInvalid(t *testing.T) {
	a := lipgloss.Color("#112233")
	b := lipgloss.Color("#445566")
	if got := string(LerpHex(a, b, -1)); got != "#112233" {
		t.Fatalf("t<0 should clamp to a: got %s", got)
	}
	if got := string(LerpHex(a, b, 2)); got != "#445566" {
		t.Fatalf("t>1 should clamp to b: got %s", got)
	}
	bad := lipgloss.Color("not-a-color")
	if got := string(LerpHex(bad, b, 0.5)); got != string(bad) {
		t.Fatalf("invalid input should return a: got %s", got)
	}
}

func TestEasings_Endpoints(t *testing.T) {
	for _, fn := range []struct {
		name string
		f    func(float64) float64
	}{
		{"out-cubic", EaseOutCubic},
		{"out-quad", EaseOutQuad},
		{"in-out", EaseInOut},
	} {
		if got := fn.f(0); got != 0 {
			t.Errorf("%s(0) = %v, want 0", fn.name, got)
		}
		if got := fn.f(1); got != 1 {
			t.Errorf("%s(1) = %v, want 1", fn.name, got)
		}
		mid := fn.f(0.5)
		if mid <= 0 || mid >= 1 {
			t.Errorf("%s(0.5) = %v, want strictly between 0 and 1", fn.name, mid)
		}
	}
}

func TestEasings_Clamp(t *testing.T) {
	if got := EaseOutCubic(-0.5); got != 0 {
		t.Errorf("EaseOutCubic(-0.5) should clamp to 0, got %v", got)
	}
	if got := EaseInOut(2.0); got != 1 {
		t.Errorf("EaseInOut(2.0) should clamp to 1, got %v", got)
	}
}
