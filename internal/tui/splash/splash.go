package splash

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"golang.org/x/term"
)

// Minimum terminal dimensions before we'll attempt the cinematic. Below
// these, the fox doesn't fit and the layout gets ugly — so we fall back
// to the instant static help.
const (
	minWidth  = 80
	minHeight = 36
)

// Run plays the splash in the alt-screen and returns when it finishes or is
// skipped. Returns true if the splash actually played, false if it was
// suppressed (non-TTY, kill switch set, terminal too small, or program
// errored). Callers should always print the static help to stdout regardless
// of the return value — the splash never writes anything permanent.
func Run() bool {
	if !styles.ShouldAnimate() {
		return false
	}

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < minWidth || h < minHeight {
		return false
	}

	prog := tea.NewProgram(New(w, h), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return false
	}
	return true
}
