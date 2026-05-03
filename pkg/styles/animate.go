package styles

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// FlashCycle is one frame of a FlashArt animation: render the art with either
// the primary or the flash style, then wait Delay before the next cycle.
type FlashCycle struct {
	UseFlash bool
	Delay    time.Duration
}

var noAnimate bool

// SetNoAnimate toggles the package-level animation kill switch. Wired up to
// the root command's --no-animate persistent flag.
func SetNoAnimate(v bool) {
	noAnimate = v
}

// ShouldAnimate reports whether terminal animations should run. Animations are
// suppressed when stdout is not a TTY, when standard "be quiet" environment
// variables are set, or when the user passed --no-animate.
func ShouldAnimate() bool {
	if noAnimate {
		return false
	}
	if os.Getenv("AILLOY_NO_ANIMATE") != "" {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}
	if t := os.Getenv("TERM"); t == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Pause sleeps for d when animations are enabled, otherwise no-ops. Lets
// callers sprinkle pauses without re-checking ShouldAnimate everywhere.
func Pause(d time.Duration) {
	if !ShouldAnimate() {
		return
	}
	time.Sleep(d)
}

// FlashArt prints the multi-line art once and then re-renders it in place for
// each cycle, alternating between primary and flashStyle. The cursor is moved
// up and each line is cleared with ANSI escapes, so the art appears to
// "flash" without scrolling. When ShouldAnimate is false, the art is printed
// once with the primary style and the function returns immediately.
func FlashArt(art string, primary, flashStyle lipgloss.Style, cycles []FlashCycle) {
	art = strings.TrimLeft(art, "\n")
	height := strings.Count(art, "\n") + 1

	fmt.Println(primary.Render(art))

	if !ShouldAnimate() {
		return
	}

	for _, c := range cycles {
		time.Sleep(c.Delay)
		fmt.Printf("\033[%dA", height)
		style := primary
		if c.UseFlash {
			style = flashStyle
		}
		for line := range strings.SplitSeq(art, "\n") {
			fmt.Print("\033[K")
			fmt.Println(style.Render(line))
		}
	}
}
