package styles

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LerpHex returns the color t-of-the-way from a to b on a per-channel linear
// interpolation. t is clamped to [0, 1]. Inputs must be `#rrggbb`. Falls back
// to a when parsing fails so callers don't have to error-check colors.
func LerpHex(a, b lipgloss.Color, t float64) lipgloss.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab, ok1 := parseHex(string(a))
	br, bg, bb, ok2 := parseHex(string(b))
	if !ok1 || !ok2 {
		return a
	}
	r := lerpByte(ar, br, t)
	g := lerpByte(ag, bg, t)
	bl := lerpByte(ab, bb, t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

func parseHex(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	// Explicit byte masks before narrowing satisfy gosec's G115
	// (integer-overflow) rule. Each masked value is in [0, 255] by
	// construction, so the uint8 conversion cannot truncate.
	return uint8((v >> 16) & 0xff), uint8((v >> 8) & 0xff), uint8(v & 0xff), true
}

func lerpByte(a, b uint8, t float64) uint8 {
	v := float64(a) + (float64(b)-float64(a))*t
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(math.Round(v))
}

// EaseOutCubic is 1 - (1-t)^3 — fast at the start, soft at the end.
func EaseOutCubic(t float64) float64 {
	t = clamp01(t)
	u := 1 - t
	return 1 - u*u*u
}

// EaseOutQuad is 1 - (1-t)^2 — gentler version of EaseOutCubic.
func EaseOutQuad(t float64) float64 {
	t = clamp01(t)
	u := 1 - t
	return 1 - u*u
}

// EaseInOut is the classic smoothstep curve.
func EaseInOut(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}
