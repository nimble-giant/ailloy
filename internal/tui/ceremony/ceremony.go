// Package ceremony provides inline cinematic accents for Ailloy's metallurgy
// commands. Each command gets a themed entrance flourish and a
// squash-and-stretch completion stamp. Designed to *wrap*, never replace,
// the command's existing stdout text — every plain-text line a CI pipeline
// or downstream tool depends on still appears post-animation, byte-identical.
//
// Inline only: no alt-screen takeover, no Bubble Tea program. The animations
// are short (~250–500ms entrance, ~200ms stamp) and use ANSI cursor
// in-place updates so they coexist with terminal scrollback. On non-TTY,
// NO_COLOR, CI, TERM=dumb, or --no-animate, the API degrades to plain text
// instantly.
package ceremony

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Theme bundles a metallurgy command's visual identity. Construction is
// declarative — each command's themes.go entry sets these once.
type Theme struct {
	// Name is a short lowercase identifier ("temper", "assay") used for
	// debugging and the no-animation fallback prefix.
	Name string

	// Verb is the present-progressive verb shown during the working
	// banner and adaptive entrance flourish, e.g. "Tempering",
	// "Assaying". Capitalized.
	Verb string

	// StampWord is the all-caps verb on the completion stamp, e.g.
	// "TEMPERED", "ASSAY COMPLETE".
	StampWord string

	// Glyph prefixes the stamp, e.g. "🔥", "⚗️", "❄️", "📦", "🔨".
	Glyph string

	// Primary is the dominant theme color (hex). Used for the entrance
	// banner, success stamp border/foreground, and strike sparks.
	Primary lipgloss.Color

	// Accent is the warm/secondary theme color (hex). Used for the
	// strike spark and the squash frame highlight on the stamp.
	Accent lipgloss.Color

	// Strikes is the number of entrance "beats" — 1 for a sweep glyph
	// (assay's magnifier), 3 for a hammer (temper), etc. Each beat
	// renders one frame of strikeChar before yielding.
	Strikes int

	// StrikeChar is the glyph rendered on each strike beat, e.g. "✦"
	// for sparks, "⚒" for hammer, "🔍" for magnifier.
	StrikeChar string

	// StrikeIntervalMs is the delay between strike frames. ~110–160ms
	// reads as "rhythm"; faster than 80ms reads as flicker; slower than
	// 200ms feels sluggish.
	StrikeIntervalMs int
}

// Open plays the entrance flourish and prints the working banner. Returns
// immediately on non-animatable terminals after writing the plain-text
// banner. Synchronous; the animation is brief by design.
func Open(t Theme) {
	if !styles.ShouldAnimate() {
		fmt.Println(styles.WorkingBanner(t.Verb + "..."))
		fmt.Println()
		return
	}

	// Print the strike beats in place on a single line, then overwrite
	// with the final working banner. The strikes pre-load the eye for
	// the verb that's about to land.
	strikes := max(t.Strikes, 1)
	interval := t.StrikeIntervalMs
	if interval <= 0 {
		interval = 130
	}

	strikeStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	primaryStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)

	for i := 1; i <= strikes; i++ {
		fmt.Print("\r\033[K")
		fmt.Print(strings.Repeat(strikeStyle.Render(t.StrikeChar)+" ", i))
		fmt.Print(primaryStyle.Render(t.Verb + "..."))
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}

	// Settle: replace the strike line with the canonical WorkingBanner
	// so the rest of the output flows from a familiar starting point.
	fmt.Print("\r\033[K")
	fmt.Println(styles.WorkingBanner(t.Verb + "..."))
	fmt.Println()
}

// Stamp prints the success completion line with a one-frame
// squash-and-stretch entrance, then settles. The summary text is the
// command's existing plain-text conclusion (e.g. "0 errors, 2 warnings");
// it appears verbatim within the stamp. Falls back to plain text when not
// animatable.
//
// Format on a TTY:
//
//	🔥 TEMPERED   — 0 errors, 2 warnings
//
// (One frame of oversized padding/bold, then settles to the regular form.)
func Stamp(t Theme, summary string) {
	stamp := composeStamp(t.Glyph, t.StampWord, summary, t.Primary, t.Accent, false)
	if !styles.ShouldAnimate() {
		fmt.Println(stamp)
		return
	}
	// Squash frame: one over-padded, white-hot variant.
	fmt.Println(composeStamp(t.Glyph, t.StampWord, summary, t.Accent, t.Accent, true))
	time.Sleep(70 * time.Millisecond)
	// Move the cursor up one line and clear it, then print the settled stamp.
	fmt.Print("\033[1A\033[K")
	fmt.Println(stamp)
}

// FailStamp is the error counterpart of Stamp. Uses the warning/error
// palette. The animated frame is dimmer rather than brighter so failure
// reads as deflation, not celebration.
func FailStamp(t Theme, summary string) {
	stamp := composeFailStamp(t.Glyph, t.StampWord, summary)
	if !styles.ShouldAnimate() {
		fmt.Println(stamp)
		return
	}
	// Mute frame: dim then settle.
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#9aa0a6")).Render(t.Glyph + " " + t.StampWord + " FAILED — " + summary)
	fmt.Println(dim)
	time.Sleep(80 * time.Millisecond)
	fmt.Print("\033[1A\033[K")
	fmt.Println(stamp)
}

// composeStamp builds the line string. If `squash` is set, the stamp word
// has bold + extra padding — used as the entrance frame.
func composeStamp(glyph, word, summary string, fg, bgAccent lipgloss.Color, squash bool) string {
	wordStyle := lipgloss.NewStyle().Foreground(fg).Bold(true)
	if squash {
		// One-frame stretch: oversize via padded style for visual pop.
		wordStyle = wordStyle.
			Background(bgAccent).
			Foreground(styles.White).
			Padding(0, 2)
	}
	tail := ""
	if strings.TrimSpace(summary) != "" {
		tail = lipgloss.NewStyle().Foreground(styles.Gray).Render(" — ") + summary
	}
	return glyph + " " + wordStyle.Render(word) + tail
}

func composeFailStamp(glyph, word, summary string) string {
	wordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f44336")).Bold(true)
	tail := ""
	if strings.TrimSpace(summary) != "" {
		tail = lipgloss.NewStyle().Foreground(styles.Gray).Render(" — ") + summary
	}
	return glyph + " " + wordStyle.Render(word+" FAILED") + tail
}
